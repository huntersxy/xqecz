package r2

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// MetaMD5 是随对象一起写入的自定义元数据头。
// 对象名可能不是 md5（第三方客户端走随机名兜底），把它随对象存下来，
// 回填任务比对「本地内容 vs 远端对象」时才有可信依据。
const MetaMD5 = "x-amz-meta-md5"

// ErrCorrupted 表示本地文件在读取过程中被改写（只可能由原地压缩引起），
// 调用方应放弃本次上传并留待下一轮重试。
var ErrCorrupted = errors.New("local file changed while reading")

// Config 是客户端配置（由 config.R2Config 映射而来，此处不依赖上层包）。
type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Prefix    string
	Region    string
	Timeout   time.Duration
}

// Client 是 R2 客户端；零值不可用，须经 New 构造。
type Client struct {
	cfg  Config
	http *http.Client
}

// New 构造客户端。Endpoint 缺失即返回错误——调用方应先用 Enabled 判定再构造。
func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("r2: endpoint 未配置")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return nil, errors.New("r2: 凭据或 bucket 未配置")
	}
	if cfg.Region == "" {
		// R2 固定 auto；签名用不到 bucket 位置，取值稳定即可。
		cfg.Region = "auto"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 15 * time.Second}).DialContext,
				TLSHandshakeTimeout: 15 * time.Second,
				MaxIdleConns:        8,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}, nil
}

// ObjectKey 返回桶内对象名：Prefix + 本地相对路径。
// 镜像保持本地 uploads/ 的结构，桶内对象与磁盘文件一一对应，便于对账。
func (c *Client) ObjectKey(rel string) string {
	key := strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
	if c.cfg.Prefix == "" {
		return key
	}
	return c.cfg.Prefix + "/" + key
}

// urlFor 返回虚拟主机风格的对象地址（R2 只支持 virtual-hosted style）。
//
// 对象名拼在 bucket 段之后，**只在两段之间保留一个斜杠**：这里若复用
// canonicalURI（它会给整条路径补前导斜杠），拼出来就是 /bucket//key，
// R2 会把多出来的那个斜杠当成对象名的一部分，对象名变成 "/key" ——
// 私有读写毫无异常，但公开地址（r2.dev / 自定义域名）永远 404。
func (c *Client) urlFor(key string) string {
	base := strings.TrimSuffix(c.cfg.Endpoint, "/")
	path := "/" + c.cfg.Bucket
	if key != "" {
		// canonicalURI 会给路径补前导斜杠（签名侧需要），此处拼对象名要去掉它。
		path += "/" + strings.TrimPrefix(canonicalURI(key), "/")
	}
	return base + path
}

// ObjectMeta 是 HeadObject 的关键字段。
type ObjectMeta struct {
	ETag         string
	ContentLen   int64
	MetaMD5      string
	LastModified time.Time
}

// Head 查询对象元信息；对象不存在时返回 (zero, false, nil)。
func (c *Client) Head(key string) (ObjectMeta, bool, error) {
	req, err := http.NewRequest(http.MethodHead, c.urlFor(key), nil)
	if err != nil {
		return ObjectMeta{}, false, err
	}
	sign(req, c.cfg.AccessKey, c.cfg.SecretKey, c.cfg.Region, "s3", "", nil, time.Now())

	resp, err := c.http.Do(req)
	if err != nil {
		return ObjectMeta{}, false, err
	}
	defer closeBody(resp)

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return ObjectMeta{}, false, nil
	case resp.StatusCode >= 300:
		return ObjectMeta{}, false, apiError("HEAD", key, resp)
	}
	var lastMod time.Time
	if raw := resp.Header.Get("Last-Modified"); raw != "" {
		if t, perr := http.ParseTime(raw); perr == nil {
			lastMod = t
		}
	}
	return ObjectMeta{
		ETag:         strings.Trim(resp.Header.Get("ETag"), "\""),
		ContentLen:   resp.ContentLength,
		MetaMD5:      resp.Header.Get(MetaMD5),
		LastModified: lastMod,
	}, true, nil
}

// Put 上传对象到指定 key（key 由调用方给全，便于归档到 original/ 等其它命名空间）。
//
// 为兼顾「内容寻址去重」与「不被原地压缩打断」，这里只读文件一次：
// 同一遍读出内容算 md5，再作为请求体发出；请求体与签名哈希共用这份内容。
// 上传前后各 stat 一次，若大小/修改时间变了说明压缩任务正在原地覆盖同一个文件，
// 此时放弃本次上传（不回写任何状态），下一轮回填会自动重传。
func (c *Client) Put(key, absPath, contentType string) error {
	before, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	body, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	after, err := os.Stat(absPath)
	if err != nil {
		return err
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return ErrCorrupted
	}

	sum := md5.Sum(body)
	req, err := http.NewRequest(http.MethodPut, c.urlFor(key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", contentType)
	extra := map[string]string{MetaMD5: hex.EncodeToString(sum[:])}
	sign(req, c.cfg.AccessKey, c.cfg.SecretKey, c.cfg.Region, "s3", hexSHA256(body), extra, time.Now())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp)
	if resp.StatusCode >= 300 {
		return apiError("PUT", key, resp)
	}
	return nil
}

// Delete 删除对象（对象不存在也视为成功，与 S3 语义一致）。
//
// 生产链路目前刻意不做删除：内容是「本地进垃圾桶、R2 留档」的 append-only 策略。
// 这里提供最小实现，供集成校验与将来可能的清理任务使用。
func (c *Client) Delete(key string) error {
	req, err := http.NewRequest(http.MethodDelete, c.urlFor(key), nil)
	if err != nil {
		return err
	}
	sign(req, c.cfg.AccessKey, c.cfg.SecretKey, c.cfg.Region, "s3", "", nil, time.Now())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer closeBody(resp)
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return apiError("DELETE", key, resp)
	}
	return nil
}

// apiError 把非 2xx 响应整理为带上下文的错误；S3 的错误体是 XML。
func apiError(op, key string, resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, readLimit(resp.ContentLength)))
	var payload struct {
		Code    string `xml:"Code"`
		Message string `xml:"Message"`
	}
	_ = xml.Unmarshal(raw, &payload)
	detail := strings.TrimSpace(payload.Code + " " + payload.Message)
	if detail == "" {
		detail = strings.TrimSpace(string(raw))
	}
	if len(detail) > 200 {
		detail = detail[:200]
	}
	return fmt.Errorf("r2 %s %s: status=%d %s", op, key, resp.StatusCode, detail)
}

// readLimit 把（可能为 -1 的）ContentLength 归一为可读上限。
func readLimit(n int64) int64 {
	if n <= 0 || n > 1<<20 {
		return 1 << 20
	}
	return n
}

func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
	}
}

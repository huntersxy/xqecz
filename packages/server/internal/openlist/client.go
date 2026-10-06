// Package openlist 是 OpenList 的最小写入/探测客户端。
//
// 与 internal/r2 的区别在于协议本身：R2 是 S3（SigV4 签名、桶语义、HEAD 带
// ETag/md5），OpenList 是它自己的 HTTP API（admin token 鉴权、挂载路径语义、
// 没有远端哈希）。二者被 internal/mirror 以 objstore.Store 同一个接口消费。
//
// 三条必须知道的协议事实：
//
//  1. **HTTP 状态码恒为 200**，成败写在响应体的 code 字段里（401 也是 200 + code:401）。
//     因此这里绝不能用 resp.StatusCode 判成败，必须解 body。
//  2. **鉴权头是 `Authorization: <token>`，不带 `Bearer`**。带 Bearer 会走到
//     jwt 分支报 `token is invalidated`；完全不带头则报 `Guest user is disabled`。
//  3. **上传路径由 `File-Path` 头指定**（不是 URL），PUT 到 `/api/fs/put`，
//     服务端会自动创建缺失的父目录；覆盖写同一路径是幂等 200。
package openlist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/objstore"
)

// Config 是 OpenList 目标的连接参数。
type Config struct {
	// Endpoint 是 OpenList 的基址，如 http://127.0.0.1:5244（不要带尾斜杠）。
	Endpoint string
	// Token 是 admin token（data.db 的 x_setting_items.key='token'），
	// 值形如 openlist-<uuid>…。唯一写入凭据，缺失即整体停用。
	Token string
	// Prefix 是挂载内的统一前缀，与 R2 保持同一套 key 结构（默认 uploads）。
	Prefix string
	// Timeout 是单次请求的超时；≤0 时回落 2 分钟。
	Timeout time.Duration
}

// Client 是 OpenList 的 HTTP 客户端。
type Client struct {
	cfg  Config
	http *http.Client
}

// New 构造客户端。基址或 token 缺失时返回错误（调用方据此停用该目标）。
func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("openlist: endpoint 未配置")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, errors.New("openlist: token 未配置")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &Client{
		cfg: Config{
			Endpoint: strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/"),
			Token:    strings.TrimSpace(cfg.Token),
			Prefix:   strings.TrimSpace(cfg.Prefix),
			Timeout:  timeout,
		},
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        8,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 15 * time.Second,
			},
		},
	}, nil
}

// ObjectKey 拼接挂载内的对象路径，与 r2.Client.ObjectKey 同一规则。
func (c *Client) ObjectKey(rel string) string {
	key := strings.TrimPrefix(strings.TrimSpace(rel), "/")
	if c.cfg.Prefix == "" {
		return key
	}
	return c.cfg.Prefix + "/" + key
}

// fsPath 把对象 key 转成 OpenList 的绝对挂载路径（挂载点必须落在 `/`）。
func fsPath(key string) string { return "/" + strings.TrimPrefix(key, "/") }

// Put 上传一份本地文件，覆盖写同路径是幂等的。
//
// 不做全量读内存（R2 需要签名载荷才那样做）：这里把文件流式交给 net/http，
// 显式设置 ContentLength，避免退化成 chunked 编码。
func (c *Client) Put(key, absPath, contentType string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	defer f.Close()

	before, err := f.Stat()
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	if before.IsDir() {
		return fmt.Errorf("openlist put %s: %s 是目录", key, absPath)
	}

	// 请求体必须包一层 io.NopCloser，不能把 *os.File 直接交出去。
	// net/http 在请求结束后一定会 Close 请求体（见 net/http/request.go：
	// 非 ReadCloser 的 body 会被自动包成 NopCloser），若直接传 *os.File，
	// Transport 会把它关掉，后面那次 f.Stat() 就只能拿到
	// `file already closed`，上传明明成功却被记成失败、每轮全量重传。
	// 关句柄的责任因此回到这里的 defer f.Close()。
	req, err := http.NewRequest(http.MethodPut, c.cfg.Endpoint+"/api/fs/put", io.NopCloser(f))
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	req.ContentLength = before.Size()
	req.Header.Set("Authorization", c.cfg.Token)
	// File-Path 会被 OpenList **解码**（实测：送 %E4%B8%AD… 落盘为 «中文…»），
	// 但 /api/fs/get 的 body path 不会。故这里送转义形态（纯 ASCII 头，
	// 对中间反代最安全），探测那边送原始形态。EscapedPath 会保留 `/`
	// 只转义各段内的字符——用 url.PathEscape 会把 `/` 也编成 %2F。
	req.Header.Set("File-Path", (&url.URL{Path: fsPath(key)}).EscapedPath())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/octet-stream")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	body, apiErr := decode(resp)
	closeBody(resp)
	if apiErr == nil && resp.StatusCode >= 300 {
		apiErr = fmt.Errorf("openlist put %s: status=%d", key, resp.StatusCode)
	}
	if apiErr != nil {
		return apiErr
	}
	_ = body

	// 上传期间文件被改写：内容已发出去但不再是本地当前那份，本轮作废。
	// 按路径 stat（与 internal/r2/client.go 一致）而不是复用 f：句柄指向的是
	// 打开那一刻的 inode，若压缩任务用「写临时文件再改名覆盖」的方式替换，
	// f.Stat() 仍是旧 inode 的元数据，会漏判。
	after, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return objstore.ErrCorrupted
	}
	return nil
}

// Head 探测远端是否已有该对象，返回其元数据。
//
// OpenList 的 Local 驱动不提供任何远端哈希（hashinfo 恒为 "null"），
// 故 MetaMD5 / ETag 恒为空，只有 ContentLen 可用。
// 对象不存在时返回 (零值, false, nil)。
func (c *Client) Head(key string) (objstore.ObjectMeta, bool, error) {
	payload, err := json.Marshal(map[string]string{"path": fsPath(key)})
	if err != nil {
		return objstore.ObjectMeta{}, false, fmt.Errorf("openlist head %s: %w", key, err)
	}
	req, err := http.NewRequest(http.MethodPost, c.cfg.Endpoint+"/api/fs/get", bytes.NewReader(payload))
	if err != nil {
		return objstore.ObjectMeta{}, false, fmt.Errorf("openlist head %s: %w", key, err)
	}
	req.Header.Set("Authorization", c.cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return objstore.ObjectMeta{}, false, fmt.Errorf("openlist head %s: %w", key, err)
	}
	raw, apiErr := decode(resp)
	closeBody(resp)
	if apiErr != nil {
		// 文件/目录不存在：OpenList 用 code=500 + "…object not found" 表达，
		// 这不是错误，是「远端还没有这一份」。
		if resp.StatusCode == http.StatusNotFound || strings.Contains(strings.ToLower(message(raw)), "not found") {
			return objstore.ObjectMeta{}, false, nil
		}
		return objstore.ObjectMeta{}, false, apiErr
	}

	// 真实响应把文件元数据放在 data 层：成功是
	// `{"code":200,"data":{"size":813433,"modified":"2026-…+08:00","hashinfo":"null"}}`，
	// 失败是 `{"code":500,"message":"…not found","data":null}`。
	// 按扁平解会把 size 解成 0，而 size 是 OpenList 侧唯一的幂等判据
	// （Local 驱动 hashinfo 恒为 "null"，没有 md5），0 会让每轮回填都把
	// 全量媒体重推一遍。
	var metaResp struct {
		Data struct {
			Size int64 `json:"size"`
			// Modified 带纳秒（2026-10-06T13:27:37.302477504+08:00），
			// time.RFC3339 能吃下；解析失败留零值即可（不参与判定）。
			Modified string `json:"modified"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &metaResp); err != nil {
		return objstore.ObjectMeta{}, false, fmt.Errorf("openlist head %s: %w", key, err)
	}
	meta := objstore.ObjectMeta{ContentLen: metaResp.Data.Size}
	if t, err := time.Parse(time.RFC3339, metaResp.Data.Modified); err == nil {
		meta.LastModified = t
	}
	return meta, true, nil
}

// apiError 是 OpenList 响应体的错误形状。
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e apiError) Error() string {
	return fmt.Sprintf("openlist: code=%d %s", e.Code, e.Message)
}

// decode 读取响应体并判定业务成败。返回原始 body 与错误（成功时为 nil）。
// 注意 HTTP 状态码只作兜底——OpenList 的业务失败同样回 200。
func decode(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("openlist: 读取响应失败: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw, nil
	}
	var ae apiError
	if err := json.Unmarshal(raw, &ae); err != nil {
		return raw, nil // 非 JSON 响应（反代错误页等）交由状态码判断
	}
	if ae.Code != 0 && ae.Code != 200 {
		return raw, ae
	}
	return raw, nil
}

// message 从原始 body 里取 message 字段，解析失败返回空串。
func message(raw []byte) string {
	var ae apiError
	if err := json.Unmarshal(raw, &ae); err != nil {
		return ""
	}
	return ae.Message
}

func closeBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}

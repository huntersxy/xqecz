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

	"github.com/huntersxy/xqecz/server/internal/objstore"
)

const MetaMD5 = "x-amz-meta-md5"

type ObjectMeta = objstore.ObjectMeta

var ErrCorrupted = objstore.ErrCorrupted

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Prefix    string
	Region    string
	Timeout   time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("r2: endpoint 未配置")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" {
		return nil, errors.New("r2: 凭据或 bucket 未配置")
	}
	if cfg.Region == "" {
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

func (c *Client) ObjectKey(rel string) string {
	key := strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
	if c.cfg.Prefix == "" {
		return key
	}
	return c.cfg.Prefix + "/" + key
}

func (c *Client) urlFor(key string) string {
	base := strings.TrimSuffix(c.cfg.Endpoint, "/")
	path := "/" + c.cfg.Bucket
	if key != "" {
		path += "/" + strings.TrimPrefix(canonicalURI(key), "/")
	}
	return base + path
}

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

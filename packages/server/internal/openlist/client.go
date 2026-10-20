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

type Config struct {
	Endpoint string
	Token    string
	Prefix   string
	Timeout  time.Duration
}

type Client struct {
	cfg  Config
	http *http.Client
}

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

func (c *Client) ObjectKey(rel string) string {
	key := strings.TrimPrefix(strings.TrimSpace(rel), "/")
	if c.cfg.Prefix == "" {
		return key
	}
	return c.cfg.Prefix + "/" + key
}

func fsPath(key string) string { return "/" + strings.TrimPrefix(key, "/") }

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

	req, err := http.NewRequest(http.MethodPut, c.cfg.Endpoint+"/api/fs/put", io.NopCloser(f))
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	req.ContentLength = before.Size()
	req.Header.Set("Authorization", c.cfg.Token)
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

	after, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("openlist put %s: %w", key, err)
	}
	if after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return objstore.ErrCorrupted
	}
	return nil
}

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
		if resp.StatusCode == http.StatusNotFound || strings.Contains(strings.ToLower(message(raw)), "not found") {
			return objstore.ObjectMeta{}, false, nil
		}
		return objstore.ObjectMeta{}, false, apiErr
	}

	var metaResp struct {
		Data struct {
			Size     int64  `json:"size"`
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

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e apiError) Error() string {
	return fmt.Sprintf("openlist: code=%d %s", e.Code, e.Message)
}

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
		return raw, nil
	}
	if ae.Code != 0 && ae.Code != 200 {
		return raw, ae
	}
	return raw, nil
}

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

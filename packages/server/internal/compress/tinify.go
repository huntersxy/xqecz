// Package compress 实现 TinyPNG 后台压缩：每轮挑一张最大的待压缩图片，压缩后就地替换，
// 原图移入垃圾桶目录，并在数据库标记 compressed_at 以免重复压缩。
package compress

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// tinifyEndpoint 是 TinyPNG 的压缩入口；上传原图后从响应中取压缩结果的下载地址。
const tinifyEndpoint = "https://api.tinify.com/shrink"

// Client 是 TinyPNG API 的最小客户端（仅用得上 shrink + 下载结果两步）。
type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http:   &http.Client{Timeout: 5 * time.Minute},
	}
}

type shrinkResponse struct {
	Output struct {
		URL   string `json:"url"`
		Size  int64  `json:"size"`
		Width int    `json:"width"`
	} `json:"output"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Shrink 上传 srcPath 并返回压缩结果，调用方负责消费 reader 后关闭。
// 失败时返回的错误包含 TinyPNG 的原始 message，便于排查配额或格式问题。
func (c *Client) Shrink(ctx context.Context, srcPath string) (io.ReadCloser, int64, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tinifyEndpoint, f)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth("api", c.apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	if st, err := f.Stat(); err == nil {
		req.ContentLength = st.Size()
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		var sr shrinkResponse
		_ = json.Unmarshal(body, &sr)
		if sr.Message != "" {
			return nil, 0, fmt.Errorf("tinify %d: %s", resp.StatusCode, sr.Message)
		}
		return nil, 0, fmt.Errorf("tinify %d: %s", resp.StatusCode, string(body))
	}

	var sr shrinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, 0, fmt.Errorf("decode tinify response: %w", err)
	}
	if sr.Output.URL == "" {
		return nil, 0, fmt.Errorf("tinify response has no output url")
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, sr.Output.URL, nil)
	if err != nil {
		return nil, 0, err
	}
	dlReq.SetBasicAuth("api", c.apiKey)
	dlResp, err := c.http.Do(dlReq)
	if err != nil {
		return nil, 0, err
	}
	if dlResp.StatusCode != http.StatusOK {
		_ = dlResp.Body.Close()
		return nil, 0, fmt.Errorf("download compressed %d", dlResp.StatusCode)
	}
	return dlResp.Body, sr.Output.Size, nil
}

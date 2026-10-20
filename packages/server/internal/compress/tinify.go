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

const tinifyEndpoint = "https://api.tinify.com/shrink"

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

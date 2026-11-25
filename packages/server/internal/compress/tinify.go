package compress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const tinifyEndpoint = "https://api.tinify.com/shrink"

var errUnsupportedImage = errors.New("tinify 415: unsupported image")

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		http: &http.Client{
			Timeout: 5 * time.Minute,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type shrinkResponse struct {
	Output struct {
		URL   string `json:"url"`
		Size  int64  `json:"size"`
		Width int    `json:"width"`
	} `json:"output"`
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

	if resp.StatusCode == http.StatusUnsupportedMediaType {
		return nil, 0, errUnsupportedImage
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("tinify %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var sr shrinkResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, 0, fmt.Errorf("decode tinify response: %w", err)
	}
	outputURL, err := url.Parse(sr.Output.URL)
	if err != nil || outputURL.Scheme != "https" || outputURL.Host != "api.tinify.com" ||
		outputURL.User != nil || !strings.HasPrefix(outputURL.Path, "/output/") {
		return nil, 0, errors.New("tinify response has invalid output url")
	}

	dlReq, err := http.NewRequestWithContext(ctx, http.MethodGet, outputURL.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	dlReq.SetBasicAuth("api", c.apiKey)
	dlResp, err := c.http.Do(dlReq)
	if err != nil {
		return nil, 0, fmt.Errorf("download compressed image: %w", errors.Unwrap(err))
	}
	if dlResp.StatusCode != http.StatusOK {
		_ = dlResp.Body.Close()
		return nil, 0, fmt.Errorf("download compressed %d", dlResp.StatusCode)
	}
	return dlResp.Body, sr.Output.Size, nil
}

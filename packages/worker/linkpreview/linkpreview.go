package linkpreview

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Result 是链接预览结果。
type Result struct {
	Title    string
	Image    string
	Platform string
}

var (
	reMetaProp = regexp.MustCompile(`(?is)<meta[^>]+(?:property|name)=["']([^"']+)["'][^>]*content=["']([^"']*)["']`)
	reMetaRev  = regexp.MustCompile(`(?is)<meta[^>]+content=["']([^"']*)["'][^>]*(?:property|name)=["']([^"']+)["']`)
	reTitle    = regexp.MustCompile(`(?is)<title[^>]*>([\s\S]*?)</title>`)

	// httpClient 带连接池复用的专用 HTTP 客户端，避免 http.DefaultClient 的短连接开销。
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

const ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"

// Fetch 抓取目标 URL 的 OG / Twitter Card 元数据。
// 任何失败（超时、非 HTML、解析无果）都返回 ok=false，调用方保留用户原值。
func Fetch(raw string) (r Result, ok bool) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Result{}, false
	}
	host := parsed.Hostname()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return Result{}, false
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, false
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" || !strings.Contains(ct, "html") {
		return Result{}, false
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return Result{}, false
	}
	html := string(body)

	title := meta(html, "og:title")
	if title == "" {
		title = meta(html, "twitter:title")
	}
	if title == "" {
		if m := reTitle.FindStringSubmatch(html); m != nil {
			title = strings.TrimSpace(m[1])
		}
	}
	image := meta(html, "og:image")
	if image == "" {
		image = meta(html, "og:image:url")
	}
	if image == "" {
		image = meta(html, "twitter:image")
	}

	if title == "" && image == "" {
		return Result{}, false
	}
	if title == "" {
		title = raw
	}
	return Result{Title: title, Image: image, Platform: detectPlatform(host)}, true
}

// meta 同时匹配 property/name 在前或在后的两种 meta 写法。
// 遍历全部匹配（页面上可能有多个 <meta>），返回第一个 property/name 命中目标的 content。
func meta(html, prop string) string {
	if ms := reMetaProp.FindAllStringSubmatch(html, -1); ms != nil {
		for _, m := range ms {
			if equalProp(m[1], prop) {
				return strings.TrimSpace(m[2])
			}
		}
	}
	if ms := reMetaRev.FindAllStringSubmatch(html, -1); ms != nil {
		for _, m := range ms {
			if equalProp(m[2], prop) {
				return strings.TrimSpace(m[1])
			}
		}
	}
	return ""
}

func equalProp(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func detectPlatform(hostname string) string {
	h := strings.ToLower(hostname)
	switch {
	case strings.Contains(h, "bilibili.com"):
		return "bilibili"
	case strings.Contains(h, "youtube.com"), strings.Contains(h, "youtu.be"):
		return "youtube"
	case strings.Contains(h, "twitter.com"), strings.Contains(h, "x.com"):
		return "twitter"
	case strings.Contains(h, "douyin.com"), strings.Contains(h, "tiktok.com"):
		return "douyin"
	case strings.Contains(h, "weibo.com"):
		return "weibo"
	default:
		return "generic"
	}
}

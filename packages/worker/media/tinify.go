package media

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// basicAuth 构造 Tinify 需要的 "Basic base64(api:<key>)" 头值。
func basicAuth(apiKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("api:"+apiKey))
}

// TinifyCompress 调用 Tinify API 压缩图片，并输出 webp 到 imagesDir。
//
// 入参：
//   - absPath 原始图片绝对路径（共享卷）
//   - imagesDir 压缩图输出目录（data/images，与 uploads 同级）
//   - apiKey    TINIFY_API_KEY
//
// 返回：相对 data 目录的压缩图路径（如 "images/xxx_tinified.webp"），供 api 映射为 /images/ URL。
// apiKey 为空时返回错误（由上层决定降级）。
func TinifyCompress(absPath, imagesDir, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("tinify API key not configured")
	}
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return "", fmt.Errorf("create images dir: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	auth := "Basic " + basicAuth(apiKey)

	f, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	// 1) 上传原图到 Tinify，拿到输出 URL（流式，不把文件全部读入内存）。
	req, _ := http.NewRequest(http.MethodPost, "https://api.tinify.com/shrink", f)
	req.Header.Set("Authorization", auth)
	req.ContentLength = fi.Size()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tinify upload failed: HTTP %d %s", resp.StatusCode, string(body))
	}
	outputURL := resp.Header.Get("Location")
	if outputURL == "" {
		return "", fmt.Errorf("tinify: missing Location header")
	}

	// 2) 下载压缩结果，并请求转换为 webp。
	stem := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	outName := stem + "_tinified.webp"
	outPath := filepath.Join(imagesDir, outName)

	dl, _ := http.NewRequest(http.MethodPost, outputURL, strings.NewReader(`{"convert": {"type": "image/webp"}}`))
	dl.Header.Set("Authorization", auth)
	dl.Header.Set("Content-Type", "application/json")
	dlResp, err := client.Do(dl)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(dlResp.Body)
		return "", fmt.Errorf("tinify download failed: HTTP %d %s", dlResp.StatusCode, string(body))
	}
	outBody, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, outBody, 0o644); err != nil {
		return "", err
	}

	// 返回相对 UPLOAD_DIR 的路径。
	rel, relErr := filepath.Rel(filepath.Dir(imagesDir), outPath)
	if relErr != nil {
		return "images/" + outName, nil
	}
	return filepath.ToSlash(rel), nil
}

package aigen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Image struct{ Bytes []byte }
type Provider interface {
	Generate(context.Context, ProviderConfig, string, Input, []string) ([]Image, error)
}
type KnownFailure struct{ Message string }

func (e KnownFailure) Error() string { return e.Message }

type OpenAIProvider struct{ Client *http.Client }

func (p OpenAIProvider) Generate(ctx context.Context, cfg ProviderConfig, key string, input Input, files []string) ([]Image, error) {
	var referenceBytes int64
	for _, path := range files {
		stat, err := os.Stat(path)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() > 20<<20 {
			return nil, KnownFailure{"参考图不可读取或过大"}
		}
		referenceBytes += stat.Size()
		if referenceBytes > 32<<20 {
			return nil, KnownFailure{"参考图总大小过大"}
		}
		if !contains([]string{".png", ".jpg", ".webp"}, filepath.Ext(path)) {
			return nil, KnownFailure{"参考图格式无效"}
		}
	}
	for index, ref := range input.References {
		if strings.TrimSpace(ref.Note) != "" {
			input.Prompt += fmt.Sprintf("\nReference %d note: %s", index+1, ref.Note)
		}
	}
	client := p.Client
	if client == nil {
		client = safeHTTPClient(time.Duration(cfg.TimeoutSeconds) * time.Second)
		defer client.CloseIdleConnections()
	}
	endpoint, contentType := cfg.Endpoint, "application/json"
	var body io.Reader
	if len(files) != 0 && cfg.EditTransport == "multipart" {
		endpoint = strings.TrimSuffix(endpoint, "/generations") + "/edits"
		reader, writer := io.Pipe()
		defer reader.Close()
		form := multipart.NewWriter(writer)
		contentType = form.FormDataContentType()
		body = reader
		go func() {
			write := func() error {
				for field, value := range map[string]string{"model": cfg.Model, "prompt": input.Prompt, "size": input.Size, "n": strconv.Itoa(input.Count)} {
					if err := form.WriteField(field, value); err != nil {
						return err
					}
				}
				if input.Quality != "" {
					if err := form.WriteField("quality", input.Quality); err != nil {
						return err
					}
				}
				for _, path := range files {
					f, err := os.Open(path)
					if err != nil {
						return err
					}
					header := make(textproto.MIMEHeader)
					header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "image[]", "filename": "reference" + filepath.Ext(path)}))
					mimeType := "image/png"
					if filepath.Ext(path) == ".jpg" {
						mimeType = "image/jpeg"
					} else if filepath.Ext(path) == ".webp" {
						mimeType = "image/webp"
					}
					header.Set("Content-Type", mimeType)
					part, err := form.CreatePart(header)
					if err != nil {
						f.Close()
						return err
					}
					_, err = io.Copy(part, f)
					f.Close()
					if err != nil {
						return err
					}
				}
				return form.Close()
			}
			writer.CloseWithError(write())
		}()
	} else {
		payload := map[string]any{"model": cfg.Model, "prompt": input.Prompt, "size": input.Size, "n": input.Count}
		if input.Quality != "" {
			payload["quality"] = input.Quality
		}
		if len(files) != 0 {
			if cfg.EditTransport != "json" {
				return nil, KnownFailure{"供应商未开启参考图能力"}
			}
			endpoint = strings.TrimSuffix(endpoint, "/generations") + "/edits"
			images := []map[string]string{}
			for _, path := range files {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, KnownFailure{"参考图不可读取"}
				}
				images = append(images, map[string]string{"image_url": "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)})
			}
			payload["images"] = images
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, KnownFailure{"生成参数无法编码"}
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return nil, KnownFailure{"生成地址无效"}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", contentType)
	var dispatched atomic.Bool
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &httptrace.ClientTrace{GotConn: func(httptrace.GotConnInfo) { dispatched.Store(true) }}))
	resp, err := client.Do(req)
	if err != nil {
		if !dispatched.Load() {
			return nil, KnownFailure{"连接供应商失败，未发送生成请求"}
		}
		return nil, errors.New("供应商请求结果不明")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 408 && resp.StatusCode != 429 {
			return nil, KnownFailure{fmt.Sprintf("供应商拒绝请求（HTTP %d）", resp.StatusCode)}
		}
		return nil, fmt.Errorf("供应商结果不明（HTTP %d）", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (56<<20)+1))
	if err != nil || len(raw) > 56<<20 {
		return nil, errors.New("供应商响应过大或不完整")
	}
	var wire struct {
		Data []struct {
			B64 string `json:"b64_json"`
			URL string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &wire) != nil || len(wire.Data) == 0 || len(wire.Data) > 2 {
		return nil, errors.New("供应商响应格式不明确")
	}
	images := make([]Image, 0, len(wire.Data))
	for _, item := range wire.Data {
		var decoded []byte
		if item.B64 != "" {
			decoded, err = io.ReadAll(io.LimitReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(item.B64)), (20<<20)+1))
		} else {
			u, parseErr := url.Parse(item.URL)
			if parseErr != nil {
				return nil, errors.New("供应商图片 URL 无效")
			}
			// Signed query parameters are allowed on results, never credentials or redirects.
			check := *u
			check.RawQuery = ""
			if validateURL(check.String()) != nil {
				return nil, errors.New("供应商图片 URL 不安全")
			}
			fetch, reqErr := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			if reqErr != nil {
				return nil, reqErr
			}
			// Use a separate protected client: never forward the provider API key to a result host.
			resultClient := safeHTTPClient(45 * time.Second)
			resultResp, fetchErr := resultClient.Do(fetch)
			if fetchErr != nil {
				resultClient.CloseIdleConnections()
				return nil, errors.New("供应商图片下载失败")
			}
			if resultResp.StatusCode == 200 {
				decoded, err = io.ReadAll(io.LimitReader(resultResp.Body, (20<<20)+1))
			} else {
				err = errors.New("供应商图片下载失败")
			}
			resultResp.Body.Close()
			resultClient.CloseIdleConnections()
		}
		if err != nil || len(decoded) == 0 || len(decoded) > 20<<20 {
			return images, errors.New("供应商图片不完整或过大")
		}
		images = append(images, Image{Bytes: decoded})
	}
	if len(images) != input.Count {
		return images, errors.New("供应商返回了部分结果")
	}
	return images, nil
}

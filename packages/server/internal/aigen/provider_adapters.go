package aigen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProviderRouter struct{ Client *http.Client }

func adapterOptions(cfg ProviderConfig) map[string]any {
	if strings.TrimSpace(cfg.Options) == "" {
		return nil
	}
	var options map[string]any
	if json.Unmarshal([]byte(cfg.Options), &options) != nil {
		return nil
	}
	return options
}

func applyImageOptions(payload map[string]any, cfg ProviderConfig) {
	options := adapterOptions(cfg)
	if len(options) == 0 {
		return
	}
	for _, key := range []string{"watermark", "optimize_prompt", "sequential_image_generation", "negative_prompt", "num_inference_steps", "guidance_scale", "output_format"} {
		if value, ok := options[key]; ok {
			switch value.(type) {
			case string, bool, float64:
				payload[key] = value
			}
		}
	}
}

func (p ProviderRouter) Generate(ctx context.Context, cfg ProviderConfig, key string, input Input, files []string) ([]Image, error) {
	switch normalizedAdapter(cfg.Adapter) {
	case "openai_chat":
		return OpenAIChatProvider{Client: p.Client}.Generate(ctx, cfg, key, input, files)
	case "gemini":
		return GeminiProvider{Client: p.Client}.Generate(ctx, cfg, key, input, files)
	case "codex_responses":
		return CodexResponsesProvider{Client: p.Client}.Generate(ctx, cfg, key, input, files)
	default:
		return OpenAIProvider{Client: p.Client}.Generate(ctx, cfg, key, input, files)
	}
}

func validateReferenceFiles(files []string) error {
	var total int64
	for _, path := range files {
		stat, err := os.Stat(path)
		if err != nil || !stat.Mode().IsRegular() || stat.Size() > 20<<20 {
			return KnownFailure{"参考图不可读取或过大"}
		}
		total += stat.Size()
		if total > 32<<20 {
			return KnownFailure{"参考图总大小过大"}
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
			return KnownFailure{"参考图格式无效"}
		}
	}
	return nil
}

func adapterPrompt(input Input) string {
	prompt := input.Prompt
	for index, ref := range input.References {
		if strings.TrimSpace(ref.Note) != "" {
			prompt += fmt.Sprintf("\nReference %d note: %s", index+1, ref.Note)
		}
	}
	return prompt
}

func adapterClient(client *http.Client, timeout int) (*http.Client, func()) {
	if client != nil {
		return client, func() {}
	}
	c := safeHTTPClient(time.Duration(timeout) * time.Second)
	return c, c.CloseIdleConnections
}

func readProviderImage(ctx context.Context, value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("供应商图片为空")
	}
	if strings.HasPrefix(value, "data:") {
		comma := strings.IndexByte(value, ',')
		if comma < 0 {
			return nil, errors.New("供应商图片 data URL 无效")
		}
		decoded, err := base64.StdEncoding.DecodeString(value[comma+1:])
		if err != nil {
			return nil, errors.New("供应商图片编码无效")
		}
		return limitImageBytes(decoded)
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return limitImageBytes(decoded)
	}
	u, err := url.Parse(value)
	if err != nil {
		return nil, errors.New("供应商图片 URL 无效")
	}
	check := *u
	check.RawQuery = ""
	if validateURL(check.String()) != nil {
		return nil, errors.New("供应商图片 URL 不安全")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("供应商图片 URL 无效")
	}
	client := safeHTTPClient(45 * time.Second)
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("供应商图片下载失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("供应商图片下载失败")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, (20<<20)+1))
	if err != nil {
		return nil, errors.New("供应商图片下载失败")
	}
	return limitImageBytes(b)
}

func limitImageBytes(b []byte) ([]byte, error) {
	if len(b) == 0 || len(b) > 20<<20 {
		return nil, errors.New("供应商图片不完整或过大")
	}
	return b, nil
}

func collectImageValues(value any, imageContext bool, out *[]string) {
	switch v := value.(type) {
	case map[string]any:
		isImage := imageContext
		if kind, ok := v["type"].(string); ok && (strings.Contains(kind, "image") || kind == "inline_data" || kind == "file_data") {
			isImage = true
		}
		for key, item := range v {
			switch key {
			case "content":
				if s, ok := item.(string); ok && strings.HasPrefix(strings.TrimSpace(s), "data:image") {
					*out = append(*out, s)
				} else {
					collectImageValues(item, isImage, out)
				}
			case "b64_json":
				if s, ok := item.(string); ok && s != "" {
					*out = append(*out, s)
				}
			case "url":
				if s, ok := item.(string); ok && (isImage || strings.HasPrefix(s, "data:image")) {
					*out = append(*out, s)
				}
			case "data", "result":
				if s, ok := item.(string); ok && isImage && s != "" {
					*out = append(*out, s)
				} else {
					collectImageValues(item, isImage, out)
				}
			case "image_url", "image", "images", "inlineData", "inline_data", "fileData", "file_data", "source":
				collectImageValues(item, true, out)
			default:
				collectImageValues(item, isImage, out)
			}
		}
	case []any:
		for _, item := range v {
			collectImageValues(item, imageContext, out)
		}
	}
}

func decodeAdapterResponse(ctx context.Context, raw []byte, count int) ([]Image, error) {
	var wire any
	if json.Unmarshal(raw, &wire) != nil {
		return nil, errors.New("供应商响应格式不明确")
	}
	values := []string{}
	collectImageValues(wire, false, &values)
	if len(values) == 0 || len(values) > 2 || len(values) < count {
		return nil, errors.New("供应商响应格式不明确")
	}
	images := make([]Image, 0, len(values))
	for _, value := range values {
		b, err := readProviderImage(ctx, value)
		if err != nil {
			return images, err
		}
		images = append(images, Image{Bytes: b})
	}
	if len(images) != count {
		return images, errors.New("供应商返回了部分结果")
	}
	return images, nil
}

type OpenAIChatProvider struct{ Client *http.Client }

func (p OpenAIChatProvider) Generate(ctx context.Context, cfg ProviderConfig, key string, input Input, files []string) ([]Image, error) {
	if err := validateReferenceFiles(files); err != nil {
		return nil, err
	}
	parts := []any{map[string]any{"type": "text", "text": adapterPrompt(input)}}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, KnownFailure{"参考图不可读取"}
		}
		parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]string{"url": "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)}})
	}
	message := map[string]any{"role": "user", "content": parts}
	payload := map[string]any{"model": cfg.Model, "messages": []any{message}, "n": input.Count, "modalities": []string{"text", "image"}}
	if input.Size != "" && input.Size != "auto" {
		payload["size"] = input.Size
	}
	if input.Quality != "" {
		payload["quality"] = input.Quality
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, KnownFailure{"生成参数无法编码"}
	}
	client, closeClient := adapterClient(p.Client, cfg.TimeoutSeconds)
	defer closeClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, KnownFailure{"生成地址无效"}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("供应商请求结果不明：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 408 && resp.StatusCode != 429 {
			return nil, KnownFailure{fmt.Sprintf("供应商拒绝请求（HTTP %d）", resp.StatusCode)}
		}
		return nil, &ProviderResponseError{StatusCode: resp.StatusCode, Snippet: readResponseSnippet(resp.Body)}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (56<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("供应商响应过大或不完整：%w", err)
	}
	if len(raw) > 56<<20 {
		return nil, errors.New("供应商响应过大或不完整")
	}
	return decodeAdapterResponse(ctx, raw, input.Count)
}

type GeminiProvider struct{ Client *http.Client }

func (p GeminiProvider) Generate(ctx context.Context, cfg ProviderConfig, key string, input Input, files []string) ([]Image, error) {
	if err := validateReferenceFiles(files); err != nil {
		return nil, err
	}
	parts := []any{map[string]any{"text": adapterPrompt(input)}}
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, KnownFailure{"参考图不可读取"}
		}
		parts = append(parts, map[string]any{"inlineData": map[string]string{"mimeType": http.DetectContentType(data), "data": base64.StdEncoding.EncodeToString(data)}})
	}
	generationConfig := map[string]any{"responseModalities": []string{"TEXT", "IMAGE"}}
	if width, height, ok := imageSize(input.Size); ok {
		generationConfig["imageConfig"] = map[string]any{"aspectRatio": geminiAspectRatio(width, height), "imageSize": geminiImageSize(width, height)}
	}
	payload := map[string]any{"contents": []any{map[string]any{"role": "user", "parts": parts}}, "generationConfig": generationConfig}
	if input.Count > 1 {
		return nil, KnownFailure{"Gemini 当前不支持单次返回多张图片，请将数量设为 1"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, KnownFailure{"生成参数无法编码"}
	}
	client, closeClient := adapterClient(p.Client, cfg.TimeoutSeconds)
	defer closeClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, KnownFailure{"生成地址无效"}
	}
	req.Header.Set("x-goog-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("供应商请求结果不明：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 408 && resp.StatusCode != 429 {
			return nil, KnownFailure{fmt.Sprintf("供应商拒绝请求（HTTP %d）", resp.StatusCode)}
		}
		return nil, &ProviderResponseError{StatusCode: resp.StatusCode, Snippet: readResponseSnippet(resp.Body)}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (56<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("供应商响应过大或不完整：%w", err)
	}
	if len(raw) > 56<<20 {
		return nil, errors.New("供应商响应过大或不完整")
	}
	return decodeAdapterResponse(ctx, raw, input.Count)
}

func imageSize(size string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(size), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	width, errW := strconv.Atoi(parts[0])
	height, errH := strconv.Atoi(parts[1])
	return width, height, errW == nil && errH == nil && width > 0 && height > 0
}

func geminiAspectRatio(width, height int) string {
	ratio := float64(width) / float64(height)
	best, distance := "1:1", 1e9
	for _, candidate := range []struct {
		name  string
		value float64
	}{
		{"1:1", 1}, {"2:3", 2.0 / 3}, {"3:2", 1.5}, {"3:4", .75}, {"4:3", 4.0 / 3}, {"9:16", 9.0 / 16}, {"16:9", 16.0 / 9}, {"21:9", 21.0 / 9},
	} {
		if d := ratio - candidate.value; d < distance && d > -distance {
			best, distance = candidate.name, absFloat(d)
		}
	}
	return best
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func geminiImageSize(width, height int) string {
	if width >= 2048 || height >= 2048 {
		return "2K"
	}
	return "1K"
}

type CodexResponsesProvider struct{ Client *http.Client }

func (p CodexResponsesProvider) Generate(ctx context.Context, cfg ProviderConfig, key string, input Input, files []string) ([]Image, error) {
	if err := validateReferenceFiles(files); err != nil {
		return nil, err
	}
	if input.Count != 1 {
		return nil, KnownFailure{"Codex Responses 当前每次只能返回 1 张图片"}
	}
	var requestInput any = adapterPrompt(input)
	if len(files) > 0 {
		parts := []any{map[string]any{"type": "input_text", "text": adapterPrompt(input)}}
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, KnownFailure{"参考图不可读取"}
			}
			parts = append(parts, map[string]any{"type": "input_image", "image_url": "data:" + http.DetectContentType(data) + ";base64," + base64.StdEncoding.EncodeToString(data)})
		}
		requestInput = []any{map[string]any{"role": "user", "content": parts}}
	}
	outputFormat := "png"
	if value, ok := adapterOptions(cfg)["output_format"].(string); ok && (value == "png" || value == "jpeg" || value == "webp") {
		outputFormat = value
	}
	payload := map[string]any{
		"model": cfg.Model,
		"input": requestInput,
		"tools": []any{map[string]any{"type": "image_generation", "output_format": outputFormat}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, KnownFailure{"生成参数无法编码"}
	}
	client, closeClient := adapterClient(p.Client, cfg.TimeoutSeconds)
	defer closeClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Endpoint, strings.NewReader(string(body)))
	if err != nil {
		return nil, KnownFailure{"生成地址无效"}
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("供应商请求结果不明：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 408 && resp.StatusCode != 429 {
			return nil, KnownFailure{fmt.Sprintf("供应商拒绝请求（HTTP %d）", resp.StatusCode)}
		}
		return nil, &ProviderResponseError{StatusCode: resp.StatusCode, Snippet: readResponseSnippet(resp.Body)}
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (56<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("供应商响应过大或不完整：%w", err)
	}
	if len(raw) > 56<<20 {
		return nil, errors.New("供应商响应过大或不完整")
	}
	return decodeAdapterResponse(ctx, raw, 1)
}

package aigen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIChatAdapterWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer chat-key" {
			t.Fatal("missing chat auth")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "chat-image" || body["modalities"].([]any)[1] != "image" {
			t.Fatalf("unexpected body: %#v", body)
		}
		if _, ok := body["size"]; ok {
			t.Fatal("automatic size must be omitted from chat requests")
		}
		url := "data:image/png;base64," + base64.StdEncoding.EncodeToString(testPNG())
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": []any{map[string]any{"type": "image_url", "image_url": map[string]string{"url": url}}}}}}})
	}))
	defer server.Close()
	images, err := (OpenAIChatProvider{Client: server.Client()}).Generate(context.Background(), ProviderConfig{Adapter: "openai_chat", Endpoint: server.URL + "/v1/chat/completions", Model: "chat-image", TimeoutSeconds: 10}, "chat-key", Input{Prompt: "a pink circle", Size: "auto", Count: 1}, nil)
	if err != nil || len(images) != 1 || len(images[0].Bytes) == 0 {
		t.Fatalf("images=%d err=%v", len(images), err)
	}
}

func TestGeminiAdapterWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "gemini-key" {
			t.Fatal("missing gemini auth")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(mustJSON(body["generationConfig"])), "IMAGE") {
			t.Fatalf("missing image modality: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"inlineData": map[string]string{"mimeType": "image/png", "data": base64.StdEncoding.EncodeToString(testPNG())}}}}}}})
	}))
	defer server.Close()
	images, err := (GeminiProvider{Client: server.Client()}).Generate(context.Background(), ProviderConfig{Adapter: "gemini", Endpoint: server.URL + "/v1beta/models/gemini-2.5-flash-image:generateContent", Model: "gemini-2.5-flash-image", TimeoutSeconds: 10}, "gemini-key", Input{Prompt: "a white canvas", Size: "auto", Count: 1}, nil)
	if err != nil || len(images) != 1 || len(images[0].Bytes) == 0 {
		t.Fatalf("images=%d err=%v", len(images), err)
	}
}

func TestCodexResponsesAdapterWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer codex-key" {
			t.Fatal("missing codex auth")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "gpt-image-1" || body["input"] != "draw a pink circle" {
			t.Fatalf("unexpected body: %#v", body)
		}
		url := base64.StdEncoding.EncodeToString(testPNG())
		_ = json.NewEncoder(w).Encode(map[string]any{"output": []any{map[string]any{"type": "image_generation_call", "result": url}}})
	}))
	defer server.Close()
	images, err := (CodexResponsesProvider{Client: server.Client()}).Generate(context.Background(), ProviderConfig{Adapter: "codex_responses", Endpoint: server.URL + "/v1/responses", Model: "gpt-image-1", TimeoutSeconds: 10}, "codex-key", Input{Prompt: "draw a pink circle", Count: 1}, nil)
	if err != nil || len(images) != 1 || len(images[0].Bytes) == 0 {
		t.Fatalf("images=%d err=%v", len(images), err)
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestProviderAdapterValidation(t *testing.T) {
	base := ProviderConfig{Name: "test", Model: "image", Enabled: true, Sizes: `[]`, Qualities: `[]`, MaxCount: 1, EditTransport: "none", TimeoutSeconds: 10, Concurrency: 1, Cost: 1, QualityCosts: `{}`, Options: `{}`}
	tests := []struct {
		name     string
		adapter  string
		endpoint string
		wantErr  bool
	}{
		{"legacy images", "", "https://example.com/v1/images/generations", false},
		{"chat", "openai_chat", "https://example.com/v1/chat/completions", false},
		{"gemini", "gemini", "https://example.com/v1beta/models/image:generateContent", false},
		{"bad suffix", "openai_chat", "https://example.com/v1/images/generations", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := base
			p.Adapter, p.Endpoint = tt.adapter, tt.endpoint
			if got := validateProvider(p); (got != nil) != tt.wantErr {
				t.Fatalf("validateProvider() error=%v wantErr=%v", got, tt.wantErr)
			}
		})
	}
}

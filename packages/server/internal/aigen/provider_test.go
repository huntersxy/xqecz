package aigen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenAIAdapterWireAndEditing(t *testing.T) {
	for _, transport := range []string{"none", "multipart", "json"} {
		t.Run(transport, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-only-key" {
					t.Fatal("missing auth")
				}
				if transport == "multipart" {
					if r.URL.Path != "/v1/images/edits" {
						t.Fatal(r.URL.Path)
					}
					if err := r.ParseMultipartForm(1 << 20); err != nil {
						t.Fatal(err)
					}
					if len(r.MultipartForm.File["image[]"]) != 1 || r.FormValue("model") != "gpt-image-2.5-sunburst-1K" {
						t.Fatal(r.MultipartForm)
					}
					if !strings.Contains(r.FormValue("prompt"), "@图片1 对应按顺序上传的第1张参考图") {
						t.Fatal("missing ordered image labels")
					}
				} else {
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					if body["n"] != float64(1) || body["size"] != "1024x1024" {
						t.Fatal(body)
					}
					if transport == "none" && len(body) != 4 {
						t.Fatal(body)
					}
					if transport == "json" && body["images"] == nil {
						t.Fatal(body)
					}
					if transport == "json" && !strings.Contains(body["prompt"].(string), "@图片1 对应按顺序上传的第1张参考图") {
						t.Fatal("missing ordered image labels")
					}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]string{"b64_json": base64.StdEncoding.EncodeToString(testPNG())}}})
			}))
			defer server.Close()
			cfg := ProviderConfig{Endpoint: server.URL + "/v1/images/generations", Model: "gpt-image-2.5-sunburst-1K", EditTransport: transport, TimeoutSeconds: 10}
			input := Input{Prompt: "circle", Size: "1024x1024", Count: 1}
			files := []string{}
			if transport != "none" {
				input.Prompt = "使用 @图片1"
				path := filepath.Join(t.TempDir(), "reference.png")
				_ = os.WriteFile(path, testPNG(), 0o600)
				files = append(files, path)
			}
			images, err := (OpenAIProvider{Client: server.Client()}).Generate(context.Background(), cfg, "test-only-key", input, files)
			if err != nil || len(images) != 1 {
				t.Fatal(err)
			}
		})
	}
}
func TestProviderFailureClassificationAndSecretRedaction(t *testing.T) {
	for _, status := range []int{400, 401, 429, 500, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "sensitive-key prompt echo")
			}))
			defer server.Close()
			_, err := (OpenAIProvider{Client: server.Client()}).Generate(context.Background(), ProviderConfig{Endpoint: server.URL + "/v1/images/generations", TimeoutSeconds: 10}, "sensitive-key", Input{Prompt: "private prompt", Count: 1}, nil)
			if err == nil || strings.Contains(err.Error(), "sensitive-key") || strings.Contains(err.Error(), "private prompt") {
				t.Fatal(err)
			}
			var known KnownFailure
			if errors.As(err, &known) != (status == 400 || status == 401) {
				t.Fatal(err)
			}
		})
	}
}
func TestNetworkBoundary(t *testing.T) {
	for _, ip := range []string{"127.0.0.1", "10.1.2.3", "169.254.169.254", "100.64.0.1", "192.168.1.1", "::1", "fc00::1", "::ffff:127.0.0.1", "198.18.0.1"} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatal(ip)
		}
	}
	if !publicIP(net.ParseIP("1.1.1.1")) {
		t.Fatal("public IP denied")
	}
	for _, address := range []string{"http://example.com/v1/images/generations", "https://127.0.0.1/v1/images/generations", "https://user:pass@example.com/v1/images/generations", "https://example.com:8080/v1/images/generations"} {
		if validateURL(address) == nil {
			t.Fatal(address)
		}
	}
	client := safeHTTPClient(0)
	if _, err := client.Get("https://localhost/"); err == nil {
		t.Fatal("local DNS accepted")
	}
	client.CloseIdleConnections()
}

// Opt-in live validation uses the explicitly supplied provider and real key.
// No credentials enter source, logs, or the production database.
func TestLiveProvider(t *testing.T) {
	key := os.Getenv("XQECZ_AI_LIVE_KEY")
	if key == "" {
		t.Skip("live provider key not supplied")
	}
	endpoint, model := os.Getenv("XQECZ_AI_LIVE_ENDPOINT"), os.Getenv("XQECZ_AI_LIVE_MODEL")
	if validateURL(endpoint) != nil {
		t.Fatal("invalid live endpoint")
	}
	adapter := OpenAIProvider{}
	if pinned := os.Getenv("XQECZ_AI_LIVE_IP"); pinned != "" {
		if !publicIP(net.ParseIP(pinned)) {
			t.Fatal("pinned test address must be public")
		}
		u, _ := url.Parse(endpoint)
		transport := &http.Transport{TLSHandshakeTimeout: 10 * time.Second}
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil || host != u.Hostname() {
				return nil, errors.New("unexpected live test host")
			}
			return (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, net.JoinHostPort(pinned, port))
		}
		adapter.Client = &http.Client{Transport: transport, Timeout: 180 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		defer adapter.Client.CloseIdleConnections()
	}
	images, err := adapter.Generate(context.Background(), ProviderConfig{Endpoint: endpoint, Model: model, TimeoutSeconds: 180}, key, Input{Prompt: "A simple pink circle on a white background, minimal flat illustration.", Size: "1024x1024", Count: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatal("result missing")
	}
	s, _, _, _ := testService(t)
	asset, err := s.saveImage(strings.NewReader(string(images[0].Bytes)), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("real provider result persisted: %s %dx%d %d bytes", asset.Mime, asset.Width, asset.Height, asset.Bytes)
}

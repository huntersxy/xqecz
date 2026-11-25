package avatar

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
)

func TestSignedAvatarPathContainsOnlyTheSubjectAndSignature(t *testing.T) {
	proxy := New("https://api.example.com/", strings.Repeat("65", 32), nil)
	address := "reader@example.com"
	avatarURL := proxy.URL(User, 32, &address)
	parsed, err := url.Parse(avatarURL)
	if err != nil || parsed.Host != "api.example.com" || parsed.RawQuery != "" ||
		!strings.HasPrefix(parsed.Path, "/api/avatars/") || strings.Contains(avatarURL, address) {
		t.Fatalf("unsafe avatar URL: %v", err)
	}
	token := path.Base(parsed.Path)
	subject, id, valid := proxy.Verify(token)
	if !valid || subject != User || id != 32 {
		t.Fatalf("signed subject=%d id=%d valid=%t", subject, id, valid)
	}
	if _, _, valid := proxy.Verify("Z" + token[1:]); valid {
		t.Fatal("accepted a modified token")
	}
	if _, _, valid := proxy.Verify(path.Base(proxy.URL(Guest, 32, &address))); !valid {
		t.Fatal("guest avatar reference failed")
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestProxyCachesPixelsWithoutReturningUpstreamMetadata(t *testing.T) {
	redis := miniredis.RunT(t)
	var cfg config.Config
	cfg.Redis.Host = redis.Host()
	cfg.Redis.Port, _ = strconv.Atoi(redis.Port())
	client := cache.Open(cfg)
	t.Cleanup(func() { _ = client.Raw().Close() })
	proxy := New("", strings.Repeat("18", 32), client)
	canvas := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	canvas.SetNRGBA(3, 4, color.NRGBA{R: 123, G: 67, B: 22, A: 255})
	var upstream bytes.Buffer
	if err := png.Encode(&upstream, canvas); err != nil {
		t.Fatal(err)
	}
	metadata := "provider fixture private metadata"
	upstream.WriteString(metadata)
	fetches := 0
	proxy.client.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		fetches++
		if req.URL.Host != "q.qlogo.cn" || req.URL.Scheme != "https" || req.Header.Get("Cookie") != "" {
			t.Fatal("unexpected avatar upstream request")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(upstream.Bytes()))}, nil
	})
	email := " " + "1000012345" + "@QQ.COM "
	for range 2 {
		body, err := proxy.Image(t.Context(), email)
		if err != nil || bytes.Contains(body, []byte(metadata)) {
			t.Fatalf("invalid proxy output: %v", err)
		}
		decoded, err := png.Decode(bytes.NewReader(body))
		if err != nil || decoded.At(3, 4) != canvas.At(3, 4) {
			t.Fatalf("avatar changed: %v", err)
		}
	}
	if fetches != 1 {
		t.Fatalf("avatar fetched %d times", fetches)
	}
	redis.FastForward(3 * time.Hour)
	proxy.client.Transport = transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: http.NoBody}, nil
	})
	if body, err := proxy.Image(t.Context(), email); err != nil || len(body) == 0 {
		t.Fatalf("last known avatar unavailable: %v", err)
	}
}

func TestProxyRejectsRedirectAndNonImage(t *testing.T) {
	proxy := New("", strings.Repeat("34", 32), nil)
	for _, status := range []int{http.StatusFound, http.StatusOK} {
		proxy.client.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host != "www.gravatar.com" {
				t.Fatal("followed an upstream redirect")
			}
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"Location": {"https://elsewhere.example.com/"}},
				Body:       io.NopCloser(strings.NewReader("<html>missing avatar</html>")),
			}, nil
		})
		if _, err := proxy.Image(t.Context(), "avatar@example.com"); err == nil {
			t.Fatalf("accepted unexpected image response with status %d", status)
		}
	}
}

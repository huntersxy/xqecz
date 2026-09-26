package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 通配规则的边界用例：一旦放行过宽，任何拿到通配域子域的人都能带凭据读接口。
func TestOriginPolicyWildcard(t *testing.T) {
	cases := []struct {
		name   string
		config []string
		origin string
		want   bool
	}{
		// 精确条目：行为与改动前保持一致
		{"exact match", []string{"https://xq.xiey.work"}, "https://xq.xiey.work", true},
		{"exact mismatch", []string{"https://xq.xiey.work"}, "https://evil.com", false},
		{"exact does not match different port", []string{"https://xq.xiey.work"}, "https://xq.xiey.work:8443", false},

		// 通配：正常放行
		{"wildcard subdomain", []string{"*.edgeone.cool"}, "https://a.edgeone.cool", true},
		{"wildcard nested subdomain", []string{"*.edgeone.cool"}, "https://a.b.edgeone.cool", true},
		{"wildcard case-insensitive", []string{"*.edgeone.cool"}, "https://A.EdgeOne.Cool", true},
		{"wildcard ignores port", []string{"*.edgeone.cool"}, "https://a.edgeone.cool:8443", true},
		{"wildcard any scheme", []string{"*.edgeone.cool"}, "http://a.edgeone.cool", true},

		// 通配：必须拒绝的边界
		{"wildcard does not cover bare domain", []string{"*.edgeone.cool"}, "https://edgeone.cool", false},
		{"wildcard requires dot boundary", []string{"*.edgeone.cool"}, "https://notedgeone.cool", false},
		{"wildcard rejects suffix injection in userinfo", []string{"*.edgeone.cool"},
			"https://a.edgeone.cool@evil.com", false},
		{"wildcard rejects other tld", []string{"*.edgeone.cool"}, "https://a.edgeone.com", false},

		// 写了 scheme 就必须一致
		{"scheme pinned allows", []string{"https://*.edgeone.cool"}, "https://a.edgeone.cool", true},
		{"scheme pinned rejects http", []string{"https://*.edgeone.cool"}, "http://a.edgeone.cool", false},

		// 混合配置与退化输入。注意 config.Load() 已按逗号拆好，这里传的是拆分后的条目。
		{"mixed exact and wildcard", []string{" https://xq.xiey.work ", " *.edgeone.cool "},
			"https://xq.xiey.work", true},
		{"mixed exact keeps other host denied", []string{" https://xq.xiey.work ", " *.edgeone.cool "},
			"https://evil.com", false},
		{"mixed ignores other host", []string{"https://xq.xiey.work", "*.edgeone.cool"}, "https://other.com", false},
		{"empty config denies", nil, "https://a.edgeone.cool", false},
		{"empty origin denies", []string{"*.edgeone.cool"}, "", false},
		{"blank entries skipped", []string{"", "  ", "*.edgeone.cool"}, "https://a.edgeone.cool", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := newOriginPolicy(tc.config).allows(tc.origin); got != tc.want {
				t.Fatalf("allows(%q) = %v, want %v (config %q)", tc.origin, got, tc.want, tc.config)
			}
		})
	}
}

// 处理器层：放行时必须回显具体 origin 并带 credentials；拒绝时不能出现任何 ACAO。
func TestCORSHandlerHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := CORS([]string{"https://xq.xiey.work", "*.edgeone.cool"})

	cases := []struct {
		name, origin, wantACAO string
		wantCreds              bool
	}{
		{"allowed by exact", "https://xq.xiey.work", "https://xq.xiey.work", true},
		{"allowed by wildcard", "https://preview.edgeone.cool", "https://preview.edgeone.cool", true},
		{"denied other host", "https://evil.com", "", false},
		{"denied bare wildcard domain", "https://edgeone.cool", "", false},
		{"no origin header", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/api/health", nil)
			if tc.origin != "" {
				c.Request.Header.Set("Origin", tc.origin)
			}
			h(c)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tc.wantACAO {
				t.Fatalf("ACAO = %q, want %q", got, tc.wantACAO)
			}
			creds := w.Header().Get("Access-Control-Allow-Credentials") == "true"
			if creds != tc.wantCreds {
				t.Fatalf("credentials = %v, want %v", creds, tc.wantCreds)
			}
			if tc.wantACAO == "" && w.Header().Get("Vary") != "" {
				t.Fatalf("denied origin should not set CORS headers, got Vary=%q", w.Header().Get("Vary"))
			}
		})
	}
}

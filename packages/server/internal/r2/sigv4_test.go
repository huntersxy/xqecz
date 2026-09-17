package r2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"
)

// fixedTime 固定签名时刻，保证测试可复现。
func fixedTime() time.Time { return time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC) }

// expectedSignature 用**独立实现**的规范化流程算出应有的签名。
//
// 刻意不复用 sign() 内部的 canonicalURI/canonicalQuery/collapseSpaces：
// 测试若与被测代码共用同一段逻辑，写错的地方会两边一起错，测了等于没测。
func expectedSignature(t *testing.T, method, rawURL, payloadHash string, headers map[string]string, extra map[string]string, accessKey, secretKey string) (string, string) {
	t.Helper()
	if payloadHash == "" {
		payloadHash = emptyPayloadHash
	}
	req, err := http.NewRequest(method, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	now := fixedTime()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")

	all := map[string]string{"host": req.URL.Host}
	for k, v := range headers {
		all[strings.ToLower(k)] = v
	}
	for k, v := range extra {
		all[strings.ToLower(k)] = v
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)

	var ch strings.Builder
	for _, n := range names {
		ch.WriteString(n + ":" + strings.Join(strings.Fields(all[n]), " ") + "\n")
	}
	signed := strings.Join(names, ";")

	// 独立编码路径：逐字节百分号编码（RFC 3986 未保留字符之外的都转义），
	// 与 canonicalURI 的编码语义一致但实现互不共享。
	path := ""
	for i, seg := range strings.Split(req.URL.Path, "/") {
		if i > 0 {
			path += "/"
		}
		path += encodeSegment(seg)
	}
	if path == "" {
		path = "/"
	}
	canonical := strings.Join([]string{method, path, req.URL.RawQuery, ch.String(), signed, payloadHash}, "\n")

	sum := sha256.Sum256([]byte(canonical))
	scope := dateStamp + "/us-east-1/service/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + hex.EncodeToString(sum[:])

	k := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	k = hmacSHA256(k, []byte("us-east-1"))
	k = hmacSHA256(k, []byte("service"))
	k = hmacSHA256(k, []byte("aws4_request"))
	return hex.EncodeToString(hmacSHA256(k, []byte(toSign))), scope
}

// TestURLForKeepsSingleSlashBetweenBucketAndKey 钉死对象地址的拼接形状。
//
// 曾经的 bug：key 也走 canonicalURI（会给整条路径补前导斜杠），拼出 /bucket//key，
// R2 于是把多余斜杠算进对象名（真实对象名变成 "/key"）：私有读写全对，
// 公开地址却永远 404 —— 只有断言 URL 本身才能拦住这类「签名没错但地址错」的问题。
func TestURLForKeepsSingleSlashBetweenBucketAndKey(t *testing.T) {
	c, err := New(Config{Endpoint: "https://acc.r2.cloudflarestorage.com", AccessKey: "ak", SecretKey: "sk", Bucket: "xqecz"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ key, want string }{
		{"uploads/ab12.webp", "https://acc.r2.cloudflarestorage.com/xqecz/uploads/ab12.webp"},
		{"uploads/original/ab12.webp", "https://acc.r2.cloudflarestorage.com/xqecz/uploads/original/ab12.webp"},
		{"/uploads/ab12.webp", "https://acc.r2.cloudflarestorage.com/xqecz/uploads/ab12.webp"},
		{"中文 名.webp", "https://acc.r2.cloudflarestorage.com/xqecz/%E4%B8%AD%E6%96%87%20%E5%90%8D.webp"},
	}
	for _, tc := range cases {
		if got := c.urlFor(tc.key); got != tc.want {
			t.Errorf("urlFor(%q) = %q，期望 %q", tc.key, got, tc.want)
		}
	}
}

// TestEmptyPayloadHashMatchesSpec 空 body 的 SHA-256 是公开常量，先钉死它。
func TestEmptyPayloadHashMatchesSpec(t *testing.T) {
	sum := sha256.Sum256(nil)
	if got := hex.EncodeToString(sum[:]); got != emptyPayloadHash {
		t.Fatalf("emptyPayloadHash 常量有误: %s != %s", emptyPayloadHash, got)
	}
}

// TestSignStructuralCorrectness 用独立实现逐项校验签名结构：
// 规范化请求、待签串、签名密钥派生与 Authorization 头的组成。
func TestSignStructuralCorrectness(t *testing.T) {
	const (
		accessKey = "AKIDEXAMPLE"
		secretKey = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	)
	cases := []struct {
		name    string
		method  string
		url     string
		payload string
		extra   map[string]string
	}{
		{name: "HEAD 无 body", method: http.MethodHead, url: "https://acc.r2.cloudflarestorage.com/bucket/uploads/d41d8cd98f00b204e9800998ecf8427e.webp"},
		{name: "PUT 带元数据", method: http.MethodPut, url: "https://acc.r2.cloudflarestorage.com/bucket/uploads/a.webp", payload: "deadbeef", extra: map[string]string{MetaMD5: "0cc175b9c0f1b6a831c399e269772661"}},
		{name: "中文对象名", method: http.MethodPut, url: "https://acc.r2.cloudflarestorage.com/bucket/uploads/中文 名.webp", payload: "x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.payload != "" {
				req.Header.Set("Content-Type", "image/webp")
			}
			payloadHash := ""
			if tc.payload != "" {
				payloadHash = hexSHA256([]byte(tc.payload))
			}
			// headers 参与签名的部分与生产实现保持一致：host + x-amz-* + content-type。
			signedPayloadHash := payloadHash
			if signedPayloadHash == "" {
				signedPayloadHash = emptyPayloadHash
			}
			hdr := map[string]string{"x-amz-content-sha256": signedPayloadHash, "x-amz-date": fixedTime().Format("20060102T150405Z")}
			if tc.payload != "" {
				hdr["content-type"] = "image/webp"
			}
			sign(req, accessKey, secretKey, "us-east-1", "service", payloadHash, tc.extra, fixedTime())

			wantExtra := map[string]string{}
			for k, v := range tc.extra {
				wantExtra[k] = v
			}
			wantSig, wantScope := expectedSignature(t, tc.method, tc.url, signedPayloadHash, hdr, wantExtra, accessKey, secretKey)

			auth := req.Header.Get("Authorization")
			if !strings.Contains(auth, "Signature="+wantSig) {
				t.Fatalf("签名不符\n got: %s\nwant signature: %s", auth, wantSig)
			}
			if !strings.Contains(auth, "Credential="+accessKey+"/"+wantScope) {
				t.Fatalf("Credential scope 不符: %s", auth)
			}
			if got := req.Header.Get("x-amz-date"); got != fixedTime().Format("20060102T150405Z") {
				t.Fatalf("x-amz-date 不符: %s", got)
			}
			if got := req.Header.Get("x-amz-content-sha256"); got != signedPayloadHash {
				t.Fatalf("payload hash 头不符: %s", got)
			}
			if tc.extra != nil {
				for k, v := range tc.extra {
					if got := req.Header.Get(k); got != v {
						t.Fatalf("元数据头 %s 未设置: %q", k, got)
					}
					if !strings.Contains(auth, strings.ToLower(k)) {
						t.Fatalf("元数据头 %s 未参与签名: %s", k, auth)
					}
				}
			}
		})
	}
}

// TestSignDeterministic 同一时刻同一请求必须得到同一签名（否则无法重放/复现）。
func TestSignDeterministic(t *testing.T) {
	mk := func() string {
		req, _ := http.NewRequest(http.MethodHead, "https://acc.r2.cloudflarestorage.com/bucket/a.webp", nil)
		sign(req, "ak", "sk", "auto", "s3", "", nil, fixedTime())
		return req.Header.Get("Authorization")
	}
	if a, b := mk(), mk(); a != b {
		t.Fatalf("签名不稳定:\n%s\n%s", a, b)
	}
}

// encodeSegment 是测试侧独立实现的 RFC 3986 分段编码（与 canonicalURI 实现互不共享）。
func encodeSegment(s string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_.~"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(unreserved, s[i]) >= 0 {
			b.WriteByte(s[i])
			continue
		}
		b.WriteString(fmt.Sprintf("%%%02X", s[i]))
	}
	return b.String()
}

// TestObjectKey 校验对象名拼接（前缀 + 相对路径，反斜杠归一）。
func TestObjectKey(t *testing.T) {
	c := &Client{cfg: Config{Prefix: "uploads"}}
	if got := c.ObjectKey("ab12.webp"); got != "uploads/ab12.webp" {
		t.Fatalf("ObjectKey = %q", got)
	}
	if got := c.ObjectKey("/ab12.webp"); got != "uploads/ab12.webp" {
		t.Fatalf("前导斜杠未归一: %q", got)
	}
	backslash := string(rune(92))
	if got := c.ObjectKey("sub" + backslash + "a.webp"); got != "uploads/sub/a.webp" {
		t.Fatalf("反斜杠未归一: %q", got)
	}
	empty := &Client{cfg: Config{}}
	if got := empty.ObjectKey("a.webp"); got != "a.webp" {
		t.Fatalf("空前缀应直接用相对路径: %q", got)
	}
}

// TestCanonicalURI 校验路径编码：保留分隔斜杠、转义特殊字符。
func TestCanonicalURI(t *testing.T) {
	cases := map[string]string{
		"/uploads/a b.webp":    "/uploads/a%20b.webp",
		"/uploads/中文.webp":     "/uploads/%E4%B8%AD%E6%96%87.webp",
		"/bucket//double.webp": "/bucket//double.webp",
		"":                     "/",
		"/a~b_c-d.e":           "/a~b_c-d.e",
	}
	for in, want := range cases {
		if got := canonicalURI(in); got != want {
			t.Errorf("canonicalURI(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestNewValidatesConfig 校验构造期的参数校验与默认值。
func TestNewValidatesConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("缺 endpoint 应报错")
	}
	if _, err := New(Config{Endpoint: "https://x"}); err == nil {
		t.Fatal("缺凭据应报错")
	}
	c, err := New(Config{Endpoint: "https://x", AccessKey: "a", SecretKey: "b", Bucket: "c"})
	if err != nil {
		t.Fatal(err)
	}
	if c.cfg.Region != "auto" {
		t.Fatalf("region 应默认为 auto，得到 %q", c.cfg.Region)
	}
}

// Package qiniu 封装七牛云的**管理凭证**（MAC 签名）与额度统计接口。
//
// 与 internal/r2 的 SigV4 是两套东西：S3 兼容接口（推送对象）走 AWS SigV4，
// 统计/管理接口（api.qiniuapi.com）走七牛自研的 HMAC-SHA1 凭证。
// 两者共用同一对 AK/SK，但签名串完全不同，不可混用。
package qiniu

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"sort"
	"strings"
)

// Signer 用七牛管理凭证为出站请求签名。
// 零值不可用，须由配置构造。
type Signer struct {
	AccessKey string
	SecretKey string
}

// Enabled 表示签名器是否具备完整凭据。
func (s Signer) Enabled() bool { return s.AccessKey != "" && s.SecretKey != "" }

// Authorize 就地为 req 加上 Authorization 头（值形如 "Qiniu <AK>:<签名>"）。
// body 是请求体原文；GET 一律传 nil。未配置 Content-Type 时不会被写入签名串。
func (s Signer) Authorize(req *http.Request, body []byte) {
	req.Header.Set("Authorization", "Qiniu "+s.token(req, body))
}

// token 计算管理凭证：`<AccessKey>:<urlsafe_base64(hmac_sha1(signingStr, SK))>`。
//
// 签名串构成（见 developer.qiniu.com/kodo/1201/access-token）：
//
//	Method SP Path [ "?" query ]
//	\nHost: <host>
//	[\nContent-Type: <ct>]            —— 仅当设置了该头
//	[\n<规范化后的 X-Qiniu-* 头>]...   —— 按 key 的 ASCII 序
//	\n\n
//	[<body>]                          —— 仅有体且 Content-Type 非 octet-stream 时
func (s Signer) token(req *http.Request, body []byte) string {
	var b strings.Builder
	b.WriteString(req.Method)
	b.WriteByte(' ')
	b.WriteString(req.URL.EscapedPath())
	if q := req.URL.RawQuery; q != "" {
		b.WriteByte('?')
		b.WriteString(q)
	}
	b.WriteString("\nHost: ")
	b.WriteString(req.URL.Host)

	if ct := req.Header.Get("Content-Type"); ct != "" {
		b.WriteString("\nContent-Type: ")
		b.WriteString(ct)
	}
	for _, line := range xqiniuHeaderLines(req) {
		b.WriteString("\n")
		b.WriteString(line)
	}
	b.WriteString("\n\n")

	// 有体且 Content-Type 非 octet-stream 时，体本身也进签名串。
	if len(body) > 0 && req.Header.Get("Content-Type") != "application/octet-stream" {
		b.Write(body)
	}

	mac := hmac.New(sha1.New, []byte(s.SecretKey))
	mac.Write([]byte(b.String()))
	return s.AccessKey + ":" + base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// xqiniuHeaderLines 把 X-Qiniu-* 头整理为 "规范化key: value" 行并按 key 排序。
// 规范化规则：每段（以 - 分隔）首字母大写、其余小写，如 x-qiniu-date → X-Qiniu-Date。
func xqiniuHeaderLines(req *http.Request) []string {
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		// 注意 "X-Qiniu-" 是 8 个字符：按 7 截取会得到 "X-Qiniu"，永远比不中。
		if len(k) < 8 || !strings.EqualFold(k[:8], "X-Qiniu-") {
			continue
		}
		keys = append(keys, normalizeXQiniuKey(k))
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		// 规范化后回查原值：Go 的 Header.Get 大小写不敏感，直接用规范化名取即可。
		lines = append(lines, key+": "+req.Header.Get(key))
	}
	return lines
}

func normalizeXQiniuKey(k string) string {
	parts := strings.Split(k, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, "-")
}

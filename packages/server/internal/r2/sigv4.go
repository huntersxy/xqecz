// Package r2 提供 Cloudflare R2（S3 兼容）的最小客户端：
// 只实现镜像所需的 PutObject 与 HeadObject，签名自行实现 AWS Signature V4，
// 不引入 AWS SDK —— 本服务以「单个静态二进制」形态部署，依赖越少越好。
package r2

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// emptyPayloadHash 是空请求体的 SHA-256（用于 HEAD/GET 这类无 body 的请求）。
const emptyPayloadHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// awsURIEncode 按 RFC 3986 对单段做百分号编码。
// 与 url.QueryEscape 的差别：空格编码为 %20（而非 +），且 ~ 不编码。
func awsURIEncode(s string) string {
	const upperhex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperhex[c>>4])
		b.WriteByte(upperhex[c&0x0f])
	}
	return b.String()
}

// canonicalURI 逐段编码路径：斜杠作为分隔符保留，每段内的特殊字符转义。
func canonicalURI(path string) string {
	if path == "" {
		return "/"
	}
	segs := strings.Split(path, "/")
	for i, s := range segs {
		segs[i] = awsURIEncode(s)
	}
	out := strings.Join(segs, "/")
	if !strings.HasPrefix(out, "/") {
		out = "/" + out
	}
	return out
}

// canonicalQuery 按键排序并编码查询串；无查询参数时返回空串。
func canonicalQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(q))
	for _, k := range keys {
		vals := append([]string(nil), q[k]...)
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, awsURIEncode(k)+"="+awsURIEncode(v))
		}
	}
	return strings.Join(parts, "&")
}

// sign 就地给请求加上 SigV4 签名头（Authorization / x-amz-date / x-amz-content-sha256）。
//
// payloadHash 是请求体的 SHA-256 十六进制串；无 body 时传空串，自动按空串哈希处理。
// extra 是要参与签名的额外头（小写名 → 值），例如 x-amz-meta-md5。
// 注意：这里只签名调用方显式声明的头，Host 单独参与，避免签进 Go 自动补充的头。
func sign(req *http.Request, accessKey, secretKey, region, service string, payloadHash string, extra map[string]string, now time.Time) {
	if payloadHash == "" {
		payloadHash = emptyPayloadHash
	}
	amzDate := now.UTC().Format("20060102T150405Z")
	dateStamp := now.UTC().Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	for k, v := range extra {
		req.Header.Set(k, v)
	}

	// 组织待签名头：host + 全部 x-amz-*（content-type 也一并纳入，避免中间层改写歧义）。
	names := []string{"host"}
	for k := range req.Header {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz-") || lk == "content-type" {
			names = append(names, lk)
		}
	}
	sort.Strings(names)
	names = dedupe(names)

	var canonicalHeaders strings.Builder
	for _, n := range names {
		v := req.Header.Get(n)
		if n == "host" {
			v = req.Host
			if v == "" {
				v = req.URL.Host
			}
		}
		canonicalHeaders.WriteString(n)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(collapseSpaces(v))
		canonicalHeaders.WriteString("\n")
	}
	signedHeaders := strings.Join(names, ";")

	// 注意用已解码的 URL.Path：canonicalURI 自己做一次百分号编码。
	// 若传 EscapedPath（已编码），中文/空格对象名会被二次编码成 %25E4…，签名必然不被接受。
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL.Path),
		canonicalQuery(req.URL.Query()),
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hexSHA256([]byte(canonicalRequest)),
	}, "\n")

	signingKey := deriveSigningKey(secretKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+accessKey+"/"+scope+
		", SignedHeaders="+signedHeaders+", Signature="+signature)
}

// collapseSpaces 把连续空白压成单空格，与 SigV4 的规范头要求一致。
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func dedupe(in []string) []string {
	out := in[:0]
	var last string
	for i, s := range in {
		if i > 0 && s == last {
			continue
		}
		out = append(out, s)
		last = s
	}
	return out
}

func deriveSigningKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func hexSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

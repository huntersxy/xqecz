package r2

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLiveCdnBehavior 对七牛 CDN 域名做端到端验收：
// 上传/读取权限、CORS（前端探测必需）、Referer 防盗链。
// 需要 QINIU_AK / QINIU_SK，缺任一则跳过，不进 CI。
//
// 时间戳防盗链已关闭，URL 不带签名也必须放行——ACME 的 HTTP-01 挑战
// 就是从这个域名取 /.well-known/acme-challenge/<token> 的，
// 一旦有签名校验，证书自动续签必然失败。
func TestLiveCdnBehavior(t *testing.T) {
	ak := os.Getenv("QINIU_AK")
	sk := os.Getenv("QINIU_SK")
	if ak == "" || sk == "" {
		t.Skip("未提供 QINIU_AK / QINIU_SK")
	}
	const region, bucket = "cn-south-1", "xy996"
	const key = "uploads/__probe__.txt"
	payload := []byte("qiniu probe payload\n")
	s3 := "https://s3." + region + ".qiniucs.com/" + bucket + "/"

	client := &http.Client{
		Timeout:   20 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	// 上传/读取权限
	put, _ := http.NewRequest(http.MethodPut, s3+key, bytes.NewReader(payload))
	put.ContentLength = int64(len(payload))
	put.Header.Set("Content-Type", "text/plain")
	sign(put, ak, sk, region, "s3", hexSHA256(payload), nil, time.Now())
	resp, err := client.Do(put)
	if err != nil {
		t.Fatalf("PUT 失败: %v", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	fmt.Printf("[上传权限] signed PUT   -> %d\n", resp.StatusCode)

	head, _ := http.NewRequest(http.MethodHead, s3+key, nil)
	sign(head, ak, sk, region, "s3", "", nil, time.Now())
	resp2, err := client.Do(head)
	if err != nil {
		t.Fatalf("HEAD 失败: %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	fmt.Printf("[读权限]   signed HEAD  -> %d\n", resp2.StatusCode)

	base := "https://img.xiey.work/" + key
	show := func(label string, mutate func(*http.Request)) {
		req, err := http.NewRequest(http.MethodGet, base, nil)
		if err != nil {
			fmt.Printf("%-30s ERR %v\n", label, err)
			return
		}
		if mutate != nil {
			mutate(req)
		}
		r, err := client.Do(req)
		if err != nil {
			fmt.Printf("%-30s ERR %v\n", label, err)
			return
		}
		b, _ := io.ReadAll(io.LimitReader(r.Body, 64))
		r.Body.Close()
		fmt.Printf("%-30s %d  ACAO=%q body=%q\n", label, r.StatusCode,
			r.Header.Get("Access-Control-Allow-Origin"), string(b))
	}

	selfReferer := func(r *http.Request) { r.Header.Set("Referer", "https://xq.xiey.work/content/1") }

	// CORS：前端跨域探测的真实形态
	show("[CORS] Origin+Range", func(r *http.Request) {
		r.Header.Set("Origin", "https://xq.xiey.work")
		r.Header.Set("Range", "bytes=0-31")
		selfReferer(r)
	})

	// 时间戳防盗链已关：无签名必须放行，否则 ACME HTTP-01 会被挡死
	show("[时间戳] 关闭·无签名", selfReferer)

	// Referer 白名单仍在
	show("[防盗链] 本站", selfReferer)
	show("[防盗链] 第三方", func(r *http.Request) {
		r.Header.Set("Referer", "https://evil.example.com/x")
	})
	show("[防盗链] 空 Referer", nil)

	// 清理
	del, _ := http.NewRequest(http.MethodDelete, s3+key, nil)
	sign(del, ak, sk, region, "s3", "", nil, time.Now())
	resp3, err := client.Do(del)
	if err != nil {
		t.Fatalf("DELETE 失败: %v", err)
	}
	io.Copy(io.Discard, resp3.Body)
	resp3.Body.Close()
	fmt.Printf("[清理] signed DELETE    -> %d\n", resp3.StatusCode)
}

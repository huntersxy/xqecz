package qiniu

import (
	"strings"
	"testing"
	"time"
)

// TestSignURLMatchesOfficialExample 用七牛文档给出的完整算例做校验。
// 见 developer.qiniu.com/fusion/3841 算法说明一节：
//
//	KEY = 9388f4ba63b89bba5b9b84aa70a92eaac099d39b
//	path = /DIR1/中文/vodfile.mp4
//	T    = 55bb9b80  （= 1438358400）
//	SIGN = b4b7f94dd7817ce0283b5491861c3936
func TestSignURLMatchesOfficialExample(t *testing.T) {
	const key = "9388f4ba63b89bba5b9b84aa70a92eaac099d39b"
	// 文档算例的过期时刻：2015-08-01 00:00:00 +08:00 → 1438358400 → 55bb9b80
	expiry := time.Unix(1438358400, 0)

	got, err := SignURL("http://xxx.yyy.com/DIR1/中文/vodfile.mp4", key, expiry)
	if err != nil {
		t.Fatal(err)
	}
	want := "http://xxx.yyy.com/DIR1/%E4%B8%AD%E6%96%87/vodfile.mp4" +
		"?sign=b4b7f94dd7817ce0283b5491861c3936&t=55bb9b80"
	if got != want {
		t.Fatalf("签名 URL 与文档不符\n got: %s\nwant: %s", got, want)
	}
}

// TestSignURLPreservesExistingQuery 带 query 的原 URL 应保留 query 并追加参数。
func TestSignURLPreservesExistingQuery(t *testing.T) {
	got, err := SignURL("https://img.xiey.work/uploads/a.webp?v=1.2", "k", time.Unix(1438358400, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "?v=1.2&sign=") || !strings.Contains(got, "&t=55bb9b80") {
		t.Fatalf("原 query 未被保留或参数顺序不对: %s", got)
	}
	// sign 必须排在 t 之前
	if strings.Index(got, "sign=") > strings.Index(got, "&t=") {
		t.Fatalf("七牛要求 sign 在前、t 在后: %s", got)
	}
}

// TestSignURLHexExpiry 十进制时间会被 CDN 当成极大过期时间（等于没开鉴权），
// 这里钉死 T 一定是 16 进制小写。
func TestSignURLHexExpiry(t *testing.T) {
	got, err := SignURL("https://img.xiey.work/x.webp", "k", time.Unix(1438358400, 0))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "&t=55bb9b80") {
		t.Fatalf("T 应为 16 进制小写 55bb9b80，实际 %s", got)
	}
	if strings.Contains(got, "1438358400") {
		t.Fatalf("绝不能出现十进制时间: %s", got)
	}
}

func TestEncodePathKeepsSlash(t *testing.T) {
	if encodePath("/DIR1/中文/vodfile.mp4") != "/DIR1/%E4%B8%AD%E6%96%87/vodfile.mp4" {
		t.Fatalf("斜线不编码 / 中文须百分号大写: %s", encodePath("/DIR1/中文/vodfile.mp4"))
	}
	// 项目实际路径是内容寻址文件名，全 ASCII，编码必须是恒等变换
	if encodePath("/uploads/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08.webp") !=
		"/uploads/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08.webp" {
		t.Fatal("内容寻址路径不应被改写")
	}
}

func TestSignURLRejectsEmpty(t *testing.T) {
	if _, err := SignURL("", "k", time.Now()); err == nil {
		t.Fatal("空 URL 应报错")
	}
	if _, err := SignURL("https://a/b", "", time.Now()); err == nil {
		t.Fatal("空密钥应报错")
	}
}

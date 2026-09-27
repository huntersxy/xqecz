package qiniu

import (
	"net/http"
	"strings"
	"testing"
)

// TestSignerMatchesOfficialExample 用七牛官方文档里的样例做字节级校验。
// 样例：developer.qiniu.com/kodo/1201/access-token
//
//	AK = "MY_ACCESS_KEY"  SK = "MY_SECRET_KEY"
//	POST /move/bmV3ZG9jczpmaW5kX21hbi50eHQ=/bmV3ZG9jczpmaW5kLm1hbi50eHQ=
//	Host: rs.qiniu.com
//	期望凭证 = MY_ACCESS_KEY:1uLvuZM6l6oCzZFqkJ6oI4oFMVQ=
func TestSignerMatchesOfficialExample(t *testing.T) {
	s := Signer{AccessKey: "MY_ACCESS_KEY", SecretKey: "MY_SECRET_KEY"}
	req, err := http.NewRequest(http.MethodPost,
		"http://rs.qiniu.com/move/bmV3ZG9jczpmaW5kX21hbi50eHQ=/bmV3ZG9jczpmaW5kLm1hbi50eHQ=", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := s.token(req, nil)
	want := "MY_ACCESS_KEY:1uLvuZM6l6oCzZFqkJ6oI4oFMVQ="
	if got != want {
		t.Fatalf("凭证不匹配\n got: %s\nwant: %s", got, want)
	}
}

func TestSignerEnabled(t *testing.T) {
	if (Signer{}).Enabled() {
		t.Fatal("空凭据应判定为未启用")
	}
	if !(Signer{AccessKey: "a", SecretKey: "b"}).Enabled() {
		t.Fatal("齐备凭据应判定为已启用")
	}
}

func TestSignerIncludesQueryString(t *testing.T) {
	s := Signer{AccessKey: "AK", SecretKey: "SK"}
	withQuery, _ := http.NewRequest(http.MethodGet, "https://api.qiniuapi.com/v6/blob_io?begin=20260101", nil)
	withoutQuery, _ := http.NewRequest(http.MethodGet, "https://api.qiniuapi.com/v6/blob_io", nil)
	if s.token(withQuery, nil) == s.token(withoutQuery, nil) {
		t.Fatal("query 必须进签名串，否则带参与不带参的签名会相同")
	}
}

func TestSignerIncludesContentType(t *testing.T) {
	s := Signer{AccessKey: "AK", SecretKey: "SK"}
	plain, _ := http.NewRequest(http.MethodPost, "https://api.qiniu.com/v6/x", nil)
	withCT, _ := http.NewRequest(http.MethodPost, "https://api.qiniu.com/v6/x", nil)
	withCT.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if s.token(plain, nil) == s.token(withCT, nil) {
		t.Fatal("Content-Type 必须进签名串")
	}
}

// TestNormalizeXQiniuKey 校验键名规范化：首字母与连字符后的字母大写，其余小写。
func TestNormalizeXQiniuKey(t *testing.T) {
	cases := map[string]string{
		"X-Qiniu-Date": "X-Qiniu-Date",
		"x-qiniu-date": "X-Qiniu-Date",
		// 规范化是机械的大小写折叠，不会「认出」正确拼写：
		// QINUI → Qinui，而非纠正为 Qiniu。
		"X-QINUI-RANGE": "X-Qinui-Range",
	}
	for in, want := range cases {
		if got := normalizeXQiniuKey(in); got != want {
			t.Errorf("normalizeXQiniuKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestXQiniuHeadersSorted 七牛要求 X-Qiniu-* 头按 key 的 ASCII 序进入签名串。
func TestXQiniuHeadersSorted(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.qiniu.com/v6/x", nil)
	req.Header.Set("X-Qiniu-Zeta", "z")
	req.Header.Set("X-Qiniu-Alpha", "a")
	lines := xqiniuHeaderLines(req)
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "X-Qiniu-Alpha") {
		t.Fatalf("X-Qiniu-* 头未按 ASCII 序排列: %v", lines)
	}
}

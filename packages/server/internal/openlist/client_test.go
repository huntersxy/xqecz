package openlist

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/objstore"
)

type putServer struct {
	*httptest.Server
	gotBody    []byte
	gotPath    string
	gotAuth    string
	gotType    string
	statusCode int
	apiBody    string
}

func newPutServer(t *testing.T) *putServer {
	t.Helper()
	ps := &putServer{apiBody: `{"code":200,"message":"success"}`}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("期望 PUT，实际 %s", r.Method)
		}
		ps.gotAuth = r.Header.Get("Authorization")
		ps.gotType = r.Header.Get("Content-Type")
		ps.gotPath = r.Header.Get("File-Path")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("读取请求体失败：%v", err)
		}
		ps.gotBody = body
		code := ps.statusCode
		if code == 0 {
			code = http.StatusOK
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(ps.apiBody))
	}))
	t.Cleanup(ps.Close)
	return ps
}

func newClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := New(Config{Endpoint: endpoint, Token: "openlist-test-token", Prefix: "uploads", Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("构造客户端失败：%v", err)
	}
	return c
}

func TestPutSucceedsAndReportsNil(t *testing.T) {
	ps := newPutServer(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "a.webp")
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	c := newClient(t, ps.URL)
	if err := c.Put("uploads/a.webp", path, "image/webp"); err != nil {
		t.Fatalf("Put 应当成功，实际返回：%v", err)
	}

	if string(ps.gotBody) != string(content) {
		t.Errorf("服务端收到的内容不符：got %q want %q", ps.gotBody, content)
	}
	if ps.gotAuth != "openlist-test-token" {
		t.Errorf("鉴权头应为裸 token（不带 Bearer），实际 %q", ps.gotAuth)
	}
	if ps.gotType != "image/webp" {
		t.Errorf("Content-Type 应为 image/webp，实际 %q", ps.gotType)
	}
	if want := (&url.URL{Path: "/uploads/a.webp"}).EscapedPath(); ps.gotPath != want {
		t.Errorf("File-Path 头不符：got %q want %q", ps.gotPath, want)
	}
}

func TestPutAfterRequestLeavesFileStatable(t *testing.T) {
	ps := newPutServer(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "b.bin")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	req, err := http.NewRequest(http.MethodPut, ps.URL, io.NopCloser(f))
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = 5
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if _, err := f.Stat(); err != nil {
		t.Fatalf("请求结束后句柄仍应可 stat，实际：%v", err)
	}
}

func TestPutDetectsMidUploadRewrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.bin")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	ps := &putServer{apiBody: `{"code":200,"message":"success"}`}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if err := os.WriteFile(path, []byte("after-after-after"), 0o644); err != nil {
			t.Errorf("改写失败：%v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(ps.apiBody))
	}))
	defer ps.Close()

	c := newClient(t, ps.URL)
	err := c.Put("uploads/c.bin", path, "application/octet-stream")
	if !errors.Is(err, objstore.ErrCorrupted) {
		t.Fatalf("期望 objstore.ErrCorrupted，实际 %v", err)
	}
}

func TestPutSurfacesAPIError(t *testing.T) {
	ps := newPutServer(t)
	ps.apiBody = `{"code":500,"message":"failed to put"}`

	dir := t.TempDir()
	path := filepath.Join(dir, "d.bin")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := newClient(t, ps.URL)
	err := c.Put("uploads/d.bin", path, "")
	if err == nil {
		t.Fatal("业务失败不应返回 nil")
	}
	if !strings.Contains(err.Error(), "failed to put") {
		t.Errorf("错误里应带上 OpenList 的 message，实际 %v", err)
	}
}

func TestPutMissingFile(t *testing.T) {
	ps := newPutServer(t)
	c := newClient(t, ps.URL)
	if err := c.Put("uploads/none.bin", filepath.Join(t.TempDir(), "none.bin"), ""); err == nil {
		t.Fatal("文件不存在应报错")
	}
}

func TestHeadSuccessAndMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("期望 POST，实际 %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "openlist-test-token" {
			t.Errorf("鉴权头不符：%q", got)
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "missing") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":500,"message":"failed to get obj: object not found","data":null}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"name":"3e3ee327ba598f0e8a5c3b371de67b76.gif","size":813433,"is_dir":false,"modified":"2026-10-06T13:27:37.302477504+08:00","created":"2026-10-06T13:27:37.300620703+08:00","sign":"","thumb":"","type":5,"hashinfo":"null","hash_info":null,"raw_url":"https://drive.xiey.work/p/uploads/3e3ee327ba598f0e8a5c3b371de67b76.gif","provider":"Local"}}`))
	}))
	defer srv.Close()

	c := newClient(t, srv.URL)

	meta, found, err := c.Head("uploads/3e.gif")
	if err != nil {
		t.Fatalf("Head 成功分支不应报错：%v", err)
	}
	if !found {
		t.Fatal("应当判定为存在")
	}
	if meta.ContentLen != 813433 {
		t.Errorf("ContentLen 不符：got %d want 813433", meta.ContentLen)
	}
	if meta.MetaMD5 != "" || meta.ETag != "" {
		t.Errorf("OpenList 不应给出哈希，实际 md5=%q etag=%q", meta.MetaMD5, meta.ETag)
	}
	if meta.LastModified.IsZero() {
		t.Error("modified 应当被解析出来")
	}

	_, found, err = c.Head("uploads/missing.bin")
	if err != nil {
		t.Fatalf("「不存在」不应是错误，实际 %v", err)
	}
	if found {
		t.Error("不存在的对象不应判定为存在")
	}
}

func TestNewRejectsIncompleteConfig(t *testing.T) {
	if _, err := New(Config{Token: "t"}); err == nil {
		t.Error("缺少 endpoint 应报错")
	}
	if _, err := New(Config{Endpoint: "http://127.0.0.1:5244"}); err == nil {
		t.Error("缺少 token 应报错")
	}
}

func TestObjectKeyAndFsPath(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:5244")
	if got, want := c.ObjectKey("/ab/cd.webp"), "uploads/ab/cd.webp"; got != want {
		t.Errorf("ObjectKey = %q want %q", got, want)
	}
	if got, want := fsPath("uploads/ab/cd.webp"), "/uploads/ab/cd.webp"; got != want {
		t.Errorf("fsPath = %q want %q", got, want)
	}
	raw, err := New(Config{Endpoint: "http://127.0.0.1:5244", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := raw.ObjectKey("ab/cd.webp"), "ab/cd.webp"; got != want {
		t.Errorf("无 prefix 时 ObjectKey = %q want %q", got, want)
	}
}

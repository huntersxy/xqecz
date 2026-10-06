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

// putServer 起一个假 OpenList，把收到的 PUT 落到内存里。
// 回调里可对请求做断言（头、鉴权、body 长度）。
type putServer struct {
	*httptest.Server
	gotBody    []byte
	gotPath    string
	gotAuth    string
	gotType    string
	statusCode int // 0 表示回 200
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

// TestPutSucceedsAndReportsNil 是本包最关键的一条回归：
//
// 曾经的实现在请求体里直接传 *os.File，net/http 在请求结束后会 Close 它，
// 于是上传成功后那次「文件是否被就地改写」的校验 stat 必然报
// `file already closed`，Put 被误判失败。上层因此不记账，每轮回填把全量
// 媒体重推一遍，生产日志表现为 scanned=309 uploaded=0 failed=309。
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
	// File-Path 送转义形态，服务端会解码；此处断言它匹配期望路径
	// （EscapedPath 保留 `/`，只转义段内字符）。
	if want := (&url.URL{Path: "/uploads/a.webp"}).EscapedPath(); ps.gotPath != want {
		t.Errorf("File-Path 头不符：got %q want %q", ps.gotPath, want)
	}
}

// TestPutAfterRequestLeavesFileStatable 直接钉住根因：请求体被包成
// NopCloser 后，Transport 不会关掉我们持有的句柄，句柄在 Put 返回后仍可用。
func TestPutAfterRequestLeavesFileStatable(t *testing.T) {
	ps := newPutServer(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "b.bin")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 复刻 Put 的请求体构造，验证包装语义（不依赖 Client 内部细节）。
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

// TestPutDetectsMidUploadRewrite 上传期间文件被就地改写 → ErrCorrupted，
// 让上层下一轮重传而不是把半旧的内容记成已同步。
func TestPutDetectsMidUploadRewrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.bin")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	ps := &putServer{apiBody: `{"code":200,"message":"success"}`}
	ps.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		// 模拟压缩任务在请求处理期间原地覆盖同一个文件。
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

// TestPutSurfacesAPIError OpenList 的业务失败写在 body 的 code 里、
// HTTP 状态码仍是 200——这种必须被判成错误。
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

// TestPutMissingFile 本地文件不存在时立即报错，不发请求。
func TestPutMissingFile(t *testing.T) {
	ps := newPutServer(t)
	c := newClient(t, ps.URL)
	if err := c.Put("uploads/none.bin", filepath.Join(t.TempDir(), "none.bin"), ""); err == nil {
		t.Fatal("文件不存在应报错")
	}
}

// TestHeadSuccessAndMissing 覆盖两条 Head 分支：拿到 size/modified，
// 以及「不存在」用 (零值, false, nil) 表达而非错误。
//
// 这里的响应体是**实盘抓下来的原样形状**（2026-10-06 从生产
// https://drive.xiey.work/api/fs/get 取得）：元数据嵌在 data 下，
// hashinfo 恒为 "null"（Local 驱动没有远端哈希）。
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
	// size 在 data 下；按扁平解会得到 0，而 0 会让 sameObject 恒判「不同步」，
	// 于是每轮回填都把全量媒体重推一遍。
	if meta.ContentLen != 813433 {
		t.Errorf("ContentLen 不符：got %d want 813433", meta.ContentLen)
	}
	// OpenList 的 Local 驱动没有远端哈希，幂等判定只能靠字节数。
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

// TestNewRejectsIncompleteConfig 端点或 token 缺失时必须构造失败，
// 上层据此整体停用该镜像目标。
func TestNewRejectsIncompleteConfig(t *testing.T) {
	if _, err := New(Config{Token: "t"}); err == nil {
		t.Error("缺少 endpoint 应报错")
	}
	if _, err := New(Config{Endpoint: "http://127.0.0.1:5244"}); err == nil {
		t.Error("缺少 token 应报错")
	}
}

// TestObjectKeyAndFsPath key 结构与 R2 保持一致，挂载点固定落在 `/`。
func TestObjectKeyAndFsPath(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:5244")
	if got, want := c.ObjectKey("/ab/cd.webp"), "uploads/ab/cd.webp"; got != want {
		t.Errorf("ObjectKey = %q want %q", got, want)
	}
	if got, want := fsPath("uploads/ab/cd.webp"), "/uploads/ab/cd.webp"; got != want {
		t.Errorf("fsPath = %q want %q", got, want)
	}
	// 无 prefix 时是裸 key。
	raw, err := New(Config{Endpoint: "http://127.0.0.1:5244", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := raw.ObjectKey("ab/cd.webp"), "ab/cd.webp"; got != want {
		t.Errorf("无 prefix 时 ObjectKey = %q want %q", got, want)
	}
}

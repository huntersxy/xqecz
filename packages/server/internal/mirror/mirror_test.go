package mirror

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/r2"
)

type fakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	md5s    map[string]string
	heads   int
	puts    int
	putErr  error
}

func (f *fakeStore) putCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.puts
}

func (f *fakeStore) failWith(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.putErr = err
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string][]byte{}, md5s: map[string]string{}}
}

func (f *fakeStore) Put(key, absPath, contentType string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.puts++
	if f.putErr != nil {
		return f.putErr
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}
	f.objects[key] = data
	f.md5s[key] = md5Hex(data)
	return nil
}

func (f *fakeStore) Head(key string) (r2.ObjectMeta, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.heads++
	data, ok := f.objects[key]
	if !ok {
		return r2.ObjectMeta{}, false, nil
	}
	return r2.ObjectMeta{
		ETag:       f.md5s[key],
		ContentLen: int64(len(data)),
		MetaMD5:    f.md5s[key],
	}, true, nil
}

func md5Hex(b []byte) string {
	res := md5Sum(b)
	return res
}

func testSetup(t *testing.T, store ObjectStore) (*Setup, string) {
	t.Helper()
	cfg := config.R2Config{
		AccountID: "acc", AccessKey: "ak", SecretKey: "sk", Bucket: "bkt",
		Endpoint: "https://acc.r2.cloudflarestorage.com", Prefix: "uploads",
		PublicBase: "https://file.example.com",
	}
	s := newWithStore(cfg, store)
	dir := t.TempDir()
	return s, dir
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	abs := filepath.Join(dir, name)
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestPushUploadsWhenMissing(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "a.webp", []byte("original-bytes"))

	uploaded, err := s.Push("a.webp", abs)
	if err != nil {
		t.Fatalf("Push 失败: %v", err)
	}
	if !uploaded {
		t.Fatal("R2 上不存在时应上传")
	}
	if store.puts != 1 {
		t.Fatalf("期望 1 次 PUT，实际 %d", store.puts)
	}
	if string(store.objects["uploads/a.webp"]) != "original-bytes" {
		t.Fatal("上传内容与本地不一致")
	}
}

func TestPushSkipsWhenRemoteMatches(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "b.webp", []byte("same-bytes"))

	if _, err := s.Push("b.webp", abs); err != nil {
		t.Fatal(err)
	}
	before := store.puts
	s2 := newWithStoreTarget(s.cfg, store)
	uploaded, err := s2.Push("b.webp", abs)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded || store.puts != before {
		t.Fatalf("内容一致时不应重复上传：uploaded=%v puts=%d->%d", uploaded, before, store.puts)
	}
	if store.heads == 0 {
		t.Fatal("跳过前应先 HEAD 比对内容")
	}
}

func TestPushReuploadsAfterInPlaceCompression(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "c.webp", []byte("uncompressed-original-content"))

	if _, err := s.Push("c.webp", abs); err != nil {
		t.Fatal(err)
	}
	if string(store.objects["uploads/c.webp"]) != "uncompressed-original-content" {
		t.Fatal("首次应为原图内容")
	}

	if err := os.WriteFile(abs, []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	uploaded, err := s.Push("c.webp", abs)
	if err != nil {
		t.Fatalf("压缩后重传失败: %v", err)
	}
	if !uploaded {
		t.Fatal("内容变化后必须重新上传")
	}
	if string(store.objects["uploads/c.webp"]) != "tiny" {
		t.Fatalf("R2 上应换成压缩后的内容，实际 %q", string(store.objects["uploads/c.webp"]))
	}
}

func TestPushReuploadsSameSizeWhenFileChanges(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "same-size.webp", []byte("before"))
	if _, err := s.Push("same-size.webp", abs); err != nil {
		t.Fatal(err)
	}
	stat, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("after!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(abs, stat.ModTime(), stat.ModTime().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	uploaded, err := s.Push("same-size.webp", abs)
	if err != nil || !uploaded {
		t.Fatalf("same-size rewrite should upload: uploaded=%v err=%v", uploaded, err)
	}
	if got := string(store.objects["uploads/same-size.webp"]); got != "after!" {
		t.Fatalf("remote content = %q, want after!", got)
	}
}

func TestPushNoopWhenLocalMissing(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	uploaded, err := s.Push("gone.webp", filepath.Join(dir, "gone.webp"))
	if err != nil {
		t.Fatalf("本地文件缺失不应报错: %v", err)
	}
	if uploaded || store.puts != 0 {
		t.Fatal("本地文件缺失时不应上传")
	}
}

func TestPushSkipsThumbnailsAndTemp(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)

	for _, rel := range []string{"thumbs/a_thumb.webp", "images/old.webp", ".upload-123.part"} {
		abs := writeFile(t, dir, filepath.Base(rel), []byte("x"))
		uploaded, err := s.Push(rel, abs)
		if err == nil && uploaded {
			t.Fatalf("%s 不应被镜像", rel)
		}
		if store.puts != 0 {
			t.Fatalf("%s 触发了上传", rel)
		}
	}
}

func TestPushTreatsCorruptedAsSkip(t *testing.T) {
	store := newFakeStore()
	store.failWith(r2.ErrCorrupted)
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "d.webp", []byte("bytes"))

	uploaded, err := s.Push("d.webp", abs)
	if err != nil {
		t.Fatalf("易变文件应静默跳过，实际: %v", err)
	}
	if uploaded {
		t.Fatal("易变文件不应记为已上传")
	}

	store.failWith(nil)
	uploaded, err = s.Push("d.webp", abs)
	if err != nil || !uploaded {
		t.Fatalf("下一轮应重试成功: uploaded=%v err=%v", uploaded, err)
	}
}

func TestPushPropagatesRealErrors(t *testing.T) {
	store := newFakeStore()
	store.failWith(errors.New("network down"))
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "e.webp", []byte("bytes"))

	if _, err := s.Push("e.webp", abs); err == nil {
		t.Fatal("网络错误应返回给调用方")
	}
}

func TestDisabledSetupIsNoop(t *testing.T) {
	s := New(Target{}, "")
	if s.Enabled() {
		t.Fatal("缺凭据时不应启用")
	}
	if s.PublicURL("a.webp") != "" {
		t.Fatal("未启用时不应给出公开地址")
	}
	uploaded, err := s.Push("a.webp", "whatever")
	if uploaded || err != nil {
		t.Fatalf("未启用时 Push 应为空操作: uploaded=%v err=%v", uploaded, err)
	}
	var nilSetup *Setup
	if nilSetup.Enabled() || nilSetup.PublicURL("a.webp") != "" {
		t.Fatal("nil Setup 应安全")
	}
	if _, err := nilSetup.Push("a.webp", "x"); err != nil {
		t.Fatalf("nil Setup 的 Push 不应报错: %v", err)
	}
}

func TestPublicURLWithoutExposedBase(t *testing.T) {
	s := newWithStore(config.R2Config{AccountID: "a", AccessKey: "k", SecretKey: "s", Bucket: "b"}, newFakeStore())
	if got := s.PublicURL("a.webp"); got != "" {
		t.Fatalf("未配置 PublicBase 时不应返回地址，实际 %q", got)
	}
	s.cfg.PublicBase = "https://file.example.com/"
	if got := s.PublicURL("a.webp"); got != "https://file.example.com/"+s.ObjectKey("a.webp") {
		t.Fatalf("公开地址拼接有误: %q（对象名 %q）", got, s.ObjectKey("a.webp"))
	}
	s.cfg.Prefix = "uploads"
	if got := s.PublicURL("a.webp"); got != "https://file.example.com/uploads/a.webp" {
		t.Fatalf("带前缀的公开地址有误: %q", got)
	}
}

func TestPublicURLPathMatchesObjectKey(t *testing.T) {
	s := newWithStore(config.R2Config{
		AccountID: "a", AccessKey: "k", SecretKey: "s", Bucket: "b",
		Prefix: "uploads", PublicBase: "https://file.example.com",
	}, newFakeStore())
	for _, rel := range []string{"ab12.webp", "uploads/ab12.webp", "/uploads/ab12.webp"} {
		u := s.PublicURL(rel)
		if !strings.HasSuffix(u, "/"+s.ObjectKey("ab12.webp")) {
			t.Fatalf("PublicURL(%q) = %q，其路径应等于对象名 %q", rel, u, s.ObjectKey("ab12.webp"))
		}
	}
}

func TestObjectKeyKeepsUploadsLayout(t *testing.T) {
	s := newWithStore(config.R2Config{Prefix: "uploads"}, newFakeStore())
	if got := s.ObjectKey("ab12.webp"); got != "uploads/ab12.webp" {
		t.Fatalf("ObjectKey = %q", got)
	}
	noPrefix := newWithStore(config.R2Config{}, newFakeStore())
	if got := noPrefix.ObjectKey("ab12.webp"); got != "ab12.webp" {
		t.Fatalf("空前缀 ObjectKey = %q", got)
	}
}

func TestLocalMD5HashesContentNotName(t *testing.T) {
	dir := t.TempDir()

	name := "d41d8cd98f00b204e9800998ecf8427e.webp"
	abs := writeFile(t, dir, name, []byte("not-the-empty-string"))
	if got, want := localMD5(abs), md5Sum([]byte("not-the-empty-string")); got != want {
		t.Fatalf("内容寻址名不得沿用文件名：得到 %q，期望内容 md5 %q", got, want)
	}

	same := writeFile(t, dir, md5Sum([]byte("hello"))+".png", []byte("hello"))
	if got, want := localMD5(same), md5Sum([]byte("hello")); got != want {
		t.Fatalf("名字与内容一致时应得到内容 md5：得到 %q，期望 %q", got, want)
	}
}

func TestPushReuploadsCompressedContentAddressedFile(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)

	const original = "uncompressed-original-bytes"
	name := md5Sum([]byte(original)) + ".webp"
	abs := writeFile(t, dir, name, []byte(original))

	if _, err := s.Push(name, abs); err != nil {
		t.Fatal(err)
	}
	if string(store.objects["uploads/"+name]) != original {
		t.Fatal("首次应为原图内容")
	}

	const compressed = "tiny"
	if err := os.WriteFile(abs, []byte(compressed), 0o644); err != nil {
		t.Fatal(err)
	}
	uploaded, err := s.Push(name, abs)
	if err != nil {
		t.Fatalf("压缩后重传失败: %v", err)
	}
	if !uploaded {
		t.Fatal("原图已换成压缩图，即便文件名未变也必须重传")
	}
	if string(store.objects["uploads/"+name]) != compressed {
		t.Fatalf("远端应换成压缩后的内容，实际 %q", string(store.objects["uploads/"+name]))
	}

	before := store.puts
	s2 := newWithStoreTarget(s.cfg, store)
	again, err := s2.Push(name, abs)
	if err != nil {
		t.Fatal(err)
	}
	if again || store.puts != before {
		t.Fatalf("内容一致后每轮都应收敛为不传：uploaded=%v puts=%d->%d", again, before, store.puts)
	}
}

func TestArchiveOriginalKeepsBothCopies(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	s.binDir = filepath.Join(dir, "bin")

	const name = "d41d8cd98f00b204e9800998ecf8427e.webp"
	abs := writeFile(t, dir, name, []byte("tiny-compressed"))
	if _, err := s.Push(name, abs); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := writeFile(t, s.binDir, name, []byte("uncompressed-original-bytes"))

	if _, err := s.ArchiveOriginal(name, binPath); err != nil {
		t.Fatalf("归档失败: %v", err)
	}
	if got := string(store.objects["uploads/"+name]); got != "tiny-compressed" {
		t.Fatalf("现役对象应为压缩图，实际 %q", got)
	}
	if got := string(store.objects["uploads/original/"+name]); got != "uncompressed-original-bytes" {
		t.Fatalf("归档对象应为压缩前原图，实际 %q", got)
	}
}

func TestArchiveOriginalSkipsSecondTime(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	s.binDir = filepath.Join(dir, "bin")
	if err := os.MkdirAll(s.binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const name = "d41d8cd98f00b204e9800998ecf8427e.webp"
	binPath := writeFile(t, s.binDir, name, []byte("orig"))

	if _, err := s.ArchiveOriginal(name, binPath); err != nil {
		t.Fatal(err)
	}
	puts, heads := store.puts, store.heads
	uploaded, err := s.ArchiveOriginal(name, binPath)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded || store.puts != puts || store.heads != heads {
		t.Fatalf("第二次归档不应再发请求：uploaded=%v puts=%d->%d heads=%d->%d",
			uploaded, puts, store.puts, heads, store.heads)
	}
}

func TestArchiveKeyNamespace(t *testing.T) {
	s := newWithStore(config.R2Config{Prefix: "uploads"}, newFakeStore())
	if got := s.ArchiveKey("ab12.webp"); got != "uploads/original/ab12.webp" {
		t.Fatalf("ArchiveKey = %q", got)
	}
	noPrefix := newWithStore(config.R2Config{}, newFakeStore())
	if got := noPrefix.ArchiveKey("ab12.webp"); got != "uploads/original/ab12.webp" {
		t.Fatalf("空前缀 ArchiveKey = %q", got)
	}
}

func TestFindInBin(t *testing.T) {
	dir := t.TempDir()
	if _, ok := FindInBin(dir, "missing.webp"); ok {
		t.Fatal("不存在时不应命中")
	}
	plain := writeFile(t, dir, "a.webp", []byte("x"))
	writeFile(t, dir, "1700000000000_a.webp", []byte("old"))
	got, ok := FindInBin(dir, "a.webp")
	if !ok || got != plain {
		t.Fatalf("同名文件应优先命中，得到 %q ok=%v", got, ok)
	}
	if got, ok := FindInBin(dir, "b.webp"); ok {
		t.Fatalf("无任何匹配不应命中，得到 %q", got)
	}
	writeFile(t, dir, "1700000000001_b.webp", []byte("orig"))
	if got, ok := FindInBin(dir, "b.webp"); !ok || filepath.Base(got) != "1700000000001_b.webp" {
		t.Fatalf("时间戳前缀版本应命中，得到 %q ok=%v", got, ok)
	}
}

func TestMD5OfMetaHandlesMultipartETag(t *testing.T) {
	if got := md5OfMeta(r2.ObjectMeta{ETag: "abc-3"}); got != "" {
		t.Fatalf("多段 ETag 不应被当成 md5，得到 %q", got)
	}
	if got := md5OfMeta(r2.ObjectMeta{ETag: "0CC175B9C0F1B6A831C399E269772661"}); got != "0cc175b9c0f1b6a831c399e269772661" {
		t.Fatalf("单段 ETag 应大小写归一后作为 md5，得到 %q", got)
	}
	if got := md5OfMeta(r2.ObjectMeta{MetaMD5: " 0cc175b9c0f1b6a831c399e269772661 "}); got != "0cc175b9c0f1b6a831c399e269772661" {
		t.Fatalf("自定义元数据应优先且去空白，得到 %q", got)
	}
}

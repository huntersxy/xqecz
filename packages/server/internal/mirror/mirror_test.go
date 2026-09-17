package mirror

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/r2"
)

// fakeStore 是内存版对象存储：记录调用次数与对象内容，用于断言镜像行为。
type fakeStore struct {
	objects map[string][]byte
	md5s    map[string]string
	heads   int
	puts    int
	putErr  error
}

func newFakeStore() *fakeStore {
	return &fakeStore{objects: map[string][]byte{}, md5s: map[string]string{}}
}

// Put 的第一个参数是桶内对象名：生产侧由 mirror 算好（含 uploads/ 与 original/ 命名空间），
// 测试侧直接用这个名字记账，避免测试跟着生产实现一起算错。
func (f *fakeStore) Put(key, absPath, contentType string) error {
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
	// 测试辅助：与生产实现同款的 md5（只用于构造期望值）
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

// TestPushUploadsWhenMissing 本地有、R2 没有 → 上传。
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

// TestPushSkipsWhenRemoteMatches 远端内容与本地一致 → 只 HEAD，不重复 PUT。
func TestPushSkipsWhenRemoteMatches(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "b.webp", []byte("same-bytes"))

	if _, err := s.Push("b.webp", abs); err != nil {
		t.Fatal(err)
	}
	before := store.puts
	// 换一个新的 Setup（清掉进程内缓存），模拟重启后再次扫描同一文件。
	s2 := newWithStore(s.cfg, store)
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

// TestPushReuploadsAfterInPlaceCompression 是本次接入最关键的一条：
// 压缩是**原地改写**（路径不变、内容变了），镜像必须识别出内容已变重传，
// 否则 R2 上会永远停在未压缩的原图上。
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

	// 模拟 TinyPNG 原地替换（同一路径、更短的字节）。Setup 实例不变，
	// 这是最容易出错的地方：任何「传过就跳过」的标记都会在这里骗过自己。
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

// TestPushNoopWhenLocalMissing 本地文件不存在（删除/入桶）→ 不上传也不报错。
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

// TestPushSkipsThumbnailsAndTemp 缩略图与隐藏临时文件都不镜像。
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

// TestPushTreatsCorruptedAsSkip 上传期间文件被原地改写 → 放弃本轮且不留下缓存标记，
// 下一轮仍会重试（不能假装成功）。
func TestPushTreatsCorruptedAsSkip(t *testing.T) {
	store := newFakeStore()
	store.putErr = r2.ErrCorrupted
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "d.webp", []byte("bytes"))

	uploaded, err := s.Push("d.webp", abs)
	if err != nil {
		t.Fatalf("易变文件应静默跳过，实际: %v", err)
	}
	if uploaded {
		t.Fatal("易变文件不应记为已上传")
	}

	// 恢复可上传后应能重试成功（说明上一轮没有写入「已同步」缓存）。
	store.putErr = nil
	uploaded, err = s.Push("d.webp", abs)
	if err != nil || !uploaded {
		t.Fatalf("下一轮应重试成功: uploaded=%v err=%v", uploaded, err)
	}
}

// TestPushPropagatesRealErrors 真实错误必须冒泡，交给回填任务记录并重试。
func TestPushPropagatesRealErrors(t *testing.T) {
	store := newFakeStore()
	store.putErr = errors.New("network down")
	s, dir := testSetup(t, store)
	abs := writeFile(t, dir, "e.webp", []byte("bytes"))

	if _, err := s.Push("e.webp", abs); err == nil {
		t.Fatal("网络错误应返回给调用方")
	}
}

// TestDisabledSetupIsNoop 未配置凭据时全部方法都是安全空操作。
func TestDisabledSetupIsNoop(t *testing.T) {
	s := New(config.R2Config{}, "")
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
	// nil 接收者也不能 panic（镜像可能整体为 nil）。
	var nilSetup *Setup
	if nilSetup.Enabled() || nilSetup.PublicURL("a.webp") != "" {
		t.Fatal("nil Setup 应安全")
	}
	if _, err := nilSetup.Push("a.webp", "x"); err != nil {
		t.Fatalf("nil Setup 的 Push 不应报错: %v", err)
	}
}

// TestPublicURLWithoutExposedBase 只配了凭据、没配公开域名时不给前端地址
// （前端测速没有候选，直接走本地）。
func TestPublicURLWithoutExposedBase(t *testing.T) {
	s := newWithStore(config.R2Config{AccountID: "a", AccessKey: "k", SecretKey: "s", Bucket: "b"}, newFakeStore())
	if got := s.PublicURL("a.webp"); got != "" {
		t.Fatalf("未配置 PublicBase 时不应返回地址，实际 %q", got)
	}
	s.cfg.PublicBase = "https://file.example.com/"
	if got := s.PublicURL("a.webp"); got != "https://file.example.com/a.webp" {
		t.Fatalf("公开地址拼接有误: %q", got)
	}
}

// TestObjectKeyKeepsUploadsLayout 对象名保持本地目录结构，便于对账。
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

// TestLocalMD5UsesContentAddressedName 内容寻址文件名直接沿用（不必读全文件）。
func TestLocalMD5UsesContentAddressedName(t *testing.T) {
	dir := t.TempDir()
	name := "d41d8cd98f00b204e9800998ecf8427e.webp"
	abs := writeFile(t, dir, name, []byte("not-the-empty-string"))
	if got := localMD5(name, abs); got != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("内容寻址名应直接沿用，得到 %q", got)
	}
	// 随机名则老实算内容 md5。
	rand := writeFile(t, dir, "1730000000_abcdef.png", []byte("hello"))
	if got := localMD5("1730000000_abcdef.png", rand); got != md5Sum([]byte("hello")) {
		t.Fatalf("随机名应计算内容 md5，得到 %q", got)
	}
}

// TestArchiveOriginalKeepsBothCopies 是「原图与压缩图两份都保留」这条要求的核心断言：
// 压缩前的原图归档到 uploads/original/<name>，压缩图仍占 uploads/<name>，互不覆盖。
func TestArchiveOriginalKeepsBothCopies(t *testing.T) {
	store := newFakeStore()
	s, dir := testSetup(t, store)
	s.binDir = filepath.Join(dir, "bin")

	const name = "d41d8cd98f00b204e9800998ecf8427e.webp"
	// 压缩后的现役文件（原地替换后的字节）。
	abs := writeFile(t, dir, name, []byte("tiny-compressed"))
	if _, err := s.Push(name, abs); err != nil {
		t.Fatal(err)
	}
	// 压缩前的原图：被 MoveToBin 留在垃圾桶里，文件名不变。
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

// TestArchiveOriginalSkipsSecondTime 归档键由内容寻址文件名推得，重复调用幂等：
// 第二次只走进程内状态，不再发请求（回填任务每轮扫一遍也不会产生额外流量）。
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

// TestArchiveKeyNamespace 归档落在 uploads/original/ 下，键名保持与本地文件名一致。
func TestArchiveKeyNamespace(t *testing.T) {
	s := newWithStore(config.R2Config{Prefix: "uploads"}, newFakeStore())
	if got := s.ArchiveKey("ab12.webp"); got != "uploads/original/ab12.webp" {
		t.Fatalf("ArchiveKey = %q", got)
	}
	// 空前缀时回落到 uploads，保证归档与现役对象仍在同一命名空间下。
	noPrefix := newWithStore(config.R2Config{}, newFakeStore())
	if got := noPrefix.ArchiveKey("ab12.webp"); got != "uploads/original/ab12.webp" {
		t.Fatalf("空前缀 ArchiveKey = %q", got)
	}
}

// TestFindInBin 垃圾桶查找：同名优先，其次兼容 MoveToBin 为防同秒覆盖而加的时间戳前缀版本。
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
	// 只有时间戳前缀版本时也要能找回原件。
	if got, ok := FindInBin(dir, "b.webp"); ok {
		t.Fatalf("无任何匹配不应命中，得到 %q", got)
	}
	writeFile(t, dir, "1700000000001_b.webp", []byte("orig"))
	if got, ok := FindInBin(dir, "b.webp"); !ok || filepath.Base(got) != "1700000000001_b.webp" {
		t.Fatalf("时间戳前缀版本应命中，得到 %q ok=%v", got, ok)
	}
}

// TestMD5OfMetaHandlesMultipartETag 多段上传的 ETag 带 -N 后缀，不能当作内容 md5。
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

package content

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/config"
)

// 这些用例锁死 2026-09-14 那次事故的修复：客户端声明的文件名不可信，
// 任何上传都不能在被覆盖前就动到磁盘上已存在的文件。

func md5Of(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

func newUploadHandler(dir string) *Handler {
	return &Handler{deps: app.Deps{Cfg: config.Config{UploadDir: dir}}}
}

// uploadPart 构造一个与线上同形态的 multipart 文件分区（客户端指定 filename）。
func uploadPart(t *testing.T, filename string, body []byte) (*multipart.Part, func()) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	head := make(textproto.MIMEHeader)
	head.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	head.Set("Content-Type", "image/webp")
	w, err := mw.CreatePart(head)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := w.Write(body); err != nil {
		t.Fatalf("写分区: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	part, err := multipart.NewReader(bytes.NewReader(buf.Bytes()), mw.Boundary()).NextPart()
	if err != nil {
		t.Fatalf("NextPart: %v", err)
	}
	return part, func() { _ = part.Close() }
}

func assertNoUploadTemp(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".upload-") {
			t.Fatalf("残留临时文件 %s", e.Name())
		}
	}
}

// TestSavePartDoesNotClobberExistingFile 复刻线上事故：拿已有内容的文件名，
// 再传一份只有 128KiB 的截断片段。原文件必须一字节未变。
func TestSavePartDoesNotClobberExistingFile(t *testing.T) {
	dir := t.TempDir()
	h := newUploadHandler(dir)

	original := bytes.Repeat([]byte("ORIGINAL-IMAGE-DATA-"), 20000)
	name := md5Of(original) + ".webp"
	origPath := filepath.Join(dir, name)
	if err := os.WriteFile(origPath, original, 0o644); err != nil {
		t.Fatalf("预置原图: %v", err)
	}

	truncated := original[:131072]
	part, done := uploadPart(t, name, truncated)
	defer done()

	got, err := h.savePart(part)
	if err != nil {
		t.Fatalf("savePart: %v", err)
	}

	after, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("原图不见了: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("原图被破坏：%d -> %d 字节", len(original), len(after))
	}
	if got.RelPath == name {
		t.Fatalf("截断内容占用了原文件名 %s", name)
	}
	if want := md5Of(truncated) + ".webp"; got.RelPath != want {
		t.Fatalf("截断内容落盘名 = %s，期望 %s", got.RelPath, want)
	}
	if _, err := os.Stat(filepath.Join(dir, got.RelPath)); err != nil {
		t.Fatalf("新内容未落盘: %v", err)
	}
	assertNoUploadTemp(t, dir)
}

// TestSavePartReusesIdenticalContent 同名且内容一致时走复用，不产生重复文件。
func TestSavePartReusesIdenticalContent(t *testing.T) {
	dir := t.TempDir()
	h := newUploadHandler(dir)

	content := bytes.Repeat([]byte("same-bytes-"), 5000)
	name := md5Of(content) + ".webp"

	for i := 1; i <= 2; i++ {
		part, done := uploadPart(t, name, content)
		got, err := h.savePart(part)
		done()
		if err != nil {
			t.Fatalf("第 %d 次上传: %v", i, err)
		}
		if got.RelPath != name {
			t.Fatalf("第 %d 次落盘名 = %s，期望 %s", i, got.RelPath, name)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("同内容两次上传应只有 1 个文件，实际 %d", len(entries))
	}
	got, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("读回: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("落盘内容与上传内容不一致")
	}
	assertNoUploadTemp(t, dir)
}

// TestSavePartRepairsTruncatedExistingFile 内容校验通过时，允许把磁盘上的半截文件修好。
func TestSavePartRepairsTruncatedExistingFile(t *testing.T) {
	dir := t.TempDir()
	h := newUploadHandler(dir)

	content := bytes.Repeat([]byte("full-image-"), 40000)
	name := md5Of(content) + ".webp"
	if err := os.WriteFile(filepath.Join(dir, name), content[:131072], 0o644); err != nil {
		t.Fatalf("预置半截文件: %v", err)
	}

	part, done := uploadPart(t, name, content)
	defer done()

	got, err := h.savePart(part)
	if err != nil {
		t.Fatalf("savePart: %v", err)
	}
	if got.RelPath != name {
		t.Fatalf("落盘名 = %s，期望 %s", got.RelPath, name)
	}
	after, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("读回: %v", err)
	}
	if !bytes.Equal(after, content) {
		t.Fatalf("半截文件未被修复，仍为 %d 字节", len(after))
	}
	assertNoUploadTemp(t, dir)
}

// TestSavePartOversizeLeavesNoFile 超限整单放弃，磁盘上不留任何残文件。
func TestSavePartOversizeLeavesNoFile(t *testing.T) {
	dir := t.TempDir()
	h := newUploadHandler(dir)

	part, done := uploadPart(t, "big.webp", make([]byte, maxUploadSize+1))
	defer done()

	if _, err := h.savePart(part); !errors.Is(err, errTooLarge) {
		t.Fatalf("err = %v，期望 errTooLarge", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("超限后残留 %d 个文件", len(entries))
	}
}

// TestSavePartRandomNameForPlainUpload 非内容寻址的来源仍按随机名兜底落盘。
func TestSavePartRandomNameForPlainUpload(t *testing.T) {
	dir := t.TempDir()
	h := newUploadHandler(dir)

	content := []byte("plain-upload-bytes")
	part, done := uploadPart(t, "my-photo.webp", content)
	defer done()

	got, err := h.savePart(part)
	if err != nil {
		t.Fatalf("savePart: %v", err)
	}
	if got.RelPath == "my-photo.webp" {
		t.Fatal("不应沿用客户端原始文件名")
	}
	if !strings.HasSuffix(got.RelPath, ".webp") {
		t.Fatalf("应保留扩展名，实际 %s", got.RelPath)
	}
	saved, err := os.ReadFile(filepath.Join(dir, got.RelPath))
	if err != nil {
		t.Fatalf("读回: %v", err)
	}
	if !bytes.Equal(saved, content) {
		t.Fatal("落盘内容与上传内容不一致")
	}
	assertNoUploadTemp(t, dir)
}

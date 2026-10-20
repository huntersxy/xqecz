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

func md5Of(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

func newUploadHandler(dir string) *Handler {
	return &Handler{deps: app.Deps{Cfg: config.Config{UploadDir: dir}}}
}

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

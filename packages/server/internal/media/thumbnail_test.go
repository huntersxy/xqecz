package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 这些用例锁死 2026-09-14 那次事故的修复：生成缩略图失败时不得留下 0 字节文件，
// 也不得破坏该内容原有的可用缩略图。

// truncatedWebP 复刻线上那份损坏文件：RIFF 头声明一个很大的 VP8L 块，实际只有 payloadLen 字节。
func truncatedWebP(payloadLen int) []byte {
	const declared = 5_178_438
	payload := make([]byte, payloadLen)
	for i := range payload {
		payload[i] = byte(i * 7)
	}
	b := make([]byte, 0, payloadLen+12)
	b = append(b, 'R', 'I', 'F', 'F')
	b = binary.LittleEndian.AppendUint32(b, declared)
	b = append(b, 'W', 'E', 'B', 'P')
	b = append(b, 'V', 'P', '8', 'L')
	b = binary.LittleEndian.AppendUint32(b, declared-12)
	return append(b, payload...)
}

func writeSourcePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 5), B: uint8((x ^ y) * 3), A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("创建源图: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("编码源图: %v", err)
	}
}

func assertNoThumbJunk(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".thumb-") {
			t.Fatalf("残留临时缩略图 %s", e.Name())
		}
		if info, err := e.Info(); err == nil && info.Size() == 0 {
			t.Fatalf("残留 0 字节文件 %s", e.Name())
		}
	}
}

// TestGenerateThumbnailSuccess 正常源图应产出 800px 宽的可解码 WebP，且不残留临时文件。
func TestGenerateThumbnailSuccess(t *testing.T) {
	root := t.TempDir()
	thumbDir := filepath.Join(root, "thumbs")

	const stem = "6726993ce12dfb2b3a104fd4f65791de"
	src := filepath.Join(t.TempDir(), stem+".png")
	writeSourcePNG(t, src, 1600, 1200)

	rel, err := GenerateThumbnail(context.Background(), src, "image", thumbDir)
	if err != nil {
		t.Fatalf("GenerateThumbnail: %v", err)
	}
	if want := "thumbs/" + stem + "_thumb.webp"; rel != want {
		t.Fatalf("rel = %q，期望 %q", rel, want)
	}

	f, err := os.Open(filepath.Join(thumbDir, stem+"_thumb.webp"))
	if err != nil {
		t.Fatalf("打开产物: %v", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatalf("产物不是可解码图片: %v", err)
	}
	if b := img.Bounds(); b.Dx() != ThumbMaxWidth {
		t.Fatalf("缩略图宽度 = %d，期望 %d", b.Dx(), ThumbMaxWidth)
	}
	assertNoThumbJunk(t, thumbDir)
}

// TestGenerateThumbnailFailureKeepsExistingThumbnail 是事故的直接回归：
// 同一内容先有一张可用缩略图，随后源图变成 128KiB 截断 WebP，再生成时必须报错
// 且已有的可用缩略图一字节未变。
func TestGenerateThumbnailFailureKeepsExistingThumbnail(t *testing.T) {
	root := t.TempDir()
	thumbDir := filepath.Join(root, "thumbs")

	const stem = "6726993ce12dfb2b3a104fd4f65791de"
	out := filepath.Join(thumbDir, stem+"_thumb.webp")

	goodSrc := filepath.Join(t.TempDir(), stem+".png")
	writeSourcePNG(t, goodSrc, 1200, 900)
	if _, err := GenerateThumbnail(context.Background(), goodSrc, "image", thumbDir); err != nil {
		t.Fatalf("首次生成失败: %v", err)
	}
	before, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("读首次产物: %v", err)
	}
	if len(before) == 0 {
		t.Fatal("首次产物为空")
	}

	badSrc := filepath.Join(t.TempDir(), stem+".webp")
	if err := os.WriteFile(badSrc, truncatedWebP(131072), 0o644); err != nil {
		t.Fatalf("写截断源图: %v", err)
	}
	if _, err := GenerateThumbnail(context.Background(), badSrc, "image", thumbDir); err == nil {
		t.Fatal("源图损坏时应当返回错误")
	}

	after, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("可用缩略图被删除: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("可用缩略图被破坏：%d -> %d 字节", len(before), len(after))
	}
	assertNoThumbJunk(t, thumbDir)
}

// TestGenerateThumbnailFailureLeavesNoFile 源图损坏且原本没有缩略图时，失败后不留任何文件
// （旧实现在 ffmpeg 路径下会留下一个 0 字节的 _thumb.webp）。
func TestGenerateThumbnailFailureLeavesNoFile(t *testing.T) {
	root := t.TempDir()
	thumbDir := filepath.Join(root, "thumbs")

	src := filepath.Join(t.TempDir(), "deadbeefdeadbeefdeadbeefdeadbeef.webp")
	if err := os.WriteFile(src, truncatedWebP(4096), 0o644); err != nil {
		t.Fatalf("写截断源图: %v", err)
	}
	if _, err := GenerateThumbnail(context.Background(), src, "image", thumbDir); err == nil {
		t.Fatal("源图损坏时应当返回错误")
	}

	entries, err := os.ReadDir(thumbDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("失败后残留 %d 个文件: %s", len(entries), strings.Join(names, ", "))
	}
}

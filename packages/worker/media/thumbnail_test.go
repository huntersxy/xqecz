package media

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// assertWebP 校验文件是合法的 WebP（RIFF + WEBP 魔数）。
func assertWebP(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read thumb: %v", err)
	}
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Fatalf("not a webp file (len=%d, magic=% x)", len(data), data[:12])
	}
}

func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	src := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 16; x++ {
			src.SetRGBA(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 32), B: 128, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, src); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

// TestGenerateWithGoOutputsWebP 验证无 ffmpeg 的纯 Go 兜底路径输出 WebP。
func TestGenerateWithGoOutputsWebP(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "src.png")
	out := filepath.Join(dir, "src_thumb.webp")
	writeTestPNG(t, in)

	if err := generateWithGo(in, out); err != nil {
		t.Fatalf("generateWithGo: %v", err)
	}
	assertWebP(t, out)
}

// TestGenerateThumbnailOutputsWebP 验证真实入口：ffmpeg 可用走 ffmpeg，
// 缺失/失败走纯 Go，无论哪条路径最终都应产出 *_thumb.webp。
func TestGenerateThumbnailOutputsWebP(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "src.png")
	writeTestPNG(t, in)

	rel, err := GenerateThumbnail(context.Background(), in, "image", dir)
	if err != nil {
		t.Fatalf("GenerateThumbnail: %v", err)
	}
	if !strings.HasSuffix(rel, "_thumb.webp") {
		t.Fatalf("unexpected thumb path: %s", rel)
	}
	assertWebP(t, filepath.Join(dir, filepath.Base(rel)))
}

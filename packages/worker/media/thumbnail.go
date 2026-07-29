package media

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	// 额外图像解码器（标准库不含 WebP/BMP/TIFF）。均为纯 Go，无 CGO 依赖。
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ffmpegAvailable 探测系统是否装有 ffmpeg（启动时一次性探测，之后走缓存）。
var ffmpegAvailable = sync.OnceValue(func() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
})

// GenerateThumbnail 为图片或视频生成 800px 宽的缩略图。
//
// 图片：优先用 ffmpeg（若可用）；ffmpeg 缺失时自动降级为纯 Go 解码→缩放→编码 JPEG，
// 零系统依赖，保证本地/生产环境无需安装 ffmpeg 也能生成图片缩略图。
// 视频：必须使用 ffmpeg（缺失即返回错误，由调用方降级）。
//
// 返回：相对 data 目录的缩略图路径（如 "thumbs/xxx_thumb.jpg"），供 api 映射为 /thumbs/ URL。
func GenerateThumbnail(absPath, contentType, thumbDir string) (string, error) {
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", fmt.Errorf("create thumb dir: %w", err)
	}

	// 视频：只能依赖 ffmpeg 抽帧。
	if contentType == "video" {
		if !ffmpegAvailable() {
			return "", fmt.Errorf("ffmpeg not found in PATH; cannot generate video thumbnail")
		}
		outPath := filepath.Join(thumbDir, thumbName(absPath, "jpg"))
		if err := runFFmpeg(absPath, outPath); err != nil {
			return "", err
		}
		return relThumb(thumbDir, outPath)
	}

	// 图片：ffmpeg 优先（统一缩放），失败/缺失则纯 Go 兜底。
	if ffmpegAvailable() {
		outPath := filepath.Join(thumbDir, thumbName(absPath, "jpg"))
		if err := runFFmpeg(absPath, outPath); err == nil {
			return relThumb(thumbDir, outPath)
		}
		// ffmpeg 失败不致命，继续走纯 Go 路径。
	}

	outPath := filepath.Join(thumbDir, thumbName(absPath, "jpg"))
	if err := generateWithGo(absPath, outPath); err != nil {
		return "", err
	}
	return relThumb(thumbDir, outPath)
}

// thumbName 由源文件名推导缩略图文件名（`<stem>_thumb.jpg`）。
func thumbName(absPath, ext string) string {
	stem := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return stem + "_thumb." + ext
}

// runFFmpeg 用 ffmpeg 把源文件缩放到宽 800（高度按比例）输出为 jpg。
func runFFmpeg(absPath, outPath string) error {
	args := []string{"-y", "-i", absPath, "-vf", "scale=800:-1", outPath}
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %v: %s", err, string(out))
	}
	if _, statErr := os.Stat(outPath); statErr != nil {
		return fmt.Errorf("thumbnail not produced: %w", statErr)
	}
	return nil
}

// relThumb 把绝对输出路径转成相对 data 目录（thumbDir 的父目录）的斜杠路径（如 "thumbs/xxx_thumb.jpg"）。
func relThumb(thumbDir, outPath string) (string, error) {
	rel, err := filepath.Rel(filepath.Dir(thumbDir), outPath)
	if err != nil {
		return "thumbs/" + filepath.Base(outPath), nil
	}
	return filepath.ToSlash(rel), nil
}

// generateWithGo 纯 Go 实现：解码图片 → 缩放到宽 800 → 编码为 JPEG。
// 不依赖任何外部二进制，适合未安装 ffmpeg 的环境（本地/生产通用）。
func generateWithGo(absPath, outPath string) error {
	f, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return fmt.Errorf("invalid image dimensions")
	}

	// 先合成到白底 RGBA，避免 PNG 透明区域在 JPEG 中变黑。
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(rgba, b, img, b.Min, draw.Over)

	const maxW = 800
	dstW := maxW
	dstH := int(math.Round(float64(srcH) * float64(dstW) / float64(srcW)))
	if dstH < 1 {
		dstH = 1
	}

	dst := resizeBilinear(rgba, b, dstW, dstH)

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create thumb: %w", err)
	}
	defer out.Close()
	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("encode jpeg: %w", err)
	}
	return nil
}

// resizeBilinear 用双线性插值把 RGBA 缩放到 dstW×dstH（alpha 视为不透明，因已合成白底）。
func resizeBilinear(src *image.RGBA, b image.Rectangle, dstW, dstH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < dstH; y++ {
		fy := (float64(y)+0.5)*float64(sh)/float64(dstH) - 0.5
		y0 := int(math.Floor(fy))
		if y0 < 0 {
			y0 = 0
		}
		if y0 > sh-1 {
			y0 = sh - 1
		}
		y1 := y0 + 1
		if y1 > sh-1 {
			y1 = sh - 1
		}
		dy := fy - float64(y0)
		for x := 0; x < dstW; x++ {
			fx := (float64(x)+0.5)*float64(sw)/float64(dstW) - 0.5
			x0 := int(math.Floor(fx))
			if x0 < 0 {
				x0 = 0
			}
			if x0 > sw-1 {
				x0 = sw - 1
			}
			x1 := x0 + 1
			if x1 > sw-1 {
				x1 = sw - 1
			}
			dx := fx - float64(x0)
			c00 := src.RGBAAt(b.Min.X+x0, b.Min.Y+y0)
			c10 := src.RGBAAt(b.Min.X+x1, b.Min.Y+y0)
			c01 := src.RGBAAt(b.Min.X+x0, b.Min.Y+y1)
			c11 := src.RGBAAt(b.Min.X+x1, b.Min.Y+y1)
			rr := uint8(clamp8((1-dx)*((1-dy)*float64(c00.R)+dy*float64(c01.R)) + dx*((1-dy)*float64(c10.R)+dy*float64(c11.R))))
			gg := uint8(clamp8((1-dx)*((1-dy)*float64(c00.G)+dy*float64(c01.G)) + dx*((1-dy)*float64(c10.G)+dy*float64(c11.G))))
			bb := uint8(clamp8((1-dx)*((1-dy)*float64(c00.B)+dy*float64(c01.B)) + dx*((1-dy)*float64(c10.B)+dy*float64(c11.B))))
			dst.SetRGBA(x, y, color.RGBA{R: rr, G: gg, B: bb, A: 255})
		}
	}
	return dst
}

func clamp8(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

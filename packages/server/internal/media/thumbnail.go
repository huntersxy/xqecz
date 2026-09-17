// Package media 提供图片/视频的缩略图生成与 WebP 转换，均为进程内纯计算，
// 原 Worker 的 gRPC 边界取消后其能力并入本服务。
package media

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// 纯 Go、零 CGO 的 WebP 编解码实现（原 Worker 使用的同一实现）。
	"github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"

	// 标准库之外的解码器（WebP/BMP/TIFF），均为纯 Go。
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ThumbMaxWidth 是缩略图的目标宽度。
const ThumbMaxWidth = 800

// ffmpegAvailable 启动时探测一次 ffmpeg 是否存在，之后走缓存。
var ffmpegAvailable = sync.OnceValue(func() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
})

// GenerateThumbnail 生成 800px 宽的缩略图，返回相对 data 目录的路径（thumbs/<stem>_thumb.webp）。
//
// 图片：ffmpeg 可用时优先用 ffmpeg；缺失或失败时降级为纯 Go 解码缩放。
// 视频：只能依赖 ffmpeg 抽帧，缺失即返回错误（由调用方降级，不阻断上传）。
//
// 两种实现都只写同目录临时文件，成功后才原子改名覆盖正式缩略图。
// 因此任何失败（源图损坏、ffmpeg 超时、编码中断）都不会在 thumbs/ 里留下半截或 0 字节
// 的文件，也不会破坏该内容原有的可用缩略图。
func GenerateThumbnail(ctx context.Context, absPath, contentType, thumbDir string) (string, error) {
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", fmt.Errorf("create thumb dir: %w", err)
	}

	outPath := filepath.Join(thumbDir, thumbName(absPath, "webp"))

	// 临时文件保留 .webp 后缀：ffmpeg 依赖输出扩展名推断封装格式。
	tmp, err := os.CreateTemp(thumbDir, ".thumb-*.webp")
	if err != nil {
		return "", fmt.Errorf("create temp thumb: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	// 改名成功后临时文件已不存在，这里的清理只会命中失败路径。
	defer func() { _ = os.Remove(tmpPath) }()

	if contentType == "video" {
		if !ffmpegAvailable() {
			return "", fmt.Errorf("ffmpeg not found in PATH; cannot generate video thumbnail")
		}
		if err := runFFmpeg(ctx, absPath, tmpPath); err != nil {
			return "", err
		}
		return commitThumb(tmpPath, outPath, thumbDir)
	}

	var ffErr error
	if ffmpegAvailable() {
		if ffErr = runFFmpeg(ctx, absPath, tmpPath); ffErr == nil {
			return commitThumb(tmpPath, outPath, thumbDir)
		}
		// ffmpeg（-y）在解码前就会建出输出文件，失败时先清掉它再降级。
		_ = os.Remove(tmpPath)
	}
	if err := generateWithGo(absPath, tmpPath); err != nil {
		if ffErr != nil {
			// 两个实现都失败时把 ffmpeg 的原因一并抛出，避免被静默吞掉。
			return "", fmt.Errorf("%w (ffmpeg: %v)", err, ffErr)
		}
		return "", err
	}
	return commitThumb(tmpPath, outPath, thumbDir)
}

// commitThumb 把生成好的临时缩略图原子替换为正式文件，并返回相对 data 目录的路径。
func commitThumb(tmpPath, outPath, thumbDir string) (string, error) {
	if _, err := os.Stat(tmpPath); err != nil {
		return "", fmt.Errorf("thumbnail not produced: %w", err)
	}
	// 临时文件属主可能与目录不一致，统一权限，避免 Web 进程读不到。
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return "", fmt.Errorf("commit thumbnail: %w", err)
	}
	return relThumb(thumbDir, outPath), nil
}

// thumbName 由源文件名推导缩略图名。
func thumbName(absPath, ext string) string {
	stem := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return stem + "_thumb." + ext
}

// runFFmpeg 取首帧并缩放到宽 800，输出 WebP（质量 85）。
func runFFmpeg(ctx context.Context, absPath, outPath string) error {
	ffCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{
		"-y", "-i", absPath,
		"-vf", "scale=800:-1",
		"-frames:v", "1",
		"-c:v", "libwebp", "-quality", "85",
		outPath,
	}
	out, err := exec.CommandContext(ffCtx, "ffmpeg", args...).CombinedOutput()
	if err != nil {
		if ffCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("ffmpeg timed out after 5 minutes")
		}
		return fmt.Errorf("ffmpeg failed: %v: %s", err, string(out))
	}
	if _, statErr := os.Stat(outPath); statErr != nil {
		return fmt.Errorf("thumbnail not produced: %w", statErr)
	}
	return nil
}

// relThumb 把绝对输出路径转为相对 data 目录的斜杠路径。
func relThumb(thumbDir, outPath string) string {
	rel, err := filepath.Rel(filepath.Dir(thumbDir), outPath)
	if err != nil {
		return "thumbs/" + filepath.Base(outPath)
	}
	return filepath.ToSlash(rel)
}

// generateWithGo 纯 Go 路径：解码 → 合成白底 → 缩放到宽 800 → 编码 WebP。
// 不依赖任何外部二进制，ffmpeg 缺失时仍能生成图片缩略图。
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

	// 合成到白底，避免透明区域编码后发黑。
	rgba := image.NewRGBA(image.Rect(0, 0, srcW, srcH))
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Over)

	dstW := ThumbMaxWidth
	if srcW < dstW {
		dstW = srcW
	}
	dstH := int(math.Round(float64(srcH) * float64(dstW) / float64(srcW)))
	if dstH < 1 {
		dstH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), rgba, rgba.Bounds(), xdraw.Over, nil)

	out, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create thumb: %w", err)
	}
	defer out.Close()
	if err := webp.Encode(out, dst, &webp.EncoderOptions{Quality: 85}); err != nil {
		return fmt.Errorf("encode webp: %w", err)
	}
	return nil
}

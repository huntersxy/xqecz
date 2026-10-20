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
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const ThumbMaxWidth = 800

var ffmpegAvailable = sync.OnceValue(func() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
})

func GenerateThumbnail(ctx context.Context, absPath, contentType, thumbDir string) (string, error) {
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		return "", fmt.Errorf("create thumb dir: %w", err)
	}

	outPath := filepath.Join(thumbDir, thumbName(absPath, "webp"))

	tmp, err := os.CreateTemp(thumbDir, ".thumb-*.webp")
	if err != nil {
		return "", fmt.Errorf("create temp thumb: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if contentType == "video" {
		if !ffmpegAvailable() {
			return "", fmt.Errorf("ffmpeg not found in PATH; cannot generate video thumbnail")
		}
		if err := runFFmpeg(ctx, absPath, tmpPath); err != nil {
			return "", err
		}
		return commitThumb(ctx, tmpPath, outPath, thumbDir)
	}

	var ffErr error
	if ffmpegAvailable() {
		if ffErr = runFFmpeg(ctx, absPath, tmpPath); ffErr == nil {
			return commitThumb(ctx, tmpPath, outPath, thumbDir)
		}
		_ = os.Remove(tmpPath)
	}
	if err := generateWithGo(absPath, tmpPath); err != nil {
		if ffErr != nil {
			return "", fmt.Errorf("%w (ffmpeg: %v)", err, ffErr)
		}
		return "", err
	}
	return commitThumb(ctx, tmpPath, outPath, thumbDir)
}

func commitThumb(ctx context.Context, tmpPath, outPath, thumbDir string) (string, error) {
	if _, err := os.Stat(tmpPath); err != nil {
		return "", fmt.Errorf("thumbnail not produced: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, outPath); err != nil {
		return "", fmt.Errorf("commit thumbnail: %w", err)
	}
	if _, err := EnsureAtlasThumbnail(ctx, thumbDir, filepath.Base(outPath)); err != nil {
		slog.Warn("星图缩略图生成失败", "file", filepath.Base(outPath), "err", err)
	}
	if _, err := EnsureMontageThumbnail(ctx, thumbDir, filepath.Base(outPath)); err != nil {
		slog.Warn("走马灯缩略图生成失败", "file", filepath.Base(outPath), "err", err)
	}
	return relThumb(thumbDir, outPath), nil
}

func thumbName(absPath, ext string) string {
	stem := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
	return stem + "_thumb." + ext
}

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

func relThumb(thumbDir, outPath string) string {
	rel, err := filepath.Rel(filepath.Dir(thumbDir), outPath)
	if err != nil {
		return "thumbs/" + filepath.Base(outPath)
	}
	return filepath.ToSlash(rel)
}

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

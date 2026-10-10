package media

import (
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"
)

// 星图缩略图与普通 800px 缩略图分开；策略变化须更换目录版本，避免 CDN 复用旧质量。
const AtlasDirectory = "atlas-v1"
const AtlasMaxEdge = 256
const AtlasQuality = 55

// 走马灯大面积展示，使用更清晰的独立档位，不复用 256px 星图小球。
const MontageDirectory = "montage-v1"
const MontageMaxEdge = 640
const MontageQuality = 78

// 请求补图与启动回填共用生成入口。限制编码并发而非浏览作品数，防首次展开压满服务。
var atlasJobs = make(chan struct{}, 2)
var atlasLocks [64]sync.Mutex

// AtlasThumbnailURL 只为本地普通 WebP 缩略图返回专用地址，不猜原图、视频或第三方路径。
func AtlasThumbnailURL(thumb string) string   { return thumbnailVariantURL(thumb, AtlasDirectory) }
func MontageThumbnailURL(thumb string) string { return thumbnailVariantURL(thumb, MontageDirectory) }

func thumbnailVariantURL(thumb, directory string) string {
	if !strings.HasPrefix(thumb, "/thumbs/") {
		return ""
	}
	name := strings.TrimPrefix(thumb, "/thumbs/")
	if !validAtlasName(name) {
		return ""
	}
	return "/thumbs/" + directory + "/" + name
}

func validAtlasName(name string) bool {
	return name != "" && filepath.Base(name) == name && !strings.ContainsAny(name, "/\\?#") && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".webp")
}

// EnsureAtlasThumbnail 从已存在的普通缩略图缩放，视频也复用抽帧图，不重复解码原媒体。
// 原图更新则重建；已有有效文件直接返回。失败保留旧产物，取消后不得提交新文件。
func EnsureAtlasThumbnail(ctx context.Context, thumbDir, name string) (string, error) {
	return ensureThumbnailVariant(ctx, thumbDir, name, AtlasDirectory, AtlasMaxEdge, AtlasQuality)
}
func EnsureMontageThumbnail(ctx context.Context, thumbDir, name string) (string, error) {
	return ensureThumbnailVariant(ctx, thumbDir, name, MontageDirectory, MontageMaxEdge, MontageQuality)
}

func ensureThumbnailVariant(ctx context.Context, thumbDir, name, directory string, maxEdge, quality int) (string, error) {
	if !validAtlasName(name) {
		return "", fmt.Errorf("invalid atlas thumbnail name")
	}
	source := filepath.Join(thumbDir, name)
	outputDir := filepath.Join(thumbDir, directory)
	output := filepath.Join(outputDir, name)
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(output))
	lock := &atlasLocks[hash.Sum32()%uint32(len(atlasLocks))]
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	st, err := os.Stat(source)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("atlas source unavailable: %v", err)
	}
	rel := "thumbs/" + directory + "/" + name
	if cached, err := os.Stat(output); err == nil && cached.Mode().IsRegular() && cached.Size() > 0 && !cached.ModTime().Before(st.ModTime()) && (directory != MontageDirectory || cached.Size() <= st.Size()) {
		return rel, nil
	}
	select {
	case atlasJobs <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-atlasJobs }()
	f, err := os.Open(source)
	if err != nil {
		return "", err
	}
	img, format, err := image.Decode(f)
	_ = f.Close()
	if err != nil {
		return "", fmt.Errorf("decode atlas source: %w", err)
	}
	bounds := img.Bounds()
	edge := max(bounds.Dx(), bounds.Dy())
	if edge < 1 {
		return "", fmt.Errorf("empty atlas source")
	}
	ratio := math.Min(1, float64(maxEdge)/float64(edge))
	width := max(1, int(math.Round(float64(bounds.Dx())*ratio)))
	height := max(1, int(math.Round(float64(bounds.Dy())*ratio)))
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(outputDir, ".atlas-*.webp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	encodeErr := webp.Encode(tmp, dst, &webp.EncoderOptions{Quality: float32(quality)})
	closeErr := tmp.Close()
	if encodeErr != nil {
		return "", fmt.Errorf("encode atlas: %w", encodeErr)
	}
	if closeErr != nil {
		return "", closeErr
	}
	// 清晰档重编码有时反而增大：原 WebP 更小时保留原清晰度和字节数。
	if directory == MontageDirectory && format == "webp" {
		encoded, statErr := os.Stat(tmpPath)
		if statErr != nil {
			return "", statErr
		}
		if encoded.Size() >= st.Size() {
			original, readErr := os.ReadFile(source)
			if readErr != nil {
				return "", readErr
			}
			if err := os.WriteFile(tmpPath, original, 0o644); err != nil {
				return "", err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, output); err != nil {
		return "", err
	}
	return rel, nil
}

// SweepAtlasThumbnails 补齐已有普通缩略图；仅扫描直属文件，绝不递归处理自己的产物。
// 无 DB 字段迁移、无首次请求全量同步生成；失败继续其它文件，关停及时取消。
func SweepAtlasThumbnails(ctx context.Context, thumbDir string) (ok, failed int, err error) {
	entries, err := os.ReadDir(thumbDir)
	if os.IsNotExist(err) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return ok, failed, ctx.Err()
		}
		if !entry.Type().IsRegular() || !validAtlasName(entry.Name()) {
			continue
		}
		if _, err := EnsureAtlasThumbnail(ctx, thumbDir, entry.Name()); err != nil {
			failed++
		} else if _, err := EnsureMontageThumbnail(ctx, thumbDir, entry.Name()); err != nil {
			failed++
		} else {
			ok++
		}
	}
	return ok, failed, nil
}

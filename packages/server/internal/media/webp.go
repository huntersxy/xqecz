// 上传图片的无损 WebP 转换。
package media

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/deepteams/webp"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ConvertResult 是无损 WebP 转换结果。
type ConvertResult struct {
	AbsPath string
	Size    int64
}

// ConvertNonGifToWebP 把上传的图片无损转为 WebP：
// GIF（动图）与已是 WebP 的文件跳过；其余图片转出同名 .webp 后删除源文件；
// 任何失败都降级为保留源文件并返回 nil，不阻断上传。
func ConvertNonGifToWebP(absPath, mimetype string) *ConvertResult {
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == ".gif" || ext == ".webp" {
		return nil
	}
	if mimetype != "" && !strings.HasPrefix(mimetype, "image/") {
		return nil
	}

	webpPath := strings.TrimSuffix(absPath, ext) + ".webp"

	if err := encodeLosslessWebP(absPath, webpPath); err != nil {
		slog.Warn("webp 无损转换失败，保留原文件", "src", absPath, "err", err)
		_ = os.Remove(webpPath)
		return nil
	}
	if err := os.Remove(absPath); err != nil {
		slog.Warn("webp 源文件删除失败（忽略）", "src", absPath, "err", err)
	}

	st, err := os.Stat(webpPath)
	if err != nil {
		return nil
	}
	return &ConvertResult{AbsPath: webpPath, Size: st.Size()}
}

func encodeLosslessWebP(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	img, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create webp: %w", err)
	}
	defer out.Close()

	// Lossless + Method 4 对应旧实现 sharp 的 webp({ lossless: true, effort: 4 })。
	if err := webp.Encode(out, img, &webp.EncoderOptions{Lossless: true, Quality: 75, Method: 4}); err != nil {
		return fmt.Errorf("encode webp: %w", err)
	}
	return nil
}

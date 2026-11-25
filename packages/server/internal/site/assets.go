package site

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"

	_ "golang.org/x/image/webp"
	_ "image/gif"
	_ "image/jpeg"
)

const (
	AssetLogo    = "logo"
	AssetFavicon = "favicon"

	maxAssetBytes  = 4 << 20
	maxAssetPixels = 20_000_000
	maxLogoEdge    = 1600
	faviconSize    = 256
)

var assetSequence atomic.Uint64

func (s *Service) SaveAsset(kind string, r io.Reader) (string, error) {
	if kind != AssetLogo && kind != AssetFavicon {
		return "", errors.New("不支持的素材类型")
	}
	data, err := io.ReadAll(io.LimitReader(r, maxAssetBytes+1))
	if err != nil {
		return "", errors.New("图片读取失败")
	}
	if len(data) == 0 {
		return "", errors.New("请选择图片")
	}
	if len(data) > maxAssetBytes {
		return "", errors.New("图片不能超过 4MB")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return "", errors.New("仅支持 PNG、JPEG、WebP 或 GIF 图片")
	}
	if cfg.Width > 10000 || cfg.Height > 10000 || cfg.Width*cfg.Height > maxAssetPixels {
		return "", errors.New("图片尺寸过大")
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", errors.New("图片解析失败")
	}

	var encoded []byte
	var ext string
	if kind == AssetFavicon {
		encoded, err = squarePNG(decoded)
		ext = ".png"
	} else {
		encoded, err = logoWebP(decoded)
		ext = ".webp"
	}
	if err != nil {
		return "", errors.New("图片处理失败")
	}

	dir := filepath.Join(s.imagesDir, "site")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", errors.New("无法创建站点素材目录")
	}
	name := fmt.Sprintf("%s-%d-%d%s", kind, time.Now().UTC().UnixNano(), assetSequence.Add(1), ext)
	if err := os.WriteFile(filepath.Join(dir, name), encoded, 0o644); err != nil {
		return "", errors.New("站点素材保存失败")
	}
	return "/images/site/" + name, nil
}

func logoWebP(img image.Image) ([]byte, error) {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width < 1 || height < 1 {
		return nil, errors.New("invalid image")
	}
	if width > maxLogoEdge || height > maxLogoEdge {
		scale := float64(maxLogoEdge) / float64(max(width, height))
		dstW := max(1, int(float64(width)*scale))
		dstH := max(1, int(float64(height)*scale))
		dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
		img = dst
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.EncoderOptions{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func squarePNG(img image.Image) ([]byte, error) {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width < 1 || height < 1 {
		return nil, errors.New("invalid image")
	}
	side := min(width, height)
	src := image.Rect(
		b.Min.X+(width-side)/2,
		b.Min.Y+(height-side)/2,
		b.Min.X+(width-side)/2+side,
		b.Min.Y+(height-side)/2+side,
	)
	dst := image.NewRGBA(image.Rect(0, 0, faviconSize, faviconSize))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, src, draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Service) removeAsset(rawURL string) {
	const prefix = "/images/site/"
	if !strings.HasPrefix(rawURL, prefix) {
		return
	}
	name := strings.TrimPrefix(rawURL, prefix)
	if name == "" || strings.ContainsAny(name, "\\/") || strings.HasPrefix(name, ".") {
		return
	}
	abs := filepath.Join(s.imagesDir, "site", name)
	rel, err := filepath.Rel(s.imagesDir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return
	}
	_ = os.Remove(abs)
}

package aigen

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
	"gorm.io/gorm"
)

var assetFilePattern = regexp.MustCompile(`^[0-9a-f]{32}\.(?:png|jpg|webp)(?:\.preview\.jpg)?$`)

func (s *Service) AssetPath(asset Asset) (string, error) {
	if !assetFilePattern.MatchString(asset.FileName) || filepath.Base(asset.FileName) != asset.FileName || strings.ContainsAny(asset.FileName, `/\`) {
		return "", ErrInvalid
	}
	return filepath.Join(s.dir, asset.FileName), nil
}

// Stream to a private temporary file; the actual decoder chooses the extension,
// not the upload MIME header or filename. Bound pixels as well as compressed bytes.
func (s *Service) saveImage(reader io.Reader, uid, taskID uint64) (Asset, error) {
	asset := Asset{UserID: uid, TaskID: taskID, DownloadAllowed: true}
	f, err := os.CreateTemp(s.dir, ".image-*")
	if err != nil {
		return asset, err
	}
	temp := f.Name()
	defer os.Remove(temp)
	hash := sha256.New()
	n, err := io.Copy(f, io.TeeReader(io.LimitReader(reader, (20<<20)+1), hash))
	if err != nil || n == 0 || n > 20<<20 {
		f.Close()
		return asset, errors.New("图片大小须在 20MB 以内")
	}
	if _, err = f.Seek(0, 0); err != nil {
		f.Close()
		return asset, err
	}
	s.imageSlots <- struct{}{}
	defer func() { <-s.imageSlots }()
	dim, format, err := image.DecodeConfig(f)
	if err != nil || dim.Width < 1 || dim.Height < 1 || dim.Width > 30000 || dim.Height > 30000 || int64(dim.Width)*int64(dim.Height) > 40_000_000 {
		f.Close()
		return asset, errors.New("图片无效或像素数过大")
	}
	_, err = f.Seek(0, 0)
	if err != nil {
		f.Close()
		return asset, err
	}
	decoded, actualFormat, err := image.Decode(f)
	closeErr := f.Close()
	if actualFormat != format {
		return asset, ErrInvalid
	}
	if err != nil || closeErr != nil || dim.Width < 1 || dim.Height < 1 || int64(dim.Width)*int64(dim.Height) > 40_000_000 {
		return asset, errors.New("仅接受有效 PNG、JPEG、WebP 图片，像素数须在 4000 万以内")
	}
	ext, mime := "", ""
	switch format {
	case "png":
		ext, mime = ".png", "image/png"
	case "jpeg":
		ext, mime = ".jpg", "image/jpeg"
	case "webp":
		ext, mime = ".webp", "image/webp"
	default:
		return asset, ErrInvalid
	}
	var token [16]byte
	if _, err = rand.Read(token[:]); err != nil {
		return asset, err
	}
	asset.FileName = hex.EncodeToString(token[:]) + ext
	asset.Digest = hex.EncodeToString(hash.Sum(nil))
	asset.Mime, asset.Bytes, asset.Width, asset.Height = mime, n, dim.Width, dim.Height
	if err = os.Rename(temp, filepath.Join(s.dir, asset.FileName)); err != nil {
		return asset, err
	}
	previewName := asset.FileName + ".preview.jpg"
	if err := writePreview(decoded, filepath.Join(s.dir, previewName)); err == nil {
		asset.PreviewFileName = previewName
	} else {
		slog.Warn("AI private preview unavailable")
	}
	return asset, nil
}

func writePreview(source image.Image, path string) error {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > 512 || height > 512 {
		if width >= height {
			height = height * 512 / width
			width = 512
		} else {
			width = width * 512 / height
			height = 512
		}
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	preview := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(preview, preview.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(preview, preview.Bounds(), source, bounds, draw.Over, nil)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	encodeErr := jpeg.Encode(f, preview, &jpeg.Options{Quality: 78})
	closeErr := f.Close()
	if encodeErr != nil {
		_ = os.Remove(path)
		return encodeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
	}
	return closeErr
}

func (s *Service) Upload(ctx context.Context, uid uint64, reader io.Reader) (Asset, error) {
	if uid == 0 {
		return Asset{}, ErrInvalid
	}
	row, err := s.saveImage(reader, uid, 0)
	if err != nil {
		return row, err
	}
	candidate := row
	duplicate := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := lockAccount(tx, uid); err != nil {
			return err
		}
		var existing Asset
		if tx.Where("user_id = ? AND digest = ? AND task_id = 0 AND archived = ? AND public = ?", uid, row.Digest, false, false).First(&existing).Error == nil {
			row = existing
			duplicate = true
			return nil
		}
		var count int64
		var bytes int64
		if err := tx.Model(&Asset{}).Where("user_id = ? AND task_id = 0 AND created_at >= ?", uid, time.Now().UTC().Truncate(24*time.Hour)).Count(&count).Error; err != nil {
			return err
		}
		if err := tx.Model(&Asset{}).Where("user_id = ?", uid).Select("COALESCE(SUM(bytes), 0)").Scan(&bytes).Error; err != nil {
			return err
		}
		if count >= 50 || bytes+row.Bytes > 512<<20 {
			return ErrQuota
		}
		return tx.Create(&row).Error
	})
	if err != nil || duplicate {
		s.removeFiles([]Asset{candidate})
	}
	return row, err
}
func (s *Service) removeFiles(assets []Asset) {
	for _, row := range assets {
		path, err := s.AssetPath(row)
		if err == nil {
			_ = os.Remove(path)
		}
		if row.PreviewFileName != "" {
			preview, err := s.AssetPath(Asset{FileName: row.PreviewFileName})
			if err == nil {
				_ = os.Remove(preview)
			}
		}
	}
}

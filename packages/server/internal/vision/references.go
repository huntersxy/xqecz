package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
)

type Reference struct {
	Name             string `json:"name"`
	DataURL          string `json:"-"`
	Bytes            int64  `json:"bytes"`
	Version          string `json:"version"`
	Description      string `json:"description"`
	NeedsDescription bool   `json:"needs_description"`
}

func ImportReferences(ctx context.Context, source, dest string) ([]Reference, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, err
	}
	library := &Service{referenceDir: dest}
	out := []Reference{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		f, err := os.Open(filepath.Join(source, entry.Name()))
		if err != nil {
			return nil, err
		}
		ref, err := library.putReference(ctx, name, f, "upsert")
		_ = f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, ref)
	}
	return out, nil
}

func (s *Service) References(ctx context.Context) ([]Reference, error) {
	s.referenceMu.RLock()
	defer s.referenceMu.RUnlock()
	return loadReferences(ctx, s.referenceDir)
}

func loadReferences(ctx context.Context, dir string) ([]Reference, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []Reference{}, nil
	}
	if err != nil {
		return nil, errors.New("角色参照库不可读取")
	}
	out := []Reference{}
	notes, err := loadReferenceNotes(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".jpg") {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(out) >= 12 {
			return nil, errors.New("角色参照图最多 12 张，请精简参照库")
		}
		info, err := os.Lstat(filepath.Join(dir, entry.Name()))
		if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<10 {
			return nil, errors.New("角色参照图不可读取或超过 512KB，请先导入压缩")
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, errors.New("角色参照图读取失败")
		}
		cfg, err := jpeg.DecodeConfig(bytes.NewReader(b))
		if err != nil || cfg.Width > 1024 || cfg.Height > 1024 {
			return nil, errors.New("角色参照图需为最长边不超过 1024 的 JPEG")
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if err := validateReferenceName(name); err != nil {
			return nil, err
		}
		version := referenceVersion(b)
		note := notes[name]
		out = append(out, Reference{Name: name, DataURL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b), Bytes: info.Size(), Version: version, Description: note.Text, NeedsDescription: note.Text == "" || note.Version != version})
	}
	return out, nil
}

func imageSample() string {
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(16, 48, 72, 112), &image.Uniform{C: color.RGBA{30, 100, 220, 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(80, 12, 112, 44), &image.Uniform{C: color.RGBA{255, 200, 40, 255}}, image.Point{}, draw.Src)
	var b bytes.Buffer
	_ = jpeg.Encode(&b, img, &jpeg.Options{Quality: 85})
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

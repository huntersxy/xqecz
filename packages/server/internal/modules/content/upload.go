package content

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxUploadSize = 20 << 20

const maxFieldSize = 1 << 20

var md5NamePattern = regexp.MustCompile(`^[a-f0-9]{32}\.[a-z0-9]{1,8}$`)

var (
	errBadMultipart = errors.New("bad multipart body")
	errTooLarge     = errors.New("file too large")
	errBadFileType  = errors.New("unsupported media type")
)

type uploadedFile struct {
	AbsPath string
	RelPath string
	Size    int64
	Mime    string
}

type form struct {
	Fields map[string][]string
	File   *uploadedFile
}

func (f *form) value(key string) string {
	if v := f.Fields[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func (h *Handler) parseMultipart(c *gin.Context) (*form, error) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		return nil, errBadMultipart
	}

	out := &form{Fields: map[string][]string{}}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errBadMultipart
		}

		name := part.FormName()
		if name == "file" && part.FileName() != "" {
			file, err := h.savePart(part)
			if err != nil {
				_ = part.Close()
				return nil, err
			}
			out.File = file
			_ = part.Close()
			continue
		}

		buf, _ := io.ReadAll(io.LimitReader(part, maxFieldSize))
		_ = part.Close()
		if name != "" {
			out.Fields[name] = append(out.Fields[name], string(buf))
		}
	}
	return out, nil
}

func (h *Handler) savePart(part *multipart.Part) (*uploadedFile, error) {
	mime := part.Header.Get("Content-Type")
	if !strings.HasPrefix(mime, "image/") && !strings.HasPrefix(mime, "video/") {
		return nil, errBadFileType
	}

	dir := h.deps.Cfg.UploadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	declared := strings.ToLower(filepath.Base(part.FileName()))
	ext := strings.ToLower(filepath.Ext(declared))
	if ext == "" {
		ext = ".bin"
	}

	tmp, err := os.CreateTemp(dir, ".upload-*.part")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	sum := md5.New()
	n, err := io.Copy(io.MultiWriter(tmp, sum), io.LimitReader(part, maxUploadSize+1))
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if n > maxUploadSize {
		return nil, errTooLarge
	}

	digest := hex.EncodeToString(sum.Sum(nil))
	filename := randomName() + ext
	if md5NamePattern.MatchString(declared) {
		if strings.TrimSuffix(declared, filepath.Ext(declared)) == digest {
			filename = declared
		} else {
			filename = digest + ext
		}
	}

	abs := filepath.Join(dir, filename)
	if st, statErr := os.Stat(abs); statErr == nil && st.Size() == n {
		return &uploadedFile{AbsPath: abs, RelPath: filename, Size: n, Mime: mime}, nil
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		return nil, err
	}
	committed = true

	return &uploadedFile{AbsPath: abs, RelPath: filename, Size: n, Mime: mime}, nil
}

func (f *form) parseTagsField() []string {
	raw := f.Fields["tags"]
	out := []string{}
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "[") {
			for _, t := range ParseTags(item) {
				add(t)
			}
			continue
		}
		for _, t := range strings.Split(item, ",") {
			add(t)
		}
	}
	return out
}

func (h *Handler) relToUpload(abs string) string {
	rel, err := filepath.Rel(h.deps.Cfg.UploadDir, abs)
	if err != nil {
		return filepath.Base(abs)
	}
	return filepath.ToSlash(rel)
}

func respondUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errTooLarge):
		softFail(c, 400, "文件大小超过 20MB 限制")
	case errors.Is(err, errBadFileType):
		softFail(c, 400, "仅支持图片或视频文件")
	default:
		softFail(c, 400, "请求参数格式错误")
	}
}

package content

import (
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/store"
)

const downloadParam = "download"

var blockedDownloadExt = map[string]bool{
	".html": true, ".htm": true, ".svg": true, ".js": true, ".mjs": true,
	".php": true, ".sh": true, ".bat": true, ".exe": true, ".dll": true,
}

func (h *Handler) RegisterMedia(r *gin.Engine) {
	for _, m := range []struct {
		route string
		dir   string
	}{
		{"/uploads/*filepath", h.deps.Cfg.UploadDir},
		{"/thumbs/*filepath", h.deps.Cfg.ThumbDir},
		{"/images/*filepath", h.deps.Cfg.ImagesDir},
	} {
		dir := m.dir
		serve := func(c *gin.Context) {
			h.serveMedia(c, dir)
		}
		r.GET(m.route, serve)
		r.HEAD(m.route, serve)
	}
}

func (h *Handler) serveMedia(c *gin.Context, dir string) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if rel == "" {
		c.Status(404)
		return
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		c.Status(404)
		return
	}
	abs := filepath.Join(dir, clean)

	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		c.Status(404)
		return
	}

	header := c.Writer.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Set("Cross-Origin-Resource-Policy", "cross-origin")
	header.Del("Access-Control-Allow-Credentials")
	header.Del("Vary")

	if c.Query(downloadParam) == "1" || c.Query(downloadParam) == "true" {
		ext := strings.ToLower(filepath.Ext(abs))
		if blockedDownloadExt[ext] {
			c.Status(403)
			return
		}
		c.Header("Content-Disposition", contentDisposition(h.downloadName(rel, ext)))
		c.Header("Content-Type", "application/octet-stream")
		c.Header("X-Content-Type-Options", "nosniff")
	} else {
		c.Header("Cache-Control", "public, max-age=2592000")
	}

	c.File(abs)
}

func (h *Handler) downloadName(rel, ext string) string {
	var title string
	if err := h.deps.DB.Model(&store.Content{}).
		Where("file_path = ? OR file_path = ?", rel, strings.TrimPrefix(rel, "/")).
		Order("id DESC").Limit(1).Pluck("title", &title).Error; err != nil {
		slog.Warn("查询下载文件名失败", "rel", rel, "err", err)
	}
	name := sanitizeFilename(title)
	if name == "" {
		return filepath.Base(rel)
	}
	return name + ext
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", "\"", "_", "\r", "", "\n", "",
		":", "_", "*", "_", "?", "_", "<", "_", ">", "_", "|", "_",
	)
	s = replacer.Replace(s)
	s = strings.Trim(s, ". ")
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}

func contentDisposition(filename string) string {
	ascii := sanitizeFilename(filename)
	for _, r := range ascii {
		if r > 127 {
			ascii = "download" + filepath.Ext(filename)
			break
		}
	}
	return "attachment; filename=\"" + ascii + "\"; filename*=UTF-8''" + url.PathEscape(filename)
}

func downloadRequested(raw string) bool {
	if raw == "" {
		return false
	}
	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	return raw == "1"
}

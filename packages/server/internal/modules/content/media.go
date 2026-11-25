package content

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

const downloadParam = "download"

var blockedDownloadExt = map[string]bool{
	".html": true, ".htm": true, ".svg": true, ".js": true, ".mjs": true,
	".php": true, ".sh": true, ".bat": true, ".exe": true, ".dll": true,
}

var publicSiteAsset = regexp.MustCompile(`^/images/site/(logo-[0-9]+-[0-9]+\.webp|favicon-[0-9]+-[0-9]+\.png)$`)

func (h *Handler) RegisterMedia(r *gin.Engine) {
	optionalAuth := web.OptionalAuth(h.deps)
	for _, m := range []struct {
		prefix string
		dir    string
	}{
		{"/uploads/", h.deps.Cfg.UploadDir},
		{"/thumbs/", h.deps.Cfg.ThumbDir},
		{"/images/", h.deps.Cfg.ImagesDir},
	} {
		serve := func(c *gin.Context) {
			h.serveMedia(c, m.prefix, m.dir)
		}
		r.GET(m.prefix+"*filepath", optionalAuth, serve)
		r.HEAD(m.prefix+"*filepath", optionalAuth, serve)
	}
}

func (h *Handler) serveMedia(c *gin.Context, prefix, dir string) {
	c.Header("Cache-Control", "private, no-store")
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	clean := filepath.FromSlash(rel)
	if !filepath.IsLocal(clean) || filepath.ToSlash(filepath.Clean(clean)) != rel ||
		strings.HasPrefix(rel, ".") || strings.Contains(rel, "/.") {
		c.Status(http.StatusNotFound)
		return
	}
	file, err := os.OpenInRoot(dir, clean)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		c.Status(http.StatusNotFound)
		return
	}

	path := prefix + rel
	siteAsset := publicSiteAsset.MatchString(path)
	var title string
	public := siteAsset
	if !siteAsset {
		row, allowed, err := h.visibleMedia(c, path)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		if !allowed {
			c.Status(http.StatusNotFound)
			return
		}
		title = row.Title
		public = store.IsPublicContentStatus(row.AuditStatus)
	}

	header := c.Writer.Header()
	header.Set("Cross-Origin-Resource-Policy", "cross-origin")
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	if public {
		header.Set("Access-Control-Allow-Origin", "*")
		header.Del("Access-Control-Allow-Credentials")
		header.Del("Vary")
		header.Set("Cache-Control", "public, no-cache, must-revalidate")
	}
	if siteAsset {
		header.Set("Cache-Control", "public, max-age=2592000")
	}

	ext := strings.ToLower(filepath.Ext(rel))
	contentType := mime.TypeByExtension(ext)
	if !(strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/")) ||
		strings.HasPrefix(contentType, "image/svg+xml") {
		contentType = "application/octet-stream"
	}
	if downloadRequested(c.Query(downloadParam)) {
		if blockedDownloadExt[ext] {
			c.Status(http.StatusForbidden)
			return
		}
		name := sanitizeFilename(title)
		if name == "" {
			name = info.Name()
		} else {
			name += ext
		}
		c.Header("Content-Disposition", contentDisposition(name))
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	http.ServeContent(c.Writer, c.Request, info.Name(), info.ModTime(), file)
}

func (h *Handler) visibleMedia(c *gin.Context, path string) (store.Content, bool, error) {
	keys := []string{path, strings.TrimPrefix(path, "/")}
	if strings.HasPrefix(path, "/uploads/") {
		keys = append(keys, strings.TrimPrefix(path, "/uploads/"))
	}
	var rows []store.Content
	err := h.deps.DB.WithContext(c.Request.Context()).
		Select("file_path", "thumb_path", "audit_status", "user_id", "title").
		Where("file_path IN ? OR thumb_path IN ?", keys, keys).
		Order("id DESC").Find(&rows).Error
	if err != nil {
		return store.Content{}, false, err
	}
	for _, row := range rows {
		if (FileURL(deref(row.FilePath)) == path || FileURL(deref(row.ThumbPath)) == path) && web.CanViewContent(c, row) {
			return row, true, nil
		}
	}
	return store.Content{}, false, nil
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

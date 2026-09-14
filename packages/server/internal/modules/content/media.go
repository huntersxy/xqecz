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

// downloadParam 为查询参数名：带 ?download=1 时以附件形式返回，浏览器直接下载。
const downloadParam = "download"

// 明确禁止下载的扩展名（可执行/脚本），避免把上传目录当成分发通道。
var blockedDownloadExt = map[string]bool{
	".html": true, ".htm": true, ".svg": true, ".js": true, ".mjs": true,
	".php": true, ".sh": true, ".bat": true, ".exe": true, ".dll": true,
}

// RegisterMedia 挂载媒体静态目录，并在 download=1 时附加 Content-Disposition。
// 使用自定义处理器替代 gin 的 r.Static：静态文件服务本身不变，
// 差别只在于能读取查询参数并回填「内容标题」作为下载文件名。
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
		r.GET(m.route, func(c *gin.Context) {
			h.serveMedia(c, dir)
		})
	}
}

func (h *Handler) serveMedia(c *gin.Context, dir string) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if rel == "" {
		c.Status(404)
		return
	}
	// 防止路径穿越：清理后仍须落在目标目录内。
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

// downloadName 优先用内容标题作为下载文件名（下载后是可读的名字而非 md5），
// 查不到对应记录（如缩略图、垃圾桶文件）时回退为原始文件名。
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

// sanitizeFilename 去掉文件系统与响应头都敏感字符。
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
	// 过长标题会撑爆响应头，按字符截断（中文按 3 字节保守处理）。
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}

// contentDisposition 同时给出 ASCII 回退名与 RFC 5987 的 UTF-8 名，
// 保证中文标题在主流浏览器都能正确落盘。
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

// 供测试与调试使用：把查询参数值解析为布尔。
func downloadRequested(raw string) bool {
	if raw == "" {
		return false
	}
	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	return raw == "1"
}

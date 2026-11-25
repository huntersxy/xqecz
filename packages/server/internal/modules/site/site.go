package site

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	sitecfg "github.com/huntersxy/xqecz/server/internal/site"
	"github.com/huntersxy/xqecz/server/internal/web"
)

type Handler struct {
	deps app.Deps
}

type settingsResponse struct {
	sitecfg.Config
	Elements []sitecfg.TextElement `json:"elements"`
}

func Register(api *gin.RouterGroup, deps app.Deps) {
	h := &Handler{deps: deps}
	g := api.Group("/site")
	g.GET("/config", h.config)
	g.GET("/manifest", h.manifest)

	admin := api.Group("/admin/site", web.RequireAuth(deps), web.RequireAdmin())
	admin.GET("/settings", h.settings)
	admin.PUT("/settings", h.save)
	admin.POST("/assets/:kind", h.upload)
}

func (h *Handler) config(c *gin.Context) {
	cfg := sysPublic(c, h.deps)
	c.Header("Cache-Control", "no-cache")
	web.OK(c, cfg, "ok")
}

func (h *Handler) manifest(c *gin.Context) {
	cfg := sysPublic(c, h.deps)
	icon := cfg.FaviconURL
	if icon == "" {
		icon = "/favicon.webp"
	}
	name := cfg.BrowserTitle
	if name == "" {
		name = cfg.Name
	}
	payload := gin.H{
		"name":             name,
		"short_name":       cfg.ShortName,
		"description":      cfg.Description,
		"start_url":        "/",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#ffffff",
		"theme_color":      "#6366f1",
		"orientation":      "portrait-primary",
		"lang":             "zh-CN",
		"categories":       []string{"entertainment", "social"},
		"icons": []gin.H{
			{"src": icon, "sizes": "any", "type": iconMIME(icon), "purpose": "any"},
			{"src": icon, "sizes": "any", "type": iconMIME(icon), "purpose": "maskable"},
		},
		"shortcuts": []gin.H{
			{"name": cfg.Texts["nav.gallery"], "short_name": cfg.Texts["nav.gallery"], "url": "/", "icons": []gin.H{{"src": icon, "sizes": "any", "type": iconMIME(icon)}}},
			{"name": cfg.Texts["nav.publish"], "short_name": cfg.Texts["nav.publish"], "url": "/quick-upload", "icons": []gin.H{{"src": icon, "sizes": "any", "type": iconMIME(icon)}}},
		},
	}
	c.Header("Content-Type", "application/manifest+json; charset=utf-8")
	c.Header("Cache-Control", "no-cache, must-revalidate")
	c.JSON(200, payload)
}

func (h *Handler) settings(c *gin.Context) {
	if h.deps.Site == nil {
		web.SoftFail(c, 503, "站点配置服务未初始化")
		return
	}
	cfg, err := h.deps.Site.Config(c.Request.Context())
	if err != nil {
		web.Fail(c, 500, "站点配置读取失败")
		return
	}
	web.OK(c, settingsPayload(cfg), "ok")
}

func (h *Handler) save(c *gin.Context) {
	if h.deps.Site == nil {
		web.SoftFail(c, 503, "站点配置服务未初始化")
		return
	}
	var req sitecfg.Config
	if err := c.ShouldBindJSON(&req); err != nil {
		web.SoftFail(c, 400, "请求参数格式错误")
		return
	}
	cfg, err := h.deps.Site.Save(c.Request.Context(), req)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, settingsPayload(cfg), "站点外观已保存")
}

func (h *Handler) upload(c *gin.Context) {
	if h.deps.Site == nil {
		web.SoftFail(c, 503, "站点配置服务未初始化")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		web.SoftFail(c, 400, "需要 multipart 图片上传")
		return
	}
	kind := c.Param("kind")
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			web.SoftFail(c, 400, "上传体过大或读取失败")
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			_ = part.Close()
			continue
		}
		url, err := h.deps.Site.SaveAsset(kind, part)
		_ = part.Close()
		if err != nil {
			web.SoftFail(c, 400, err.Error())
			return
		}
		web.OK(c, gin.H{"kind": kind, "url": url}, "素材已上传")
		return
	}
	web.SoftFail(c, 400, "请选择图片")
}

func settingsPayload(cfg sitecfg.Config) settingsResponse {
	return settingsResponse{Config: cfg, Elements: sitecfg.Elements()}
}

func sysPublic(c *gin.Context, deps app.Deps) sitecfg.Config {
	if deps.Site == nil {
		return sitecfg.PublicDefaults()
	}
	return deps.Site.Public(c.Request.Context())
}

func iconMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "image/webp"
	}
}

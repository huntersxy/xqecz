// Package aigen connects AI services to the existing session and admin boundary.
package aigen

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

type Handler struct {
	s       *core.Service
	deps    app.Deps
	content *content.Handler
}

func Register(api *gin.RouterGroup, deps app.Deps, contentHandler *content.Handler) {
	h := &Handler{s: deps.AI, deps: deps, content: contentHandler}
	ready := func(c *gin.Context) {
		if h.s == nil {
			web.Fail(c, 503, "AI 创作暂不可用")
			return
		}
		c.Next()
	}
	public := api.Group("", ready)
	public.GET("/ideas", h.ideas)
	public.GET("/ideas/:id", h.idea)
	public.GET("/ideas/:id/assets/:asset", h.ideaAsset)
	public.GET("/creative/assets/:id", web.OptionalAuth(deps), h.image)
	public.GET("/creative/assets", h.publicAssets)
	g := api.Group("/ai", ready, web.RequireAuth(deps), sessionOnly)
	g.GET("/config", h.config)
	g.GET("/account", h.account)
	g.GET("/ledger", h.ledger)
	g.GET("/sessions", h.sessions)
	g.GET("/sessions/:id", h.getSession)
	g.POST("/sessions", h.createSession)
	g.PATCH("/sessions/:id", h.saveSession)
	g.GET("/tasks", h.tasks)
	g.POST("/tasks", h.submit)
	g.POST("/quote", h.quote)
	g.POST("/tasks/:id/cancel", h.cancel)
	g.GET("/assets", h.assets)
	g.POST("/assets", h.upload)
	g.GET("/assets/:id", h.image)
	g.DELETE("/assets/:id", h.archiveAsset)
	g.POST("/assets/:id/publish", h.publish)
	g.POST("/providers", h.byok)
	g.DELETE("/providers/:id", h.deleteBYOK)
	g.POST("/providers/:id/test", h.testBYOK)
	g.POST("/ideas", h.createIdea)
	g.POST("/ideas/:id/sponsor", h.sponsor)
	g.DELETE("/ideas/:id", h.withdraw)
	a := api.Group("/admin/ai", ready, web.RequireAuth(deps), web.RequireAdmin(), sessionOnly)
	a.GET("/providers", h.adminProviders)
	a.GET("/users", h.grantUsers)
	a.POST("/providers", h.saveProvider)
	a.POST("/providers/:id/test", h.testSite)
	a.GET("/tasks", h.adminTasks)
	a.POST("/tasks/:id/resolve", h.resolve)
	a.POST("/grants", h.grant)
	a.GET("/ledger", h.adminLedger)
	a.GET("/ideas", h.adminIdeas)
	a.POST("/ideas/:id/hide", h.hide)
	a.GET("/ideas/:id/events", h.events)
	a.GET("/assets", h.adminAssets)
	a.PATCH("/assets/:id", h.assetMetadata)
}
func sessionOnly(c *gin.Context) {
	if web.MustIdentity(c).IsAPIKey {
		web.Fail(c, 403, "AI 创作需要网页登录")
		return
	}
	c.Next()
}
func uid(c *gin.Context) uint64    { return web.MustIdentity(c).UID }
func pathID(c *gin.Context) uint64 { id, _ := strconv.ParseUint(c.Param("id"), 10, 64); return id }
func bind(c *gin.Context, dst any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	return c.ShouldBindJSON(dst)
}
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, core.ErrInvalid), errors.Is(err, core.ErrBusy), errors.Is(err, core.ErrQuota), errors.Is(err, core.ErrUnavailable), errors.Is(err, core.ErrConflict):
		web.SoftFail(c, 400, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		web.Fail(c, 404, "记录不存在")
	default:
		web.Fail(c, 500, "操作失败，请重试")
	}
}

// Normalize model times through the site's single timestamp serializer. The
// models keep native time.Time for GORM, and HTTP owns the external contract.
func wire(v any) (any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&out); err != nil {
		return nil, err
	}
	var walk func(any) any
	walk = func(item any) any {
		switch value := item.(type) {
		case []any:
			for i := range value {
				value[i] = walk(value[i])
			}
		case map[string]any:
			for k, v := range value {
				if k == "created_at" || k == "updated_at" {
					if raw, ok := v.(string); ok {
						if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
							value[k] = web.TimeOf(t)
							continue
						}
					}
				}
				value[k] = walk(v)
			}
		}
		return item
	}
	return walk(out), nil
}
func ok(c *gin.Context, v any) {
	out, err := wire(v)
	if err != nil {
		web.Fail(c, 500, "响应无法读取")
		return
	}
	web.OK(c, out, "ok")
}
func (h *Handler) db(c *gin.Context) *gorm.DB { return h.s.Database().WithContext(c.Request.Context()) }

func (h *Handler) config(c *gin.Context) {
	rows, err := h.s.VisibleProviders(c.Request.Context(), uid(c))
	if err != nil {
		fail(c, err)
		return
	}
	out := []any{}
	for _, row := range rows {
		out = append(out, providerView(row))
	}
	ok(c, gin.H{"providers": out})
}
func providerView(row core.ProviderConfig) any {
	return gin.H{"id": row.ID, "revision": row.Revision, "name": row.Name, "model": row.Model, "endpoint": row.Endpoint, "owner": row.UserID, "allowed_provider_id": row.AllowedProviderID, "enabled": row.Enabled, "default": row.Default, "has_key": row.CipherKey != "", "sizes": json.RawMessage(row.Sizes), "qualities": json.RawMessage(row.Qualities), "max_count": row.MaxCount, "max_references": row.MaxReferences, "edit_transport": row.EditTransport, "timeout_seconds": row.TimeoutSeconds, "concurrency": row.Concurrency, "cost": row.Cost, "quality_costs": json.RawMessage(row.QualityCosts)}
}
func (h *Handler) account(c *gin.Context) {
	var row core.Account
	err := h.db(c).First(&row, "user_id = ?", uid(c)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row.UserID = uid(c)
	} else if err != nil {
		fail(c, err)
		return
	}
	ok(c, row)
}
func (h *Handler) ledger(c *gin.Context) { h.listLedger(c, uid(c)) }
func (h *Handler) listLedger(c *gin.Context, user uint64) {
	rows := []core.Ledger{}
	query := h.db(c)
	if user != 0 {
		query = query.Where("user_id = ?", user)
	}
	if err := query.Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}
func (h *Handler) sessions(c *gin.Context) {
	rows := []core.Session{}
	q := h.db(c).Where("user_id = ?", uid(c))
	if before, _ := strconv.ParseUint(c.Query("before_id"), 10, 64); before != 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}
func (h *Handler) getSession(c *gin.Context) {
	var row core.Session
	if err := h.db(c).Where("id = ? AND user_id = ?", pathID(c), uid(c)).First(&row).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, row)
}
func (h *Handler) createSession(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if bind(c, &req) != nil || len([]rune(req.Name)) > 100 {
		fail(c, core.ErrInvalid)
		return
	}
	if req.Name == "" {
		req.Name = "未命名创作"
	}
	row := core.Session{UserID: uid(c), Name: req.Name, Draft: `{}`, Version: 1}
	if err := h.db(c).Create(&row).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, row)
}
func (h *Handler) saveSession(c *gin.Context) {
	var req struct {
		Name    string          `json:"name"`
		Draft   json.RawMessage `json:"draft"`
		Version uint64          `json:"version"`
	}
	if bind(c, &req) != nil || req.Name == "" || len([]rune(req.Name)) > 100 || len(req.Draft) > 64<<10 || !json.Valid(req.Draft) {
		fail(c, core.ErrInvalid)
		return
	}
	result := h.db(c).Model(&core.Session{}).Where("id = ? AND user_id = ? AND version = ?", pathID(c), uid(c), req.Version).Updates(map[string]any{"name": req.Name, "draft": string(req.Draft), "version": req.Version + 1})
	if result.Error != nil {
		fail(c, result.Error)
		return
	}
	if result.RowsAffected != 1 {
		fail(c, core.ErrConflict)
		return
	}
	var row core.Session
	if err := h.db(c).First(&row, pathID(c)).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, row)
}
func (h *Handler) taskViews(c *gin.Context, rows []core.Task) ([]any, error) {
	ids := []uint64{}
	userIDs := []uint64{}
	for _, r := range rows {
		ids = append(ids, r.ID)
		userIDs = append(userIDs, r.UserID)
	}
	// Fetch display names once for the whole page, including historical owners
	// outside the grant selector's limit; never serialize account details.
	users := []store.User{}
	if len(userIDs) > 0 {
		if err := h.db(c).Select("id,username").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
	}
	names := map[uint64]string{}
	for _, user := range users {
		names[user.ID] = user.Username
	}
	assets := []core.Asset{}
	if len(ids) > 0 {
		if err := h.db(c).Where("task_id IN ?", ids).Find(&assets).Error; err != nil {
			return nil, err
		}
	}
	out := []any{}
	for _, row := range rows {
		own := []core.Asset{}
		for _, a := range assets {
			if a.TaskID == row.ID && !a.Archived {
				own = append(own, a)
			}
		}
		// Decode only reference metadata; never return the encrypted provider snapshot.
		var snapshot struct {
			References []core.Asset `json:"references"`
		}
		if err := json.Unmarshal([]byte(row.Snapshot), &snapshot); err != nil {
			snapshot.References = nil
		}
		if snapshot.References == nil {
			snapshot.References = []core.Asset{}
		}
		name := names[row.UserID]
		if name == "" {
			name = "已注销用户"
		}
		out = append(out, gin.H{"task": row, "user": gin.H{"id": row.UserID, "username": name}, "assets": own, "references": snapshot.References})
	}
	return out, nil
}
func (h *Handler) tasks(c *gin.Context) { h.listTasks(c, false) }
func (h *Handler) listTasks(c *gin.Context, admin bool) {
	rows := []core.Task{}
	q := h.db(c)
	if before, _ := strconv.ParseUint(c.Query("before_id"), 10, 64); before != 0 {
		q = q.Where("id < ?", before)
	}
	if !admin {
		q = q.Where("user_id = ?", uid(c))
	}
	if c.Query("sponsored") == "1" {
		q = q.Where("idea_id <> 0")
	}
	if id, _ := strconv.ParseUint(c.Query("session_id"), 10, 64); id != 0 {
		q = q.Where("session_id = ?", id)
	}
	if search := c.Query("search"); search != "" {
		q = q.Where("prompt LIKE ?", "%"+search+"%")
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	views, err := h.taskViews(c, rows)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, views)
}
func (h *Handler) submit(c *gin.Context) {
	var input core.Input
	if bind(c, &input) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	task, err := h.s.Submit(c.Request.Context(), uid(c), input)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, task)
}
func (h *Handler) quote(c *gin.Context) {
	var input core.Input
	if bind(c, &input) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	cost, err := h.s.Quote(c.Request.Context(), uid(c), input)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"cost": cost})
}
func (h *Handler) cancel(c *gin.Context) {
	if err := h.s.Cancel(c.Request.Context(), uid(c), pathID(c)); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) assets(c *gin.Context) {
	rows := []core.Asset{}
	q := h.db(c).Where("user_id = ? AND archived = ?", uid(c), false)
	if before, _ := strconv.ParseUint(c.Query("before_id"), 10, 64); before != 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}
func (h *Handler) publicAssets(c *gin.Context) {
	rows := []core.Asset{}
	q := h.db(c).Where("public = ? AND archived = ?", true, false)
	if raw := c.Query("ids"); raw != "" {
		ids := []uint64{}
		for _, value := range strings.Split(raw, ",") {
			id, err := strconv.ParseUint(value, 10, 64)
			if err != nil || id == 0 || len(ids) >= 40 {
				fail(c, core.ErrInvalid)
				return
			}
			ids = append(ids, id)
		}
		q = q.Where("id IN ?", ids)
	}
	if err := q.Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}
func (h *Handler) upload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, (20<<20)+(64<<10))
	reader, err := c.Request.MultipartReader()
	if err != nil {
		fail(c, core.ErrInvalid)
		return
	}
	var row core.Asset
	seen := false
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fail(c, core.ErrInvalid)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" || seen {
			part.Close()
			fail(c, core.ErrInvalid)
			return
		}
		row, err = h.s.Upload(c.Request.Context(), uid(c), part)
		part.Close()
		if err != nil {
			web.SoftFail(c, 400, "请上传有效的 PNG、JPEG、WebP 图片，最大 20MB、4000 万像素")
			return
		}
		seen = true
	}
	if !seen {
		fail(c, core.ErrInvalid)
		return
	}
	ok(c, row)
}
func (h *Handler) image(c *gin.Context) {
	var row core.Asset
	if err := h.db(c).First(&row, pathID(c)).Error; err != nil {
		fail(c, err)
		return
	}
	identity, authenticated := web.IdentityOf(c)
	if !(row.Public && !row.Archived) && (!authenticated || (row.UserID != identity.UID && !identity.IsAdmin)) {
		web.Fail(c, 404, "图片不存在")
		return
	}
	h.sendImage(c, row, authenticated && (row.UserID == identity.UID || identity.IsAdmin))
}
func (h *Handler) sendImage(c *gin.Context, row core.Asset, owner bool) {
	if c.Query("preview") == "1" && row.PreviewFileName != "" {
		if candidate, err := h.s.AssetPath(core.Asset{FileName: row.PreviewFileName}); err == nil {
			if stat, err := os.Stat(candidate); err == nil && stat.Mode().IsRegular() {
				row.FileName = row.PreviewFileName
			}
		}
	}
	path, err := h.s.AssetPath(row)
	if err != nil {
		web.Fail(c, 404, "图片不存在")
		return
	}
	if c.Query("download") == "1" && !owner && !row.DownloadAllowed {
		web.Fail(c, 403, "此素材未授权下载")
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	mime := "image/png"
	switch filepath.Ext(path) {
	case ".jpg":
		mime = "image/jpeg"
	case ".webp":
		mime = "image/webp"
	}
	c.Header("Content-Type", mime)
	if c.Query("download") == "1" {
		c.Header("Content-Disposition", "attachment; filename=reference"+filepath.Ext(path))
	}
	c.File(path)
}
func (h *Handler) archiveAsset(c *gin.Context) {
	result := h.db(c).Model(&core.Asset{}).Where("id = ? AND user_id = ?", pathID(c), uid(c)).Update("archived", true)
	if result.Error != nil {
		fail(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		web.Fail(c, 404, "图片不存在")
		return
	}
	ok(c, nil)
}

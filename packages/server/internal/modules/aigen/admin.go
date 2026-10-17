package aigen

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/web"
)

type providerRequest struct {
	Revision          uint64           `json:"revision"`
	ID                uint64           `json:"id"`
	Name              string           `json:"name"`
	Adapter           string           `json:"adapter"`
	Endpoint          string           `json:"endpoint"`
	Model             string           `json:"model"`
	Key               string           `json:"api_key"`
	AllowedProviderID uint64           `json:"allowed_provider_id"`
	Enabled           bool             `json:"enabled"`
	Default           bool             `json:"default"`
	Sizes             []string         `json:"sizes"`
	Qualities         []string         `json:"qualities"`
	MaxCount          int              `json:"max_count"`
	MaxReferences     int              `json:"max_references"`
	EditTransport     string           `json:"edit_transport"`
	TimeoutSeconds    int              `json:"timeout_seconds"`
	Concurrency       int              `json:"concurrency"`
	Cost              int64            `json:"cost"`
	QualityCosts      map[string]int64 `json:"quality_costs"`
	Options           map[string]any   `json:"options"`
}

func encode(v any) string { raw, _ := json.Marshal(v); return string(raw) }
func encodeOptions(v map[string]any) string {
	if v == nil {
		return "{}"
	}
	return encode(v)
}
func (r providerRequest) model() core.ProviderConfig {
	return core.ProviderConfig{ID: r.ID, Revision: r.Revision, Name: r.Name, Adapter: r.Adapter, Endpoint: r.Endpoint, Model: r.Model, AllowedProviderID: r.AllowedProviderID, Enabled: r.Enabled, Default: r.Default, Sizes: encode(r.Sizes), Qualities: encode(r.Qualities), MaxCount: r.MaxCount, MaxReferences: r.MaxReferences, EditTransport: r.EditTransport, TimeoutSeconds: r.TimeoutSeconds, Concurrency: r.Concurrency, Cost: r.Cost, QualityCosts: encode(r.QualityCosts), Options: encodeOptions(r.Options)}
}
func (h *Handler) adminProviders(c *gin.Context) {
	rows := []core.ProviderConfig{}
	if err := h.db(c).Where("user_id = 0").Order("id asc").Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	out := []any{}
	for _, row := range rows {
		out = append(out, providerView(row))
	}
	ok(c, out)
}
func (h *Handler) saveProvider(c *gin.Context) {
	var req providerRequest
	if bind(c, &req) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	row, err := h.s.SaveProvider(c.Request.Context(), 0, req.model(), req.Key)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, providerView(row))
}
func (h *Handler) setProviderEnabled(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if bind(c, &req) != nil || req.Enabled == nil {
		fail(c, core.ErrInvalid)
		return
	}
	if err := h.s.SetSiteProviderEnabled(c.Request.Context(), pathID(c), *req.Enabled); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) deleteProvider(c *gin.Context) {
	if err := h.s.DeleteSiteProvider(c.Request.Context(), pathID(c)); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) adminTasks(c *gin.Context) { h.listTasks(c, true) }
func (h *Handler) resolve(c *gin.Context) {
	var req struct {
		Consumed bool   `json:"consumed"`
		Note     string `json:"note"`
	}
	if bind(c, &req) != nil {
		web.SoftFail(c, 400, "核查参数无效，请刷新后重试")
		return
	}
	if err := h.s.Resolve(c.Request.Context(), uid(c), pathID(c), req.Consumed, req.Note); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) grant(c *gin.Context) {
	var req struct {
		UserID   uint64   `json:"user_id"`
		UserIDs  []uint64 `json:"user_ids"`
		AllUsers bool     `json:"all_users"`
		Amount   int64    `json:"amount"`
		Key      string   `json:"idempotency_key"`
		Note     string   `json:"note"`
	}
	if bind(c, &req) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	if req.AllUsers || len(req.UserIDs) > 0 {
		if req.UserID != 0 {
			web.SoftFail(c, 400, "请选择用户或全体用户，不要混用发放方式")
			return
		}
		count, err := h.s.GrantBatch(c.Request.Context(), uid(c), req.UserIDs, req.AllUsers, req.Amount, req.Key, req.Note)
		if err != nil {
			if err == core.ErrInvalid {
				web.SoftFail(c, 400, "选择的用户已不存在或发放额度无效，请刷新用户列表后重试")
				return
			}
			fail(c, err)
			return
		}
		ok(c, gin.H{"count": count})
		return
	}
	if err := h.s.Grant(c.Request.Context(), uid(c), req.UserID, req.Amount, req.Key, req.Note); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) adminLedger(c *gin.Context) {
	user, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)
	h.listLedger(c, user)
}
func (h *Handler) grantUsers(c *gin.Context) {
	var rows []struct {
		ID       uint64 `json:"id"`
		Username string `json:"username"`
	}
	if err := h.db(c).Table("users").Select("id", "username").Order("id asc").Limit(10000).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = make([]struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
		}, 0)
	}
	ok(c, rows)
}
func (h *Handler) adminAssets(c *gin.Context) {
	rows := []core.Asset{}
	if err := h.db(c).Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}
func (h *Handler) assetMetadata(c *gin.Context) {
	var req struct {
		Title           string `json:"title"`
		Description     string `json:"description"`
		License         string `json:"license"`
		Purpose         string `json:"purpose"`
		Public          bool   `json:"public"`
		DownloadAllowed bool   `json:"download_allowed"`
		Archived        bool   `json:"archived"`
	}
	// Metadata errors belong to the asset editor, not the generation/session
	// validation boundary. Keep the site's HTTP 200 + business-code contract.
	message := ""
	switch {
	case bind(c, &req) != nil:
		message = "设定图信息格式无效，请刷新后重试"
	case len([]rune(req.Title)) > 150:
		message = "设定图标题不能超过150字"
	case len([]rune(req.Description)) > 3000:
		message = "设定图描述不能超过3000字"
	case len([]rune(req.License)) > 300:
		message = "来源与授权说明不能超过300字"
	case len([]rune(req.Purpose)) > 80:
		message = "设定图用途不能超过80字"
	case req.Public && strings.TrimSpace(req.License) == "":
		message = "公开素材前请填写来源与授权说明"
	}
	if message != "" {
		web.SoftFail(c, 400, message)
		return
	}
	var asset core.Asset
	if err := h.db(c).First(&asset, pathID(c)).Error; err != nil {
		fail(c, err)
		return
	}
	if req.Public && asset.TaskID != 0 {
		web.SoftFail(c, 400, "生成结果不能设为公开素材，请在 AI 工作台发布到站内")
		return
	}
	if req.Public && asset.UserID != uid(c) {
		web.SoftFail(c, 400, "只能将自己上传的设定图设为公开素材")
		return
	}
	result := h.db(c).Model(&core.Asset{}).Where("id = ?", pathID(c)).Updates(map[string]any{"title": req.Title, "description": req.Description, "license": req.License, "purpose": req.Purpose, "public": req.Public, "download_allowed": req.DownloadAllowed, "archived": req.Archived})
	if result.Error != nil {
		fail(c, result.Error)
		return
	}
	// MySQL may report zero affected rows for unchanged metadata; the asset
	// existence was checked above, so this remains a successful idempotent save.
	ok(c, nil)
}
func (h *Handler) testSite(c *gin.Context) { h.queueTest(c, true) }
func (h *Handler) queueTest(c *gin.Context, site bool) {
	var req struct {
		Key string `json:"idempotency_key"`
	}
	if bind(c, &req) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	task, err := h.s.QueueTest(c.Request.Context(), uid(c), pathID(c), site, req.Key)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, task)
}

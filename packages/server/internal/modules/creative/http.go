package creative

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/app"
	core "github.com/huntersxy/xqecz/server/internal/creative"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

type Handler struct{ deps app.Deps }

func Register(api *gin.RouterGroup, deps app.Deps) {
	h := &Handler{deps: deps}
	g := api.Group("/creative")
	g.GET("/resources", h.publicList)
	g.GET("/resources/:id", h.detail)
	g.POST("/templates/:id/expand", h.expand)
	a := api.Group("/admin/creative", web.RequireAuth(deps), web.RequireAdmin())
	a.GET("/resources", h.adminList)
	a.POST("/resources", h.save)
	a.PUT("/resources/:id", h.save)
	a.DELETE("/resources/:id", h.delete)
}
func (h *Handler) db(c *gin.Context) *gorm.DB { return h.deps.DB.WithContext(c.Request.Context()) }
func id(c *gin.Context) uint64                { v, _ := strconv.ParseUint(c.Param("id"), 10, 64); return v }
func dto(row core.Resource) any {
	return gin.H{"id": row.ID, "kind": row.Kind, "slug": row.Slug, "title": row.Title, "summary": row.Summary, "payload": json.RawMessage(row.Payload), "status": row.Status, "sort_order": row.SortOrder, "version": row.Version, "created_at": web.TimeOf(row.CreatedAt), "updated_at": web.TimeOf(row.UpdatedAt)}
}
func (h *Handler) publicList(c *gin.Context) { h.list(c, false) }
func (h *Handler) adminList(c *gin.Context)  { h.list(c, true) }
func (h *Handler) list(c *gin.Context, admin bool) {
	rows := []core.Resource{}
	q := h.db(c)
	if !admin {
		q = q.Where("status = ?", "published")
	}
	if kind := c.Query("kind"); kind != "" {
		q = q.Where("kind = ?", kind)
	}
	if slug := c.Query("slug"); slug != "" {
		q = q.Where("slug = ?", slug)
	}
	if search := c.Query("search"); search != "" {
		q = q.Where("(title LIKE ? OR summary LIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if err := q.Order("sort_order asc,id desc").Limit(200).Find(&rows).Error; err != nil {
		web.Fail(c, 500, "资源读取失败")
		return
	}
	out := []any{}
	for _, row := range rows {
		out = append(out, dto(row))
	}
	web.OK(c, out, "ok")
}
func (h *Handler) detail(c *gin.Context) {
	var row core.Resource
	if h.db(c).Where("id = ? AND status = ?", id(c), "published").First(&row).Error != nil {
		web.Fail(c, 404, "资源不存在")
		return
	}
	web.OK(c, dto(row), "ok")
}
func (h *Handler) save(c *gin.Context) {
	var req struct {
		core.Resource
		Payload json.RawMessage `json:"payload"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	if c.ShouldBindJSON(&req) != nil {
		web.SoftFail(c, 400, "请求格式无效")
		return
	}
	row := req.Resource
	row.ID = id(c)
	if row.Slug == "" && row.ID == 0 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			web.Fail(c, 500, "资源标识生成失败")
			return
		}
		row.Slug = row.Kind + "-" + hex.EncodeToString(suffix[:])
	}
	row.Payload = string(req.Payload)
	payload, err := core.Validate(row)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	if row.Status == "published" {
		for _, relation := range []struct {
			IDs  []uint64
			Kind string
		}{{payload.CharacterIDs, "character"}, {[]uint64{payload.PresetID}, "preset"}} {
			for _, relatedID := range relation.IDs {
				if relatedID == 0 {
					continue
				}
				var n int64
				if h.db(c).Model(&core.Resource{}).Where("id = ? AND kind = ? AND status = ?", relatedID, relation.Kind, "published").Count(&n).Error != nil || n != 1 {
					web.SoftFail(c, 400, "关联角色或组合尚未发布")
					return
				}
			}
		}
		for _, asset := range payload.AssetIDs {
			var n int64
			if h.db(c).Model(&aigen.Asset{}).Where("id = ? AND public = ? AND download_allowed = ? AND archived = ?", asset, true, true, false).Count(&n).Error != nil || n != 1 {
				web.SoftFail(c, 400, "关联图片须公开并允许用于生成与二创，请检查图片授权")
				return
			}
		}
	}
	if row.ID == 0 {
		row.Version = 1
		if err = h.db(c).Create(&row).Error; err != nil {
			web.SoftFail(c, 400, "保存失败，请检查 Slug 是否重复")
			return
		}
	} else {
		result := h.db(c).Model(&core.Resource{}).Where("id = ? AND kind = ? AND version = ?", row.ID, row.Kind, row.Version).Updates(map[string]any{"slug": row.Slug, "title": row.Title, "summary": row.Summary, "payload": row.Payload, "status": row.Status, "sort_order": row.SortOrder, "version": row.Version + 1})
		if result.Error != nil {
			web.SoftFail(c, 400, "保存失败，请检查 Slug 是否重复")
			return
		}
		if result.RowsAffected != 1 {
			web.SoftFail(c, 409, "资源已更新，请刷新")
			return
		}
		if h.db(c).First(&row, row.ID).Error != nil {
			web.Fail(c, 500, "资源读取失败")
			return
		}
	}
	web.OK(c, dto(row), "已保存")
}

var resourceInUse = errors.New("资源仍被其他资源引用")

func resourceReferences(row core.Resource, target uint64) bool {
	var payload core.Payload
	if json.Unmarshal([]byte(row.Payload), &payload) != nil {
		return false
	}
	if payload.PresetID == target {
		return true
	}
	for _, id := range payload.CharacterIDs {
		if id == target {
			return true
		}
	}
	return false
}

func (h *Handler) delete(c *gin.Context) {
	rid := id(c)
	if rid == 0 {
		web.SoftFail(c, 400, "资源编号无效")
		return
	}
	err := h.db(c).Transaction(func(tx *gorm.DB) error {
		var row core.Resource
		if err := tx.First(&row, rid).Error; err != nil {
			return err
		}
		var rows []core.Resource
		if err := tx.Where("id <> ?", rid).Find(&rows).Error; err != nil {
			return err
		}
		for _, other := range rows {
			if resourceReferences(other, rid) {
				return resourceInUse
			}
		}
		return tx.Delete(&row).Error
	})
	if errors.Is(err, resourceInUse) {
		web.SoftFail(c, 400, "资源仍被其他角色、组合或模板引用，请先解除关联")
		return
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			web.Fail(c, 404, "资源不存在或已删除")
		} else {
			web.Fail(c, 500, "资源删除失败")
		}
		return
	}
	web.OK(c, nil, "已删除")
}

func (h *Handler) expand(c *gin.Context) {
	var row core.Resource
	if h.db(c).Where("id = ? AND kind = ? AND status = ?", id(c), "template", "published").First(&row).Error != nil {
		web.Fail(c, 404, "模板不存在")
		return
	}
	var req struct {
		Values  map[string]string `json:"values"`
		Version uint64            `json:"version"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	if c.ShouldBindJSON(&req) != nil {
		web.SoftFail(c, 400, "变量参数无效")
		return
	}
	if req.Version != row.Version {
		web.SoftFail(c, 409, "模板已更新，请重新打开资源库")
		return
	}
	payload, _ := core.Validate(row)
	prompt, err := core.Expand(payload, req.Values)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, gin.H{"prompt": prompt}, "ok")
}

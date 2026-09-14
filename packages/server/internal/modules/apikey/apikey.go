// Package apikey 实现 API 密钥的创建、列表、更新与删除。
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

type Handler struct {
	deps app.Deps
}

func New(deps app.Deps) *Handler { return &Handler{deps: deps} }

// Register 挂载 /api-keys 路由。
func Register(api *gin.RouterGroup, deps app.Deps) {
	h := New(deps)
	g := api.Group("/api-keys", web.RequireAuth(deps))
	g.POST("", h.create)
	g.GET("", h.list)
	g.PUT("/:id", h.update)
	g.DELETE("/:id", h.remove)
}

type createReq struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) create(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	var req createReq
	_ = c.ShouldBindJSON(&req)

	raw := "xq_" + randomHex(24)
	name := req.Name
	if name == "" {
		name = "default"
	}
	perms, err := json.Marshal(nonNil(req.Permissions))
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	row := store.APIKey{
		UserID:      identity.UID,
		Name:        name,
		KeyPrefix:   raw[:10],
		KeyHash:     sha256Hex(raw),
		Permissions: string(perms),
		IsActive:    1,
	}
	if err := h.deps.DB.WithContext(ctx).Create(&row).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	payload := decorate(row)
	payload["key"] = raw // 完整密钥仅在创建时返回一次
	web.OK(c, payload, "API 密钥已创建")
}

func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	var rows []store.APIKey
	if err := h.deps.DB.WithContext(ctx).Where("user_id = ?", identity.UID).
		Order("id DESC").Find(&rows).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	list := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		list = append(list, decorate(r))
	}
	web.OK(c, gin.H{"list": list}, "ok")
}

type updateReq struct {
	Name        *string   `json:"name"`
	Permissions *[]string `json:"permissions"`
	IsActive    *bool     `json:"is_active"`
}

func (h *Handler) update(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "密钥不存在")
		return
	}
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}

	db := h.deps.DB.WithContext(ctx)
	var row store.APIKey
	err = db.Where("id = ? AND user_id = ?", id, identity.UID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "密钥不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	updates := map[string]any{}
	if req.Name != nil && *req.Name != "" {
		updates["name"] = *req.Name
	}
	if req.Permissions != nil {
		perms, err := json.Marshal(nonNil(*req.Permissions))
		if err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
		updates["permissions"] = string(perms)
	}
	if req.IsActive != nil {
		updates["is_active"] = boolToInt(*req.IsActive)
	}
	if len(updates) > 0 {
		if err := db.Model(&store.APIKey{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
	}

	var updated store.APIKey
	if err := db.First(&updated, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, decorate(updated), "ok")
}

func (h *Handler) remove(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "密钥不存在")
		return
	}
	db := h.deps.DB.WithContext(ctx)
	var row store.APIKey
	err = db.Where("id = ? AND user_id = ?", id, identity.UID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "密钥不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if err := db.Delete(&store.APIKey{}, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, nil, "密钥已删除")
}

// decorate 输出密钥对外字段（完整密钥与哈希均不外泄）。
func decorate(row store.APIKey) gin.H {
	var perms []string
	if err := json.Unmarshal([]byte(row.Permissions), &perms); err != nil || perms == nil {
		perms = []string{}
	}
	var lastUsed any
	if row.LastUsedAt != nil {
		lastUsed = web.TimeOf(*row.LastUsedAt)
	}
	return gin.H{
		"id": row.ID, "name": row.Name, "key_prefix": row.KeyPrefix,
		"permissions": perms, "is_active": row.IsActive == 1,
		"last_used_at": lastUsed, "created_at": web.TimeOf(row.CreatedAt),
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(b)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func boolToInt(b bool) int8 {
	if b {
		return 1
	}
	return 0
}

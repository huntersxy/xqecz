// Package admin 实现管理端接口：审核、用户管理、统计、举报与认领处理、缩略图批处理。
package admin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

type Handler struct {
	deps    app.Deps
	content *content.Handler
}

func New(deps app.Deps, contentHandler *content.Handler) *Handler {
	return &Handler{deps: deps, content: contentHandler}
}

// Register 挂载 /admin 路由（全部要求登录 + 管理员）。
func Register(api *gin.RouterGroup, deps app.Deps, contentHandler *content.Handler) {
	h := New(deps, contentHandler)
	g := api.Group("/admin", web.RequireAuth(deps), web.RequireAdmin())

	g.POST("/audit/:id", h.audit)
	g.GET("/pending", h.pending)
	g.GET("/content/all", h.allContent)
	g.PUT("/content/:id/author", h.updateAuthor)
	g.DELETE("/content/purge", h.purge)
	g.GET("/users", h.users)
	g.GET("/dashboard", h.dashboard)
	g.PUT("/users/:id/role", h.updateRole)
	g.PUT("/users/:id/ban", h.updateBan)
	g.DELETE("/users/:id", h.deleteUser)
	g.GET("/comments/reports", h.reports)
	g.POST("/comments/reports/:id/handle", h.handleReport)
	g.GET("/claims", h.claims)
	g.POST("/claims/:id/handle", h.handleClaim)
	g.POST("/content/:id/regenerate-thumbnail", h.regenerateThumbnail)
	g.POST("/content/regenerate-all-thumbnails", h.regenerateAll)
	g.GET("/content/regenerate-all-thumbnails/status", h.regenerateAllStatus)
	g.POST("/content/refresh-recommend", h.refreshRecommend)
}

type auditReq struct {
	Status string `json:"status"`
	Remark string `json:"remark"`
}

func (h *Handler) audit(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "内容不存在")
		return
	}
	var req auditReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		web.Fail(c, 400, "审核状态取值不合法")
		return
	}
	if _, exists, err := h.content.RowByID(ctx, id); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	} else if !exists {
		web.Fail(c, 404, "内容不存在")
		return
	}
	if err := h.content.SetAuditStatus(ctx, id, req.Status); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	// 与旧实现一致：审核后经详情路径返回最新内容（沿用其缓存与浏览量语义）。
	item, err := h.content.DetailInternal(ctx, id)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, item, "审核完成")
}

func (h *Handler) pending(c *gin.Context) {
	page, err := h.content.ListPage(c.Request.Context(), content.ListOpts{
		Page:        atoiDefault(c.Query("page"), 1),
		PageSize:    atoiDefault(c.Query("page_size"), 20),
		AuditStatus: "pending",
		SortBy:      "created_at",
		Order:       "asc",
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, page, "ok")
}

func (h *Handler) allContent(c *gin.Context) {
	page, err := h.content.ListPage(c.Request.Context(), content.ListOpts{
		Page:        atoiDefault(c.Query("page"), 1),
		PageSize:    atoiDefault(c.Query("page_size"), 20),
		AuditStatus: strings.TrimSpace(c.Query("audit_status")),
		Tag:         strings.TrimSpace(c.Query("tag")),
		Keyword:     strings.TrimSpace(c.Query("keyword")),
		SortBy:      c.Query("sort_by"),
		Order:       c.Query("order"),
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, page, "ok")
}

type authorReq struct {
	UserID uint64 `json:"user_id"`
}

func (h *Handler) updateAuthor(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "内容不存在")
		return
	}
	var req authorReq
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == 0 {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if _, exists, err := h.content.RowByID(ctx, id); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	} else if !exists {
		web.Fail(c, 404, "内容不存在")
		return
	}
	var target store.User
	if err := h.deps.DB.WithContext(ctx).First(&target, req.UserID).Error; err != nil {
		web.Fail(c, 404, "目标用户不存在")
		return
	}
	oldUserID, newUsername, err := h.content.UpdateAuthor(ctx, id, req.UserID)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{"content_id": id, "oldUserId": oldUserID, "newUsername": newUsername}, "ok")
}

func (h *Handler) purge(c *gin.Context) {
	count, err := h.content.PurgeDeleted(c.Request.Context())
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{"count": count}, fmt.Sprintf("已清理 %d 条", count))
}

func (h *Handler) users(c *gin.Context) {
	ctx := c.Request.Context()
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(c.Query("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}
	keyword := strings.TrimSpace(c.Query("keyword"))

	db := h.deps.DB.WithContext(ctx).Model(&store.User{})
	if keyword != "" {
		db = db.Where("username LIKE ?", "%"+keyword+"%")
	}
	var total int64
	// Session 复用同一组条件：Count 与 Find 共用同一份 where，否则总数与列表会不一致。
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	var rows []store.User
	if err := db.Session(&gorm.Session{}).Order("id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	list := make([]gin.H, 0, len(rows))
	for _, u := range rows {
		list = append(list, formatUser(u))
	}
	web.OK(c, gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
		"total_page": totalPages(total, pageSize),
	}, "ok")
}

type roleReq struct {
	IsAdmin bool `json:"is_admin"`
}

func (h *Handler) updateRole(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "用户不存在")
		return
	}
	var req roleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if _, err := h.updateUserField(ctx, id, "is_admin", boolToInt(req.IsAdmin)); err != nil {
		web.Fail(c, errStatus(err), err.Error())
		return
	}
	var u store.User
	if err := h.deps.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, formatUser(u), "ok")
}

type banReq struct {
	IsBanned bool `json:"is_banned"`
}

func (h *Handler) updateBan(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "用户不存在")
		return
	}
	identity := web.MustIdentity(c)
	var req banReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if req.IsBanned && identity.UID == id {
		web.Fail(c, 403, "不能封禁自己")
		return
	}
	if _, err := h.updateUserField(ctx, id, "is_banned", boolToInt(req.IsBanned)); err != nil {
		web.Fail(c, errStatus(err), err.Error())
		return
	}
	var u store.User
	if err := h.deps.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, formatUser(u), "ok")
}

func (h *Handler) deleteUser(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "用户不存在")
		return
	}
	identity := web.MustIdentity(c)
	if identity.UID == id {
		web.Fail(c, 403, "不能删除自己")
		return
	}
	var u store.User
	if err := h.deps.DB.WithContext(ctx).First(&u, id).Error; err != nil {
		web.Fail(c, 404, "用户不存在")
		return
	}
	if err := h.deps.DB.WithContext(ctx).Delete(&store.User{}, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	// 作者信息参与内容装饰，删除用户后失效全部内容缓存。
	h.deps.Redis.ClearAllContentCaches(ctx)
	web.OK(c, nil, "用户已删除")
}

// reports 返回未处理举报（附带评论原文与举报人用户名）。
func (h *Handler) reports(c *gin.Context) {
	ctx := c.Request.Context()
	type row struct {
		ID           uint64    `json:"id"`
		CommentID    uint64    `json:"comment_id"`
		UserID       uint64    `json:"user_id"`
		Reason       string    `json:"reason"`
		Handled      int8      `json:"handled"`
		CreatedAt    time.Time `json:"-"`
		CommentText  *string   `json:"-"`
		UserUsername *string   `json:"-"`
	}
	var rows []row
	err := h.deps.DB.WithContext(ctx).Raw(`
		SELECT cr.id, cr.comment_id, cr.user_id, cr.reason, cr.handled, cr.created_at,
		       c.text AS comment_text, u.username AS user_username
		FROM comment_reports cr
		LEFT JOIN comments c ON c.id = cr.comment_id
		LEFT JOIN users u ON u.id = cr.user_id
		WHERE cr.handled = 0
		ORDER BY cr.created_at DESC`).Scan(&rows).Error
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	list := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		item := gin.H{
			"id": r.ID, "comment_id": r.CommentID, "user_id": r.UserID,
			"reason": r.Reason, "handled": r.Handled != 0,
			"created_at": web.TimeOf(r.CreatedAt),
		}
		if r.CommentText != nil {
			item["Comment"] = gin.H{"id": r.CommentID, "text": *r.CommentText}
		}
		if r.UserUsername != nil {
			item["User"] = gin.H{"id": r.UserID, "username": *r.UserUsername}
		}
		list = append(list, item)
	}
	web.OK(c, list, "ok")
}

func (h *Handler) handleReport(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "举报不存在")
		return
	}
	if err := h.deps.DB.WithContext(c.Request.Context()).Model(&store.CommentReport{}).
		Where("id = ?", id).Update("handled", 1).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, nil, "举报已处理")
}

func (h *Handler) claims(c *gin.Context) {
	ctx := c.Request.Context()
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(c.Query("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}

	db := h.deps.DB.WithContext(ctx).Model(&store.Claim{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	var rows []store.Claim
	if err := db.Session(&gorm.Session{}).Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	contents, users := h.loadClaimRefs(ctx, rows)
	list := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		item := gin.H{
			"id": r.ID, "content_id": r.ContentID, "user_id": r.UserID,
			"reason": deref(r.Reason), "status": r.Status,
			"approved_by": r.ApprovedBy, "remark": deref(r.Remark),
			"created_at": web.TimeOf(r.CreatedAt), "updated_at": web.TimeOf(r.UpdatedAt),
		}
		if c, ok := contents[r.ContentID]; ok {
			item["content"] = h.content.DecorateOne(ctx, c, true)
		} else {
			item["content"] = nil
		}
		if u, ok := users[r.UserID]; ok {
			item["user"] = gin.H{"id": u.ID, "username": u.Username}
		} else {
			item["user"] = gin.H{"id": r.UserID, "username": "unknown"}
		}
		list = append(list, item)
	}

	web.OK(c, gin.H{
		"list": list, "total": total, "page": page, "page_size": pageSize,
		"total_page": totalPages(total, pageSize),
	}, "ok")
}

type claimHandleReq struct {
	Action *string `json:"action"`
	Remark string  `json:"remark"`
}

func (h *Handler) handleClaim(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "认领申请不存在")
		return
	}
	identity := web.MustIdentity(c)
	var req claimHandleReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Action == nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	action := *req.Action
	if action != "approve" && action != "reject" {
		web.Fail(c, 400, "操作类型取值不合法")
		return
	}

	var claim store.Claim
	err := h.deps.DB.WithContext(ctx).First(&claim, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "认领申请不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	status := "rejected"
	if action == "approve" {
		status = "approved"
	}
	operator := identity.UID
	remark := req.Remark
	if err := h.deps.DB.WithContext(ctx).Model(&store.Claim{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "remark": remark, "approved_by": operator,
	}).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if action == "approve" {
		if _, _, err := h.content.UpdateAuthor(ctx, claim.ContentID, claim.UserID); err != nil {
			slogWarn("认领通过后变更作者失败", err)
		}
	}
	web.OK(c, nil, "认领已处理")
}

func (h *Handler) regenerateThumbnail(c *gin.Context) {
	ctx := c.Request.Context()
	id, ok := pathID(c)
	if !ok {
		web.Fail(c, 404, "内容不存在")
		return
	}
	if _, exists, err := h.content.RowByID(ctx, id); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	} else if !exists {
		web.Fail(c, 404, "内容不存在")
		return
	}
	item, err := h.content.RegenerateThumbnail(ctx, id)
	if errors.Is(err, content.ErrNoOriginalFile()) {
		web.Fail(c, 400, "无原始文件")
		return
	}
	if err != nil {
		web.Fail(c, 400, err.Error())
		return
	}
	web.OK(c, item, "缩略图已生成")
}

func (h *Handler) regenerateAll(c *gin.Context) {
	res := h.content.RegenerateAllThumbnails(c.Request.Context())
	message := fmt.Sprintf("已开始处理 %d 条", res.Count)
	if res.Running {
		message = "批量生成已在后台进行中"
	}
	web.OK(c, res, message)
}

func (h *Handler) regenerateAllStatus(c *gin.Context) {
	web.OK(c, h.content.RegenerateAllStatus(c.Request.Context()), "ok")
}

func (h *Handler) refreshRecommend(c *gin.Context) {
	if err := h.content.RefreshRecommend(c.Request.Context()); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, nil, "推荐位已刷新")
}

// updateUserField 更新单个用户字段，用户不存在时返回错误。
func (h *Handler) updateUserField(ctx context.Context, id uint64, column string, value int8) (store.User, error) {
	db := h.deps.DB.WithContext(ctx)
	var u store.User
	if err := db.First(&u, id).Error; err != nil {
		return u, errUserNotFound
	}
	if err := db.Model(&store.User{}).Where("id = ?", id).Update(column, value).Error; err != nil {
		return u, err
	}
	return u, nil
}

var errUserNotFound = errors.New("用户不存在")

func errStatus(err error) int {
	if errors.Is(err, errUserNotFound) {
		return 404
	}
	return 500
}

func (h *Handler) loadClaimRefs(ctx context.Context, rows []store.Claim) (map[uint64]store.Content, map[uint64]store.User) {
	contentIDs := []uint64{}
	userIDs := []uint64{}
	seenC := map[uint64]bool{}
	seenU := map[uint64]bool{}
	for _, r := range rows {
		if !seenC[r.ContentID] {
			seenC[r.ContentID] = true
			contentIDs = append(contentIDs, r.ContentID)
		}
		if !seenU[r.UserID] {
			seenU[r.UserID] = true
			userIDs = append(userIDs, r.UserID)
		}
	}
	contents := map[uint64]store.Content{}
	users := map[uint64]store.User{}
	if len(contentIDs) > 0 {
		var cs []store.Content
		if err := h.deps.DB.WithContext(ctx).Unscoped().Where("id IN ?", contentIDs).Find(&cs).Error; err == nil {
			for _, c := range cs {
				contents[c.ID] = c
			}
		}
	}
	if len(userIDs) > 0 {
		var us []store.User
		if err := h.deps.DB.WithContext(ctx).Where("id IN ?", userIDs).Find(&us).Error; err == nil {
			for _, u := range us {
				users[u.ID] = u
			}
		}
	}
	return contents, users
}

func formatUser(u store.User) gin.H {
	out := gin.H{
		"id": u.ID, "username": u.Username,
		"is_admin": u.IsAdmin == 1, "is_banned": u.IsBanned == 1,
		"created_at": web.TimeOf(u.CreatedAt), "updated_at": web.TimeOf(u.UpdatedAt),
	}
	if u.Email != nil && *u.Email != "" {
		out["email"] = *u.Email
	}
	return out
}

func pathID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func totalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 1
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func boolToInt(b bool) int8 {
	if b {
		return 1
	}
	return 0
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func slogWarn(msg string, err error) {
	slog.Warn(msg, "err", err)
}

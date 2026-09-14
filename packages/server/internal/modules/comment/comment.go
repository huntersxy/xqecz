// Package comment 实现树形评论、评论计数与举报。
package comment

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

const cacheTTL = 300 * time.Second

type Handler struct {
	deps app.Deps
}

func New(deps app.Deps) *Handler { return &Handler{deps: deps} }

// Register 挂载 /comment 路由。
func Register(api *gin.RouterGroup, deps app.Deps) {
	h := New(deps)
	g := api.Group("/comment")
	g.GET("/list/:content_id", h.list)
	g.GET("/count/:content_id", h.count)
	g.POST("/add", web.RequireAuth(deps), h.add)
	g.DELETE("/:id", web.RequireAuth(deps), h.remove)
	g.POST("/report", web.RequireAuth(deps), h.report)
}

// UserBrief 是评论内嵌的作者信息。
type UserBrief struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// DTO 是单条评论的对外形状。
type DTO struct {
	ID        uint64    `json:"id"`
	ContentID uint64    `json:"content_id"`
	UserID    uint64    `json:"user_id"`
	Text      string    `json:"text"`
	ParentID  *uint64   `json:"parent_id"`
	IsBanned  bool      `json:"is_banned"`
	CreatedAt web.Time  `json:"created_at"`
	UpdatedAt web.Time  `json:"updated_at"`
	User      UserBrief `json:"user"`
}

// TopDTO 是顶层评论（额外携带一层回复列表）。
type TopDTO struct {
	DTO
	Replies []DTO `json:"replies"`
}

// Page 是评论分页响应。
type Page struct {
	List      []TopDTO `json:"list"`
	Total     int64    `json:"total"`
	Page      int      `json:"page"`
	PageSize  int      `json:"page_size"`
	TotalPage int      `json:"total_page"`
}

func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	contentID, err := strconv.ParseUint(c.Param("content_id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(c.Query("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}

	key := "comments:" + strconv.FormatUint(contentID, 10) + ":" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize)
	data, err := cache.GetOrSetJSON(ctx, h.deps.Redis, key, cacheTTL, func() (Page, error) {
		return h.query(ctx, contentID, page, pageSize)
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, data, "ok")
}

func (h *Handler) query(ctx context.Context, contentID uint64, page, pageSize int) (Page, error) {
	db := h.deps.DB.WithContext(ctx)

	base := db.Model(&store.Comment{}).
		Where("content_id = ? AND parent_id IS NULL AND is_banned = 0", contentID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return Page{}, err
	}

	var tops []store.Comment
	if err := db.Where("content_id = ? AND parent_id IS NULL AND is_banned = 0", contentID).
		Order("created_at ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&tops).Error; err != nil {
		return Page{}, err
	}

	topIDs := make([]uint64, 0, len(tops))
	for _, t := range tops {
		topIDs = append(topIDs, t.ID)
	}

	var replies []store.Comment
	if len(topIDs) > 0 {
		if err := db.Where("parent_id IN ? AND is_banned = 0", topIDs).
			Order("created_at ASC").Find(&replies).Error; err != nil {
			return Page{}, err
		}
	}

	// 批量取作者，消除 N+1。
	ids := make([]uint64, 0, len(tops)+len(replies))
	for _, r := range append(append([]store.Comment{}, tops...), replies...) {
		if r.UserID > 0 {
			ids = append(ids, r.UserID)
		}
	}
	users := map[uint64]store.User{}
	if len(ids) > 0 {
		var rows []store.User
		if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
			return Page{}, err
		}
		for _, u := range rows {
			users[u.ID] = u
		}
	}

	replyMap := map[uint64][]DTO{}
	for _, r := range replies {
		if r.ParentID == nil {
			continue
		}
		replyMap[*r.ParentID] = append(replyMap[*r.ParentID], toDTO(r, users))
	}

	list := make([]TopDTO, 0, len(tops))
	for _, t := range tops {
		children := replyMap[t.ID]
		if children == nil {
			children = []DTO{}
		}
		list = append(list, TopDTO{DTO: toDTO(t, users), Replies: children})
	}

	totalPage := 1
	if pageSize > 0 {
		totalPage = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return Page{List: list, Total: total, Page: page, PageSize: pageSize, TotalPage: totalPage}, nil
}

func (h *Handler) count(c *gin.Context) {
	ctx := c.Request.Context()
	contentID, err := strconv.ParseUint(c.Param("content_id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	key := "comment_count:" + strconv.FormatUint(contentID, 10)
	data, err := cache.GetOrSetJSON(ctx, h.deps.Redis, key, cacheTTL, func() (gin.H, error) {
		var n int64
		if err := h.deps.DB.WithContext(ctx).Model(&store.Comment{}).
			Where("content_id = ? AND is_banned = 0", contentID).Count(&n).Error; err != nil {
			return nil, err
		}
		return gin.H{"content_id": contentID, "count": n}, nil
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, data, "ok")
}

type addReq struct {
	ContentID uint64  `json:"content_id"`
	Text      string  `json:"text"`
	ParentID  *uint64 `json:"parent_id"`
}

func (h *Handler) add(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	var req addReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if req.ContentID == 0 || strings.TrimSpace(req.Text) == "" {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}

	row := store.Comment{
		ContentID: req.ContentID,
		UserID:    identity.UID,
		Text:      strings.TrimSpace(req.Text),
		ParentID:  req.ParentID,
	}
	db := h.deps.DB.WithContext(ctx)
	if err := db.Create(&row).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	h.deps.Redis.ClearCommentCache(ctx, req.ContentID)

	users := map[uint64]store.User{}
	var u store.User
	if err := db.First(&u, identity.UID).Error; err == nil {
		users[u.ID] = u
	}
	web.OK(c, toDTO(row, users), "评论成功")
}

func (h *Handler) remove(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "评论不存在")
		return
	}
	identity := web.MustIdentity(c)

	var row store.Comment
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "评论不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if identity.UID != row.UserID && !identity.IsAdmin {
		web.Fail(c, 403, "无权删除该评论")
		return
	}
	if err := h.deps.DB.WithContext(ctx).Delete(&store.Comment{}, id).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	h.deps.Redis.ClearCommentCache(ctx, row.ContentID)
	web.OK(c, nil, "已删除")
}

type reportReq struct {
	CommentID uint64 `json:"comment_id"`
	Reason    string `json:"reason"`
}

func (h *Handler) report(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	var req reportReq
	if err := c.ShouldBindJSON(&req); err != nil || req.CommentID == 0 {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}

	report := store.CommentReport{CommentID: req.CommentID, UserID: identity.UID, Reason: req.Reason}
	if err := h.deps.DB.WithContext(ctx).Create(&report).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{
		"id":         report.ID,
		"comment_id": report.CommentID,
		"user_id":    report.UserID,
		"reason":     report.Reason,
		"handled":    report.Handled,
		"created_at": web.TimeOf(report.CreatedAt),
	}, "举报已提交")
}

func toDTO(row store.Comment, users map[uint64]store.User) DTO {
	var author UserBrief
	if u, ok := users[row.UserID]; ok {
		author = UserBrief{ID: u.ID, Username: u.Username}
	} else {
		author = UserBrief{ID: row.UserID, Username: "unknown"}
	}
	return DTO{
		ID:        row.ID,
		ContentID: row.ContentID,
		UserID:    row.UserID,
		Text:      row.Text,
		ParentID:  row.ParentID,
		IsBanned:  row.IsBanned == 1,
		CreatedAt: web.TimeOf(row.CreatedAt),
		UpdatedAt: web.TimeOf(row.UpdatedAt),
		User:      author,
	}
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

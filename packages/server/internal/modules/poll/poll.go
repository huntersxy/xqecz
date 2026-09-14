// Package poll 实现投票的列表、详情、投票、创建与删除。
package poll

import (
	"context"
	"crypto/rand"
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

// visitorCookie 是游客投票去重用的标识 Cookie。
const visitorCookie = "visitor_id"

type Handler struct {
	deps app.Deps
}

func New(deps app.Deps) *Handler { return &Handler{deps: deps} }

// Register 挂载 /poll 路由。
func Register(api *gin.RouterGroup, deps app.Deps) {
	h := New(deps)
	g := api.Group("/poll")
	g.GET("/list", h.list)
	g.GET("/:id", web.OptionalAuth(deps), h.detailHandler)
	g.POST("/:id/vote", web.OptionalAuth(deps), h.vote)
	g.POST("/create", web.RequireAuth(deps), h.create)
	g.DELETE("/:id", web.RequireAuth(deps), h.remove)
}

// UserBrief 是投票内嵌的发起人信息。
type UserBrief struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// DTO 是投票的对外形状。
type DTO struct {
	ID          uint64     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Options     []string   `json:"options"`
	VoteCount   int64      `json:"vote_count"`
	UserID      uint64     `json:"user_id"`
	User        *UserBrief `json:"user,omitempty"`
	CreatedAt   web.Time   `json:"created_at"`
	UpdatedAt   web.Time   `json:"updated_at"`
}

// Detail 是投票详情响应。
type Detail struct {
	Poll       DTO              `json:"poll"`
	VoteCounts map[string]int64 `json:"vote_counts"`
	TotalVotes int64            `json:"total_votes"`
	MyVote     *int             `json:"my_vote"`
}

// Page 是投票分页响应。
type Page struct {
	List      []DTO `json:"list"`
	Total     int64 `json:"total"`
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	TotalPage int   `json:"total_page"`
}

func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	page := atoiDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := atoiDefault(c.Query("page_size"), 20)
	if pageSize < 1 {
		pageSize = 20
	}

	db := h.deps.DB.WithContext(ctx).Model(&store.Poll{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	var rows []store.Poll
	if err := h.deps.DB.WithContext(ctx).Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	users := h.userMap(ctx, rows)
	list := make([]DTO, 0, len(rows))
	for _, r := range rows {
		list = append(list, decorate(r, users))
	}

	totalPage := 1
	if pageSize > 0 {
		totalPage = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	web.OK(c, Page{List: list, Total: total, Page: page, PageSize: pageSize, TotalPage: totalPage}, "ok")
}

func (h *Handler) detailHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "投票不存在")
		return
	}
	identity, _ := web.IdentityOf(c)
	visitor, _ := c.Cookie(visitorCookie)

	data, err := h.detail(c.Request.Context(), id, identity.UID, visitor)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "投票不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, data, "ok")
}

type voteReq struct {
	OptionIndex *int `json:"option_index"`
}

func (h *Handler) vote(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "投票不存在")
		return
	}
	var req voteReq
	if err := c.ShouldBindJSON(&req); err != nil || req.OptionIndex == nil {
		web.Fail(c, 400, "选项不存在")
		return
	}
	identity, _ := web.IdentityOf(c)

	var row store.Poll
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "投票不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	options := parseOptions(row.Options)
	idx := *req.OptionIndex
	if idx < 0 || idx >= len(options) {
		web.Fail(c, 400, "选项不存在")
		return
	}

	// 已登录用户按 user_id 去重，游客按 visitor_id 去重（必要时下发 Cookie）。
	visitor := ""
	if identity.UID == 0 {
		visitor = h.ensureVisitor(c)
	}

	db := h.deps.DB.WithContext(ctx)
	var existing store.PollVote
	q := db.Where("poll_id = ?", id)
	if identity.UID > 0 {
		q = q.Where("user_id = ?", identity.UID)
	} else {
		q = q.Where("visitor_id = ?", visitor)
	}
	err = q.First(&existing).Error

	switch {
	case err == nil:
		if existing.OptionIndex != idx {
			if err := db.Model(&store.PollVote{}).Where("id = ?", existing.ID).
				Update("option_index", idx).Error; err != nil {
				web.Fail(c, 500, "服务异常")
				return
			}
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		vote := store.PollVote{PollID: id, OptionIndex: idx}
		if identity.UID > 0 {
			uid := identity.UID
			vote.UserID = &uid
		} else {
			v := visitor
			vote.VisitorID = &v
		}
		if err := db.Create(&vote).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
		if err := db.Model(&store.Poll{}).Where("id = ?", id).
			UpdateColumn("vote_count", gorm.Expr("vote_count + 1")).Error; err != nil {
			web.Fail(c, 500, "服务异常")
			return
		}
	default:
		web.Fail(c, 500, "服务异常")
		return
	}

	data, err := h.detail(ctx, id, identity.UID, visitor)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, data, "ok")
}

type createReq struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
}

func (h *Handler) create(c *gin.Context) {
	ctx := c.Request.Context()
	identity := web.MustIdentity(c)
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if len(req.Options) < 2 {
		web.Fail(c, 400, "至少需要两个选项")
		return
	}

	encoded, err := json.Marshal(req.Options)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	row := store.Poll{
		Title:       strings.TrimSpace(req.Title),
		Description: &req.Description,
		Options:     string(encoded),
		UserID:      identity.UID,
	}
	if err := h.deps.DB.WithContext(ctx).Create(&row).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	users := h.userMap(ctx, []store.Poll{row})
	web.OK(c, decorate(row, users), "创建成功")
}

func (h *Handler) remove(c *gin.Context) {
	ctx := c.Request.Context()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "投票不存在")
		return
	}
	identity := web.MustIdentity(c)

	var row store.Poll
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "投票不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if identity.UID != row.UserID && !identity.IsAdmin {
		web.Fail(c, 403, "无权删除该投票")
		return
	}
	// 物理删除：投票记录一并清理，避免留下悬空的 poll_votes。
	if err := h.deps.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("poll_id = ?", id).Delete(&store.PollVote{}).Error; err != nil {
			return err
		}
		return tx.Delete(&store.Poll{}, id).Error
	}); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, nil, "已删除")
}

// detail 组装投票详情：各选项票数、总票数与本投票人的选择。
func (h *Handler) detail(ctx context.Context, id, uid uint64, visitor string) (Detail, error) {
	var row store.Poll
	if err := h.deps.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return Detail{}, err
	}

	type countRow struct {
		OptionIndex int
		C           int64
	}
	var rows []countRow
	if err := h.deps.DB.WithContext(ctx).Model(&store.PollVote{}).
		Select("option_index, COUNT(*) AS c").
		Where("poll_id = ?", id).
		Group("option_index").
		Scan(&rows).Error; err != nil {
		return Detail{}, err
	}
	counts := map[string]int64{}
	var total int64
	for _, r := range rows {
		counts[strconv.Itoa(r.OptionIndex)] = r.C
		total += r.C
	}

	var myVote *int
	q := h.deps.DB.WithContext(ctx).Where("poll_id = ?", id)
	if uid > 0 {
		q = q.Where("user_id = ?", uid)
	} else if visitor != "" {
		q = q.Where("visitor_id = ?", visitor)
	} else {
		q = nil
	}
	if q != nil {
		var vote store.PollVote
		if err := q.First(&vote).Error; err == nil {
			idx := vote.OptionIndex
			myVote = &idx
		}
	}

	users := h.userMap(ctx, []store.Poll{row})
	return Detail{
		Poll:       decorate(row, users),
		VoteCounts: counts,
		TotalVotes: total,
		MyVote:     myVote,
	}, nil
}

// ensureVisitor 读取或下发游客标识 Cookie（一年有效），用于投票去重。
func (h *Handler) ensureVisitor(c *gin.Context) string {
	if v, err := c.Cookie(visitorCookie); err == nil && v != "" {
		return v
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	v := hex.EncodeToString(b)
	web.SetCookie(c, visitorCookie, v, 365*24*3600, true)
	return v
}

func (h *Handler) userMap(ctx context.Context, rows []store.Poll) map[uint64]store.User {
	ids := []uint64{}
	seen := map[uint64]bool{}
	for _, r := range rows {
		if r.UserID > 0 && !seen[r.UserID] {
			seen[r.UserID] = true
			ids = append(ids, r.UserID)
		}
	}
	out := map[uint64]store.User{}
	if len(ids) == 0 {
		return out
	}
	var users []store.User
	if err := h.deps.DB.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out
}

func decorate(row store.Poll, users map[uint64]store.User) DTO {
	dto := DTO{
		ID:          row.ID,
		Title:       row.Title,
		Description: deref(row.Description),
		Options:     parseOptions(row.Options),
		VoteCount:   row.VoteCount,
		UserID:      row.UserID,
		CreatedAt:   web.TimeOf(row.CreatedAt),
		UpdatedAt:   web.TimeOf(row.UpdatedAt),
	}
	if u, ok := users[row.UserID]; ok {
		dto.User = &UserBrief{ID: u.ID, Username: u.Username}
	}
	return dto
}

func parseOptions(raw string) []string {
	out := []string{}
	if raw == "" {
		return out
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return out
	}
	for _, o := range arr {
		out = append(out, o)
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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

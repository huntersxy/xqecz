package content

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/recommend"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
)

const (
	ContentTTL = 300 * time.Second
	ListTTL    = 300 * time.Second
	TagsTTL    = 300 * time.Second
)

type Options struct {
	Mirror app.MediaMirror
}

type Handler struct {
	deps   app.Deps
	rec    *recommend.Refresher
	mirror app.MediaMirror
}

func New(deps app.Deps, opts ...Options) *Handler {
	h := &Handler{deps: deps, rec: recommend.NewRefresher(deps)}
	if len(opts) > 0 {
		h.mirror = opts[0].Mirror
	}
	return h
}

func Register(api *gin.RouterGroup, deps app.Deps, opts ...Options) *Handler {
	h := New(deps, opts...)
	g := api.Group("/content")
	g.GET("/list", h.list)
	g.GET("/search", h.search)
	g.GET("/recommend", h.recommend)
	g.GET("/tags", h.tags)
	g.GET("/metadata-options", h.metadataOptions)
	g.GET("/my", web.RequireAuth(deps), h.my)

	g.POST("/upload", web.RequireAuth(deps), web.RequireAPIKeyPermission("upload"), h.upload)
	g.POST("/quick-upload", h.quickUpload)
	g.PUT("/:id", web.RequireAuth(deps), web.RequireAPIKeyPermission("upload"), h.update)
	g.DELETE("/:id", web.RequireAuth(deps), web.RequireAPIKeyPermission("delete"), h.remove)
	g.POST("/:id/claim", web.RequireAuth(deps), h.claim)
	g.POST("/:id/like", web.RequireAuth(deps), h.toggleLike)
	g.GET("/:id/like-status", web.RequireAuth(deps), h.likeStatus)
	g.POST("/:id/favorite", web.RequireAuth(deps), h.toggleFavorite)

	g.GET("/:id", web.OptionalAuth(deps), h.detail)
	return h
}

type listOpts struct {
	Page          int
	PageSize      int
	Tag           string
	AuditStatus   string
	AuditStatuses []string
	Keyword       string
	SortBy        string
	Order         string
	UserID        uint64
}

func (o *listOpts) normalize() {
	if o.Page < 1 {
		o.Page = 1
	}
	if o.PageSize < 1 {
		o.PageSize = 20
	}
	if o.PageSize > 100 {
		o.PageSize = 100
	}
	if o.SortBy != "created_at" && o.SortBy != "view_count" && o.SortBy != "id" {
		o.SortBy = "created_at"
	}
	if o.Order != "asc" {
		o.Order = "desc"
	}
}

func (o *listOpts) cacheKey() string {
	statuses := append([]string{}, o.AuditStatuses...)
	sortStrings(statuses)
	norm := struct {
		Page          int      `json:"page"`
		PageSize      int      `json:"pageSize"`
		Tag           string   `json:"tag"`
		AuditStatus   string   `json:"auditStatus"`
		AuditStatuses []string `json:"auditStatuses"`
		Keyword       string   `json:"keyword"`
		SortBy        string   `json:"sortBy"`
		Order         string   `json:"order"`
		UserID        uint64   `json:"userId"`
	}{o.Page, o.PageSize, o.Tag, o.AuditStatus, statuses, o.Keyword, o.SortBy, o.Order, o.UserID}

	b, _ := json.Marshal(norm)
	sum := sha1.Sum(b)
	return "content_list:" + hex.EncodeToString(sum[:])
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func (h *Handler) list(c *gin.Context) {
	opts := listOpts{
		Page:        atoiDefault(c.Query("page"), 1),
		PageSize:    atoiDefault(c.Query("page_size"), 20),
		Tag:         strings.TrimSpace(c.Query("tag")),
		AuditStatus: strings.TrimSpace(c.Query("audit_status")),
		Keyword:     strings.TrimSpace(c.Query("keyword")),
		SortBy:      c.Query("sort_by"),
		Order:       c.Query("order"),
	}
	if opts.AuditStatus == "" {
		opts.AuditStatuses = []string{"approved", "pending"}
	}
	page, err := h.query(c.Request.Context(), opts)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, page, "ok")
}

func (h *Handler) search(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		web.OK(c, Page{List: []Item{}, Total: 0, Page: 1, PageSize: 20, TotalPage: 1}, "ok")
		return
	}
	opts := listOpts{
		Page:          atoiDefault(c.Query("page"), 1),
		PageSize:      atoiDefault(c.Query("page_size"), 20),
		Tag:           strings.TrimSpace(c.Query("tag")),
		Keyword:       keyword,
		AuditStatuses: []string{"approved", "pending"},
	}
	page, err := h.query(c.Request.Context(), opts)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, page, "ok")
}

func (h *Handler) my(c *gin.Context) {
	id := web.MustIdentity(c)
	opts := listOpts{
		Page:        atoiDefault(c.Query("page"), 1),
		PageSize:    atoiDefault(c.Query("page_size"), 20),
		AuditStatus: strings.TrimSpace(c.Query("audit_status")),
		UserID:      id.UID,
	}
	page, err := h.query(c.Request.Context(), opts)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, page, "ok")
}

func (h *Handler) query(ctx context.Context, opts listOpts) (Page, error) {
	opts.normalize()
	key := opts.cacheKey()
	return cache.GetOrSetJSON(ctx, h.deps.Redis, key, ListTTL, func() (Page, error) {
		return h.queryDB(ctx, opts)
	})
}

func (h *Handler) queryDB(ctx context.Context, opts listOpts) (Page, error) {
	db := h.deps.DB.WithContext(ctx).Model(&store.Content{})

	switch {
	case len(opts.AuditStatuses) > 0:
		db = db.Where("audit_status IN ?", opts.AuditStatuses)
	case opts.AuditStatus != "":
		db = db.Where("audit_status = ?", opts.AuditStatus)
	}
	if opts.UserID > 0 {
		db = db.Where("user_id = ?", opts.UserID)
	}
	if opts.Keyword != "" {
		db = db.Where("title LIKE ?", "%"+opts.Keyword+"%")
	}
	if tags := ParseTagsParam(opts.Tag); len(tags) > 0 {
		cond := ""
		args := make([]any, 0, len(tags))
		for i, t := range tags {
			if i > 0 {
				cond += " OR "
			}
			cond += "JSON_CONTAINS(tags, JSON_QUOTE(?))"
			args = append(args, t)
		}
		db = db.Where("("+cond+")", args...)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return Page{}, err
	}

	var rows []store.Content
	err := db.
		Order(opts.SortBy + " " + opts.Order).
		Offset((opts.Page - 1) * opts.PageSize).
		Limit(opts.PageSize).
		Find(&rows).Error
	if err != nil {
		return Page{}, err
	}

	list, err := h.decorateRows(ctx, rows)
	if err != nil {
		return Page{}, err
	}
	return Page{
		List:      list,
		Total:     total,
		Page:      opts.Page,
		PageSize:  opts.PageSize,
		TotalPage: int(math.Ceil(float64(total) / float64(opts.PageSize))),
	}, nil
}

func (h *Handler) detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		web.Fail(c, 404, "内容不存在")
		return
	}
	silent := c.Query("silent") == "1"
	identity, hasViewer := web.IdentityOf(c)
	ctx := c.Request.Context()

	if !hasViewer && !silent {
		var cached Item
		if hit, _ := h.deps.Redis.GetJSON(ctx, "content:"+strconv.FormatUint(id, 10), &cached); hit {
			h.incView(ctx, id)
			web.OK(c, cached, "ok")
			return
		}
	}

	var row store.Content
	err = h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 404, "内容不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}

	if row.AuditStatus == "rejected" {
		isOwner := hasViewer && identity.UID == row.UserID
		if !isOwner && !(hasViewer && identity.IsAdmin) {
			web.Fail(c, 404, "内容不存在或未通过审核")
			return
		}
	}

	if !silent {
		h.incView(ctx, id)
	}

	var likeCount int64
	h.deps.DB.WithContext(ctx).Model(&store.ContentLike{}).
		Where("content_id = ?", id).Count(&likeCount)

	item := decorateWith(h.mirror, row, h.userMapFor(ctx, []store.Content{row}), likeCount, false)

	if !hasViewer && row.AuditStatus != "rejected" {
		_ = h.deps.Redis.SetJSON(ctx, "content:"+strconv.FormatUint(id, 10), item, ContentTTL)
	}
	web.OK(c, item, "ok")
}

func (h *Handler) incView(ctx context.Context, id uint64) {
	_ = h.deps.DB.WithContext(ctx).Model(&store.Content{}).
		Where("id = ?", id).UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
	h.deps.Redis.IncrementView(ctx, id)
}

func (h *Handler) recommend(c *gin.Context) {
	count := atoiDefault(c.Query("count"), 20)
	if count < 1 {
		count = 20
	}
	if count > 100 {
		count = 100
	}
	pageNum := atoiDefault(c.Query("page"), 1)
	if pageNum < 1 {
		pageNum = 1
	}

	ctx := c.Request.Context()

	if ids := h.deps.Redis.GetRecommendList(ctx, pageNum, count); len(ids) > 0 {
		var rows []store.Content
		if err := h.deps.DB.WithContext(ctx).
			Where("id IN ? AND audit_status = ?", ids, "approved").
			Find(&rows).Error; err != nil {
			slog.Warn("recommend query failed, fallback to db", "err", err)
		} else {
			byID := make(map[uint64]store.Content, len(rows))
			for _, r := range rows {
				byID[r.ID] = r
			}
			ordered := make([]store.Content, 0, len(ids))
			for _, id := range ids {
				if r, ok := byID[id]; ok {
					ordered = append(ordered, r)
				}
			}
			list, err := h.decorateRows(ctx, ordered)
			if err == nil {
				web.OK(c, gin.H{"list": list, "count": h.deps.Redis.GetRecommendTotal(ctx)}, "ok")
				return
			}
		}
	}

	var rows []store.Content
	err := h.deps.DB.WithContext(ctx).
		Where("audit_status = ?", "approved").
		Order("view_count DESC, created_at DESC").
		Offset((pageNum - 1) * count).Limit(count).
		Find(&rows).Error
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	list, err := h.decorateRows(ctx, rows)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	var total int64
	h.deps.DB.WithContext(ctx).Model(&store.Content{}).
		Where("audit_status = ?", "approved").Count(&total)
	web.OK(c, gin.H{"list": list, "count": total}, "ok")
}

func (h *Handler) tags(c *gin.Context) {
	ctx := c.Request.Context()
	key := "tags"
	list, err := cache.GetOrSetJSON(ctx, h.deps.Redis, key, TagsTTL, func() ([]string, error) {
		var rows []store.Content
		if err := h.deps.DB.WithContext(ctx).
			Select("tags").
			Where("audit_status IN ?", []string{"approved", "pending"}).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		out := []string{}
		for _, r := range rows {
			for _, t := range ParseTags(r.Tags) {
				if !seen[t] {
					seen[t] = true
					out = append(out, t)
				}
			}
		}
		return out, nil
	})
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, list, "ok")
}

func (h *Handler) decorateRows(ctx context.Context, rows []store.Content) ([]Item, error) {
	return h.decorateRowsWith(ctx, rows, true)
}

func (h *Handler) decorateRowsWith(ctx context.Context, rows []store.Content, includeViewCount bool) ([]Item, error) {
	if len(rows) == 0 {
		return []Item{}, nil
	}
	ids := make([]uint64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}

	type likeRow struct {
		ContentID uint64
		Cnt       int64
	}
	var likes []likeRow
	if err := h.deps.DB.WithContext(ctx).Model(&store.ContentLike{}).
		Select("content_id, COUNT(*) AS cnt").
		Where("content_id IN ?", ids).
		Group("content_id").
		Scan(&likes).Error; err != nil {
		return nil, err
	}
	likeMap := make(map[uint64]int64, len(likes))
	for _, l := range likes {
		likeMap[l.ContentID] = l.Cnt
	}

	userMap := h.userMapFor(ctx, rows)
	list := make([]Item, 0, len(rows))
	for _, r := range rows {
		list = append(list, decorateWith(h.mirror, r, userMap, likeMap[r.ID], includeViewCount))
	}
	return list, nil
}

func (h *Handler) userMapFor(ctx context.Context, rows []store.Content) map[uint64]store.User {
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
	if err := h.deps.DB.WithContext(ctx).Select("id", "username", "email").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out
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

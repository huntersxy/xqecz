package admin

import (
	"context"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

const dashboardCacheKey = "admin:dashboard"
const dashboardCacheTTL = 60 * time.Second

type dashboardContentStats struct {
	Total    int64 `json:"total"`
	Pending  int64 `json:"pending"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
	Today    int64 `json:"today"`
}

type dashboardUsersStats struct {
	Total  int64 `json:"total"`
	Admins int64 `json:"admins"`
	Banned int64 `json:"banned"`
	Today  int64 `json:"today"`
}

type dashboardCommentsStats struct {
	Total int64 `json:"total"`
	Today int64 `json:"today"`
}

type dashboardClaimsStats struct {
	Total    int64 `json:"total"`
	Pending  int64 `json:"pending"`
	Approved int64 `json:"approved"`
	Rejected int64 `json:"rejected"`
}

type dashboardReportsStats struct {
	Total     int64 `json:"total"`
	Unhandled int64 `json:"unhandled"`
	Handled   int64 `json:"handled"`
}

type dashboardPollsStats struct {
	Total int64 `json:"total"`
	Votes int64 `json:"votes"`
}

type dashboardTagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

type dashboardData struct {
	Content        dashboardContentStats  `json:"content"`
	Users          dashboardUsersStats    `json:"users"`
	Comments       dashboardCommentsStats `json:"comments"`
	Claims         dashboardClaimsStats   `json:"claims"`
	Reports        dashboardReportsStats  `json:"reports"`
	Polls          dashboardPollsStats    `json:"polls"`
	Views          int64                  `json:"views"`
	TopTags        []dashboardTagCount    `json:"topTags"`
	RecentContents []content.Item         `json:"recentContents"`
	RecentUsers    []gin.H                `json:"recentUsers"`
}

// dashboard 返回仪表盘聚合数据（60 秒缓存；fresh=1 时跳过读缓存）。
func (h *Handler) dashboard(c *gin.Context) {
	ctx := c.Request.Context()
	if c.Query("fresh") != "1" {
		var cached dashboardData
		if hit, _ := h.deps.Redis.GetJSON(ctx, dashboardCacheKey, &cached); hit {
			web.OK(c, cached, "ok")
			return
		}
	}
	data, err := h.computeDashboard(ctx)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	_ = h.deps.Redis.SetJSON(ctx, dashboardCacheKey, data, dashboardCacheTTL)
	web.OK(c, data, "ok")
}

// computeDashboard 每张表只发一条条件聚合查询，避免并发占用连接池。
func (h *Handler) computeDashboard(ctx context.Context) (dashboardData, error) {
	db := h.deps.DB.WithContext(ctx)
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)

	var contentRow dashboardContentStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total,
		       COALESCE(SUM(audit_status = 'pending'), 0) AS pending,
		       COALESCE(SUM(audit_status = 'approved'), 0) AS approved,
		       COALESCE(SUM(audit_status = 'rejected'), 0) AS rejected,
		       COALESCE(SUM(created_at >= ?), 0) AS today
		FROM contents`, today).Scan(&contentRow).Error; err != nil {
		return dashboardData{}, err
	}

	var userRow dashboardUsersStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total,
		       COALESCE(SUM(is_admin = 1), 0) AS admins,
		       COALESCE(SUM(is_banned = 1), 0) AS banned,
		       COALESCE(SUM(created_at >= ?), 0) AS today
		FROM users`, today).Scan(&userRow).Error; err != nil {
		return dashboardData{}, err
	}

	var commentRow dashboardCommentsStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total, COALESCE(SUM(created_at >= ?), 0) AS today
		FROM comments`, today).Scan(&commentRow).Error; err != nil {
		return dashboardData{}, err
	}

	var claimRow dashboardClaimsStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total,
		       COALESCE(SUM(status = 'pending'), 0) AS pending,
		       COALESCE(SUM(status = 'approved'), 0) AS approved,
		       COALESCE(SUM(status = 'rejected'), 0) AS rejected
		FROM claims`).Scan(&claimRow).Error; err != nil {
		return dashboardData{}, err
	}

	var reportRow dashboardReportsStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total, COALESCE(SUM(handled = 0), 0) AS unhandled
		FROM comment_reports`).Scan(&reportRow).Error; err != nil {
		return dashboardData{}, err
	}
	reportRow.Handled = reportRow.Total - reportRow.Unhandled

	var pollRow dashboardPollsStats
	if err := db.Raw(`
		SELECT COUNT(*) AS total, COALESCE(SUM(vote_count), 0) AS votes
		FROM polls`).Scan(&pollRow).Error; err != nil {
		return dashboardData{}, err
	}

	var viewsRow struct {
		Views int64 `json:"views"`
	}
	if err := db.Raw(`
		SELECT COALESCE(SUM(view_count), 0) AS views
		FROM contents`).Scan(&viewsRow).Error; err != nil {
		return dashboardData{}, err
	}

	// 热门标签只统计已通过内容，避免待审与垃圾数据污染。
	var tagRows []store.Content
	if err := db.Select("tags").Where("audit_status = ?", "approved").Find(&tagRows).Error; err != nil {
		return dashboardData{}, err
	}
	tagCount := map[string]int{}
	order := []string{}
	for _, row := range tagRows {
		for _, t := range content.ParseTags(row.Tags) {
			if _, seen := tagCount[t]; !seen {
				order = append(order, t)
			}
			tagCount[t]++
		}
	}
	ranked := make([]dashboardTagCount, 0, len(order))
	for _, t := range order {
		ranked = append(ranked, dashboardTagCount{Tag: t, Count: tagCount[t]})
	}
	// 稳定排序：同数量时保持首次出现顺序，保证结果可复现。
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].Count > ranked[j].Count })
	if len(ranked) > 10 {
		ranked = ranked[:10]
	}

	var recentRows []store.Content
	if err := db.Order("created_at DESC").Limit(6).Find(&recentRows).Error; err != nil {
		return dashboardData{}, err
	}
	recentContents := make([]content.Item, 0, len(recentRows))
	for _, row := range recentRows {
		recentContents = append(recentContents, h.content.DecorateOne(ctx, row, true))
	}

	var recentUserRows []store.User
	if err := db.Order("created_at DESC").Limit(6).Find(&recentUserRows).Error; err != nil {
		return dashboardData{}, err
	}
	recentUsers := make([]gin.H, 0, len(recentUserRows))
	for _, u := range recentUserRows {
		recentUsers = append(recentUsers, formatUser(u))
	}

	return dashboardData{
		Content: contentRow, Users: userRow, Comments: commentRow,
		Claims: claimRow, Reports: reportRow, Polls: pollRow,
		Views: viewsRow.Views, TopTags: ranked,
		RecentContents: recentContents, RecentUsers: recentUsers,
	}, nil
}

// Package recommend 实现推荐位打分与刷新：读取已审核内容 → 纯函数打分 → 原子写 Redis ZSet。
// 打分逻辑移植自原 Go Worker 的 computeRecommend（纯函数，无外部依赖）。
package recommend

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/store"
)

const (
	lockKey = "lock:refresh-recommend"
	lockTTL = 300 * time.Second
	// 周期刷新节奏与旧实现一致：每 10 分钟一次。
	refreshInterval = 10 * time.Minute
)

// Item 是打分输入。
type Item struct {
	ContentID     uint64
	CreatedAtUnix int64
	ViewCount     int64
	LikeCount     int64
}

// timeDecayScore 时间衰减分数：1 天内 100，7 天内线性衰减到 0，更久为 0。
func timeDecayScore(createdAtUnix int64) float64 {
	daysAgo := time.Since(time.Unix(createdAtUnix, 0)).Hours() / 24
	switch {
	case daysAgo < 1:
		return 100.0
	case daysAgo < 7:
		return 50.0 * (1.0 - daysAgo/7.0)
	default:
		return 0.0
	}
}

// ScoreItem 综合时间衰减、浏览量、点赞数为单条内容打分。
// 点赞是公开指标，权重高于浏览量：likeScore 满额 100，viewScore 满额 30。
func ScoreItem(it Item) float64 {
	timeScore := timeDecayScore(it.CreatedAtUnix)
	viewScore := math.Min(float64(it.ViewCount), 1000) / 1000.0 * 30.0
	likeScore := math.Min(float64(it.LikeCount), 500) / 500.0 * 100.0
	return timeScore + viewScore + likeScore
}

// Compute 对输入逐个打分并按分数降序返回。
func Compute(items []Item) []cache.ScoredItem {
	out := make([]cache.ScoredItem, 0, len(items))
	for _, it := range items {
		out = append(out, cache.ScoredItem{ContentID: it.ContentID, Score: ScoreItem(it)})
	}
	slices.SortFunc(out, func(a, b cache.ScoredItem) int {
		return int(b.Score - a.Score)
	})
	return out
}

// Refresher 负责把 MySQL 中的已审核内容重新打分并写入 Redis ZSet。
type Refresher struct {
	deps app.Deps
}

func NewRefresher(deps app.Deps) *Refresher { return &Refresher{deps: deps} }

// Refresh 执行一次刷新。多实例通过 Redis 分布式锁防抖，同一时刻只有一个实例在执行。
func (r *Refresher) Refresh(ctx context.Context) error {
	if !r.deps.Redis.AcquireLock(ctx, lockKey, lockTTL) {
		slog.Debug("another instance is refreshing recommend, skip")
		return nil
	}
	defer r.deps.Redis.ReleaseLock(ctx, lockKey)

	var rows []store.Content
	if err := r.deps.DB.WithContext(ctx).
		Select("id", "created_at", "view_count").
		Where("audit_status = ?", "approved").
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	likeMap, err := likeCounts(ctx, r.deps, ids)
	if err != nil {
		return err
	}

	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, Item{
			ContentID:     row.ID,
			CreatedAtUnix: row.CreatedAt.Unix(),
			ViewCount:     row.ViewCount,
			LikeCount:     likeMap[row.ID],
		})
	}

	scored := Compute(items)
	if err := r.deps.Redis.WriteRecommendList(ctx, scored); err != nil {
		return err
	}
	slog.Info("recommend refreshed", "items", len(scored))
	return nil
}

// Start 启动后立即刷新一次，并按固定节奏周期刷新，直到 ctx 结束。
func (r *Refresher) Start(ctx context.Context) {
	go func() {
		if err := r.Refresh(ctx); err != nil {
			slog.Warn("initial recommend refresh failed", "err", err)
		}
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.Refresh(ctx); err != nil {
					slog.Warn("scheduled recommend refresh failed", "err", err)
				}
			}
		}
	}()
}

// RefreshAsync 供写路径（点赞、审核通过等）触发，不阻塞响应。
func (r *Refresher) RefreshAsync() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := r.Refresh(ctx); err != nil {
			slog.Warn("recommend refresh failed", "err", err)
		}
	}()
}

func likeCounts(ctx context.Context, deps app.Deps, ids []uint64) (map[uint64]int64, error) {
	type row struct {
		ContentID uint64
		Cnt       int64
	}
	var rows []row
	if err := deps.DB.WithContext(ctx).Model(&store.ContentLike{}).
		Select("content_id, COUNT(*) AS cnt").
		Where("content_id IN ?", ids).
		Group("content_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]int64, len(rows))
	for _, r := range rows {
		out[r.ContentID] = r.Cnt
	}
	return out, nil
}

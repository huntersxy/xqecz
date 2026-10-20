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
	lockKey         = "lock:refresh-recommend"
	lockTTL         = 300 * time.Second
	refreshInterval = 10 * time.Minute
)

type Item struct {
	ContentID     uint64
	CreatedAtUnix int64
	ViewCount     int64
	LikeCount     int64
}

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

func ScoreItem(it Item) float64 {
	timeScore := timeDecayScore(it.CreatedAtUnix)
	viewScore := math.Min(float64(it.ViewCount), 1000) / 1000.0 * 30.0
	likeScore := math.Min(float64(it.LikeCount), 500) / 500.0 * 100.0
	return timeScore + viewScore + likeScore
}

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

type Refresher struct {
	deps app.Deps
}

func NewRefresher(deps app.Deps) *Refresher { return &Refresher{deps: deps} }

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

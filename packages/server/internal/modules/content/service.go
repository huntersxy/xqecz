package content

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/media"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
)

// 批量缩略图任务的 Redis 键与节奏（与旧实现一致）。
const (
	regenLockKey    = "lock:regen-all-thumbs"
	regenStatusKey  = "task:regen-all-thumbs:status"
	regenLockTTL    = 120 * time.Second
	regenHeartbeat  = 30 * time.Second
	regenStatusTTL  = 7 * 24 * time.Hour
	regenJobTimeout = 6 * time.Hour
)

// ListOpts 是列表查询参数（管理端与内部复用）。
type ListOpts struct {
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

// ListPage 执行列表查询（含读穿缓存）。
func (h *Handler) ListPage(ctx context.Context, o ListOpts) (Page, error) {
	return h.query(ctx, listOpts{
		Page: o.Page, PageSize: o.PageSize, Tag: o.Tag,
		AuditStatus: o.AuditStatus, AuditStatuses: o.AuditStatuses,
		Keyword: o.Keyword, SortBy: o.SortBy, Order: o.Order, UserID: o.UserID,
	})
}

// RowByID 取单行原始记录（软删除行不可见）。
func (h *Handler) RowByID(ctx context.Context, id uint64) (store.Content, bool, error) {
	var row store.Content
	err := h.deps.DB.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return store.Content{}, false, nil
	}
	if err != nil {
		return store.Content{}, false, err
	}
	return row, true, nil
}

// DetailInternal 走与公开详情一致的缓存与浏览量计数路径，供管理端审核后返回最新详情。
func (h *Handler) DetailInternal(ctx context.Context, id uint64) (Item, error) {
	key := "content:" + strconv.FormatUint(id, 10)
	var cached Item
	if hit, _ := h.deps.Redis.GetJSON(ctx, key, &cached); hit {
		h.incView(ctx, id)
		return cached, nil
	}

	var row store.Content
	if err := h.deps.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return Item{}, err
	}
	h.incView(ctx, id)
	item := decorate(row, h.userMapFor(ctx, []store.Content{row}), h.likeCount(ctx, id), false)
	if row.AuditStatus != "rejected" {
		_ = h.deps.Redis.SetJSON(ctx, key, item, ContentTTL)
	}
	return item, nil
}

// SetAuditStatus 更新审核状态并失效缓存；通过时触发推荐刷新。
func (h *Handler) SetAuditStatus(ctx context.Context, id uint64, status string) error {
	if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).
		Where("id = ?", id).Update("audit_status", status).Error; err != nil {
		return err
	}
	h.invalidate(ctx, id)
	if status == "approved" {
		h.rec.RefreshAsync()
	}
	return nil
}

// UpdateAuthor 变更内容作者，返回原作者 id 与新作者用户名。
func (h *Handler) UpdateAuthor(ctx context.Context, contentID, userID uint64) (uint64, string, error) {
	db := h.deps.DB.WithContext(ctx)
	var row store.Content
	if err := db.First(&row, contentID).Error; err != nil {
		return 0, "", err
	}
	oldUserID := row.UserID
	if err := db.Model(&store.Content{}).Where("id = ?", contentID).
		Update("user_id", userID).Error; err != nil {
		return oldUserID, "", err
	}
	h.invalidate(ctx, contentID)
	var u store.User
	if err := db.First(&u, userID).Error; err != nil {
		return oldUserID, "", nil
	}
	return oldUserID, u.Username, nil
}

// DecorateOne 把单行内容装饰为对外形状（管理端认领列表等复用）。
func (h *Handler) DecorateOne(ctx context.Context, row store.Content, includeViewCount bool) Item {
	return decorate(row, h.userMapFor(ctx, []store.Content{row}), h.likeCount(ctx, row.ID), includeViewCount)
}

// RefreshRecommend 立即重算推荐位。
func (h *Handler) RefreshRecommend(ctx context.Context) error {
	return h.rec.Refresh(ctx)
}

// RegenerateThumbnail 为单条内容重建缩略图。
func (h *Handler) RegenerateThumbnail(ctx context.Context, id uint64) (Item, error) {
	var row store.Content
	if err := h.deps.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return Item{}, err
	}
	if row.FilePath == nil || *row.FilePath == "" {
		return Item{}, errNoOriginalFile
	}
	abs := h.absUploadPath(*row.FilePath)
	rel, err := media.GenerateThumbnail(ctx, abs, MediaTypeForPath(*row.FilePath), h.deps.Cfg.ThumbDir)
	if err != nil {
		return Item{}, err
	}
	if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).
		Where("id = ?", id).Update("thumb_path", rel).Error; err != nil {
		return Item{}, err
	}
	h.invalidate(ctx, id)
	var updated store.Content
	if err := h.deps.DB.WithContext(ctx).First(&updated, id).Error; err != nil {
		return Item{}, err
	}
	return h.DecorateOne(ctx, updated, false), nil
}

// errNoOriginalFile 表示内容没有原始文件，无法生成缩略图。
var errNoOriginalFile = errors.New("无原始文件")

// ErrNoOriginalFile 供调用方判定错误类型。
func ErrNoOriginalFile() error { return errNoOriginalFile }

// thumbSweepLimit 是启动补图单次处理上限，避免历史积压过多时启动期 CPU 风暴。
const thumbSweepLimit = 200

// SweepMissingThumbnails 为「有原文件但缺缩略图」的内容补图。
// 旧实现在启动迁移里做同样的事；与批量重建共用同一把 Redis 锁，多实例只有一个执行，
// 也不会与手工触发的批量任务打架。任何失败只记日志，不影响服务。
func (h *Handler) SweepMissingThumbnails(ctx context.Context) {
	if !h.deps.Redis.AcquireLock(ctx, regenLockKey, regenLockTTL) {
		slog.Info("跳过启动补图：已有缩略图任务在进行")
		return
	}
	defer h.deps.Redis.ReleaseLock(context.Background(), regenLockKey)

	var rows []store.Content
	err := h.deps.DB.WithContext(ctx).
		Select("id", "file_path").
		Where("file_path IS NOT NULL AND file_path <> '' AND (thumb_path IS NULL OR thumb_path = '')").
		Limit(thumbSweepLimit).
		Find(&rows).Error
	if err != nil {
		slog.Warn("启动补图查询失败", "err", err)
		return
	}
	if len(rows) == 0 {
		// 无待补内容时不产生日志噪音（正常情况下每次启动都会走到这里）。
		return
	}

	ok, fail := 0, 0
	for _, row := range rows {
		if ctx.Err() != nil {
			break
		}
		if row.FilePath == nil {
			continue
		}
		rel, err := media.GenerateThumbnail(ctx, h.absUploadPath(*row.FilePath), MediaTypeForPath(*row.FilePath), h.deps.Cfg.ThumbDir)
		if err != nil {
			fail++
			continue
		}
		if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).
			Where("id = ?", row.ID).Update("thumb_path", rel).Error; err != nil {
			fail++
			continue
		}
		h.deps.Redis.ClearContentCache(ctx, row.ID)
		ok++
	}
	if ok > 0 {
		// 缩略图嵌入列表卡片，补图后失效列表缓存。
		h.deps.Redis.ClearContentListCache(ctx)
	}
	slog.Info("启动补图完成", "ok", ok, "fail", fail, "scanned", len(rows))
}

// RegenResult 是批量缩略图接口的即时返回。
type RegenResult struct {
	OK      int64 `json:"ok"`
	Fail    int64 `json:"fail"`
	Total   int64 `json:"total"`
	Count   int64 `json:"count"`
	Running bool  `json:"running,omitempty"`
}

// RegenStatus 是批量缩略图任务的进度快照。
type RegenStatus struct {
	Status     string `json:"status"`
	Total      int64  `json:"total"`
	OK         int64  `json:"ok"`
	Fail       int64  `json:"fail"`
	StartedAt  *int64 `json:"started_at,omitempty"`
	UpdatedAt  *int64 `json:"updated_at,omitempty"`
	FinishedAt *int64 `json:"finished_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

// regenerateAllThumbnails 立即返回、后台异步处理；多实例用 Redis 锁 + 心跳防重。
func (h *Handler) RegenerateAllThumbnails(ctx context.Context) RegenResult {
	if !h.deps.Redis.AcquireLock(ctx, regenLockKey, regenLockTTL) {
		return RegenResult{Running: true}
	}

	var rows []store.Content
	err := h.deps.DB.WithContext(ctx).
		Select("id", "file_path").
		Where("file_path IS NOT NULL AND file_path <> ''").
		Find(&rows).Error
	if err != nil {
		h.deps.Redis.ReleaseLock(ctx, regenLockKey)
		slog.Warn("查询待重建缩略图失败", "err", err)
		return RegenResult{}
	}
	if len(rows) == 0 {
		h.deps.Redis.Del(ctx, regenStatusKey)
		h.deps.Redis.ReleaseLock(ctx, regenLockKey)
		return RegenResult{}
	}

	startedAt := time.Now().UnixMilli()
	h.writeRegenStatus(ctx, RegenStatus{
		Status: "running", Total: int64(len(rows)), StartedAt: &startedAt,
	})

	go h.runRegenerateAll(rows, startedAt)
	return RegenResult{Total: int64(len(rows)), Count: int64(len(rows))}
}

func (h *Handler) runRegenerateAll(rows []store.Content, startedAt int64) {
	ctx, cancel := context.WithTimeout(context.Background(), regenJobTimeout)
	defer cancel()

	// 心跳续期：长任务不会被其它实例重复触发。
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(regenHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				h.deps.Redis.RenewLock(ctx, regenLockKey, regenLockTTL)
			}
		}
	}()
	defer func() {
		close(stop)
		wg.Wait()
		h.deps.Redis.ReleaseLock(context.Background(), regenLockKey)
	}()

	var ok, fail int64
	for i, row := range rows {
		if row.FilePath == nil {
			continue
		}
		abs := h.absUploadPath(*row.FilePath)
		rel, err := media.GenerateThumbnail(ctx, abs, MediaTypeForPath(*row.FilePath), h.deps.Cfg.ThumbDir)
		if err != nil {
			fail++
		} else if err := h.deps.DB.WithContext(ctx).Model(&store.Content{}).
			Where("id = ?", row.ID).Update("thumb_path", rel).Error; err != nil {
			fail++
		} else {
			h.deps.Redis.ClearContentCache(ctx, row.ID)
			ok++
		}
		if (i+1)%10 == 0 || i == len(rows)-1 {
			updatedAt := time.Now().UnixMilli()
			h.writeRegenStatus(ctx, RegenStatus{
				Status: "running", Total: int64(len(rows)), OK: ok, Fail: fail,
				StartedAt: &startedAt, UpdatedAt: &updatedAt,
			})
		}
	}

	finishedAt := time.Now().UnixMilli()
	h.writeRegenStatus(ctx, RegenStatus{
		Status: "done", Total: int64(len(rows)), OK: ok, Fail: fail,
		StartedAt: &startedAt, FinishedAt: &finishedAt,
	})
	// 缩略图嵌入列表卡片，全部结束后统一失效列表缓存。
	h.deps.Redis.ClearContentListCache(ctx)
	slog.Info("批量缩略图完成", "ok", ok, "fail", fail, "total", len(rows))
}

func (h *Handler) writeRegenStatus(ctx context.Context, st RegenStatus) {
	_ = h.deps.Redis.SetJSON(ctx, regenStatusKey, st, regenStatusTTL)
}

// RegenerateAllStatus 返回批量缩略图任务进度（无任务时为 idle）。
func (h *Handler) RegenerateAllStatus(ctx context.Context) RegenStatus {
	var st RegenStatus
	if hit, _ := h.deps.Redis.GetJSON(ctx, regenStatusKey, &st); hit && st.Status != "" {
		return st
	}
	return RegenStatus{Status: "idle"}
}

// absUploadPath 把相对路径还原为共享上传目录下的绝对路径。
func (h *Handler) absUploadPath(rel string) string {
	return filepath.Join(h.deps.Cfg.UploadDir, filepath.FromSlash(rel))
}

// absMediaPath 按前缀还原媒体文件的绝对路径：
// thumbs/ 与 images/ 各自对应独立目录，其余（裸文件名）落在 uploads。
func (h *Handler) absMediaPath(rel string) string {
	switch {
	case strings.HasPrefix(rel, "thumbs/"):
		return filepath.Join(h.deps.Cfg.ThumbDir, filepath.FromSlash(strings.TrimPrefix(rel, "thumbs/")))
	case strings.HasPrefix(rel, "images/"):
		return filepath.Join(h.deps.Cfg.ImagesDir, filepath.FromSlash(strings.TrimPrefix(rel, "images/")))
	case strings.HasPrefix(rel, "/"):
		return filepath.FromSlash(rel)
	default:
		return h.absUploadPath(rel)
	}
}

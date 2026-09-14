package compress

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/media"
	"github.com/huntersxy/xqecz/server/internal/store"
)

const (
	// lockKey 让多实例部署时同一时刻只有一个实例在压缩（与推荐刷新同一套机制）。
	lockKey = "lock:tinify-compress"
	// lockTTL 是锁的兜底过期时间，取「单张最大文件压缩耗时的量级」即可：
	// 进程在压缩中被强杀时无法执行 defer 释放，只能靠 TTL 自愈，故不宜过长。
	lockTTL = 3 * time.Minute

	// failCooldown 是单条内容压缩失败后的冷却期。
	// 失败若不加冷却，下一轮仍会挑中同一条最大的图片反复调用 API，白白消耗配额
	//（实测：垃圾桶目录权限问题上失败重试 4 次，等于 4 次无效 API 调用）。
	failCooldown = 30 * time.Minute

	// candidateBatch 候选批大小：一次多取几条，用于跳过处于冷却期的内容。
	candidateBatch = 20
)

// Worker 是 TinyPNG 压缩任务的调度器：每 CompressEvery 处理一张最大的待压缩图片。
type Worker struct {
	deps   app.Deps
	client *Client

	mu     sync.Mutex
	failed map[uint64]time.Time // 内容 id → 最近一次失败时间
}

// NewWorker 构造任务；未配置 TINIFY_API_KEY 时返回休眠状态的任务（Enabled() == false）。
func NewWorker(deps app.Deps) *Worker {
	w := &Worker{deps: deps, failed: map[uint64]time.Time{}}
	if deps.Cfg.TinyPNGAPIKey != "" {
		w.client = NewClient(deps.Cfg.TinyPNGAPIKey)
	}
	return w
}

// Enabled 表示压缩任务是否可用（缺 Key 时为 false，任务静默跳过，不报错）。
func (w *Worker) Enabled() bool { return w.client != nil }

// Start 启动周期任务：等到第一个间隔才执行，避免与启动期的其他任务争抢资源。
func (w *Worker) Start(ctx context.Context) {
	if !w.Enabled() {
		slog.Info("tinypng 压缩任务未启用（未配置 TINIFY_API_KEY）")
		return
	}
	slog.Info("tinypng 压缩任务已启用",
		"interval", w.deps.Cfg.CompressEvery,
		"min_size", w.deps.Cfg.CompressMinSize,
	)
	go func() {
		ticker := time.NewTicker(w.deps.Cfg.CompressEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				// 退出时主动释放锁：main 在 ctx 结束时会直接退出进程，
				// 压缩 goroutine 可能来不及执行 CompressOne 的 defer，导致锁残留到 TTL 到期。
				w.deps.Redis.ReleaseLock(context.Background(), lockKey)
				return
			case <-ticker.C:
				if err := w.CompressOne(ctx); err != nil {
					slog.Warn("tinypng 压缩失败", "err", err)
				}
			}
		}
	}()
}

// CompressOne 取当前体积最大的待压缩图片处理一次；没有候选时直接返回。
func (w *Worker) CompressOne(ctx context.Context) error {
	if !w.Enabled() {
		return nil
	}
	if !w.deps.Redis.AcquireLock(ctx, lockKey, lockTTL) {
		return nil // 另一实例正在压缩
	}
	defer w.deps.Redis.ReleaseLock(ctx, lockKey)

	row, ok, err := w.pickCandidate(ctx)
	if err != nil || !ok {
		return err
	}

	abs := w.absPath(*row.FilePath)
	st, err := os.Stat(abs)
	if err != nil {
		// 文件已不在（可能被删除或移动）：标记已处理，避免每分钟重复尝试。
		slog.Warn("待压缩文件不存在，标记跳过", "id", row.ID, "file", *row.FilePath)
		return w.markProcessed(ctx, row.ID, row.FileSize)
	}
	// 以磁盘真实体积为准：小于阈值的不压缩（顺带修正库里过期的 file_size）。
	if st.Size() < w.deps.Cfg.CompressMinSize {
		return w.markProcessed(ctx, row.ID, st.Size())
	}

	newSize, err := w.shrinkInPlace(ctx, abs, st.Size())
	if err != nil {
		// 进入冷却期，避免下一轮又挑中同一条反复消耗 API 配额。
		w.markFailed(row.ID)
		return err
	}

	// 压缩未变小：保留原图，仅标记已处理，避免持续消耗配额。
	if newSize >= st.Size() {
		slog.Info("压缩无收益，保留原图", "id", row.ID, "orig", st.Size(), "got", newSize)
		return w.markProcessed(ctx, row.ID, st.Size())
	}

	if err := w.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ?", row.ID).
		Updates(map[string]any{"file_size": newSize, "compressed_at": time.Now()}).Error; err != nil {
		return err
	}
	w.deps.Redis.ClearContentCache(ctx, row.ID)
	w.deps.Redis.ClearContentListCache(ctx)
	slog.Info("tinypng 压缩完成",
		"id", row.ID, "file", *row.FilePath,
		"before", st.Size(), "after", newSize,
		"saved", st.Size()-newSize,
	)
	return nil
}

// pickCandidate 选出体积最大的待压缩图片：跳过 GIF（动图不支持）、已压缩过的记录，
// 以及处于失败冷却期的内容。
func (w *Worker) pickCandidate(ctx context.Context) (store.Content, bool, error) {
	var rows []store.Content
	err := w.deps.DB.WithContext(ctx).
		Where("file_path IS NOT NULL AND file_path <> ''").
		Where("compressed_at IS NULL").
		Where("file_size >= ?", w.deps.Cfg.CompressMinSize).
		Where("LOWER(file_path) NOT LIKE ?", "%.gif").
		Order("file_size DESC").
		Limit(candidateBatch).
		Find(&rows).Error
	if err != nil {
		return store.Content{}, false, err
	}
	for _, row := range rows {
		if !w.inCooldown(row.ID) {
			return row, true, nil
		}
	}
	return store.Content{}, false, nil
}

func (w *Worker) inCooldown(id uint64) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	at, ok := w.failed[id]
	if !ok {
		return false
	}
	if time.Since(at) > failCooldown {
		delete(w.failed, id)
		return false
	}
	return true
}

func (w *Worker) markFailed(id uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.failed[id] = time.Now()
}

// shrinkInPlace 原地替换为压缩结果：先落临时文件，成功后才把原图移入垃圾桶再改名，
// 任一步失败都保留原图，不会出现「原图没了、新图没写上」的空缺。
func (w *Worker) shrinkInPlace(ctx context.Context, abs string, origSize int64) (int64, error) {
	body, _, err := w.client.Shrink(ctx, abs)
	if err != nil {
		return 0, err
	}
	defer body.Close()

	tmp := abs + ".tinify-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(out, body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return 0, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return 0, closeErr
	}
	if written == 0 {
		_ = os.Remove(tmp)
		return 0, io.ErrUnexpectedEOF
	}
	if written >= origSize {
		// 结果没有更小：丢弃临时文件，交由调用方保留原图。
		_ = os.Remove(tmp)
		return written, nil
	}

	if err := media.MoveToBin(abs, w.deps.Cfg.BinDir); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	// 原文件属主可能被替换，保持与目录一致的权限，避免 Web 进程读不到。
	_ = os.Chmod(abs, 0o644)
	return written, nil
}

// markProcessed 记录「已处理」并把 file_size 校准为磁盘真实值。
// 未压缩变小的图片同样标记，否则每轮都会被重新挑中而白白消耗配额。
func (w *Worker) markProcessed(ctx context.Context, id uint64, size int64) error {
	return w.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ?", id).
		Updates(map[string]any{"file_size": size, "compressed_at": time.Now()}).Error
}

// absPath 把库里的相对路径还原为绝对路径（thumbs/、images/ 前缀各自对应独立目录）。
func (w *Worker) absPath(rel string) string {
	switch {
	case strings.HasPrefix(rel, "thumbs/"):
		return filepath.Join(w.deps.Cfg.ThumbDir, filepath.FromSlash(strings.TrimPrefix(rel, "thumbs/")))
	case strings.HasPrefix(rel, "images/"):
		return filepath.Join(w.deps.Cfg.ImagesDir, filepath.FromSlash(strings.TrimPrefix(rel, "images/")))
	default:
		return filepath.Join(w.deps.Cfg.UploadDir, filepath.FromSlash(rel))
	}
}

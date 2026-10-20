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
	"github.com/huntersxy/xqecz/server/internal/mirror"
	"github.com/huntersxy/xqecz/server/internal/store"
)

const (
	lockKey = "lock:tinify-compress"
	lockTTL = 3 * time.Minute

	failCooldown = 30 * time.Minute

	candidateBatch = 20
)

type Worker struct {
	deps   app.Deps
	client *Client

	mu     sync.Mutex
	failed map[uint64]time.Time
}

func NewWorker(deps app.Deps) *Worker {
	w := &Worker{deps: deps, failed: map[uint64]time.Time{}}
	if deps.Cfg.TinyPNGAPIKey != "" {
		w.client = NewClient(deps.Cfg.TinyPNGAPIKey)
	}
	return w
}

func (w *Worker) Enabled() bool { return w.client != nil }

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

func (w *Worker) CompressOne(ctx context.Context) error {
	if !w.Enabled() {
		return nil
	}
	if !w.deps.Redis.AcquireLock(ctx, lockKey, lockTTL) {
		return nil
	}
	defer w.deps.Redis.ReleaseLock(ctx, lockKey)

	row, ok, err := w.pickCandidate(ctx)
	if err != nil || !ok {
		return err
	}

	abs := w.absPath(*row.FilePath)
	st, err := os.Stat(abs)
	if err != nil {
		slog.Warn("待压缩文件不存在，标记跳过", "id", row.ID, "file", *row.FilePath)
		return w.markProcessed(ctx, row.ID, row.FileSize)
	}
	if st.Size() < w.deps.Cfg.CompressMinSize {
		return w.markProcessed(ctx, row.ID, st.Size())
	}

	newSize, binPath, err := w.shrinkInPlace(ctx, abs, st.Size())
	if err != nil {
		w.markFailed(row.ID)
		return err
	}

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

	if w.deps.Mirror != nil && row.FilePath != nil && *row.FilePath != "" {
		w.deps.Mirror.PushAsync(*row.FilePath, abs)
		if binPath != "" {
			w.deps.Mirror.ArchiveOriginalAsync(*row.FilePath, binPath)
		}
	}
	slog.Info("tinypng 压缩完成",
		"id", row.ID, "file", *row.FilePath,
		"before", st.Size(), "after", newSize,
		"saved", st.Size()-newSize,
	)
	return nil
}

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

func (w *Worker) shrinkInPlace(ctx context.Context, abs string, origSize int64) (int64, string, error) {
	body, _, err := w.client.Shrink(ctx, abs)
	if err != nil {
		return 0, "", err
	}
	defer body.Close()

	tmp := abs + ".tinify-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return 0, "", err
	}
	written, copyErr := io.Copy(out, body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return 0, "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return 0, "", closeErr
	}
	if written == 0 {
		_ = os.Remove(tmp)
		return 0, "", io.ErrUnexpectedEOF
	}
	if written >= origSize {
		_ = os.Remove(tmp)
		return written, "", nil
	}

	if err := media.MoveToBin(abs, w.deps.Cfg.BinDir); err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	if err := os.Rename(tmp, abs); err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	_ = os.Chmod(abs, 0o644)
	binPath, _ := mirror.FindInBin(w.deps.Cfg.BinDir, filepath.Base(abs))
	return written, binPath, nil
}

func (w *Worker) markProcessed(ctx context.Context, id uint64, size int64) error {
	return w.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ?", id).
		Updates(map[string]any{"file_size": size, "compressed_at": time.Now()}).Error
}

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

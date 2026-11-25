package compress

import (
	"context"
	"errors"
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
	lockKey = "lock:tinify-compress"
	lockTTL = 6 * time.Minute

	failCooldown = 30 * time.Minute
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
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		w.deps.Redis.ReleaseLock(releaseCtx, lockKey)
	}()

	row, ok, err := w.pickCandidate(ctx)
	if err != nil || !ok {
		return err
	}

	abs := w.absPath(*row.FilePath)
	st, err := os.Stat(abs)
	if errors.Is(err, os.ErrNotExist) {
		slog.Warn("待压缩文件不存在，标记跳过", "id", row.ID)
		return w.markProcessed(ctx, row, row.FileSize)
	}
	if err != nil {
		w.markFailed(row.ID)
		return err
	}
	if !st.Mode().IsRegular() {
		return w.markProcessed(ctx, row, row.FileSize)
	}
	if st.Size() < w.deps.Cfg.CompressMinSize {
		return w.markProcessed(ctx, row, st.Size())
	}

	compressCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	newSize, binPath, err := w.shrinkInPlace(compressCtx, abs, st.Size())
	if errors.Is(err, errUnsupportedImage) {
		slog.Info("文件格式不支持压缩，保留原图", "id", row.ID)
		return w.markProcessed(ctx, row, st.Size())
	}
	if err != nil {
		w.markFailed(row.ID)
		return err
	}

	if newSize >= st.Size() {
		slog.Info("压缩无收益，保留原图", "id", row.ID, "orig", st.Size(), "got", newSize)
		return w.markProcessed(ctx, row, st.Size())
	}

	if err := w.markProcessed(ctx, row, newSize); err != nil {
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
		"id", row.ID,
		"before", st.Size(), "after", newSize,
		"saved", st.Size()-newSize,
	)
	return nil
}

func (w *Worker) pickCandidate(ctx context.Context) (store.Content, bool, error) {
	query := w.deps.DB.WithContext(ctx).
		Where("audit_status = ?", "approved").
		Where("compressed_at IS NULL").
		Where("file_size >= ?", w.deps.Cfg.CompressMinSize).
		Where("(LOWER(file_path) LIKE ? OR LOWER(file_path) LIKE ? OR LOWER(file_path) LIKE ? OR LOWER(file_path) LIKE ? OR LOWER(file_path) LIKE ? OR LOWER(file_path) LIKE ?)",
			"%.png", "%.jpg", "%.jpeg", "%.webp", "%.avif", "%.jxl")
	if ids := w.coolingIDs(); len(ids) != 0 {
		query = query.Where("id NOT IN ?", ids)
	}
	var row store.Content
	result := query.Order("file_size DESC, id ASC").Limit(1).Find(&row)
	return row, result.RowsAffected > 0, result.Error
}

func (w *Worker) coolingIDs() []uint64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	ids := make([]uint64, 0, len(w.failed))
	for id, at := range w.failed {
		if time.Since(at) >= failCooldown {
			delete(w.failed, id)
		} else {
			ids = append(ids, id)
		}
	}
	return ids
}

func (w *Worker) markFailed(id uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.failed[id] = time.Now()
}

func (w *Worker) shrinkInPlace(ctx context.Context, abs string, origSize int64) (int64, string, error) {
	body, size, err := w.client.Shrink(ctx, abs)
	if err != nil {
		return 0, "", err
	}
	defer body.Close()

	out, err := os.CreateTemp(filepath.Dir(abs), ".tinify-*")
	if err != nil {
		return 0, "", err
	}
	tmp := out.Name()
	defer os.Remove(tmp)
	written, copyErr := io.Copy(out, io.LimitReader(body, origSize))
	closeErr := out.Close()
	if copyErr != nil {
		return 0, "", copyErr
	}
	if closeErr != nil {
		return 0, "", closeErr
	}
	if written >= origSize {
		return written, "", nil
	}
	if written == 0 || written != size {
		return 0, "", io.ErrUnexpectedEOF
	}

	if err := os.Chmod(tmp, 0o644); err != nil {
		return 0, "", err
	}
	binPath, err := media.CopyToBin(abs, w.deps.Cfg.BinDir)
	if err != nil {
		return 0, "", err
	}
	if err := os.Rename(tmp, abs); err != nil {
		return 0, "", err
	}
	return written, binPath, nil
}

func (w *Worker) markProcessed(ctx context.Context, row store.Content, size int64) error {
	return w.deps.DB.WithContext(ctx).Model(&store.Content{}).Where("id = ? AND file_path = ?", row.ID, *row.FilePath).
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

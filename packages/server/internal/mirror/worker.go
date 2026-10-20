package mirror

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/store"
)

const (
	lockTTL = 10 * time.Minute
	batch   = 500
)

type Worker struct {
	deps    app.Deps
	m       *Setup
	lockKey string
}

func NewWorker(deps app.Deps, m *Setup) *Worker {
	if !m.Enabled() {
		return nil
	}
	return &Worker{deps: deps, m: m, lockKey: "lock:mirror:" + m.cfg.LogName()}
}

func (w *Worker) tag() string { return w.m.cfg.LogName() }

func (w *Worker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	slog.Info(w.m.cfg.LogName()+" 镜像任务已启用",
		"prefix", w.m.cfg.Prefix,
		"public_base", w.m.cfg.PublicBase,
		"interval", w.m.cfg.SyncEvery,
	)
	go func() {
		w.runRound(ctx)
		ticker := time.NewTicker(w.m.cfg.SyncEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				w.deps.Redis.ReleaseLock(context.Background(), w.lockKey)
				return
			case <-ticker.C:
				w.runRound(ctx)
			}
		}
	}()
}

func (w *Worker) runRound(ctx context.Context) {
	if !w.deps.Redis.AcquireLock(ctx, w.lockKey, lockTTL) {
		return
	}
	defer w.deps.Redis.ReleaseLock(ctx, w.lockKey)

	var lastID uint64
	var scanned, uploaded, archived, failed int
	for {
		if ctx.Err() != nil {
			return
		}
		var rows []store.Content
		err := w.deps.DB.WithContext(ctx).
			Select("id", "file_path").
			Where("file_path IS NOT NULL AND file_path <> '' AND id > ?", lastID).
			Order("id ASC").Limit(batch).Find(&rows).Error
		if err != nil {
			slog.Warn(w.tag()+" 回填查询失败", "err", err)
			return
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			if ctx.Err() != nil {
				return
			}
			lastID = row.ID
			if row.FilePath == nil || *row.FilePath == "" {
				continue
			}
			rel := *row.FilePath
			scanned++
			didUpload, err := w.m.Push(rel, w.absUpload(rel))
			if err != nil {
				failed++
				slog.Warn(w.tag()+" 回填失败", "id", row.ID, "rel", rel, "err", err)
				continue
			}
			if didUpload {
				uploaded++
			}
		}
	}
	archived = w.archiveBackfill(ctx)
	if uploaded > 0 || archived > 0 || failed > 0 {
		slog.Info(w.tag()+" 回填完成", "scanned", scanned, "uploaded", uploaded,
			"archived", archived, "failed", failed)
	}
}

func (w *Worker) archiveBackfill(ctx context.Context) int {
	if w.m.BinDir() == "" {
		return 0
	}
	var lastID uint64
	done := 0
	for {
		if ctx.Err() != nil {
			return done
		}
		var rows []store.Content
		err := w.deps.DB.WithContext(ctx).
			Select("id", "file_path").
			Where("compressed_at IS NOT NULL").
			Where("file_path IS NOT NULL AND file_path <> ''").
			Where("id > ?", lastID).
			Order("id ASC").Limit(batch).Find(&rows).Error
		if err != nil {
			slog.Warn(w.tag()+" 原图归档查询失败", "err", err)
			return done
		}
		if len(rows) == 0 {
			return done
		}
		for _, row := range rows {
			if ctx.Err() != nil {
				return done
			}
			lastID = row.ID
			if row.FilePath == nil || *row.FilePath == "" {
				continue
			}
			rel := normRel(*row.FilePath)
			if w.m.archivedRemotely(rel) {
				continue
			}
			if _, found, err := w.m.store.Head(w.m.ArchiveKey(rel)); err != nil {
				slog.Warn(w.tag()+" 原图归档探测失败", "id", row.ID, "rel", rel, "err", err)
				continue
			} else if found {
				w.m.MarkArchived(rel)
				continue
			}
			binPath, ok := FindInBin(w.m.BinDir(), rel)
			if !ok {
				continue
			}
			if _, err := w.m.ArchiveOriginal(rel, binPath); err != nil {
				slog.Warn(w.tag()+" 原图归档失败", "id", row.ID, "rel", rel, "err", err)
				continue
			}
			done++
		}
	}
}

func (w *Worker) absUpload(rel string) string {
	rel = strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
	return filepath.Join(w.deps.Cfg.UploadDir, filepath.FromSlash(rel))
}

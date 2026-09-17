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
	// lockKey 与压缩/推荐任务同一套机制：多实例部署时只有一个实例在回填。
	lockKey = "lock:r2-mirror"
	// lockTTL 覆盖「扫描一批 + 上传若干对象」的耗时；进程被强杀时靠 TTL 自愈。
	lockTTL = 10 * time.Minute
	// batch 是每轮扫描的内容条数，逐批推进，不会一次性把库拉满。
	batch = 500
)

// Worker 负责把「本地有、R2 上没有（或内容不一致）」的文件补齐。
//
// 首次启动立即跑一轮：R2 接入之前的历史数据全部在此时补齐，
// 之后按 R2_SYNC_INTERVAL_SECONDS 周期扫描，兜住上传/压缩时的偶发失败。
type Worker struct {
	deps app.Deps
	m    *Setup
}

// NewWorker 构造回填任务；镜像未启用时返回 nil，调用方无需启动。
func NewWorker(deps app.Deps, m *Setup) *Worker {
	if !m.Enabled() {
		return nil
	}
	return &Worker{deps: deps, m: m}
}

// Start 启动周期回填（立即跑一轮，之后按配置间隔重复）。
func (w *Worker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	slog.Info("r2 镜像任务已启用",
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
				// 退出时主动释放锁，避免残留到 TTL 到期。
				w.deps.Redis.ReleaseLock(context.Background(), lockKey)
				return
			case <-ticker.C:
				w.runRound(ctx)
			}
		}
	}()
}

// runRound 扫描一轮全量内容并补齐缺失对象。
func (w *Worker) runRound(ctx context.Context) {
	if !w.deps.Redis.AcquireLock(ctx, lockKey, lockTTL) {
		return // 另一实例正在回填
	}
	defer w.deps.Redis.ReleaseLock(ctx, lockKey)

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
			slog.Warn("r2 回填查询失败", "err", err)
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
				slog.Warn("r2 回填失败", "id", row.ID, "rel", rel, "err", err)
				continue
			}
			if didUpload {
				uploaded++
			}
		}
	}
	archived = w.archiveBackfill(ctx)
	if uploaded > 0 || archived > 0 || failed > 0 {
		slog.Info("r2 回填完成", "scanned", scanned, "uploaded", uploaded,
			"archived", archived, "failed", failed)
	}
}

// archiveBackfill 补齐「压缩前的原图」归档：只扫 compressed_at 非空的行
//（只有这些行发生过原地替换，原图才在垃圾桶里），远端已有则直接跳过。
// 这是压缩时归档失败的兜底：那一刻原图只剩垃圾桶一份，丢了就再也补不回来。
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
			slog.Warn("r2 原图归档查询失败", "err", err)
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
				slog.Warn("r2 原图归档探测失败", "id", row.ID, "rel", rel, "err", err)
				continue
			} else if found {
				w.m.MarkArchived(rel)
				continue
			}
			binPath, ok := FindInBin(w.m.BinDir(), rel)
			if !ok {
				continue // 原件已不在垃圾桶（人工清理过），无从补传
			}
			if _, err := w.m.ArchiveOriginal(rel, binPath); err != nil {
				slog.Warn("r2 原图归档失败", "id", row.ID, "rel", rel, "err", err)
				continue
			}
			done++
		}
	}
}

// absUpload 把库里的相对路径还原为 uploads 目录下的绝对路径。
// 只处理 uploads（原图/压缩图），thumbs 与 images 由 Push 内部直接跳过。
func (w *Worker) absUpload(rel string) string {
	rel = strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
	return filepath.Join(w.deps.Cfg.UploadDir, filepath.FromSlash(rel))
}

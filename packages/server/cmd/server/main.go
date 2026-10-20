package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/api"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/cli"
	"github.com/huntersxy/xqecz/server/internal/compress"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/creative"
	"github.com/huntersxy/xqecz/server/internal/logx"
	"github.com/huntersxy/xqecz/server/internal/media"
	"github.com/huntersxy/xqecz/server/internal/mirror"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/huntersxy/xqecz/server/internal/recommend"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"github.com/huntersxy/xqecz/server/internal/web"
	"github.com/joho/godotenv"
)

func main() {
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	_ = godotenv.Overload(filepath.Join(root, ".env.local"))

	cfg := config.Load()

	if len(os.Args) > 1 && os.Args[1] == "admin" {
		cli.RunAdmin(cfg, os.Args[2:])
		return
	}
	logs := logx.NewBuffer(500)
	slog.SetDefault(slog.New(logx.NewWithBuffer(os.Stdout, logs)))

	db, err := store.Open(cfg)
	if err != nil {
		slog.Error("mysql init failed", "err", err)
		os.Exit(1)
	}
	mediaMirror := mirror.NewChain(
		mirror.New(mirror.TargetFromR2(cfg.R2), cfg.BinDir),
		mirror.New(mirror.TargetFromOpenList(cfg.OpenList), cfg.BinDir),
	)
	deps := app.Deps{Cfg: cfg, DB: db, Redis: cache.Open(cfg), Mirror: mediaMirror, Logs: logs}
	web.Check(deps)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	images := aigen.New(db, cfg)
	if err := creative.Init(ctx, db); err != nil {
		slog.Error("creative tables init failed", "err", err)
		os.Exit(1)
	}
	if err := images.Init(ctx); err != nil {
		slog.Error("AI image tables init failed", "err", err)
		os.Exit(1)
	}
	images.Start(ctx)
	deps.AI = images
	defer func() { stop(); images.Close() }()
	metadata := vision.New(db, deps.Redis, cfg.UploadDir, filepath.Join(root, "data", "vision-references"))
	if err := metadata.Init(ctx); err != nil {
		slog.Warn("视觉生成初始化失败，普通上传仍可用", "err", err)
	} else {
		deps.Vision = metadata
		metadata.Start(ctx)
		defer func() { stop(); metadata.Close() }()
	}

	recommend.NewRefresher(deps).Start(ctx)

	compress.NewWorker(deps).Start(ctx)

	for _, target := range mediaMirror.Setups() {
		mirror.NewWorker(deps, target).Start(ctx)
	}

	go func() {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		content.New(deps, content.Options{Mirror: deps.Mirror}).SweepMissingThumbnails(ctx)
		ok, fail, err := media.SweepAtlasThumbnails(ctx, cfg.ThumbDir)
		slog.Info("星图缩略图回填完成", "ok", ok, "fail", fail, "err", err)
	}()

	if err := api.Run(deps, content.Options{Mirror: deps.Mirror}); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

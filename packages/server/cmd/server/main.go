package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/huntersxy/xqecz/server/internal/api"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/cli"
	"github.com/huntersxy/xqecz/server/internal/compress"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/huntersxy/xqecz/server/internal/recommend"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"github.com/joho/godotenv"
)

func main() {
	// .env 统一放项目根一份（前端与后端共用）；.env.local 可覆盖，便于本地指向影子库。
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	_ = godotenv.Overload(filepath.Join(root, ".env.local"))

	cfg := config.Load()

	// 运维子命令：部署机只有二进制、没有 Go 工具链，管理员初始化必须由二进制自身提供。
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		cli.RunAdmin(cfg, os.Args[2:])
		return
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	db, err := store.Open(cfg)
	if err != nil {
		slog.Error("mysql init failed", "err", err)
		os.Exit(1)
	}
	deps := app.Deps{Cfg: cfg, DB: db, Redis: cache.Open(cfg)}
	web.Check(deps)

	// 推荐位：启动即刷新一次，之后每 10 分钟刷新（与旧实现节奏一致）。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	recommend.NewRefresher(deps).Start(ctx)

	// TinyPNG 后台压缩：每分钟挑一张最大的待压缩图片（未配置 Key 时自动休眠）。
	compress.NewWorker(deps).Start(ctx)

	// 启动后异步补图（等价于旧实现的启动迁移）：延迟几秒，避免与服务启动争抢 CPU。
	go func() {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		content.New(deps).SweepMissingThumbnails(ctx)
	}()

	if err := api.Run(deps); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

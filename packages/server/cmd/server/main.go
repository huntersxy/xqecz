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
	"github.com/huntersxy/xqecz/server/internal/logx"
	"github.com/huntersxy/xqecz/server/internal/mirror"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/huntersxy/xqecz/server/internal/qiniu"
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
	// 证书续签辅助：certbot 的 auth/cleanup hook 调用，把 HTTP-01 挑战写进七牛桶——
	// img.xiey.work 回源到桶，挑战请求根本到不了部署机。
	if len(os.Args) > 1 && os.Args[1] == "acme" {
		cli.RunAcme(cfg, os.Args[2:])
		return
	}

	slog.SetDefault(slog.New(logx.New(os.Stdout)))

	db, err := store.Open(cfg)
	if err != nil {
		slog.Error("mysql init failed", "err", err)
		os.Exit(1)
	}
	// 媒体镜像：R2 为主，七牛为一级替补（额度闸门放行时才对外下发地址）。
	// 凭据不全时对应目标为停用态，后续推送与回填都自动跳过。
	gate := qiniuGate(cfg)
	mediaMirror := mirror.NewChain(
		mirror.New(mirror.TargetFromR2(cfg.R2), cfg.BinDir),
		mirror.New(mirror.TargetFromQiniu(cfg.Qiniu), cfg.BinDir),
		gate,
		mirror.Signer(cfg.Qiniu.TimeKey),
	)
	deps := app.Deps{Cfg: cfg, DB: db, Redis: cache.Open(cfg), Mirror: mediaMirror}
	web.Check(deps)

	// 推荐位：启动即刷新一次，之后每 10 分钟刷新（与旧实现节奏一致）。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 七牛额度闸门：启动即查一轮，之后按配置周期刷新（凭据不全时不设闸）。
	// 初始为关闭——没查过就不放行，宁可先少一层替补也不要先超支。
	gate.Start(ctx, cfg.Qiniu.QuotaEvery)

	recommend.NewRefresher(deps).Start(ctx)

	// TinyPNG 后台压缩：每分钟挑一张最大的待压缩图片（未配置 Key 时自动休眠）。
	compress.NewWorker(deps).Start(ctx)

	// 回填：每个可用目标各跑一个任务、各持一把锁，互不阻塞
	//（替补层被额度闸门停用时不启动，也不会白推）。
	for _, target := range mediaMirror.Setups() {
		mirror.NewWorker(deps, target).Start(ctx)
	}

	// 启动后异步补图（等价于旧实现的启动迁移）：延迟几秒，避免与服务启动争抢 CPU。
	go func() {
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}
		content.New(deps, content.Options{Mirror: deps.Mirror}).SweepMissingThumbnails(ctx)
	}()

	if err := api.Run(deps, content.Options{Mirror: deps.Mirror}); err != nil {
		slog.Error("server exited", "err", err)
		os.Exit(1)
	}
}

// qiniuGate 构造七牛额度闸门：凭据齐备才建，否则返回 nil（替补层随之停用）。
// 闸门不在此处启动——它需要 ctx，由 main 在 ctx 就绪后调用 Start。
func qiniuGate(cfg config.Config) *qiniu.Gate {
	if !cfg.Qiniu.Enabled() {
		return nil
	}
	client, err := qiniu.New(qiniu.Config{
		Signer: qiniu.Signer{
			AccessKey: cfg.Qiniu.AccessKey,
			SecretKey: cfg.Qiniu.SecretKey,
		},
		Bucket:    cfg.Qiniu.Bucket,
		CDNDomain: cfg.Qiniu.CDNDomain(),
	})
	if err != nil {
		slog.Warn("七牛额度闸门不可用，替补层停用", "err", err)
		return nil
	}
	q := qiniu.DefaultQuota()
	q.Threshold = cfg.Qiniu.Threshold
	q.EgressBytes = cfg.Qiniu.EgressCapBytes
	return qiniu.NewGate(client, q)
}

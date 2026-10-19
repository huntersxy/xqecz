# AGENTS.md — xqecz monorepo

## 项目与栈

小泉动漫二创站（xqecz）：用户上传/浏览二次创作内容（文本 + 可选图片），含评论、投票、认领与管理后台。Monorepo = Go 单进程后端 + Vue 3 前端，前端 pnpm workspace、后端 go module、任务经 moon 编排。

技术栈：后端 Go / gin / GORM(MySQL·TiDB) / go-redis / slog；媒体 ffmpeg + 纯 Go WebP + TinyPNG；前端 Vue 3 / TS / Vite / Tailwind / Arco Design Vue / zod；数据 MySQL·TiDB + Redis；生产后端 CI+OpenRC、前端 EdgeOne Makers。

## 快速开始

前置：Node ≥ 20 + pnpm、Go 1.26+、FFmpeg（可选，缺失时图片缩略图降级纯 Go、视频缩略图不可用）；不用 Docker，开发期 MySQL/Redis 连云端；项目根 `.env` 需存在（gitignore）。

```bash
pnpm install          # 前端依赖（含 moon CLI）
pnpm dev              # 并发起后端(:3000) + Vite(:5173)，端口自适应，退出递归清理
pnpm dev:server       # 仅后端（go run ./cmd/server）；pnpm dev:fe 仅前端
pnpm build            # 后端二进制 + 前端产物（Go 侧剥符号表）
pnpm server:test      # go test ./...    pnpm server:vet  # go vet ./...
pnpm fe:test          # vitest 全量      pnpm fe:typecheck # vue-tsc
pnpm accept           # 对运行中的后端跑全接口验收并清理数据（XQECZ_BASE，默认 127.0.0.1:3000）
pnpm exec moon run server:build   # 单目标；moon query projects 看工程图；多目标 moon run server:test frontend:test
cd packages/server && go run ./cmd/dbinfo # 账号权限与表规模；dbsql "SELECT 1" 查库
./packages/server/xqecz-server admin -username admin [-email a@b.com] [-reset]  # 无 Go 工具链的部署机也可用
```

## 架构速览

```
前端(Vue3) ──/api──→ Go 单进程（packages/server）
  ├─ gin 中间件（Session / API 密钥 / 管理员 / CORS）；GORM → MySQL/TiDB；go-redis → Redis（会话/缓存/浏览量/推荐 ZSet）
  ├─ 进程内：缩略图（ffmpeg，缺失降级纯 Go）、推荐打分、视觉模型客户端
  ├─ 后台：TinyPNG 定时压缩（每轮挑最大待压缩图，原图入 data/bin）；可选镜像（OpenList 与 R2 并行推送）
  └─ 静态 /uploads、/thumbs、/images 由同进程托管，物理目录为项目根 data/
```

## 红线

- 不把任一份 `.env`、会话、默认凭据或整份配置打印进日志、提交或贴进对话。
- 不对线上库跑非正式 migration 或破坏性数据操作（`scripts/migrations/` 是唯一入口，不开 `AutoMigrate`）。
- 不擅自提交、推送、发布；发布不以「构建通过 / 部署完成」代替站点验收。
- 不在 `data/bin` 之外真删媒体；删内容走 `purgeContent` + 引用计数。


# AGENTS.md — xqecz monorepo

## 项目概述

小泉动漫二创站（xqecz）— 用户上传/浏览二次创作内容（图片、图文，内容类型已缩窄为 image/text），含评论、管理后台。

**Monorepo 架构**：NestJS 主后端（API）+ Go 无状态 Worker（文件处理 / 推荐打分）+ Vue 3 前端，通过 pnpm workspace 统一管理。

## 架构

```
前端(Vue3) ──HTTP──→ NestJS API ──TypeORM──→ MySQL
                       │   │  ▲
                       │   │  └── Redis（session / cache / 浏览量 / 推荐 ZSet）
                       │   │
                       │   └──gRPC──→ Go Worker（无状态计算）
                       │                  ├─ GenerateThumbnail / CompressImage（文件处理，读写共享上传目录）
                       │                  ├─ FetchLinkPreview（OG 解析）
                       │                  └─ RefreshRecommend（纯打分，输入来自 NestJS，输出评分回传 NestJS）
                       │
                       └── 共享上传目录（UPLOAD_DIR）：NestJS 与 Worker 指向同一目录、互认绝对路径，处理缩略图/压缩
```

**推荐链路（重要）**：NestJS 独占 DB/Redis —— worker **不**直连 MySQL/Redis。
推荐刷新由 api 主导：`ContentService.refreshRecommend()` 读 MySQL approved 内容 → 组装 `RecommendItem[]` 经 gRPC `RefreshRecommend` → worker 纯计算返回 `ScoredItem[]` → api 用 `RedisService.writeRecommendList()` 原子写入 Redis ZSet `recommend:hot`（ioredis `keyPrefix=xqecz:` → 实际 key `xqecz:recommend:hot`）。读取时 `recommend()` 优先读 ZSet，无数据/异常降级到 `view_count` 排序。

## 目录结构

```
D:\xqecz/
├── packages/
│   ├── api/               # NestJS 主后端（TypeORM + MySQL + ioredis）
│   │   ├── src/
│   │   │   ├── auth/      # 认证模块（Redis Session）
│   │   │   ├── content/   # 内容模块（CRUD / 搜索 / 推荐 / 媒体管线）
│   │   │   ├── comment/   # 评论模块（树形评论 / 举报）
│   │   │   ├── poll/      # 投票模块
│   │   │   ├── admin/     # 管理后台（含 refresh-recommend / regenerate 端点）
│   │   │   ├── api-key/   # API 密钥管理
│   │   │   ├── entities/  # TypeORM Entity（8 张表）
│   │   │   ├── guards/    # AuthGuard / AdminGuard / OptionalAuthGuard
│   │   │   ├── redis/     # RedisService（session / cache / view / recommend 读写）
│   │   │   ├── worker/    # WorkerService（gRPC client → Go worker）
│   │   │   └── decorators/# CurrentUser
│   │   ├── .env           # 本地测试配置（gitignore，含 DB/Redis 凭据）
│   │   └── package.json
│   │
│   ├── worker/            # Go 无状态 gRPC 微服务（不连 DB/Redis，无 cron）
│   │   ├── cmd/server/    # 入口（gRPC server）
│   │   ├── server/        # WorkerServer 实现（worker.go / recommend.go / worker_test.go）
│   │   ├── config/        # env 驱动配置（TINIFY_*/UPLOAD_DIR，无 DB/Redis 配置）
│   │   ├── media/         # 缩略图（ffmpeg）、Tinify 压缩
│   │   ├── linkpreview/   # OG/Twitter Card 解析
│   │   ├── proto/         # Go gRPC stub（从 proto/ 生成后复制）
│   │   └── go.mod
│   │
│   └── frontend/          # Vue 3 前端（本仓内 packages/frontend）
│       └── src/views/     # 路由页：首页瀑布流 WaterfallTheme；详情页 ContentDetailView 为全屏覆盖式路由页（/content/:id）
│
├── proto/
│   ├── xqecz.proto        # gRPC protobuf 定义（4 文件处理/推荐方法 + Health）
│   ├── gen/               # 生成产物（go/ + ts/）
│   └── package.json
│
├── scripts/
│   └── run-worker.mjs     # 启动 Go Worker（从 packages/api/.env 注入 UPLOAD_DIR）
│
├── pnpm-workspace.yaml
├── package.json            # 根脚本（dev / build / start / worker:build）
└── AGENTS.md               # ← 本文件
```

## gRPC 接口（proto/xqecz.proto）

| 方法 | 入参 | 返回 | 说明 |
|------|------|------|------|
| `Health` | — | status/version | 健康检查 |
| `GenerateThumbnail` | file_path, content_type | thumb_path, success, error | ffmpeg 抽帧/缩放 → webp |
| `CompressImage` | file_path | compressed_path, success, error | Tinify 压缩（无 key 即跳过） |
| `FetchLinkPreview` | url | title/image/platform, success, error | 解析 OG 元数据 |
| `RefreshRecommend` | items[]（content_id/created_at_unix/view_count/like_count） | results[]（content_id/score）, success | **纯打分**，不碰 DB/Redis；like_count 权重高于 view_count |

> gRPC client 已设 `loader: { keepCase: true }`，proto 字段用 snake_case，与 Go worker 一致。

## 快速命令

> 项目**不使用 Docker**，全部通过 pnpm 脚本本地直启（MySQL/Redis 连云端实例，本机无需安装）。
> 前置要求：Node.js ≥ 20 + pnpm、Go 1.25+（worker 编译需 GOPROXY 可达）、FFmpeg（可选，缩略图用，缺失即降级）。

```bash
# ── 首次准备 ──
pnpm install --shamefully-hoist            # 装依赖（传递依赖需提升；CI=true 可跳过 TTY 确认）
# 确认 packages/api/.env 存在（gitignore，含云端 MySQL/Redis 凭据与 UPLOAD_DIR/WORKER_URL）

# ── 一键开发（推荐）──
pnpm dev                                   # 并发起三端：api(nest --watch :3000) + worker(go run :50051) + 前端(Vite :5173)
# 说明：api 读 packages/api/.env；worker 经 scripts/run-worker.mjs 启动，自动从同一 .env 注入 UPLOAD_DIR，两端目录天然一致。
# 前端 Vite 已把 /api、/uploads 等代理到 http://localhost:3000（可用 VITE_PROXY_TARGET 覆盖）。

# ── 一键生产运行 ──
pnpm start                                 # = pnpm build（三端全量构建）+ pnpm start:services
pnpm start:services                        # 跳过构建直接起：api(node dist/main :3000) + worker + 前端(vite preview :4173)

# ── 拆分命令 ──
pnpm dev:api                               # 仅 NestJS API 开发（热重载，:3000）
pnpm dev:worker                            # 仅 Go Worker（go run :50051）
pnpm dev:fe                                # 仅前端 Vite 开发服务器（:5173）
pnpm build                                 # 三端全量构建（api dist/ + 前端 dist/ + worker 二进制）
pnpm worker:build                          # 仅 Worker 构建
pnpm --filter ./packages/api run typecheck # API 类型检查
cd packages/worker && go test ./...        # Worker 测试
pnpm --filter ./packages/frontend run type-check # 前端类型检查
pnpm --filter ./packages/frontend run build # 前端生产构建
pnpm proto:generate                        # 生成 ts/go stub
```

## 核心约束

- **前端是契约** — `packages/frontend/src/api/index.ts` 是唯一接口定义（前端已并入本仓 `packages/frontend`，不再独立仓库）
- **NestJS 独占 DB/Redis** — Go worker 不访问 MySQL/Redis，也不含 cron 调度；它只做 gRPC 无状态计算，数据经 gRPC 从 NestJS 传入、结果回传 NestJS 落库/写缓存
- **API 密钥认证** — `AuthGuard` 双模式：请求头 `X-API-Key`（sha256 比对 `api_keys.key_hash`，`req.user.api_key` 携带权限）或 Session Cookie；`ApiKeyPermissionGuard` + `@RequireApiKeyPermission('upload'|'delete'|'read')` 仅约束密钥调用，Session 用户不受限。新增受保护接口时按此模式挂守卫
- **共享上传目录** — 文件处理路径通过 gRPC 传入绝对路径，NestJS 与 Go 必须指向同一 `UPLOAD_DIR`；单一配置源为 `packages/api/.env`，worker 由 `scripts/run-worker.mjs` 启动时自动读取该 .env 注入 `UPLOAD_DIR/THUMB_DIR/IMAGES_DIR`
- **统一响应** — `{ code, message, data }` 包装格式
- **内容类型已缩窄** — `content.type` 值域仅 `image` / `text`（按“是否有 file”自动推导）；存量 `video` / `link` 记录走一次性迁移归并为 `text`（`POST /admin/content/migrate-old-types`）
- **软删除** — 所有删除写 `deleted_at`；Entity 已声明 `@DeleteDateColumn()`，TypeORM 的 `find/findOne/findAndCount` 查询自动附加 `WHERE deleted_at IS NULL`，业务代码无需手动过滤
- **降级优先** — 外部依赖（Tinify/Worker）缺失即降级，gRPC 永不返 rpc error，只返 `success=false` + `error` 文本

## 技术栈

| 层 | 技术 |
|----|------|
| API 后端 | NestJS 11 + TypeORM + MySQL + ioredis + @nestjs/microservices(gRPC) |
| Worker | Go 1.25 + gRPC + FFmpeg（**无** DB/Redis/cron 依赖） |
| 前端 | Vue 3.5 + TypeScript + Vite + Tailwind CSS + Arco Design Vue |
| 通信 | gRPC（NestJS → Worker，snake_case via keepCase） |
| 数据库 | MySQL + Redis |
| 运行方式 | pnpm 脚本本地直启（concurrently 并发三端，无 Docker） |

## 修改指南

1. **先读 `packages/frontend/AGENTS.md`** — 理解前端接口契约
2. **API 改动** — 在 `packages/api/src/` 对应模块中改（entity → service → controller）；模块需用 `@UseGuards` 类引用 guard 时，必须 `import { AuthModule }` 并让其 export 该 guard + `TypeOrmModule`
3. **Worker 改动** — 在 `packages/worker/server/` 中实现纯计算逻辑；涉及文件处理就读 `file_path` 绝对路径、写回结果路径
4. **新增/修改 gRPC 接口** — 先改 `proto/xqecz.proto` → `pnpm --filter @xqecz/proto run generate` 生成 stub → 实现 Go 端 + 在 `packages/api/src/worker/worker.service.ts` 调
5. **推荐算法改动** — 只改 `packages/worker/server/recommend.go:computeRecommend()`（纯函数，输入 `RecommendItem`，输出 `ScoredItem`）；刷新节奏/落库在 api `content.service.ts:refreshRecommend()`（多实例通过 Redis 分布式锁防抖，见 `RedisService.acquireLock()`）
6. **数据库变更** — 改 `packages/api/src/entities/`，生产用正式 migration；`synchronize` 仅本地/测试开，勿在生产长期开启
7. **踩坑记忆** — TypeORM `bigint` 主键返字符串，与 Redis ZSet 数值成员比对需 `String()` 归一化；`tsconfig.json` 需 `esModuleInterop:true`（CJS 默认导入）

## 已归档

- 旧后端 `xqecz-golang/`、`xqecz-nodejs/` 已完整迁移进本 monorepo，并归档至 `D:/xqecz/archive/`（各自独立 git 仓库，完整历史保留；均打本地 tag `archive/monorepo-migration-2026-07-22`，远程 `xqecz-all.git` 的 `golang`/`nodejs` 分支亦保留）。monorepo 已 gitignore `/archive/`。
- 独立前端仓库 `xqecz_frontend` 已并入本仓 `packages/frontend`（源文件直接纳入 monorepo，不再独立仓库/symlink）；原独立仓库整体移至 `D:/xqecz/archive/xqecz_frontend` 保留历史（远程 `xqecz_frontend.git` 的 `dev` 分支亦保留）。

# xqecz 后端 Go 化改造方案（历史文档）

> 状态：**已完成**（2026-09 上线，NestJS + Go Worker 双进程已被单一 Go 进程取代）。
> 本文件保留为迁移期的设计记录与接口契约参照；**现状请以 `AGENTS.md` 与代码为准**。
> 原目标：在**不改变前端调用方式**的前提下，用单一 Go 进程替换原有 NestJS API + Go Worker 双进程后端。

## 1. 背景

现有后端为 NestJS（HTTP/DB/Redis/会话）+ Go Worker（gRPC 无状态计算）双进程：
接口面 60 条 REST 路由，业务逻辑分散在 TypeORM Repository 与手写 SQL 之间，运行期常驻两个进程、两套依赖。

## 2. 不可动摇的约束（前端零改动契约）

以下约定须逐字节保持，否则前端即需改动：

| 项 | 约定 |
|----|------|
| 基础路径 | `/api` 前缀（前端 `VITE_API_BASE_URL` 默认 `/api`） |
| 响应包装 | `{ code, message, data }`，`code === 200` 为成功 |
| 错误通道 | HTTP 状态码 + body `{ message }`（前端 `pickServerMessage` 读 `message` 或 `data.message`） |
| 会话 | Cookie `session_id`，`httpOnly; sameSite=lax; path=/; maxAge=30d`；Redis key `xqecz:session:{64hex}`，读取时滑动续期 |
| 游客标识 | Cookie `visitor_id`（投票去重用），`maxAge=365d` |
| 密码 | bcrypt（`$2a$/$2b$`），与 Go `x/crypto/bcrypt` 互通，存量哈希可直接复用 |
| 静态资源 | `/uploads` → `data/uploads`、`/thumbs` → `data/thumbs`、`/images` → `data/images` |
| 上传约定 | multipart 字段名 `file`；原文件扁平落 `data/uploads/<md5>.<ext>`；单文件 ≤ 20MB；仅 `image/*`、`video/*` |
| 字段命名 | 请求/响应一律 snake_case（`page_size`、`audit_status`、`like_count`…） |
| 端口 | 开发 API :3000（前端 Vite 代理 `/api`、`/uploads` 至此） |

## 3. 冻结接口清单（60 条）

认证类：`health` 1 条、`auth` 6 条、`api-keys` 4 条。
业务类：`content` 14 条、`comment` 5 条、`poll` 5 条。
管理类：`admin` 18 条（全部要求 `AuthGuard + AdminGuard`）。

### auth / health

| 方法 | 路径 | 认证 | 入参 → 出参要点 |
|------|------|------|-----------------|
| GET | /api/health | — | `{ok:true}` |
| POST | /api/auth/register | — | `{username,email,password}` → `{user_id}` |
| POST | /api/auth/login | — | `{username,password}` → Set-Cookie + `{user,needs_email}` |
| POST | /api/auth/logout | — | 清 Cookie，`data:null` |
| POST | /api/auth/change-password | Session | `{oldPassword,newPassword}` |
| GET | /api/auth/me | Session | 用户对象 |
| PUT | /api/auth/email | Session | `{email}`，成功后失效全部内容缓存 |

### content

| 方法 | 路径 | 认证 | 备注 |
|------|------|------|------|
| GET | /api/content/list | — | `page,page_size,tag,keyword,sort_by,order,audit_status`；默认可见范围 `approved+pending` |
| GET | /api/content/search | — | keyword 为空直接返回空分页 |
| GET | /api/content/recommend | — | `count,page`；Redis ZSet `xqecz:recommend:hot`，缺数据降级 view_count |
| GET | /api/content/tags | — | 标签数组 |
| GET | /api/content/my | Session | 我的内容 |
| GET | /api/content/:id | 可选 Session | `silent=1` 不计浏览量 |
| POST | /api/content/upload | Session 或 API-Key(upload) | multipart |
| POST | /api/content/quick-upload | 公开 | 有文件时按 IP 限频 20 次/小时（Redis `quick_upload:ip:*`） |
| PUT | /api/content/:id | Session 或 API-Key(upload) | 仅作者或管理员；编辑后回到待审 |
| DELETE | /api/content/:id | Session 或 API-Key(delete) | 软删除 |
| POST | /api/content/:content_id/claim | Session | 提交认领 |
| POST | /api/content/:content_id/like | Session | 切换点赞，返回 `{liked,like_count}` |
| GET | /api/content/:content_id/like-status | Session | `{liked,favorited,like_count}` |
| POST | /api/content/:content_id/favorite | Session | 切换收藏 |

### comment / poll

| 方法 | 路径 | 认证 |
|------|------|------|
| GET | /api/comment/list/:content_id | — |
| GET | /api/comment/count/:content_id | — |
| POST | /api/comment/add | Session |
| DELETE | /api/comment/:id | Session |
| POST | /api/comment/report | Session |
| GET | /api/poll/list | — |
| GET | /api/poll/:id | 可选 Session |
| POST | /api/poll/:id/vote | 可选 Session（游客用 visitor_id） |
| POST | /api/poll/create | Session |
| DELETE | /api/poll/:id | Session |

### admin（全部 Session + 管理员）

`POST /api/admin/audit/:id`、`GET /api/admin/pending`、`GET /api/admin/content/all`、
`PUT /api/admin/content/:id/author`、`DELETE /api/admin/content/purge`、
`GET /api/admin/users`、`GET /api/admin/dashboard`、`PUT /api/admin/users/:id/role`、
`PUT /api/admin/users/:id/ban`、`DELETE /api/admin/users/:id`、
`GET /api/admin/comments/reports`、`POST /api/admin/comments/reports/:id/handle`、
`GET /api/admin/claims`、`POST /api/admin/claims/:id/handle`、
`POST /api/admin/content/:id/regenerate-thumbnail`、
`POST /api/admin/content/regenerate-all-thumbnails`、
`GET /api/admin/content/regenerate-all-thumbnails/status`、
`POST /api/admin/content/refresh-recommend`。

### api-keys（Session）

`POST /api/api-keys`、`GET /api/api-keys`、`PUT /api/api-keys/:id`、`DELETE /api/api-keys/:id`。

## 4. 目标架构

```
packages/frontend (Vue 3，改动最小)
        │  /api/*  /uploads/* /thumbs/* /images/*
        ▼
packages/server (单一 Go 二进制)
        ├─ HTTP 层：路由 + 中间件（会话/鉴权/限流/CORS/日志）
        ├─ 领域层：content / comment / poll / auth / admin
        ├─ 数据层：MySQL（sqlc 生成类型安全查询）
        ├─ 缓存层：Redis（会话 / 读穿缓存 / 浏览量 / 推荐 ZSet）
        └─ 媒体层：缩略图与转码走 ffmpeg 子进程，图片转 WebP 走纯 Go 实现
```

原 Go Worker 的 gRPC 边界取消（打分与媒体处理并入同一进程，仍是纯函数 + 子进程，不引入额外网络跳数）。

## 5. 技术选型候选

### 5.1 HTTP

| 方案 | 取舍 |
|------|------|
| **stdlib `net/http` + Go 1.22+ `ServeMux`** | 零依赖、常驻内存最低、路径参数原生支持；中间件需手写约 60 行 |
| `chi` | 极轻量、与 stdlib 完全兼容，多一层依赖 |
| `gin` / `echo` | 生态最大，自带绑定/校验，反射与分配更多 |
| `fiber` | 基于 fasthttp，吞吐高但脱离 stdlib 生态，内存模型不同 |

### 5.2 数据访问

| 方案 | 取舍 |
|------|------|
| **`sqlc`（SQL → 类型安全 Go）** | 零反射、零运行期开销、编译期校验 SQL；需维护 schema.sql 与查询文件 |
| `sqlx` | 手写 SQL + 结构体扫描，灵活但无编译期校验 |
| `ent` / `GORM` | 实体抽象接近 TypeORM，迁移成本低，但反射/分配更重 |

### 5.3 Redis

| 方案 | 取舍 |
|------|------|
| **`rueidis`** | 自动流水线、RESP3、客户端缓存，分配与延迟更优；生态较新 |
| `go-redis/v9` | 主流稳妥，行为与 ioredis 最接近（含 keyPrefix 语义） |

### 5.4 图片处理

| 方案 | 取舍 |
|------|------|
| **纯 Go（`gen2brain/webp` WASM libwebp + `disintegration/imaging`）** | 无 cgo，交叉编译与部署简单，内存可控 |
| cgo libwebp | 性能略好，需 CGO 与目标平台工具链，产物变大 |
| 保留独立媒体子进程 | 隔离性好，但多一个进程与 IPC 成本 |

## 6. 数据层兼容

- 表结构保持现状（10 张表），不新增迁移；Go 侧以 `schema.sql` 描述既有结构供 sqlc 生成。
- `id` 为 bigint，Go 侧统一用 `int64`，对外 JSON 仍输出数字。
- 软删除语义保留：所有查询附加 `deleted_at IS NULL`。
- 启动时不再自动建管理员账号（原 NestJS 行为改为显式 CLI 子命令，避免生产意外写入）。

## 7. 媒体与推荐链路

- 缩略图：ffmpeg 子进程抽帧/缩放 → WebP，落 `data/thumbs`；ffmpeg 缺失时降级为成功但不产出文件（与原 worker 行为一致）。
- 推荐打分：纯函数内联（原 `computeRecommend`），刷新由定时/写操作触发，结果写 Redis ZSet `xqecz:recommend:hot`，多实例用 Redis 锁防抖。

## 8. 仓库 / 脚手架 / CI / 部署

- 前端仍由 pnpm workspace 管理，新增 `packages/server`（独立 Go module），根 `pnpm` 脚本作为统一入口。
- 代码生成工具（sqlc 等）用 Go 1.24+ 的 `go.mod` `tool` 指令声明，避免全局安装漂移。
- CI：Go 端 `go vet` + `go test`；前端沿用现有 job；二者仍由同一 push 触发。
- 部署：产物由「NestJS dist + worker 二进制 + 前端 dist」变为「Go 单二进制 + 前端 dist」。

## 9. 迁移步骤与验收

1. **契约对拍**：以现有 NestJS 为基准，对同一批请求录制响应；Go 实现逐字段比对（JSON 归一化后深比较），差异即回归。
2. 骨架 → 认证/会话 → 内容读路径 → 写路径与上传 → 评论/投票 → 管理端 → 媒体与推荐。
3. 每阶段以对拍脚本 + 前端联调（真实页面跑通首页/详情/上传/后台）作为验收。
4. 合并前删除 NestJS 与 proto/gRPC 残留，并同步 `AGENTS.md`。

## 10. 风险

- 现有业务逻辑分散在 service 与手写 SQL 中，存在未文档化的隐式约定（缓存失效点、审核状态流转、限频）。
- 生产为宝塔面板 Node 项目（`xqecz2`）通过 FTP 上传 dist 后重启，切换为 Go 二进制需同步调整部署方式。
- 云 MySQL/Redis 为共享实例，对拍与联调会产生真实读写，需要可回滚的数据策略。
## 11. 已定决策（本轮）

| 决策项 | 结论 |
|--------|------|
| HTTP 框架 | gin（成熟、绑定与校验开箱可用，中间件生态完整） |
| 数据访问 | GORM v2 + MySQL 驱动；复杂聚合用原生 SQL，不做过度抽象 |
| Redis | go-redis v9（行为最接近现有 ioredis，前缀语义由包装层统一实现） |
| 其他 | log/slog 结构化日志、godotenv 读根 .env、x/crypto/bcrypt |
| 迁移节奏 | 一次性全量重写，60 条接口齐备后整体切换；旧实现保留至切换前作为对拍基准 |
| 同仓管理 | moon（moonrepo）统一任务图与缓存 + go module（后端）/ pnpm workspace（前端） |
| 部署 | 本轮不动；后端本地跑通并完成对拍后单独处理 |

选型理由：以「成熟完善的库直接用、避免自研基础设施」为准绳；常驻内存与延迟不作为约束条件，
但相比 NestJS 双进程仍有数量级改善。

## 12. 实施进度

| 阶段 | 状态 | 证据 |
|------|------|------|
| 骨架（配置/数据/缓存/HTTP 外壳） | 完成 | `/api/health` 返回标准包装，MySQL 与 Redis 连通 |
| 认证与会话 | 完成 | 13 项接口场景冒烟通过，文案与状态码与旧接口一致 |
| API 密钥通道 | 完成 | 有效密钥返回调用者身份，伪造密钥返回 401 |
| 内容读路径 | 完成 | 列表/搜索/推荐/标签/详情，含读穿缓存、浏览量、可见性规则 |
| 内容写路径与媒体 | 完成 | 上传/快速上传/编辑/删除/认领/点赞/收藏，含 WebP 无损转换、ffmpeg 缩略图、软删除与缓存失效 |
| 评论 / 投票 | 完成 | 树形评论（一层回复）、评论计数、举报；投票的列表/详情/投票/创建/删除，游客按 visitor_id 去重 |
| 管理端 | 完成 | 审核/待审/全量内容、用户管理、仪表盘聚合、举报与认领处理、缩略图单条与批量重建、推荐刷新触发 |
| 推荐打分与刷新 | 完成 | 纯函数打分并入本进程，Redis 锁防抖，启动即刷新 + 每 10 分钟周期刷新 |

迁移期辅助工具（`packages/server/cmd/`）：`dbsync` 复制库结构、`dbinfo` 查看权限与表规模、`dbsql` 执行排查语句。

### 迁移中修正的既有缺陷

- **标签筛选未生效**：旧 `ContentService.list()` 只在缓存键里使用 `tag`，查询条件里没有标签过滤，首页标签点击后返回的仍是全部内容。
  新实现按 `JSON_CONTAINS(tags, JSON_QUOTE(?))` 实现多标签任一匹配，列表与搜索两条路径都生效。
- **rejected 内容可见性**：保持旧语义（仅作者本人与管理员可见），并确保这类内容不进入读穿缓存。

### 媒体与上传的实现取向

- **上传不落内存**：用 `Request.MultipartReader()` 流式解析，文件边收边写盘（多读 1 字节判定超限），
  字段值单独收集；替代 multer 的内存缓冲方案，20MB 上限下常驻内存不随上传增长。
- **媒体处理进程内完成**：原 Worker 的缩略图生成与推荐打分并入同一进程，取消 gRPC 与跨进程文件共享约定；
  ffmpeg 缺失时图片缩略图自动降级为纯 Go 解码 + `x/image/draw` 缩放，视频仍依赖 ffmpeg（缺失仅告警）。
- **原图无损 WebP 化**：非 GIF 图片上传时转 WebP 后删除源文件，失败则保留原图并继续上传。

### 评论与投票的行为要点

- 评论树只组装一层回复：顶层按 `parent_id IS NULL` 分页，回复按顶层 id 批量取回后按父节点归组，
  顶层响应固定带 `replies` 字段（无回复时为 `[]`），回复项本身不再嵌套。
- 评论列表、评论计数分别按 `comments:{cid}:{page}:{size}` 与 `comment_count:{cid}` 缓存；
  发表/删除评论时按内容 id 失效这两类缓存。
- 投票去重：登录用户按 `user_id`，游客按 `visitor_id` Cookie；改投其他选项只更新选择，
  不重复累加 `vote_count`（仅首次投票累加）。
- 举报接口返回原始记录（`handled` 为 0/1 数值），管理端列表才转换为布尔值，与旧实现一致。

### 管理端实现要点

- 仪表盘每张表只发一条条件聚合 SQL（`SUM(col = 'x')` 直接统计），结果缓存 60 秒，`fresh=1` 跳过读缓存但仍写回。
- 热门标签沿用「只统计已通过内容」的口径；同数量标签用稳定排序保持首次出现顺序，避免 Go map 迭代随机导致结果漂移。
- 批量缩略图：Redis 锁 + 30 秒心跳续期，进度写 `task:regen-all-thumbs:status`（7 天 TTL），接口立即返回、后台逐条处理。
- 认领通过后会把内容作者变更给认领人，并失效内容缓存。
- 迁移中修正的缺陷：用户列表与认领列表的分页总数与列表内容原先可能不一致（过滤条件只作用于计数查询），
  现统一用同一份条件会话执行 Count 与 Find。

## 13. 切换记录

后端实现完整后已移除 NestJS 与 Go Worker 双进程形态：

- 删除 `packages/api`、`packages/worker`、`proto/` 与 `scripts/run-worker.mjs`、`scripts/sync-deps.mjs`；
  仓库只保留 `packages/server`（Go）与 `packages/frontend`（Vue）。
- `scripts/dev.mjs` 由三端编排改为两端：Go 后端（`go -C packages/server run ./cmd/server`）+ Vite，
  端口自适应与进程树自愈逻辑保留。
- 根脚本改为 `dev` / `build:server` / `server:test` / `start:backend` 等；
  `pnpm-workspace.yaml` 去掉 `proto` 包与仅 NestJS 需要的构建放行项（protobufjs、multer 覆盖）。
- 前端新增契约冒烟测试 `src/api/__tests__/contract.live.test.ts`：设置 `XQECZ_LIVE_API` 时，
  用前端自己的 zod schema 校验运行中后端的真实响应；未设置时自动跳过，不影响 CI。

### 同仓管理随切换的调整

- CI（`ci.yml`）：job 由 `api` 改为 `server`（go vet + go test + CGO_ENABLED=0 构建），前端 job 不变；
  覆盖率上传路径与 flag 同步改为 server。
- 部署（`deploy.yml`）：产物由「NestJS dist + worker 二进制 + 前端 dist」变为「Go 单二进制（linux/amd64）+ 前端 dist」；
  仍走 FTP + 宝塔项目停启，但宝塔侧启动命令需改为运行 `packages/server/xqecz-server`。
- 依赖机器人（`dependabot.yml`）：gomod 目录改为 `/packages/server`。
- PR 模板：验证项与降级说明同步为 Go + 前端两端的命令。

### 启动补图与一次审计自误

- 旧实现在启动迁移里为「有原文件但缺缩略图」的内容补图，新实现以 `SweepMissingThumbnails` 恢复该行为：
  延迟 5 秒后台执行、单次上限 200 条、与批量重建共用同一把 Redis 锁，无待补内容时保持静默。
- 审计该行为时出现一次自误：审计 SQL 只写了 `file_path IS NOT NULL`，把 6 条 `file_path = ''` 的纯文本行
  误判为「缺图内容」，而服务端查询（额外带 `file_path <> ''`）正确地返回 0 条，两者对不上导致排查数轮。
  结论：`contents.file_path` 的「无媒体」有两种写法（NULL 与空串），审计与过滤都要同时判断。
- 同一轮补上的单元测试：`recommend.ScoreItem` / `Compute` 的权重与饱和边界、
  `listOpts` 缓存键的规范化（顺序无关、非法排序回落、参数不同键不同）与分页边界收敛。

### 索引补齐与可重复验收

- **索引体检**：`contents` / `comments` / `api_keys` 此前只有主键索引，首页列表、评论列表与 API 密钥鉴权
  在 EXPLAIN 里都是 `type=ALL` + `Using filesort`。已按「等值列 → IN 列 → 排序列」补 7 组二级索引
  （脚本 `scripts/migrations/2026-09-13-add-hot-path-indexes.sql`，已在线应用）。效果：
  `api_keys` 鉴权由全表扫变为 `ref const,const` 命中 1 行；内容列表由 `ALL` 变为 `range` + `Using index`（覆盖索引）；
  评论顶层列表与举报列表均走 `ref`。`contents.tags` 的 JSON_CONTAINS 无法用 B-Tree 索引，数据量大时需另做标签表。
- **验收脚本** `scripts/acceptance.mjs`（`pnpm accept`）：覆盖公开读路径、登录态、上传与互动、投票去重、
  API 密钥鉴权与权限拒绝、管理端审核与统计、软删除可见性，共 44 项断言；结束后清理全部测试数据与上传文件。
  实测 44/44 通过、残留为 0。
- **时延观测**：三条路径的外部往返次数不同，读数差异主要来自本机到云 MySQL/Redis 的网络距离——
  列表（缓存命中）1 次 Redis 往返、详情（缓存命中）含浏览量写库与计数、详情（`silent=1`）按旧语义跳过缓存读需回表组装。

### 并发与时延实测（第 12 轮）

- **生产产物**：`vite preview` 起 `dist` 后代理到 Go 后端可正常取数；`index.html` 引用的 11 个资源全部存在，产物自洽。
- **并发分层探测**（20 并发，本机到云 MySQL/Redis RTT 约 30ms）：
  `/api/health` 2ms（纯内存）→ `/api/content/list` 57ms（1 次 Redis 往返）→ `/api/content/:id?silent=1` 616ms。
  定位到瓶颈是 `MYSQL_POOL_SIZE=5`：silent 路径跳过缓存读、每请求 3 次查询，20 并发即打满连接池排队。
  把池调到 30 复测，同一负载下 p50 降到 304ms、墙钟从 1178ms 降到 339ms（吞吐 3.5 倍）——示例配置已更新该参数与依据。
- **200 并发混合流量**：0 失败、无服务端告警、压测后健康检查正常；列表路径无串号。
  `-race` 需要 cgo（本机无 C 编译器），竞态检测未在本机执行。
- **前端契约层修正**：`ContentSchema` 等此前用 `num` 解析时间字段，`Number(ISO)` 为 NaN 会静默变成 0。
  现改用 `dateNum`（ISO → Unix 秒，数字原样，空值 0），并补 `src/types/__tests__/schemas.test.ts`。
  该缺陷此前无可见症状（详情页与后台走的是未解析的原始字符串），属潜伏问题。

### 验收脚本覆盖面（第 13 轮扩展）

`scripts/acceptance.mjs` 由 44 项扩展到 62 项，新增覆盖此前从未被脚本跑过的路径：
静态文件服务（`/uploads` / `/thumbs`）、`/content/my`、游客快速上传与限频计数键、
举报处理闭环（提交 → 管理端可见 → 处理后从待办消失）、认领通过（作者转移）、管理端改作者、
未过审内容的可见性（匿名 404 / 管理员 200）、批量缩略图触发与状态机。实测 62/62 通过、清理零残留。

过程中的两个发现：
- **认领通过会改变内容归属**，因此后续「本人删除」会正确地返回 403——测试脚本自身的前提被破坏，
  不是服务端缺陷；脚本已改为在验证后把作者改回（顺带覆盖 `PUT /admin/content/:id/author`）。
- **批量缩略图进程被中断时状态会停留在 `running`**（与旧实现一致：状态写 Redis、锁 120 秒后自动释放）。
  管理端重新触发会覆盖状态，因此不做额外处理，仅记录。

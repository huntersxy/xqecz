# AGENTS.md — xqecz monorepo

## 项目概述

小泉动漫二创站（xqecz）：用户上传/浏览二次创作内容（统一为「文本 + 可选媒体」，不分类，贴吧/动态式），含评论、投票、认领、管理后台。

**Monorepo**：Go 单体后端 + Vue 3 前端；前端依赖由 pnpm workspace 管理，后端由 go module 管理，两端任务由 moon 编排。本文件是全仓唯一的 agent 约束入口，前后端约定都写在这里，前端不另设分册。

## 工作流程与完成标准

1. **先看证据** — 先 `git status` 与现有 diff，保留用户未提交改动；未经要求不暂存、提交、推送或发布。线上问题先核对实际页面版本、请求地址、状态码与响应头，区分地址错误、CORS、超时、缓存、后端不可用和浏览器降级，不凭截图猜根因。
2. **先定归属** — 共享规则放已有单一入口（见下表），不在组件里复制 URL 拼接、来源探测、布局算法或主题判断；改公共入口要搜索全部调用方并补回归。新文件必须能说明职责，不另起同名工具。
3. **守住业务契约** — 视觉与动效改动不能破坏上传校验、媒体来源优先级、评论、点赞回写、筛选与 KeepAlive 行为。配置、依赖或后端改动须与问题直接相关；调试只读需要的配置项，不打印整份 `.env`、会话或凭据。
4. **异步与资源必须有结束路径** — 请求、定时器、帧循环、Observer、事件监听、动画、GPU 资源都要说明谁创建、暂停、取消、释放；过期结果不得写回已卸载组件；装饰性失败不能阻断内容加载、交互或状态推进；错误在边界处理，不允许空 `catch` 吞掉主流程失败。
5. **注释写约束和原因** — 在公共函数、状态机边界、降级点与清理点写清输入归属、失败策略和不可破坏的原因，不逐行复述代码；测试固定可观察行为与失败条件，禁止只断言源码字符串或复刻算法。
6. **按影响验收并如实报告** — 修复要有能在修复前失败的回归用例；界面问题沿真实操作路径验证；明确区分「构建通过 / 本地预览 / 已发布线上」，模拟场景不等于真实 WebGL/CORS 验收。改完并本地验证后提交推送，等自动部署结束再访问 `https://xq.xiey.work` 走一遍；遇到 502、CORS、缓存或后端不可用必须单独说明，不能把部署完成当作功能通过。

### 常见改动的单一入口

| 要修改的规则 | 先读/先改的位置 |
|--------------|------------------|
| 启动链：主题、登录态刷新、整页过渡、路由预热 | `main.ts` 只做装配挂载；应用级状态归 `App.vue`；路由表 `router/routes.ts`、守卫 `router/index.ts`、意图预热 `router/preload.ts` |
| AI 工作室、角色设定库、创意广场的页面与控件外观 | `assets/creative-pages.css`；配色复用 `main.css` 的 `--creative-*`；弹层显式带 `creative-theme creative-dialog`。`views/AIStudioView.vue` 只保留舞台与详情，任务侧栏用 `<Teleport defer>` 注入 `PublicShell.vue` 的 `#public-shell-workspace`；手机抽屉关闭经 `utils/shellSidebar.ts`；多工作区、任务隔离与「喜欢」归 `composables/useAIStudio.ts` |
| 角色/模板/旧组合关系、图片引用编号与授权预设 | `utils/creativeResources.ts` / `utils/creativeMentions.ts` / `utils/creativeLicenses.ts`；资源展开后角色与参考图一起导入，重排图片要保持提示词引用目标；`StudioReferences.vue` 只负责拖曳、键盘排序与手势清理，实际顺序归 `useAIStudio.ts` |
| 首页灵感接力与创意广场的提示词摘要 | `utils/creativeIdeas.ts`；首页栏目宽度由 `HomeView.vue` 与其它首页区块统一约束，子组件不得扩成全屏 |
| 公开投票入口、参与与统计 | `PollPreview.vue` / `views/PollsView.vue`；列表归 `usePollList.ts`，投票与详情统计归 `usePollBallot.ts`；首页不嵌投票，发起/管理留在后台，票数与本人选择以后端回写为准 |
| 素材与 API 媒体地址、媒体目录归属、生产同路径回退 | `utils/index.ts`；设计说明见 `docs/frontend-media.md` |
| 详情大图来源优先级与探测缓存 | `utils/imageSource.ts`，调用方 `ContentMedia.vue` / `MediaImage.vue` |
| 星图全作品预加载、去重、显影进度与取消 | `utils/universeImages.ts`；不在场景里另写限量图片队列 |
| Canvas/WebGL 贴图加载与跨域缓存 | `utils/artworkPlanet.ts:planetImageSource/loadPlanetImage` |
| 星图贴图合批、帧调度、选中信息卡定位 | `universeAtlas.ts` / `universeFrame.ts` / `universeSelection.ts`；保持全部作品驻留与原有光影，不以丢图、降像素比或限帧换性能 |
| 星图/走马灯派生小图 | `server/internal/media/atlas.go`；策略按版本目录区分，经既有 `/thumbs` 托管，字段由后端下发，前端不猜文件名 |
| 星图随机坐标与可见候选 | `utils/artworkUniverse.ts`（纯算法）；渲染资源归 `universeScene.ts` |
| 星球/星图交互与动画阶段 | `ArtworkPlanet.vue` / `ArtworkUniverse.vue`；组件不重新定义共享媒体规则 |
| 公开导航、首页列表模式与推荐条 | `PublicShell.vue`（顶栏只切「工作台/作品广场」；侧栏「发现」只放创意广场/角色设定/投票广场，同一路由不在两处重复；`/studio` 保留真实侧栏）与 `HomeView.vue` + `useRecommendLoader.ts`；模式状态在 `stores/home.ts` |
| 前后端字段形状与请求/响应契约 | `api/index.ts` / `types/schemas.ts`，服务端对应模块 |

### 按改动选择验证

| 改动范围 | 必须完成的验证 |
|----------|----------------|
| 仅文档/注释 | 核对引用文件、函数和命令存在，`git diff --check`；不跑无关全量测试 |
| 前端行为或共享工具 | 相关回归测试 + `pnpm fe:typecheck`；共享工具/组件跑 `pnpm fe:test`；生产相关行为跑 `build-only` |
| 媒体路径、环境变量、Canvas/WebGL | 覆盖媒体基址为空与跨域非空两种配置；前端素材、三类媒体目录、外部/签名地址、失败回退与取消；按 `docs/frontend-media.md` 做生产构建和真实浏览器验证 |
| 动效、弹层、主题或 KeepAlive | 正常/减少动效、打开/关闭/中途取消、返回后可点击；主题跑日→夜→日，响应式跑桌面与手机；资源清理不能只靠静态截图 |
| 后端/接口/数据库 | 相关 Go 测试 + `pnpm server:vet`；响应形状跑契约冒烟；写链路按范围验收并清理测试数据 |

前端自动检查在 `.github/workflows/frontend-check.yml`：类型、单测、非空媒体基址的生产构建。它不发布站点，也不能替代浏览器交互/CORS 验收；发布方式见 `docs/deploy.md`。

## 架构

```
前端(Vue3) ──HTTP /api──→ Go 服务（单进程，packages/server）
                             ├─ gin 路由 + 中间件（Session / API 密钥 / 管理员 / CORS）
                             ├─ GORM ─────────→ MySQL
                             ├─ go-redis ─────→ Redis（session / 读穿缓存 / 浏览量 / 推荐 ZSet）
                             ├─ 进程内：缩略图（ffmpeg，缺失降级纯 Go）、推荐打分、视觉模型客户端
                             ├─ 后台任务：TinyPNG 定时压缩（每轮挑最大待压缩图，原图入 data/bin）
                             └─ 可选：媒体镜像（自建 OpenList 与 Cloudflare R2 并行推送）
静态资源：/uploads、/thumbs 由同一进程托管（/images 仅历史兼容），物理目录为项目根 data/
```

**推荐链路**：`internal/recommend` 读已通过内容 → 纯函数打分（时间衰减 + 浏览量 30 + 点赞 100）→ `cache.WriteRecommendList` 以「临时键 + RENAME」原子写入 ZSet `recommend:hot`（实际 key `xqecz:recommend:hot`）；读取优先 ZSet 并按序回表，无数据或异常降级 `view_count` 排序。刷新在启动时一次 + 每 10 分钟一次，多实例经 Redis 锁防抖。公开列表/推荐由 `decorateRows` 带出 `view_count`。

## 目录结构

```
packages/
├── server/                     # Go 后端（唯一后端进程）
│   ├── cmd/                    # server 入口；dbsync/dbinfo/dbsql/rediskeys 迁移排查；vision-import 离线参考图
│   ├── internal/
│   │   ├── app/ api/           # Deps 共享依赖；gin 路由装配
│   │   ├── config/ mysqldsn/   # .env 配置；MySQL/TiDB DSN 统一组装
│   │   ├── store/              # GORM 核心模型与连接（表模型散在 aigen/creative/vision）
│   │   ├── cache/ web/         # Redis 封装；响应包装/身份/CORS/静态目录/端口/优雅关停
│   │   ├── media/ compress/    # 缩略图与派生图；TinyPNG 后台压缩
│   │   ├── mirror/ r2/ openlist/ objstore/   # 镜像链、对象存储实现与公共面
│   │   ├── recommend/ avatar/ project/ logx/ # 推荐打分；头像地址；项目根定位；控制台日志
│   │   ├── aigen/ creative/ contenttags/ vision/  # AI 创作域、角色资源、内容标签、视觉模型
│   │   ├── modules/            # HTTP 装配：auth content comment poll admin apikey aigen creative
│   │   └── cli/                # 运维子命令（admin 初始化）
│   └── moon.yml
└── frontend/
    └── src/
        ├── main.ts App.vue     # 启动装配；应用级状态（主题/登录态/整页过渡/外壳）
        ├── router/             # routes.ts 路由表、index.ts 守卫、preload.ts 预热
        ├── api/ types/         # HTTP 层与 zod schema；契约冒烟在 api/__tests__/
        ├── components/         # 通用组件（WaterfallCard 用 ResizeObserver 驱动重排）
        ├── composables/        # 组合式函数（useWaterfallLayout 为纯算法）
        ├── stores/             # home / theme / user / admin
        ├── views/              # 路由页
        └── utils/ assets/      # 共享工具、全局样式与主题变量

scripts/                        # dev.mjs 开发编排；build-server.mjs 产物命名；start-backend.mjs 拉起；acceptance.mjs 全接口验收
.github/workflows/              # frontend-check（类型/单测/生产构建）等
.moon/workspace.yml             # moon 工作区（frontend / server）
```

## HTTP 接口

全部挂在 `/api` 下；统一响应 `{ code, message, data }`，`code === 200` 为成功。

| 模块 | 路由 | 实现 | 说明 |
|------|------|------|------|
| 健康 | `/health` | `internal/api/router.go` | 存活检查 |
| 认证 | `/auth/*` | `modules/auth` | 注册/登录/登出/改密/me/改邮箱，Cookie 会话 |
| 内容 | `/content/*` | `modules/content` | 列表/搜索/推荐/标签/我的/详情/上传/快速上传/编辑/删除/认领/点赞/收藏 |
| 评论 | `/comment/*` | `modules/comment` | 树形列表（一层回复）/计数/发表/删除/举报 |
| 投票 | `/poll/*` | `modules/poll` | 列表/详情/投票/创建/删除；游客按 visitor_id 去重 |
| 管理端 | `/admin/*` | `modules/admin` | 审核、待审、全量内容、用户、仪表盘、举报、认领、缩略图重建、推荐刷新 |
| API 密钥 | `/api-keys` | `modules/apikey` | 创建（完整密钥只返回一次）/列表/更新/删除 |
| AI 创作 | `/ai/*` `/ideas/*` `/creative/assets*` | `modules/aigen` + `internal/aigen` | 工作区会话、按 `session_id` 隔离的任务、生成、资产点赞、额度与创意赞助 |
| AI 管理 | `/admin/ai/*` | `modules/aigen/admin.go` | 供应商、任务核查、资产、额度发放 |
| 角色资源 | `/creative/resources*` `/creative/templates/*` `/admin/creative/*` | `modules/creative` | 角色与图片、提示词模板，兼容旧组合 |

静态挂载：`/uploads`→`data/uploads`、`/thumbs`→`data/thumbs`、`/images`→`data/images`（历史兼容）。由 `content.RegisterMedia` 提供而非 `r.Static`：带 `?download=1` 改为附件下载（文件名取内容标题），并拒绝 `.html`/`.svg` 等可执行或注入扩展名。

## 快速命令

> 不使用 Docker：后端是 Go 单二进制，前端是静态产物；开发期 MySQL/Redis 连云端实例。
> 前置：Node.js ≥ 20 + pnpm、Go 1.26+、FFmpeg（可选，缺失时图片缩略图降级纯 Go、视频缩略图不可用）。

```bash
pnpm install                    # 前端依赖（含 moon CLI）；项目根 .env 需存在（gitignore）
pnpm dev                        # 并发起 Go 后端(:3000) + Vite(:5173)，端口自适应，退出递归清理
pnpm dev:server                 # 仅后端（go run ./cmd/server）
pnpm dev:fe                     # 仅前端
pnpm build                      # 后端二进制 + 前端产物（Go 侧剥符号表）
pnpm start:backend              # 直接运行后端二进制
pnpm start:backend:node         # 经 scripts/start-backend.mjs 拉起

pnpm server:test                # go test ./...
pnpm server:vet                 # go vet ./...
pnpm server:fmt                 # gofmt（只对改动文件跑，见踩坑）

pnpm accept                     # 对运行中的后端跑全接口验收并清理数据（XQECZ_BASE，默认 127.0.0.1:3000）
pnpm fe:test                    # vitest 全量
pnpm fe:typecheck               # vue-tsc
XQECZ_LIVE_API=http://localhost:5173/api pnpm --filter ./packages/frontend exec vitest run src/api/__tests__/contract.live.test.ts

pnpm exec moon run server:build                  # 单目标任务
pnpm exec moon run server:test frontend:test     # 多目标
pnpm exec moon query projects                    # 工程图

cd packages/server && go run ./cmd/dbinfo        # 账号权限与表规模
cd packages/server && go run ./cmd/dbsql "SELECT 1"
./packages/server/xqecz-server admin -username admin [-email a@b.com] [-reset]   # 部署机无 Go 工具链也可用
```

## 核心约束

- **启动链职责** — `index.html` 只预置 `html.dark` 防首帧闪色；`main.ts` 只创建 app、装 pinia/router、挂载、初始化 Web Vitals；主题初始化、登录态刷新、整页过渡、Toaster/ConfirmDialog 归 `App.vue`。**主题必须在任何路由组件挂载前落到 `body[arco-theme]`**：详情页不经过 `PublicShell`，只靠 `index.html` 的 `html.dark` 会让 Arco 弹层与详情调色板仍是浅色。路由拆成 `routes.ts`（表）、`index.ts`（守卫）、`preload.ts`（意图与空闲预热），预热只拉代码、不挂载页面、不请求私有数据。

- **前端是契约** — `api/index.ts` 与 `types/schemas.ts` 定义接口形状；后端响应必须能被 zod 直接解析（改接口后跑契约冒烟）。

- **统一响应** — `{ code, message, data }`；错误文案放 `message` 供前端直接展示。校验类失败用 HTTP 200 + 业务码（如 `{code:400,message:"描述正文与媒体文件至少填一项"}`），鉴权类失败才改 HTTP 状态码。

- **用户头像** — 内容 `avatar_url` 与评论 `user.avatar_url` 都由 `internal/avatar.URL` 生成，公开响应不返回作者邮箱；前端统一用 `UserAvatar.vue`（QQ 邮箱走 QQ，其余 Gravatar），空地址/加载中/远端失败显示打包的本地默认头像，切换作者或地址时重置失败状态。首页卡片头像尺寸由容器查询控制，不影响瀑布流测量。

- **时间字段** — 一律 `web.Time` 序列化为 UTC 毫秒（`2026-09-13T11:28:04.865Z`），与前端 `Date.toJSON()` 逐字节一致。

- **CORS 白名单** — `internal/web/middleware.go` 按 `CORS_ORIGINS`（逗号分隔）放行，命中才回 `ACAO` + `Allow-Credentials`，未命中不回任何 CORS 头；`OPTIONS` 一律 204。条目分精确 origin（整串比对）与子域通配（`*.edgeone.cool`，按主机名后缀、要求点边界、忽略端口与大小写、不覆盖裸域）；生产来源写精确条目，通配只用于平台动态分配的调试域名。

- **认证双通道** — `X-API-Key`（sha256 比对 `api_keys.key_hash`，权限数组随身份注入）优先，其次 Session Cookie；`web.RequireAPIKeyPermission` 只约束密钥调用，Session 用户不受限。新增受保护接口按此挂中间件。

- **上传约定** — multipart 字段名 `file`；流式解析边收边写盘（`Request.MultipartReader`），文件名沿用前端 `<md5>.<ext>`，不合规随机兜底；单文件 ≤ 20MB，仅 `image/*` 与 `video/*`；**上传不转码**（保留源格式与后缀），压缩交给后台 TinyPNG 就地替换。

- **媒体下载** — `/uploads`、`/thumbs`、`/images` 由 `content.RegisterMedia` 挂载；`?download=1` 以附件返回，文件名取内容标题（ASCII 回退 + RFC 5987）；`.html`/`.svg`/`.js` 等拒绝下载（403），避免上传目录变成分发通道。

- **前端素材与媒体地址归属** — Vite import 的 `/assets/*`、开发 `/src/*` 与 `public` 素材由前端托管，禁止补 `VITE_MEDIA_BASE_URL` 或回退媒体源站；只有 `/thumbs/`、`/uploads/`、`/images/` 经共享 `isContentMediaPath/getImageUrl` 解析。绝对地址、签名参数、blob/data 原样保留，第三方地址不参与本站兜底。线上前端域名不是媒体域名；生产兜底常量只在共享工具定义。新增媒体目录须同步服务端路由、共享判定、开发代理与契约测试。

- **作品星球与星图的加载边界** — 普通图片能显示不代表允许读入 Canvas/WebGL；星球、展开阵列、星图共用 `loadPlanetImage`，不得绕开 CORS 加载与取消机制。开发代理只允许固定媒体源站的公开目录，不传 Cookie/Authorization，不扩成任意 URL 代理。体验优先：展开期间全量预加载并常驻，不设 64 张上限，不逐帧串行；共享 URL 只请求一次。默认近景漫游、全景另行触发；隐藏/停用时暂停渲染，卸载取消请求并释放 GPU 与监听。详见 `docs/frontend-media.md`。

- **星图/走马灯派生图** — 普通缩略图成功后派生 `atlas_thumb`（最长边 256、WebP 55）与 `montage_thumb`（640、WebP 78），目录 `thumbs/atlas-v1/`、`thumbs/montage-v1/`；不改原图与普通缩略图、不镜像、不新增 DB 列。上传、手工重建、启动扫描与缺图请求共用幂等入口，最多两路编码、同文件去重、临时文件原子提交；失败保留旧产物并让前端回退普通缩略图；策略或尺寸变化须升目录版本避免 CDN 旧缓存；删普通缩略图前确认无引用，一并把派生图移入垃圾桶。

- **共享上传目录** — 物理目录为项目根 `data/`（`UPLOAD_DIR`/`THUMB_DIR`/`IMAGES_DIR`/`BIN_DIR` 由 .env 覆盖），媒体处理与静态托管在同一进程。

- **Redis 缓存** — 公开读路径读穿缓存：`content:{id}`、`content_list:{sha1(规范化参数)}`、`tags`、`comments:{cid}:threads-v3:{page}:{size}`、`comment_count:{cid}`、`admin:dashboard`；TTL 仅兜底，**所有写路径必须显式失效**（`ClearContentCache` / `ClearContentListCache` / `ClearCommentCache` / `ClearAllContentCaches`）。

- **评论回复链** — DB `parent_id` 保留直接回复对象，列表按顶层评论分页；必须批量查询本页楼内全部后代，`threads.go:buildThreads` 把多级回复铺在顶层 `replies` 一层并附直接 `parent`，否则 A←B←C 的 C 会消失。缓存版本仍放 `comments:{cid}:*` 以便统一失效；删父评论时事务内把直接回复 `parent_id` 置空；`CommentSections` 监听内容 id 自动加载并丢弃过期请求，提交回复后刷新当前页而非跳回第一页。

- **物理删除** — 删除即 `Delete()` 真删（无 `deleted_at`，模型不含 `gorm.DeletedAt`）。删内容走 `purgeContent`（同事务清评论及举报、点赞、收藏；回复 `parent_id` 置空），再经 `removeOrphanMedia` 把**无其它引用**的媒体移入 `data/bin`（同一文件可能被多条内容共用，先查引用计数）。

- **TinyPNG 后台压缩** — 每 `COMPRESS_INTERVAL_SECONDS`（默认 60s）挑一张最大的待压缩图就地替换（`.tinify-tmp` → 原图入 bin → 改名，任一步失败保留原图）；跳过 GIF 与 `< COMPRESS_MIN_KB`（默认 400KB）；`compressed_at` 非空即已处理（压缩未变小也标记，避免反复消耗配额）；未配 `TINIFY_API_KEY` 时静默休眠。

- **媒体镜像链** — 公共面 `internal/objstore`（`Store` 只有 `Put`/`Head`）。实现为 `internal/r2`（S3 子集，手写 SigV4）与 `internal/openlist`；`Target.Driver` 分流（空值按 `r2`）。**两个目标都推，装配顺序不代表优先级**，取用优先级只在前端 `bestSource()`。本地落盘后 `PushAsync`，TinyPNG 压缩成功后再推压缩图，缩略图纯本地；对象 append-only（删内容只进本地垃圾桶）。幂等 `sameObject()`：优先 md5，OpenList 拿不到远端哈希时退 `ContentLen`——**本地 md5 必须实读内容，不能取文件名中的 md5**（压缩原地改写会让文件名永远描述压缩前）。回填每目标一任务一锁（`lock:mirror:r2` / `lock:mirror:openlist`）；凭据不全整体停用。**SigV4 规范化 URI 必须用已解码的 `URL.Path`**，用 `EscapedPath()` 会把中文/空格对象名二次编码导致拒签。

- **详情页大图源选择** — 后端在 `img`/`video` 外下发绝对地址 `mirror_img`/`mirror_video`（R2）与 `mirror2_img`/`mirror2_video`（OpenList），两级各自独立。前端 `utils/imageSource.ts` 按 **OpenList → R2 → 源站** 串行短路探测，只探镜像侧（源站本来就是回退目标）；OpenList 在前是因为自有广州机直出，R2 走 Cloudflare 国内常被拖慢。字段名 `mirror*` 是历史命名，顺序只看 `bestSource()`。结论存 `localStorage` 键 `xqecz:image-source-v3`，按网络标识（公网 IP 为主 + `navigator.connection` 指纹为辅）复用，标识未变不重探，未探级别留空而非记 false；缓存命中的级别若当前无地址即视为失效。**来源未定不渲染媒体**（`ContentMedia` 的 `resolved` 门），避免「先源站再换源」的重复请求。渲染兜底走链式回退，调用方用 `markUnreachable(tier)` 只标死那一级；渲染失败只记内存负标记（60s），不写 localStorage；探测单级试满 3 次才判不可达，负标记过期自动恢复、换网清空。

- **推荐算法单一入口** — 只改 `internal/recommend/recommend.go:ScoreItem()`；刷新节奏与落库在 `Refresher.Refresh()`。

- **降级优先** — ffmpeg 缺失时图片缩略图降级纯 Go 解码缩放，视频缩略图失败仅告警不影响上传；Redis 不可用时读路径直查 MySQL。

- **公开界面配色与弹层作用域** — 首页、详情、快速上传、登录注册及导航/版权区共用 `main.css` 的 `--creative-*`（含暗色）；Arco 主色只由 `.creative-theme` 局部覆盖；工作台与管理端用 `.admin-theme.creative-theme` 复用语义色并保留状态色。认领、举报、确认及工作台 Modal 用 `creative-dialog` 外壳，**Teleport 弹层不继承祖先主题类，必须在弹层根显式带 `creative-theme`**；Arco Drawer 外壳用组件专属前缀限定（`:global(.qu-sheet .arco-drawer...)`），勿改全局 `.arco-drawer`。**Vue scoped 的 `:global()` 必须包住完整选择器**，否则编译后后半段丢失（夜间遮罩的 `pointer-events:none` 与定位落到 body，全页无法点击）。

- **前端动效与背景锁优先用 CSS 原生** — `prefers-reduced-motion` 统一在 `main.css` 末尾压掉过渡/动画时长，**不要写全局 `transform: none`**（Arco 居中偏移与走马灯位移靠 transform 承担定位）；加载指示器例外，冻结等于无反馈。JS 显式 `behavior:'smooth'` 的滚动优先级高于 CSS，必须经 `utils.scrollBehavior()` 判定。详情覆盖层锁滚用 `html:has(.cd-root){overflow:hidden}` + `html{scrollbar-gutter:stable}`，不再在 `onMounted` 改 body 内联样式（漏还原会全站锁死）。

- **View Transition 只有一个入口** — 一律经 `utils/viewTransition.ts:withRevealTransition(update, origin?)`，当前唯一使用方是 `PublicShell.vue` 的日/夜切换（圆形揭示从按钮位置铺开）。① 减少动效只能在发起前判（UA 默认淡出挂在快照伪元素上，`main.css` 的媒体查询压不到）；② 回调里改响应式状态必须 `await nextTick()` 再返回，否则快照拍在 DOM 更新前；③ `main.css` 末尾已把根快照默认淡出全局关掉，后续 VT 要淡出得自己排动画。**没有给首页↔详情加路由过渡**：详情媒体受 `resolved` 门控，快照时刻往往只有骨架屏，不如组件自带入场。

- **迁移期工具** — `cmd/` 下 dbsync/dbinfo/dbsql/rediskeys 用于排查与对拍，改数据库相关行为先核实再断言。

- **数据库连接单一来源** — DSN 一律经 `internal/mysqldsn.Format()` 组装（服务端与四个 cmd 工具共用）；加密由 `MYSQL_TLS` 控制（留空=不加密），**TiDB Cloud Serverless 强制加密**，明文报 `1105 insecure transport`；逐字节搬运时须关 `ParseTime`，否则驱动时区换算会改写时间值。

- **TiDB 兼容性** — ① `TEXT`/`BLOB`/`JSON` 不允许 DEFAULT（1101）；② 不支持 `CHECKSUM TABLE`，一致性校验用「按主键有序导出后比对 md5」；③ 系统库对业务账号只读，必须建独立库；④ 自增游标别查 `information_schema.tables.auto_increment`（CLUSTERED 主键恒 0），用 `SHOW CREATE TABLE` 或 `SHOW TABLE <t> NEXT_ROW_ID`；⑤ 同一条 `ALTER TABLE` 不能给刚新增的列建索引（1072），拆成两条。

- **AI 管理与资源** — 管理员站点生成/赞助不扣额度，权限判定归后端，普通用户仍要预留与结算。任务列表作者由后端批量返回 id/username，不能靠额度选择器的截断用户列表推断；创意广场预览以提示词为主、参考图为辅。额度发放按用户名或全体用户；批量余额/账本同事务提交，全体用户幂等重试复用首次快照。角色和图片统一在「角色资源」维护，旧 `preset` 保留兼容；公开资源关联图须有授权并允许生成与二创。工作台复用真实侧栏：任务侧栏经 `#public-shell-workspace` 注入，槽位固定在「发现 / 我的库」之后，公共导航在作品广场与工作台之间不移动；默认进入「全部结果」瀑布流任务画廊，可切「单个任务」看舞台；画廊与任务列表共用筛选，失败任务以状态卡穿插；结果操作条只留继续编辑、复用提示词、喜欢、下载、发布与移除（同一动作只有一个入口），保存草稿、投稿创意、一键清空常驻输入区，不重复大图弹窗；窄屏用同一个侧栏抽屉、隐藏详情，输入区吸底并留出操作间距。任务列表按 `session_id` 隔离，显式新建工作区才带 `new:true`；`ai_assets.liked` 是作者私有收藏，公开列表与创意广场响应必须清零。参考图上传的取消与会话归属由 `useAIStudio.ts` 管理，拖入/粘贴不能绕过待确认草稿的锁定；`useStudioDock.ts` 测量输入区占位并跟随可视视口避让软键盘，观察器、视口监听和帧调度随卸载清理；输入内容独立滚动，提交栏保持可达。详见 `docs/ai-creation.md`。

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 + gin + GORM v2（MySQL）+ go-redis v9 + log/slog |
| 媒体 | ffmpeg（抽帧/缩放）+ `deepteams/webp`（纯 Go WebP，零 CGO）+ `x/image/draw`；压缩用外部 TinyPNG API |
| 前端 | Vue 3.5 + TypeScript + Vite + Tailwind CSS + Arco Design Vue + zod |
| 数据 | MySQL（22 张表）+ Redis（会话/缓存/浏览量/推荐 ZSet） |
| 编排 | moon（任务图与缓存）+ pnpm workspace + go module |
| 运行 | 本地直启（`scripts/dev.mjs`）；生产前后端分开发布：后端 Go 二进制经 CI SSH 直传、OpenRC 接管；前端由 EdgeOne Makers 构建（`VITE_API_BASE_URL`/`VITE_MEDIA_BASE_URL` 只在控制台配），见 `docs/deploy.md` |

## 修改指南

1. **前端改动** — 接口形状以 `types/schemas.ts` 与 `api/index.ts` 为准，改接口后跑契约冒烟；前端约定写在本文「核心约束」与本指南第 7 条。
2. **后端模块** — 在 `internal/modules/<模块>/` 改；Handler 依赖 `app.Deps`（Cfg/DB/Redis），路由在各自 `Register()` 挂载，再由 `internal/api/router.go` 装配（web 包不反向依赖业务模块）。
3. **新增接口** — 响应走 `web.OK`/`web.Fail`（校验类用 `web.SoftFail`）；身份从 `web.MustIdentity(c)` 取；列表同时要 Count 与 Find 时必须共用同一 `db.Session(&gorm.Session{})`，否则总数与列表错位。
4. **数据库变更** — 改 `internal/store/models.go`（显式 `column` 标签 + `TableName()`）；生产用 `scripts/migrations/` 的正式 migration，不开 `AutoMigrate`；热点查询索引见 `2026-09-13-add-hot-path-indexes.sql`。
5. **媒体管线** — 缩略图优先 ffmpeg、失败降级纯 Go（WebP 85）；镜像链见 `internal/mirror`，`Push` 与回填共用判定，**改推送范围只动 `mirrorable()`**，新增目标只需加一个 `TargetFrom*`；`MoveToBin` 负责搬垃圾桶。上传已不做 WebP 无损转换，压缩归 `compress/worker.go` 的候选筛选与 `shrinkInPlace`。
6. **推荐算法** — 见「核心约束」单一入口条目。
7. **瀑布流（首页）** — 纯算法在 `composables/useWaterfallLayout.ts:computeLayout()`（与 DOM 解耦，输出 `Map<id, Position>`），改逻辑先改纯函数并补 `__tests__/useWaterfallLayout.test.ts`。约定：**稳定列**（落列后不换列，仅列数/列宽变化全量重排）、**full 全量重排**（数据集合变化强制平衡列底）、**列底失衡收敛**（max-min 超过阈值补一次带锚定重排）、**单一调度**（图片加载/尺寸/宽度合并到一帧 rAF）、**滚动锚定**（重算前固定视口顶部卡片），高度由 `[data-wf-id]` 批量量取，`restore`/`reset` 管 keep-alive 缓存。**卡片排版用容器查询而非视口媒体查询**：`.wf-card` 声明 `container: wfc / inline-size`，内部用 `@container wfc (...)`；只能用 `inline-size`，写成 `size` 会让固有高度归零、整页塌掉。**加载**：首页进入即连续拉全部页（`loadAllPages`，每页 100 条），图片懒加载；keep-alive 往返用 `syncLatestOnActivated` 增量同步；`listCache` 在 `onBeforeRouteLeave` 写一次；筛选/搜索用自增 `loadSeq` 丢弃过期响应。**卡片写操作必须回传父级**（`emit('liked')` 由首页更新 `allContents`，否则 keep-alive 往返被旧值复原）。**卡片媒体区要隐藏 Arco `.arco-image-footer`**（`:deep()` 限定在 `.wf-card-media`）；**事件回调里的异步失败必须就地 catch**，未捕获的拒绝会冒泡到 `ErrorBoundary` 把整页换成错误页。
8. **踩坑记忆** — `MYSQL_POOL_SIZE` 过小是并发瓶颈（20 并发×3 查询：池 5 → p50 616ms，池 30 → 304ms）；`silent=1` 的详情请求按旧语义跳过缓存读（只写不读），时延要分开评估；`contents.file_path` 无媒体既有 NULL 也有空串，过滤须同时判；moon 常驻任务用 `preset: 'server'`（`local: true` 已移除）；GORM Count/Find 不共用条件会话会总数与列表错位；bigint 主键在 JSON 与 ZSet 之间显式转字符串；pnpm 11 起不读 `package.json` 的 `pnpm` 字段，`overrides` 写 `pnpm-workspace.yaml`；Windows git-bash 把命令行中文按 GBK 传给 curl（`Incorrect string value`），用 `-F "field=<文件"` 或 `--data-binary @文件`；控制台 `taskkill /PID` 可能静默失败，清端口用 `Stop-Process` 并复核 `netstat`，**不要用 `taskkill /T`**（会杀自身会话）；**SigV4 对时钟敏感**，R2 全 403 先看 `chronyc tracking` 的 offset，用 `chronyc makestep` 校正并在 `chrony.conf` 补 `makestep 1.0 3`；工作区文件是 CRLF，**不要对目录跑 `gofmt -w`**（会把无关文件改成 LF），只格式化本次改动文件；部署 shell 必须 LF（`.gitattributes` 已钉死）；Windows 上 Vite dev 会被编辑器原子保存的临时文件打死（`EBUSY`，前后端一起退出），看到 fe exit 1 先翻 stderr，重启即可；**OpenList 三处反直觉**：鉴权头是 `Authorization: <token>` 不带 `Bearer`、推送 `PUT /api/fs/put` 的路径放 `File-Path` 头且服务端会 URL 解码（送 `EscapedPath()`）、`/api/fs/get` 的 body 不解码且对象不存在返回 `code:500 object not found`（免签 `/d/*` 同样 500）；另需 `sign_all=false` 才能裸链直出，`OPENLIST_PUBLIC_BASE` 必须带直链前缀 `/d`（裸域是它自己的 SPA，否则图片变成 HTML）；**Vditor 自托管资源要跟包版本同步**，升级后复制 `node_modules/vditor/dist/js/lute/lute.min.js`，否则启动报 `lute.SetCallout is not a function`。

## 文档迭代规范

本文件是全仓唯一约束入口；**代码改动 + 对应文档迭代应同一次提交**，git 历史即版本记录，不维护独立版本号或 CHANGELOG。

**触发条件**（满足任一即改对应章节）：新增/删除后端模块或前端页面、组件、composable；新增/修改 HTTP 接口；表结构或列语义变化；密钥权限、缓存键、删除、降级等约束变化；依赖或技术栈变化；根脚本或 moon 任务变化；出现新的踩坑（写进修改指南 8）。

**沉淀原则**：只写「再次需要时无法从代码/现有文档推出」的信息——文件级定位、模块依赖、隐含约定、踩坑、跨端契约；能从代码直接看出的签名细节、常量值不写；沉淀约束与推理链，不写实现快照。一次改动通常只新增/修订一两条。

**流程**：`git status` + `git diff --stat` 定位真实变更 → 对照触发条件找章节 → 最小修订（不重排无关内容）→ 自校验（目录树与真实目录一致、路径/函数可搜索到、命令可执行、无自相矛盾）→ 提交前 `git diff AGENTS.md` 复核。保留 2 空格缩进、中文说明、表格与代码块风格。

## 已归档

- 旧后端 `xqecz-golang/`、`xqecz-nodejs/` 与独立前端仓库 `xqecz_frontend` 已并入本 monorepo，原仓库移至 `D:/xqecz/archive/` 保留历史（tag `archive/monorepo-migration-2026-07-22`）。
- NestJS 后端（`packages/api`）、Go Worker（`packages/worker`）与 gRPC 定义（`proto/`）已被 Go 单体取代，见 `docs/go-backend-migration.md`。仓库历史于 2026-09-14 清洗重建。

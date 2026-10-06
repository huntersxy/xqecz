# AGENTS.md — xqecz monorepo

## 项目概述

小泉动漫二创站（xqecz）— 用户上传/浏览二次创作内容（内容统一为"文本 + 可选媒体"，不分类，贴吧/动态式），含评论、投票、认领、管理后台。

**Monorepo 架构**：Go 单体后端 + Vue 3 前端。前端依赖由 pnpm workspace 管理，后端由 go module 管理，两端任务由 moon 统一编排。

## 架构

```
前端(Vue3) ──HTTP /api──→ Go 服务（单进程，packages/server）
                             ├─ gin 路由 + 中间件（Session / API 密钥 / 管理员 / CORS）
                             ├─ GORM ──────────→ MySQL
                             ├─ go-redis ──────→ Redis（session / 读穿缓存 / 浏览量 / 推荐 ZSet）
                             ├─ 进程内计算：缩略图（ffmpeg，缺失降级纯 Go 缩放）、推荐打分（纯函数）
                             ├─ 后台任务：TinyPNG 定时压缩（每轮挑最大的待压缩图，原图入 data/bin）
                             └─ 可选：媒体镜像到对象存储（自建 OpenList 与 Cloudflare R2 并行推送）
静态资源：/uploads、/thumbs 由同一进程托管（/images 仅历史兼容），物理目录为项目根 data/
```

**推荐链路**：`internal/recommend` 读取已通过内容 → 纯函数打分（时间衰减 + 浏览量 30 + 点赞 100）→
`cache.WriteRecommendList` 用「写临时键 + RENAME」原子写入 ZSet `recommend:hot`（实际 key `xqecz:recommend:hot`）；
读取时优先 ZSet 并按 ZSet 顺序回表，无数据或异常时降级为 `view_count` 排序。刷新由启动时一次 + 每 10 分钟周期触发，多实例经 Redis 锁防抖。

## 目录结构

```
packages/
├── server/                     # Go 后端（唯一后端进程）
│   ├── cmd/server/             # 入口：读 .env → 连 MySQL/Redis → 启动 HTTP + 推荐刷新
│   ├── cmd/dbsync/             # 迁移期工具：把源库结构（可选数据）复制到目标库
│   ├── cmd/dbinfo/             # 排查工具：账号权限与库表规模
│   ├── cmd/dbsql/              # 排查工具：执行单条 SQL（查/清理测试数据）
│   ├── cmd/rediskeys/          # 排查工具：按模式列出业务 Redis 键
│   ├── internal/app/           # Deps：各层共享依赖集（避免 web ↔ modules 循环依赖）
│   ├── internal/api/           # 路由装配：创建 gin 引擎并挂载全部模块
│   ├── internal/config/        # env 配置（.env + .env.local 覆盖）
│   ├── internal/mysqldsn/      # MySQL/TiDB 连接串统一组装（TLS 参数收敛到一处）
│   ├── internal/store/         # GORM 模型（10 张表）与连接
│   ├── internal/cache/         # Redis 封装：前缀、会话、读穿缓存、ZSet、失效、限频、锁
│   ├── internal/web/           # 响应包装、身份中间件、CORS、静态目录、端口自适应与优雅关停
│   ├── internal/media/         # 缩略图生成、垃圾桶目录搬运
│   ├── internal/objstore/      # 对象存储的最小公共面：Store 接口 + ObjectMeta/ErrCorrupted
│   ├── internal/r2/            # S3 子集客户端（手写 SigV4，不引 AWS SDK）
│   ├── internal/openlist/      # 自建 OpenList 客户端（/api/fs/put 推送、/api/fs/get 探存在）
│   ├── internal/mirror/        # 媒体镜像链：目标抽象、多目标编排、推送判定与存量回填
│   ├── internal/compress/      # TinyPNG 后台压缩（客户端 + 单张调度）
│   ├── internal/recommend/     # 推荐打分与刷新任务
│   ├── internal/cli/           # 运维子命令（admin：管理员初始化）
│   └── internal/modules/       # 业务模块：auth / content / comment / poll / admin / apikey
│
└── frontend/                   # Vue 3 前端
    └── src/
        ├── api/                # HTTP 层；__tests__/contract.live.test.ts 为契约冒烟
        ├── components/         # 通用组件（WaterfallCard 用 ResizeObserver 驱动瀑布流重排）
        ├── composables/        # 组合式函数；useWaterfallLayout.ts 瀑布流纯算法
        ├── stores/home.ts      # 首页筛选/分页/滚动位置与瀑布流布局缓存
        └── views/              # 路由页：HomeView 瀑布流首页；ContentDetailView 全屏覆盖式详情页

scripts/
├── dev.mjs                     # 开发编排器（Go 后端 + 前端，端口自适应 + 进程树自愈）
├── build-server.mjs            # 后端构建包装：按平台选产物名（Windows 必须 .exe）并剥离符号表
├── start-backend.mjs           # 部署启动器（宝塔 Node 项目用，把二进制作为子进程拉起）
└── migrations/                 # 一次性 SQL 迁移（历史归档，如内容统一模型清理脚本）

.moon/workspace.yml             # moon 工作区定义（工程映射：frontend / server）
packages/server/moon.yml        # 后端任务（build / run / test / lint / fmt）
packages/frontend/moon.yml      # 前端任务（dev / build / test / type-check / lint）
pnpm-workspace.yaml             # 前端依赖与传递依赖安全覆盖
```

## HTTP 接口

全部挂在 `/api` 前缀下，共 52 条；统一响应 `{ code, message, data }`，`code === 200` 为成功。

| 模块 | 路由 | 实现文件 | 说明 |
|------|------|----------|------|
| 健康 | `/health` | `internal/api/router.go` | 存活检查 |
| 认证 | `/auth/*`（6） | `modules/auth` | 注册/登录/登出/改密/me/改邮箱，Cookie 会话 |
| 内容 | `/content/*`（14） | `modules/content` | 列表/搜索/推荐/标签/我的/详情/上传/快速上传/编辑/删除/认领/点赞/收藏 |
| 评论 | `/comment/*`（5） | `modules/comment` | 树形列表（一层回复）/计数/发表/删除/举报 |
| 投票 | `/poll/*`（5） | `modules/poll` | 列表/详情/投票/创建/删除；游客按 visitor_id 去重 |
| 管理端 | `/admin/*`（18） | `modules/admin` | 审核、待审、全量内容、用户管理、仪表盘、举报、认领、缩略图重建、推荐刷新 |
| API 密钥 | `/api-keys`（4） | `modules/apikey` | 创建（完整密钥仅返回一次）/列表/更新/删除 |

静态挂载点：`/uploads` → `data/uploads`、`/thumbs` → `data/thumbs`、`/images` → `data/images`（历史兼容，已无新写入）。挂载由 `content.RegisterMedia` 提供而非 gin `r.Static`：带 `?download=1` 时改为附件下载（文件名取内容标题），并屏蔽 `.html`/`.svg` 等可直接执行或注入的扩展名。

## 快速命令

> 项目**不使用 Docker**：后端是 Go 单二进制，前端是静态产物；开发期 MySQL/Redis 连云端实例，本机无需安装。
> 前置要求：Node.js ≥ 20 + pnpm、Go 1.26+、FFmpeg（可选，缺失时图片缩略图降级为纯 Go，视频缩略图不可用）。

```bash
# ── 首次准备 ──
pnpm install                                   # 前端依赖（含 moon CLI）
# 确认项目根 .env 存在（gitignore，含云端 MySQL/Redis 凭据与 UPLOAD_DIR 等）

# ── 一键开发（推荐）──
pnpm dev                                       # scripts/dev.mjs：并发起 Go 后端(:3000) + 前端(:5173)，
                                               #   端口被占自适应顺延，退出时递归清理子进程树
pnpm dev:server                                # 仅 Go 后端（go run ./cmd/server）
pnpm dev:fe                                    # 仅前端 Vite
pnpm build                                     # 后端二进制 + 前端产物（Go 侧剥离符号表）
pnpm start                                     # 构建后 concurrently 起后端 + 前端预览
pnpm start:backend                             # 部署用：直接运行后端二进制
pnpm start:backend:node                        # 部署用：经 scripts/start-backend.mjs 拉起（面板保持 Node 项目形态）

# ── 后端 ──
pnpm server:test                               # go test ./...
pnpm server:vet                                # go vet ./...
pnpm server:fmt                                # gofmt
cd packages/server && go run ./cmd/dbinfo      # 查看账号权限与表规模
cd packages/server && go run ./cmd/dbsql "SELECT 1"
./packages/server/xqecz-server admin -username admin [-email a@b.com] [-reset]
                                               # 运维子命令：创建管理员 / 重置密码（部署机无 Go 工具链也可用）

# ── 端到端验收 ──
pnpm accept                                    # 对运行中的后端跑全接口验收（公开/登录/管理员/密钥链路，
                                               #   含时延采样，结束后清理测试数据）；需先起后端
                                               #   目标可用 XQECZ_BASE 指定，默认 http://127.0.0.1:3000

# ── 前端 ──
pnpm fe:test                                   # vitest 全量
pnpm fe:typecheck                              # vue-tsc 类型检查
XQECZ_LIVE_API=http://localhost:5173/api pnpm --filter ./packages/frontend exec vitest run src/api/__tests__/contract.live.test.ts
                                               # 契约冒烟：用前端 zod schema 校验运行中的后端响应

# ── 任务编排（moon）──
pnpm exec moon run server:build                # 单目标任务（带增量缓存与输入指纹）
pnpm exec moon run server:test frontend:test   # 多目标任务：两端测试
pnpm exec moon query projects                  # 查看工程图（当前为 frontend / server 两个工程）
```

## 核心约束

- **前端是契约** — `packages/frontend/src/api/index.ts` 与 `src/types/schemas.ts` 定义接口形状；后端响应必须能被前端 zod schema 直接解析（改接口后跑契约冒烟）
- **统一响应** — `{ code, message, data }`；错误文案放 `message`（前端直接展示）。**校验类失败沿用 HTTP 200 + 业务码**（如 `{code:400,message:"描述正文与媒体文件至少填一项"}`），鉴权类失败才改 HTTP 状态码
- **时间字段** — 一律用 `web.Time` 序列化为 UTC 毫秒（`2026-09-13T11:28:04.865Z`），与前端 `Date.toJSON()` 逐字节一致
- **CORS 白名单** — `internal/web/middleware.go` 按 `CORS_ORIGINS`（逗号分隔，由 `config.Load` 拆分）放行，**命中才回 `ACAO` + `Allow-Credentials`，未命中不回任何 CORS 头**；`OPTIONS` 一律 204。两种条目：精确 origin（`https://xq.xiey.work`，整串比对，端口也算在内）与子域通配（`*.edgeone.cool` 或带协议 `https://*.edgeone.cool`）——通配按主机名后缀匹配，**不覆盖裸域、要求点边界**（防 `notedgeone.cool` 借尾巴），忽略端口、主机名大小写不敏感。通配只用于平台按项目动态分配的调试域名，生产来源应写精确条目
- **认证双通道** — 请求头 `X-API-Key`（sha256 比对 `api_keys.key_hash`，权限数组随身份注入）优先，其次 Session Cookie（`session_id`）；`web.RequireAPIKeyPermission` 仅约束密钥调用，Session 用户不受限。新增受保护接口按此挂中间件
- **上传约定** — multipart 字段名 `file`；**流式解析边收边写盘**（`Request.MultipartReader`），文件名沿用前端的 `<md5>.<ext>`、不合规则随机兜底；单文件 ≤ 20MB、仅 `image/*` 与 `video/*`；**上传不做转码**（保留源格式与后缀），压缩统一由后台 TinyPNG 任务就地替换
- **媒体下载** — `/uploads`、`/thumbs`、`/images` 由 `content.RegisterMedia` 挂载（非 gin `r.Static`）：带 `?download=1` 时以附件返回，文件名取内容标题（ASCII 回退名 + RFC 5987 UTF-8 名）；`.html`/`.svg`/`.js` 等可直接执行或注入的扩展名拒绝下载（403），避免上传目录变分发通道
- **共享上传目录** — 物理目录为项目根 `data/`（`UPLOAD_DIR`/`THUMB_DIR`/`IMAGES_DIR`/`BIN_DIR` 由 .env 覆盖），媒体处理与静态托管在同一进程内，无需跨进程路径约定
- **Redis 缓存** — 公开读路径走读穿缓存：`content:{id}`、`content_list:{sha1(规范化参数)}`、`tags`、`comments:{cid}:{page}:{size}`、`comment_count:{cid}`、`admin:dashboard`；TTL 仅作兜底，**所有写路径必须显式失效**（`ClearContentCache` / `ClearContentListCache` / `ClearCommentCache` / `ClearAllContentCaches`）
- **物理删除** — 删除一律是 `Delete()` 真删（无 `deleted_at` 软删除列，模型亦不含 `gorm.DeletedAt`）；历史软删行已由 `scripts/migrations/2026-09-14-drop-soft-delete-columns.sql` 物理清理并删列。删内容走 `purgeContent`（同一事务清评论及其举报、点赞、收藏；回复的 `parent_id` 置空而非连带删除），再经 `removeOrphanMedia` 把**无其它引用的**媒体文件移入 `data/bin`（同一文件可能被多条内容共用，必须先查引用计数）
- **TinyPNG 后台压缩** — `internal/compress`：每 `COMPRESS_INTERVAL_SECONDS`（默认 60s）挑一张最大的待压缩图片，就地替换（先落 `.tinify-tmp` 再 `MoveToBin` 原图后改名，任一步失败都保留原图）；跳过 GIF 与 `< COMPRESS_MIN_KB`（默认 400KB）；`compressed_at` 非空即视为已处理（压缩未变小也标记，避免反复消耗配额）；**未配置 `TINIFY_API_KEY` 时任务静默休眠**
- **媒体镜像链** — `internal/mirror`：公共面收敛到 `internal/objstore`（`Store` 接口只有 `Put`/`Head`，`ObjectMeta` 只保证 `ContentLen` 所有实现都给），两个实现是 `internal/r2`（S3 子集，手写 SigV4）与 `internal/openlist`（自建 OpenList 的 HTTP 接口）。`Target` 用 `Driver` 字段（`r2`/`openlist`，空值按 `r2` 处理）分流到对应实现，`New` 经 `newStore()` 装配；`Chain` 编排**两个目标（首位 OpenList、次位 R2）**——推送侧不分主次、两边都推，所谓优先级只体现在前端取用顺序上（见「详情页大图源选择」）：本地落盘后异步推一份（`prepareUploadFile` → `PushAsync`），TinyPNG 压缩成功后把**压缩图**也推上去（`compress/worker.go`），**缩略图保持纯本地**；两侧 append-only（内容删除时本地进垃圾桶、对象不删）。幂等判定走 `sameObject()`：**优先 md5**，OpenList 的 Local 驱动 `hashinfo` 恒为 `null`（拿不到远端 md5），退路是 `ContentLen == 本地字节数`——OpenList 无远端哈希，坚持「无哈希就重传」会让每轮回填全量重推。判错的代价是不对称的：字节数判错只是少传一轮，而压缩原地改写后下一轮 `Head` 就会暴露差异；回填任务**每个目标各一个、各持一把锁**（`lock:mirror:r2` / `lock:mirror:openlist`），互不阻塞，日志前缀取 `LogName()`。凭据不全即整体停用（`Enabled()==false`，与 TinyPNG 缺 Key 同一策略）。SigV4 签名的规范化 URI 必须用**已解码**的 `URL.Path`——用 `EscapedPath()` 会把中文/空格对象名二次编码成 `%25E4…`，服务端必然拒签
- **详情页大图源选择** — 后端在 `img`/`video` 之外给**绝对地址** `mirror_img`/`mirror_video`（R2）与 `mirror2_img`/`mirror2_video`（自建 OpenList），两级**各自独立下发**（某一级停用时只是少一个候选，不影响另一级）；换公开域名只改服务端 .env，前端不必重建。前端 `utils/imageSource.ts` 按 **OpenList → R2 → 源站** 逐级回退：**按优先级串行短路探测**（首选通了就不再探次选，首选可达是常态，并行探两个等于每次冷启动都白付一倍探测流量）、**只探测镜像侧**（源站本来就是回退目标，探它白发一次整图请求）。**优先级为什么是 OpenList 在前**：它跑在自有广州机上、硬盘直出，不受第三方 CDN 计费与欠费停服的牵制；R2 走 Cloudflare，国内访问常被拖慢甚至不通（实测 `img.xqecz.bond` 明显更差），作为次选——但仍比源站优先。注意 `mirror*` 这组字段名是**历史命名**，不再表示优先级，真正的顺序在 `bestSource()` 里，改顺序只需动它和 `probeMirror` 的入参次序。结论存本机 `localStorage`（键 `xqecz:image-source-v3`，结构 `{openlist?, r2?}`，不上传；**带 v3 是因为首选从七牛换成 OpenList 后旧记录的结论会继续把人引向已下线域名**，换键让它们整体作废比重写干净），判定键是**网络标识**：公网 IP（`1.1.1.1/cdn-cgi/trace`，识别代理开/关）为主 + `navigator.connection` 本地指纹为辅（仅 Chromium，接口失败就只用本地指纹、**拿不准不推翻旧结论**），**标识没变就复用、变了才重探**。因为 IP 是异步取的，同步读只能先比本地指纹，IP 回来发现变了再重探。缓存命中的那一级若**当前没有地址**（如首选未配 `OPENLIST_PUBLIC_BASE` 时 `mirror2_*` 变空）即视为失效并重探，否则会一直指着一个不存在的源。未探的那一级结论**留空而非记 false**——记成「不可达」会让首选失效时直接掉到源站，白白绕开还能用的次选。**来源未确定时不渲染媒体**（`ContentMedia` 的 `resolved` 门），避免「先源站渲染、换源再取一次」的重复请求——F12 里表现为 `api39 → r2 → api39` 来回跳。渲染层兜底走**链式回退**（`MediaImage` 的 `fallbackSrc` 传数组，`@fallback` 回传失败地址）：调用方据此判断是**哪一级**失败，用 `markUnreachable(tier)` 只标死那一级——OpenList 挂了不连累 R2，它正是链上该顶上的下一级；再不行退生产同路径 → 坏图
- **推荐算法单一入口** — 只改 `internal/recommend/recommend.go:ScoreItem()`（纯函数）；刷新节奏与落库在 `Refresher.Refresh()`
- **降级优先** — ffmpeg 缺失时图片缩略图降级为纯 Go 解码缩放，视频缩略图失败仅告警不影响上传；Redis 不可用时读路径直查 MySQL
- **前端动效与背景锁滚优先用 CSS 原生** — `prefers-reduced-motion` 统一在 `assets/main.css` 末尾按偏好压掉过渡/动画时长，**不要**写全局 `transform: none`（Arco 的居中偏移与走马灯位移都靠 transform 承担定位职责，归零等于打坏布局）；加载指示器例外，它表达「正在进行」，冻结会把加载态变成无反馈空白，故保留循环动画。JS 里显式传 `behavior:'smooth'` 的滚动**优先级高于 CSS**，媒体查询压不住，必须经 `utils.scrollBehavior()` 判定。详情页覆盖层的背景锁滚改由 `html:has(.cd-root){overflow:hidden}` 声明，配 `html{scrollbar-gutter:stable}` 常驻槽位吸收横向跳动——不再在 `onMounted` 改 body 内联样式、`onBeforeUnmount` 还原（那套要先把旧值存下来，漏一处就把全站锁死在 `overflow:hidden`）。锁在 html 而非 body：改 body 依赖「body 的 overflow 传播到视口」这条规则，而滚动条槽归 html 的 `scrollbar-gutter`，两层容易对不上
- **View Transition 只有一个入口** — 一律经 `utils/viewTransition.ts:withRevealTransition(update, origin?)` 发起，不要各处裸调 `document.startViewTransition`。当前唯一使用方是日/夜切换（`App.vue:onThemeToggle`，圆形揭示从按钮位置铺开）。三条必须知道的：① **减少动效只能在发起前判**——UA 给 `::view-transition-*` 的默认交叉淡出挂在快照伪元素树上，`main.css` 里用 `*` 的媒体查询压不到它，所以 `prefers-reduced-motion` 的处理写在该函数里（不支持 / 要求减少动效 → 直接执行 update，视觉等同改动前）；② **回调里改 Vue 响应式状态必须 `await nextTick()` 再返回**，否则新快照拍在 DOM 更新之前（主题的 `applyMode` 挂在 `watch` 上，落地要等一次调度微任务），动画就成了「铺了个寂寞」；③ `main.css` 末尾把根快照的默认淡出全局关掉了（`animation:none` + `mix-blend-mode:normal` + 新的在上），**任何后续 VT 都拿不到 UA 默认淡出**，要淡出得自己排动画。**没有给首页↔详情加路由级过渡**，不是漏做：详情媒体受 `imageSource` 的 `resolved` 门控，快照时刻大图往往还没渲染，拍到的只有骨架屏，反而不如组件自带的 `cd-fade-in`；共享元素（卡片缩略图→大图）同理，两侧必须同时在场才谈得上形变
- **迁移期工具** — `cmd/` 下的四个小工具（dbsync/dbinfo/dbsql/rediskeys）是排查与对拍用的，改动数据库相关行为时优先用它们核实，不要凭记忆断言
- **数据库连接单一来源** — DSN 一律经 `internal/mysqldsn.Format()` 组装（服务端与四个 cmd 工具共用），不要在调用点各写一份：新增连接参数时分散写法必然漏配。加密由 `MYSQL_TLS` 控制（透传驱动的 `tls` 参数，留空=不加密）；**TiDB Cloud Serverless 强制加密**，明文连接报 `1105 insecure transport`，必须设 `MYSQL_TLS=true`。逐字节搬运数据（迁移校验）时须关 `ParseTime`，否则驱动做时区换算会改写时间值
- **TiDB 兼容性** — 生产库为 TiDB 时有三处与 MySQL/MariaDB 不同：① `TEXT`/`BLOB`/`JSON` 列**不允许 DEFAULT**（报 1101），建表须去掉默认值（业务代码对这些列总是显式赋值，故无语义变化）；② 不支持 `CHECKSUM TABLE`，一致性校验改用「按主键有序导出后比对 md5」（见 `scripts/migrations/2026-09-26-migrate-to-tidb.sh`）；③ `sys`/`mysql` 等系统库对业务账号**只读**，必须建独立库

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 + gin + GORM v2（MySQL）+ go-redis v9 + log/slog |
| 媒体 | ffmpeg 子进程（视频抽帧/缩放）+ `deepteams/webp`（纯 Go、零 CGO 的 WebP 编解码，用于生成缩略图）+ `x/image/draw` 缩放；压缩依赖外部 TinyPNG API |
| 前端 | Vue 3.5 + TypeScript + Vite + Tailwind CSS + Arco Design Vue + zod（响应校验） |
| 数据 | MySQL（10 张表）+ Redis（会话/缓存/浏览量/推荐 ZSet） |
| 同仓编排 | moon（任务图与缓存）+ pnpm workspace（前端依赖）+ go module（后端依赖） |
| 运行方式 | 本地直启（`scripts/dev.mjs` 编排两端）；生产**前后端分开发布**——后端是 Go 单二进制，CI 经 SSH 直传部署机后由 OpenRC 接管（`doas rc-service xqecz`），服务器上不跑包管理器；前端由 EdgeOne Makers 自行构建（`VITE_API_BASE_URL` / `VITE_MEDIA_BASE_URL` 只能在其控制台配，仓库根 `.env` 被 gitignore），发布流程与 CI 见 `docs/deploy.md` |

## 修改指南

1. **前端改动** — 接口形状以 `packages/frontend/src/types/schemas.ts` 与 `src/api/index.ts` 为准（后端响应必须能被 zod schema 直接解析，改接口后跑契约冒烟）；前端相关约定分散在本文「核心约束」与修改指南 7。**本仓只有这一份 `AGENTS.md`**，前端不另设分册——新增前端约定直接补进本文对应章节，不要另起文件
2. **后端模块改动** — 在 `packages/server/internal/modules/<模块>/` 内改；Handler 依赖 `app.Deps`（Cfg/DB/Redis），路由在各自 `Register()` 中挂载，再由 `internal/api/router.go` 统一装配（web 包不反向依赖业务模块，避免循环依赖）
3. **新增接口** — 响应统一走 `web.OK` / `web.Fail`（校验类失败用 `web.SoftFail`）；身份从 `web.MustIdentity(c)` 取；列表查询若同时要 Count 与 Find，必须共用同一份条件会话（`db.Session(&gorm.Session{})`），否则总数与列表会不一致
4. **数据库变更** — 改 `internal/store/models.go`（显式 `column` 标签 + `TableName()`）；生产用正式 migration（存 `scripts/migrations/`），勿开 `AutoMigrate`。热点查询的索引见 `scripts/migrations/2026-09-13-add-hot-path-indexes.sql`
5. **媒体管线** — `internal/media/`：缩略图优先 ffmpeg、失败降级纯 Go（WebP 质量 85）；镜像链见 `internal/mirror`（`Target` 定义目标、`Chain` 编排多目标，推送侧不分主次），`Push` 与回填共用同一判定，**改镜像范围只需动 `mirrorable()`**；新增一个镜像目标只需加一个 `TargetFrom*` 映射，不必碰推送与回填逻辑。`MoveToBin` 负责把文件搬进垃圾桶目录。**上传不再做 WebP 无损转换**（`internal/media/webp.go` 已删除），压缩交给 `internal/compress` 的 TinyPNG 任务；改压缩策略只需动 `compress/worker.go` 的候选筛选与 `shrinkInPlace`
6. **推荐算法** — 见「核心约束」单一入口条目
7. **瀑布流布局改动（前端首页）** — 纯布局算法在 `packages/frontend/src/composables/useWaterfallLayout.ts:computeLayout()`（与 DOM 解耦，输出 `Map<id, Position>`，可直接单测，勿写死在组件里）；改布局逻辑优先改纯函数并补 `__tests__/useWaterfallLayout.test.ts`。核心约定：**稳定列**（卡片落列后不再换列，`preserveColumns` 默认 true，仅列数/列宽变化时全量最短列重排）、**full 全量重排**（数据集合变化——分页追加/diff 更新/列表替换——时强制重新平衡列底，避免增量分配被懒加载测量失真带偏导致短列空缺；图片尺寸变化仍走增量顺移）、**列底失衡收敛**（增量后 max-min 列高差超过 `IMBALANCE_THRESHOLD` 时自动补一次带锚定的全量重排）、**单一调度**（图片加载/尺寸/宽度变化合并到一帧 `requestAnimationFrame` 只 layout 一次）、**滚动锚定**（重算前 captureAnchor 固定视口顶部卡片）、卡片高度由 `[data-wf-id]` 批量量取、`restore`/`reset` 管 keep-alive 缓存。**卡片内部排版按容器查询而非视口媒体查询**——列宽由 JS 按容器实测宽算出，与视口不等价（375px 窄屏 2 列时每列约 170px），故 `.wf-card` 声明为 `container: wfc / inline-size`、内部字号/行数用 `@container wfc (...)`。只能用 `inline-size`：它只隔离行内方向、块高仍由内容驱动；写成 `size` 会给卡片加全尺寸包含使固有高度归零，瀑布流量到的全是 0 高，整页塌掉。**加载策略**：首页进入即自动连续拉取全部页（`loadAllPages`，每页 100 条），不依赖滚动触发，图片保持懒加载；keep-alive 往返用**增量同步**（`syncLatestOnActivated`）。**缓存**：`listCache`（localStorage）统一在 `onBeforeRouteLeave` 离开时写一次；`diffLists` 有全量快照守卫。列表筛选/搜索用自增 `loadSeq` 丢弃过期响应防竞态
8. **踩坑记忆** — `MYSQL_POOL_SIZE` 过小会在并发下成为瓶颈（实测 20 并发 × 每请求 3 次查询：池 5 → p50 616ms、池 30 → p50 304ms），示例值见 `.env.example`；`silent=1` 的详情请求**按旧语义跳过缓存读**（只写不读），因此会比命中缓存的请求多出三次回表，评估时延时要分开看；`contents.file_path` 表示「无媒体」时**既有 NULL 也有空串**两种写法，过滤必须同时判 `IS NOT NULL AND <> ''`，否则会把纯文本行当成缺图内容；`moon` 的常驻任务用 `preset: 'server'`（不是 `local: true`，后者在 2.x 已移除会直接解析失败）；GORM 的 `Count` 与 `Find` 若不共用条件会话，会出现「总数带过滤、列表不带过滤」的错位；`bigint` 主键在 JSON 与 ZSet 之间比对要显式转字符串；**pnpm 11 起不再读取 `package.json` 的 `pnpm` 字段**——`overrides` 等设置必须写在 `pnpm-workspace.yaml`；Windows 下 git-bash 会把命令行里的中文按 GBK 传给 curl，导致 MySQL 拒收非法 UTF-8（`Incorrect string value`），涉及中文的接口测试要用 `-F "field=<文件"` 或 `--data-binary @文件` 传参；**控制台里跑 `taskkill /PID` 可能静默失败**，清理占用端口的旧服务改用 `powershell Stop-Process` 并复核 `netstat`；**不要用 `taskkill /T`**（会连带杀掉自身会话）；**SigV4 对时钟敏感**——新机器未同步时间时 R2 全部请求返回 403（实测偏差 13.7 小时，`HEAD`/`PUT` 一律 403，日志刷 `r2 回填失败`），表现为「凭据明明没错却全部被拒」，排查 R2 403 先看 `chronyc tracking` 的 offset，用 `chronyc makestep` 校正并在 `chrony.conf` 补 `makestep 1.0 3` 让开机也步进；**本仓库工作区文件是 CRLF**（`git` autocrlf 检出所致，仓库内存的是 LF），因此 `gofmt -l` 会把大批未改动文件报成「未格式化」——**不要对目录跑 `gofmt -w`**，那会把无关文件的行尾从 CRLF 改成 LF 造成大面积污染，只对本次改动的具体文件执行；**部署用的 shell 脚本必须保持 LF**（`.gitattributes` 里 `*.sh text eol=lf` 已钉死）——CRLF 的 shebang 在 Linux 上是 `bad interpreter`，出事时没人盯着；**Windows 上 Vite 的 dev 服务器会被编辑器的原子保存打死**——保存 .vue 时若先在同目录落临时文件再改名（`.WaterfallCard.vue.<uuid>.tmpdir/`），chokidar 会 `EBUSY: resource busy or locked, watch '...'` 抛错退出，`scripts/dev.mjs` 随即销毁整个进程树，表现为「什么都没干、前后端一起没了」。看到 fe 退出 code=1 先翻 stderr 找 EBUSY，是工具链问题不是代码问题，重启即可；**OpenList 的 HTTP 接口三处反直觉**——① 鉴权头是 `Authorization: <token>`，**不带 `Bearer`**，带了必被拒；② 推送走 `PUT /api/fs/put` 且路径放 **`File-Path` 头**，该头会被服务端 URL 解码，故送 `EscapedPath()`（纯 ASCII 最安全），而 `/api/fs/get` 的 body `path` **不解码**，送原始 Unicode；③ 对象不存在时 `/api/fs/get` 返回 `code:500 object not found` 而非 404，判存在只能靠匹配 message，且免签下载 `/d/*` 对不存在的文件同样回 **500**；另需实例侧 `sign_all=false` 才能免签直链，否则 `/d/` 与 `/p/` 裸访问一律 401；**且 `OPENLIST_PUBLIC_BASE` 必须带直链前缀 `/d`（如 `https://drive.xiey.work/d`）**——OpenList 的裸域名是它自己的 SPA 前端，未加前缀的路径一律回 `200 text/html` 的 index.html，下发的 `mirror2_*` 会「换源成功但图全是 HTML」

## 迭代规范（AGENTS.md 自身）

每次完成代码改动、提交前（或随代码改动一并提交）应自检是否需迭代本文件。本规范即**本文件的维护 SOP**：代理由此判断"改动多大、哪些重点值得沉淀"，避免文档过期或过度膨胀。

### 触发条件（满足任一即应迭代）

改动若触及以下任一层面，应更新对应章节：

| 触发点 | 需更新的章节 |
|--------|--------------|
| 新增/删除后端模块、`packages/server/internal/` 下目录结构变化 | 目录结构 / 修改指南 2 |
| 新增/修改 HTTP 接口或其分组 | HTTP 接口表 / 修改指南 3 |
| 新增/删除数据库表、模型字段或列语义变化 | 目录结构 store / 修改指南 4 / 核心约束·内容统一模型 |
| 新增/删除前端组件、composable、页面或目录调整 | 目录结构 frontend / 修改指南 7 |
| API 密钥权限、Redis 缓存键、软删除、降级策略等约束变化 | 核心约束对应条目 |
| 增减依赖版本或换技术栈 | 技术栈表 |
| 新增/调整根脚本、moon 任务或常用命令 | 快速命令 |
| 出现新的"踩坑记忆"（类型比对、构建、平台差异等） | 修改指南 8 |
| 产生新的迭代教训（下文"沉淀原则"命中者） | 视情况新增小节 |

### 沉淀原则（何时才值得写进文档）

- **只沉淀"再次需要时无法轻易从代码/现有文档推出"的信息**：文件级定位（哪个文件哪个函数）、模块依赖关系、隐含约定、踩坑、跨端契约。
- **可轻易从代码推断的内容不要写**（如某函数签名细节、随版本的常量值）——保持文档精简，避免与代码漂移。
- **沉淀的应是"约束/约定/推理链"，而非"实现快照"**。实现细节会变，约定长期有效。
- 一次改动通常**只新增/修订一两条**；多条大改请拆分提交，避免单次 diff 过大难审。

### 迭代流程（每步均可执行命令核实）

1. **定位真实变更**：用 `git status` 看改了哪些文件、`git diff --stat` 看规模，先读改动后再写，禁止凭记忆描述。
2. **对照上表判断命中**：若无命中则不需改本文件（避免为小改动制造噪音）；有命中则找到对应章节。
3. **最小修订**：只改命中条目，不重排非相关内容；措辞沿用中文、与上下文风格一致。
4. **单一来源**：本仓只有这一份 `AGENTS.md`，前后端约定都在本文内。改前端时同样更新本文对应章节（前端契约见「核心约束」首条、瀑布流见修改指南 7），不要新建前端分册——同一事实写两遍必然有一天互相矛盾。
5. **自校验（提交前必做）**：重新 `git diff AGENTS.md`，确认
   - 目录树图与实际目录一致（新增文件别漏、删掉的别残留）；
   - 文件路径/函数名/章节名可被搜索定位到真实代码；
   - 命令可在仓库内真实执行（不要写不存在/改名过的脚本）；
   - 全文无自相矛盾（陈旧条目如同名功能须同步更新）。
6. **保留版式**：本文件用 2 空格缩进、中文说明、markdown 表格与 `code` 块定位关键文件；新增内容沿用此风格。

### 版本与变更记录约定

- 迭代以**流水修订**为主，不维护独立"版本号"或"CHANGELOG 章节"——本文件的 git 历史即版本记录。
- 一次提交应包含"代码改动 + 相应的 AGENTS.md 迭代"，确保文档与代码同源同步，避免事后追补。

## 已归档

- 旧后端 `xqecz-golang/`、`xqecz-nodejs/` 已完整迁移进本 monorepo，并归档至 `D:/xqecz/archive/`（各自独立 git 仓库，完整历史保留；均打本地 tag `archive/monorepo-migration-2026-07-22`，远程 `xqecz-all.git` 的 `golang`/`nodejs` 分支亦保留）。monorepo 已 gitignore `/archive/`。
- 独立前端仓库 `xqecz_frontend` 已并入本仓 `packages/frontend`（源文件直接纳入 monorepo，不再独立仓库/symlink）；原独立仓库整体移至 `D:/xqecz/archive/xqecz_frontend` 保留历史（远程 `xqecz_frontend.git` 的 `dev` 分支亦保留）。
- NestJS 后端（`packages/api`）、Go Worker（`packages/worker`）与 gRPC 定义（`proto/`）已被 Go 单体后端取代并删除，
  对应实现与迁移记录见 `docs/go-backend-migration.md`。仓库历史已于 2026-09-14 清洗重建（单一初始提交），旧实现不再可从 git 历史回溯。

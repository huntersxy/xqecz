# 小泉动漫二创站（xqecz）

二次创作内容的上传与浏览站点。内容统一为「文本 + 可选媒体」，不做分类，以贴吧 / 动态流的方式呈现；附带评论、投票、内容认领与管理后台。

前端是 Vue 3 单页应用，后端是**单一 Go 进程**（HTTP + MySQL + Redis + 进程内媒体处理）。前端依赖由 pnpm workspace 管理，后端由 go module 管理，两端任务由 moon 统一编排。

## 架构

```
前端 (Vue 3)  ──HTTP /api──▶  Go 单体进程（packages/server）
                                ├─ gin 路由 + 中间件（会话 / API 密钥 / 管理员 / CORS）
                                ├─ GORM ──────────▶ MySQL（10 张表）
                                ├─ go-redis ──────▶ Redis（会话 / 读穿缓存 / 浏览量 / 推荐 ZSet）
                                ├─ 进程内媒体处理：缩略图（ffmpeg，缺失时降级纯 Go）
                                └─ 后台压缩：TinyPNG 定时挑最大图压缩，原图入 data/bin
静态资源：/uploads、/thumbs 由同一进程托管（/images 仅历史兼容），物理目录为项目根 data/
```

**推荐链路**：`internal/recommend` 读取已通过内容 → 纯函数打分（时间衰减 + 浏览量 + 点赞）→ 通过「写临时键 + RENAME」原子写入 Redis ZSet；读取优先 ZSet 并按序回表，无数据或异常时降级为 `view_count` 排序。启动刷新一次，之后每 10 分钟一次，多实例经 Redis 锁防抖。

## 目录结构

```
packages/
├── server/                     # Go 后端（唯一后端进程）
│   ├── cmd/server/             # 入口：读 .env → 连 MySQL/Redis → 启动 HTTP + 推荐刷新
│   ├── cmd/{dbsync,dbinfo,dbsql,rediskeys}/   # 排查与对拍用的小工具
│   ├── internal/modules/       # 业务模块：auth / content / comment / poll / admin / apikey
│   ├── internal/{api,web,store,cache,media,recommend,config,cli,project}/
│   └── internal/app/           # Deps：各层共享依赖集
└── frontend/                   # Vue 3 前端
    └── src/{api,components,composables,stores,views,router,types,utils}/

scripts/                        # 开发编排、构建包装、部署启动器、一次性 SQL 迁移
data/                           # 运行期媒体目录（uploads / thumbs / bin），不入库
docs/deploy.md                  # 部署与 CI 说明
AGENTS.md                       # 面向贡献者与 AI 的工程约定（改代码前先读）
```

## 快速开始

前置：Node.js ≥ 20 + pnpm、Go 1.26+、FFmpeg（可选，缺失时图片缩略图降级纯 Go，视频缩略图不可用）。
开发期 MySQL / Redis 直连远端实例，本机无需安装；项目不使用 Docker。

```bash
pnpm install                    # 前端依赖（含 moon CLI）
cp .env.example .env            # 填入 MySQL / Redis 凭据
pnpm dev                        # 并发起 Go 后端(:3000) + 前端(:5173)，端口占用自动顺延
```

## 环境变量

配置来源为**项目根 `.env`**（前后端共用），`.env.local` 可覆盖（便于本地指向影子库）。

| 变量 | 说明 |
|------|------|
| `MYSQL_HOST` / `MYSQL_PORT` / `MYSQL_USER` / `MYSQL_PASSWORD` / `MYSQL_DATABASE` | MySQL 连接 |
| `MYSQL_POOL_SIZE` | 连接池上限（默认 10；并发下建议 30，见 `.env.example` 实测数据） |
| `MYSQL_CONNECT_TIMEOUT` | 建连超时（秒） |
| `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` / `REDIS_DB` | Redis 连接 |
| `REDIS_PREFIX` | 业务 key 统一前缀，多实例共存时须区分 |
| `PORT` | 后端监听端口（默认 3000） |
| `CORS_ORIGINS` | 允许的跨域来源，逗号分隔；同源部署时留空即可 |
| `APP_ENV` | `production` 时 gin 走 release 模式 |
| `UPLOAD_DIR` / `THUMB_DIR` / `IMAGES_DIR` | 媒体目录，留空则用项目根 `data/` 下同名目录 |
| `VITE_API_BASE_URL` / `VITE_MEDIA_BASE_URL` | 前端注入的接口与媒体地址（默认同源 `/api`） |

## 常用命令

```bash
# 开发
pnpm dev                        # 后端 + 前端（scripts/dev.mjs，端口自适应、退出清理子进程树）
pnpm dev:server                 # 仅后端
pnpm dev:fe                     # 仅前端

# 构建与运行
pnpm build                      # 后端二进制（按平台命名、剥离符号表）+ 前端产物
pnpm start:backend              # 直接运行后端二进制

# 质量检查
pnpm server:test                # go test ./...
pnpm server:vet                 # go vet ./...
pnpm server:fmt                 # gofmt
pnpm fe:test                    # vitest 全量
pnpm fe:typecheck               # vue-tsc 类型检查
pnpm accept                     # 对运行中的后端跑全接口验收（需先起后端）

# 任务编排（moon：带增量缓存与输入指纹）
pnpm exec moon run server:build
pnpm exec moon run server:test frontend:test

# 运维（部署机无 Go 工具链也可用）
./packages/server/xqecz-server admin -username admin -email you@example.com
./packages/server/xqecz-server admin -username admin -reset
```

## 生产部署

生产形态是**前后端分开发布**，服务器上不执行包管理器：

| 侧 | 形态 | 产物 |
|----|------|------|
| 后端 | 宝塔「Go 项目」 | `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 单二进制 + `.env` |
| 前端 | 宝塔「静态站点」 | `dist` 静态文件，`/api` 反代到后端端口 |

推送到 `master` 后由 GitHub Actions 自动构建、上传并重启后端，并以生产 `/api/health` 探针收尾。完整流程、`.env` 定位规则、nginx 参考配置与回滚方式见 **[docs/deploy.md](docs/deploy.md)**。

## 接口约定（摘要）

- 全部挂在 `/api` 前缀下，共 60 条；统一响应 `{ code, message, data }`，`code === 200` 为成功
- 校验类失败沿用 HTTP 200 + 业务码；鉴权类失败才改 HTTP 状态码
- 时间字段统一序列化为 UTC 毫秒（与前端 `Date.toJSON()` 逐字节一致）
- 认证双通道：请求头 `X-API-Key` 优先，其次 Cookie 会话 `session_id`
- 上传：multipart 字段名 `file`，流式落盘，单文件 ≤ 20MB，仅 `image/*` 与 `video/*`；上传不做转码，压缩由后台 TinyPNG 任务就地替换（GIF 与 <400KB 跳过，压缩后后缀不变）
- 下载：媒体地址带 `?download=1` 即以附件返回（`Content-Disposition`），文件名取内容标题
- **前端是契约**：`packages/frontend/src/api/index.ts` 与 `src/types/schemas.ts` 定义接口形状，后端响应必须能被前端 zod schema 直接解析

完整的接口清单与改动约束见 **[AGENTS.md](AGENTS.md)**。

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 + gin + GORM v2 + go-redis v9 + log/slog |
| 媒体 | ffmpeg 子进程 + `deepteams/webp`（纯 Go、零 CGO）+ `x/image/draw` |
| 前端 | Vue 3.5 + TypeScript + Vite + Tailwind CSS + Arco Design Vue + zod |
| 数据 | MySQL（10 张表）+ Redis（会话 / 缓存 / 浏览量 / 推荐 ZSet） |
| 编排 | moon（任务图与缓存）+ pnpm workspace + go module |

## License

[GPL-3.0](LICENSE)。站内用户上传内容遵循 CC BY-NC 4.0（非商业使用）。

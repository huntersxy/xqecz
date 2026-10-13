# 图片自动编目

首页上传抽屉、快速上传页、创作工作台支持勾选「自动生成图片标题与标签」。只支持图片，正文与媒体、大小、审核和身份规则沿用现有上传契约。关闭选项恢复手动草稿。

服务端收到 `auto_metadata=true` 后强制使用「待命名作品」与「待生成标签」，除用户声明的 `AIGC` 标记外，忽略客户端标题/标签。内容和任务在同一个事务提交；上传返回后，后台串行调用模型，成功回写标题和标签并失效详情、列表、标签及管理仪表盘缓存。不改变审核状态。普通上传不调用视觉模型。

三个上传入口还支持独立的「这是 AI 生成的作品」选项（`is_ai_generated=true`），图片、视频和纯文字均可标记。勾选后服务端合并 `AIGC`，不改其他手动标签；已有 `AIGC`、`aigc` 或 `#AIGC` 统一为一份 `AIGC`。前端合并不写入手动草稿，因此取消勾选只撤去选项附加的标记，不删除用户自己填写的标记。

自动补全时占位标签可同时包含 `AIGC`，生成回写依据任务保存的用户声明恢复该标记；失败/重试期间也不丢失。模型自行输出的 `AIGC` 会被忽略，不靠图片外观判断是否 AI 生成。此用户声明不占模型标签数量上限。共享合并规则分别位于前端 `utils/contentTags.ts` 和后端 `internal/contenttags`。

## 配置与预设

管理员在「管理控制台 → 视觉模型」配置 API 基址（或完整 `/chat/completions` 端点）、模型、API Key、提示词、标签上限、超时、思考参数及角色参照开关。默认停用且无 Key。待处理任务保留，配置 Key 并启用后继续执行。

预设：`https://api.siliconflow.cn/v1`、`Qwen/Qwen3.5-4B`，`enable_thinking=false`，`max_tokens=1024`。硅基流动官方介绍该 4B 模型支持多模态且免费，具体模型可用性和额度以账号实际调用为准：

- <https://www.siliconflow.cn/news/yg6n19y2g6frnp4koxu8ye48>
- <https://docs.siliconflow.cn/docs/api/chat-completions-post>

其他服务使用 OpenAI Chat Completions 兼容格式，支持 Base64 `image_url`。可关闭硅基流动思考参数，并按模型选择 `max_tokens` / `max_completion_tokens`。本版没有实现 Responses、Anthropic Messages 等独立协议适配。

「保存并测试视觉接口」使用内置色块图，验证请求、JSON 解析及标签规则，不写作品。实际角色识别仍需真实图片验收。

## 角色参照与压缩

角色库是服务端私有目录 `data/vision-references/`，不通过媒体路由公开。文件名即角色名。新增/替换参照图先运行：

```powershell
go -C packages/server run ./cmd/vision-import -source 'E:\Pr\小泉动漫工程\人物素材\经典' -dest 'D:\CODE\xqecz\data\vision-references'
```

导入器保留原图，以 JPEG 85 / 最长边 1024 创建副本。当前五位角色：陈晓晓、方源、柳如烟、小泉、严小希；五张压缩图合计 287456 字节，原图合计 4653528 字节。角色库最多 12 张，每张最多 512KB。模型请求会包含命名参照图和最后一张目标作品，所有图片均从本地读取，以 Base64 传输，不依赖媒体源站/CORS，也不传上传者邮箱或昵称。

部署时必须单独复制 `data/vision-references/` 到后端同名目录；该运行时目录已 gitignore，不在前端构建中，也不随代码提交。管理员页显示实际可读的角色名。关闭参照开关后仅按作品画面生成。

提示词要求按发色、挑染、发型、瞳色及配饰综合匹配，允许换装，不按单一服饰认人。结果额外包含 `characters: [{name, confidence}]`；服务端只接受参照库中且模型置信度 ≥ 0.85 的名字，优先加入角色标签，移除未确认参照角色标签，拒绝包含未确认角色名的标题。模型自报置信度不是准确率保证，需拿不同画风/换装/多人图实际校验。

## 标签与任务边界

提示词负责角色/主体/动作/场景/画风的语义提取；服务端负责格式：NFKC、去前缀 `#`、中英文分隔符拆词、去重、已有标签大小写归一、过滤泛词/网址/句子、每词 2–20 字、总量 1–12 个（默认 8）。多词概念如「黑色长发」保持完整。提供最近 500 条已审核内容中的高频标签（最多 120 个）供模型复用。

请求最多 180 秒，默认 90 秒；单进程串行处理，每 3 秒检查队列。失败退避 15/30 秒，最多 3 次，之后管理员可重试。任务拥有 5 分钟租约和唯一 claim token，崩溃后可被接管；旧 worker 的结果不会覆盖新 claim。停止服务会取消 HTTP 请求，未完成任务保留租约以供恢复。启用多个实例时通过数据库条件更新竞争 claim。

编辑标题、标签或替换媒体会事务内取消待处理/运行中/失败任务；结果保存还会锁定并逐字比较原标题、标签和文件路径，避免人工编辑被覆盖。删除作品同时删除任务，迟到结果不会复活作品。服务端压缩/生成缩略图不取消任务。

启动会只迁移 `vision_settings` 和 `vision_jobs` 两张新表，不改 `contents` 列。部署账号需要建这两张表的权限；初始化失败仅关闭该可选功能，普通上传保持可用。当前并未对生产数据库执行迁移或写入。

Key 保存在 `vision_settings` 的后端配置中，不返回前端，查询/保存该配置使用静默 SQL logger；Key 留空保持原值，清除需显式勾选。上游响应正文及请求错误地址不进入任务错误信息，避免第三方错误正文回显凭据。数据库及备份仍按服务端凭据管理。

## 验证

```powershell
pnpm server:test
pnpm server:vet
pnpm fe:typecheck
pnpm fe:test
pnpm --filter ./packages/frontend run build-only
```

后端新增测试采用临时 SQLite 与内存 Redis，不载入 `.env`。SQLite 只用于隔离测试；生产仍是 MySQL/TiDB。真实 MySQL 并发锁、硅基流动模型和角色准确率未被这些模拟测试替代。

浏览器隔离服务（以本地假视觉端点验证完整上传/配置/回写路径）：

```powershell
$env:XQECZ_VISION_FIXTURE_ADDR='127.0.0.1:31313'
$env:XQECZ_VISION_FIXTURE_REFS='D:\CODE\xqecz\data\vision-references'
go -C packages/server test ./internal/api -run '^TestVisionBrowserFixture$' -v -timeout 30m
# 另一个终端
$env:VITE_PROXY_TARGET='http://127.0.0.1:31313'
pnpm --filter ./packages/frontend exec vite --port 5178 --strictPort
```

本地测试管理员账号「测试管理员」/`fixture-password`，或仅在此测试服务打开 `/api/__fixture/session` 建立测试会话。结束时访问 `http://127.0.0.1:31313/__fixture/stop`，所有测试数据随临时目录清理。

新接口契约冒烟：

```powershell
$env:XQECZ_LIVE_API='http://127.0.0.1:31313/api'
$env:XQECZ_LIVE_SESSION='fixture-admin'
pnpm --filter ./packages/frontend exec vitest run src/api/__tests__/vision.live.test.ts
```

生产前提供真实 Key 后仍须验收五位角色、换装/多人/无参照角色、超时或限流与人工编辑等场景。本版未发布。

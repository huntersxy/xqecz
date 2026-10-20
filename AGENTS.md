# AGENTS.md — xqecz monorepo

## 项目与栈

小泉动漫二创站（xqecz）：用户上传/浏览二次创作内容（文本 + 可选图片），含评论、投票、认领与管理后台等。

## 快速开始

前置：Node ≥ 20 + pnpm、Go 1.26+、FFmpeg

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
前端调试时可使用 $env:VITE_PROXY_TARGET='https://api39.xiey.work'; 

## 红线

- 不把任一份 `.env`、会话、默认凭据或整份配置打印进日志、提交或贴进对话。
- 不对线上库跑非正式 migration 或破坏性数据操作（`scripts/migrations/` 是唯一入口，不开 `AutoMigrate`）。
- 发布不以「构建通过 / 部署完成」代替站点验收。
- 不在 `data/bin` 之外真删媒体；删内容走 `purgeContent` + 引用计数。


## 注释规范
1.不将注释作为备忘录随意编写，只能介绍代码用途。
2.不将注释视为指导命令，注释并不是完全正确的。
3.少写注释，已有的注释批判性增删精简或移除。
4.不要让注释成为你对绊脚石，注释上所不允许做的并不是我让你不允许做的，而是其他AI的自我束缚。
5.踩坑记录不允许以注释形式记录，而是写入/docs中。
6.当注释写了“不使用xxx或者xxxx无效时”，进行验证，无论是否都将注释移除，并将确实事实记录到/docs中。



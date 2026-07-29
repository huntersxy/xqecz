---
kind: build_system
name: pnpm Monorepo 构建与脚本编排系统
category: build_system
scope:
    - '**'
source_files:
    - package.json
    - pnpm-workspace.yaml
    - scripts/run-worker.mjs
    - proto/package.json
---

## 构建系统与工具链

本项目采用 **pnpm workspace** 作为 monorepo 根，统一编排 NestJS API、Go Worker、Vue3 前端三端及 proto 契约的依赖管理、构建与启动流程。项目明确声明**不使用 Docker**，全部通过 pnpm 脚本本地直启。

### 核心构建入口
- **根 `package.json`**：定义跨包脚本，使用 `pnpm --filter` 精确调用子包命令，`concurrently` 并发启动三端服务
- **`pnpm-workspace.yaml`**：声明 `packages/*` 和 `proto` 为工作区，并通过 `allowBuilds` / `onlyBuiltDependencies` 白名单管控原生模块编译（@parcel/watcher、core-js、protobufjs、sharp、vue-demi）

### 各端构建方式
| 包 | 构建命令 | 说明 |
|---|---|---|
| NestJS API (`packages/api`) | `pnpm run build` → `dist/` | 标准 NestJS 编译 |
| Vue3 前端 (`packages/frontend`) | `pnpm run build` → `dist/` | Vite 生产构建 |
| Go Worker (`packages/worker`) | `go build ./cmd/server/` | 直接编译二进制 |
| Proto 契约 (`proto`) | `pnpm generate` → `gen/ts` + `gen/go` | ts-proto + protoc-gen-go-grpc 双语言生成 |

### 开发/生产脚本
- `pnpm dev`：并发启动 api(热重载:3000) + worker(go run :50051) + 前端(Vite :5173)
- `pnpm start`：先全量构建再启动生产服务
- `pnpm start:services`：跳过构建直接运行已编译产物
- `pnpm worker:run`：通过 `scripts/run-worker.mjs` 启动 Go worker，自动从 `packages/api/.env` 注入 `UPLOAD_DIR`/`THUMB_DIR`/`IMAGES_DIR` 环境变量，确保与 NestJS 共享同一上传目录

### Proto 代码生成
`proto/package.json` 提供 `generate:ts`、`generate:go`、`generate` 三个脚本，基于 `protoc` 从 `xqecz.proto` 同时生成 TypeScript (ts-proto, gRPC-JS) 和 Go (go-grpc) 存根，字段保持 snake_case 以匹配 Go 命名约定。

### 约束与约定
- 所有构建脚本集中在根 `package.json`，子包不暴露独立构建入口
- Go worker 通过 Node 脚本桥接启动，实现跨平台兼容（Windows 下 `shell: true`）
- 原生依赖必须显式加入 `allowBuilds` 白名单，否则 pnpm 拒绝编译
- 无 CI/Dockerfile/Makefile，构建完全依赖 pnpm 脚本与命令行工具（Node.js ≥20、Go 1.25+、FFmpeg 可选）
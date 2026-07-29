---
kind: dependency_management
name: pnpm Monorepo + Go 多语言依赖管理
category: dependency_management
scope:
    - '**'
source_files:
    - package.json
    - pnpm-workspace.yaml
    - packages/api/package.json
    - packages/frontend/package.json
    - proto/package.json
    - packages/worker/go.sum
    - scripts/run-worker.mjs
---

## 1. 使用的系统与工具
- **Node.js 生态**：使用 pnpm workspace 作为 monorepo 包管理器，通过 `pnpm-workspace.yaml` 声明工作区范围（`packages/*` 与 `proto`），并借助 `allowBuilds` / `onlyBuiltDependencies` 精确控制原生模块构建。
- **Go 生态**：Worker 服务使用标准 Go 模块（`go.mod`/`go.sum`），依赖通过 `go.sum` 锁定版本，未启用 vendor 目录（`.gitignore` 中显式忽略 `packages/worker/vendor/`）。
- **Proto 契约**：`proto` 子包通过 `ts-proto` 从 `xqecz.proto` 生成 TypeScript 与 Go 客户端代码，供 API 与 Worker 共享 gRPC 接口。

## 2. 关键文件与位置
- 根级编排：`package.json`、`pnpm-workspace.yaml`
- Node 包定义：`packages/api/package.json`、`packages/frontend/package.json`、`proto/package.json`
- Go 依赖锁定：`packages/worker/go.sum`（存在 go.sum，go.mod 未在仓库中提交）
- 脚本入口：`scripts/run-worker.mjs`（用于启动 Go worker）、根 `package.json` 中的 `worker:build`/`worker:run` 脚本

## 3. 架构与约定
- **Monorepo 分层**：API（NestJS）、Frontend（Vue3/Vite）、Worker（Go）各自维护独立 `package.json`，通过 pnpm workspace 统一安装与工作区脚本调用；proto 契约集中管理并通过脚本生成两端代码。
- **构建脚本聚合**：根 `package.json` 提供 `dev`/`build`/`start` 等命令，内部通过 `pnpm --filter ./packages/<pkg>` 定向到子包，并使用 `concurrently` 并行启动三端。
- **原生依赖白名单**：`pnpm-workspace.yaml` 的 `allowBuilds` 与 `onlyBuiltDependencies` 仅允许 `@parcel/watcher`、`core-js`、`protobufjs`、`sharp`、`vue-demi` 五个包执行构建钩子，避免意外触发 C++ 编译。
- **前端分包策略**：Vite 配置中使用 `manualChunks` 将 `vue`/`pinia`、`antd`、`utils`、`motion-v` 等拆分为独立 vendor chunk，减少首屏体积。
- **Go 依赖锁定**：Worker 侧通过 `go.sum` 记录哈希校验，确保可重复构建；计划文档指出后续需引入 `go.mod` 并添加 MySQL/Redis 驱动以支持纯 Go 构建（`CGO_ENABLED=0`）。

## 4. 约定与约束
- **包名命名空间**：API 与 proto 包使用 `@xqecz/` 私有 scope（`@xqecz/api`、`@xqecz/proto`），便于未来发布至私有 registry。
- **引擎版本约束**：前端 `package.json` 通过 `engines.node` 限定 Node 版本为 `^20.19.0 || >=22.12.0`，保证开发/构建环境一致性。
- **Husky 钩子**：前端子包通过 `prepare` 脚本自动初始化 husky，配合 `lint-staged` 对 `.vue/.ts/.js/.css` 执行 eslint 修复。
- **Proto 生成流程**：`pnpm run proto:generate` 在根脚本中调用 `@xqecz/proto` 包的 `generate` 脚本，同时生成 TS 与 Go 代码并输出到 `gen/ts` 与 `gen/go`。
- **禁止未授权构建**：`pnpm-workspace.yaml` 的 `onlyBuiltDependencies` 列表是硬性约束，不在白名单内的包不会执行 `prebuild`/`install` 等生命周期脚本。
- **Go vendor 被忽略**：`.gitignore` 明确排除 `packages/worker/vendor/`，说明项目不采用 vendoring 策略，而是依赖 go.sum 锁定。

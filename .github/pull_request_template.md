## 目的
<!-- 这个 PR 解决什么问题 / 关联 issue（如 #123） -->

## 改动概要
<!-- 关键文件与逻辑，1-3 句 -->

## 验证方式
<!-- 本地命令 / 手动步骤 / 测试覆盖 -->
- [ ] `pnpm -r typecheck` 通过
- [ ] `packages/api` build / `go build ./cmd/server/` / 前端 build 通过
- [ ] 对应单测通过（`go test ./...` / `pnpm --filter ... test`）

## 风险与降级
<!-- 影响范围、回滚方案、外部依赖降级情况（Tinify/S3/Worker） -->

## 自查（对照 docs/code-review.md）
- [ ] 未破坏前端契约（`packages/frontend/src/api/index.ts` + `src/types/schemas.ts`）
- [ ] 无密钥 / `.env` / 大二进制入库
- [ ] 业务删除走软删除（未物理删）
- [ ] 外部依赖缺失有降级，gRPC 未返 rpc error

## 审查关注点
<!-- 提示 Reviewer 重点看哪里：架构红线 / 安全 / 性能 -->

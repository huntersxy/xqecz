## 目的
<!-- 这个 PR 解决什么问题 / 关联 issue（如 #123） -->

## 改动概要
<!-- 关键文件与逻辑，1-3 句 -->

## 验证方式
<!-- 本地命令 / 手动步骤 / 测试覆盖 -->
- [ ] `go vet ./...` + `go test ./...`（packages/server）通过
- [ ] `pnpm --filter ./packages/frontend run type-check` 通过
- [ ] 涉及接口改动时，跑过端到端验收：`pnpm accept`（需先起后端，脚本自带清理）
- [ ] 涉及响应字段改动时，跑过契约冒烟：`XQECZ_LIVE_API=http://localhost:5173/api pnpm --filter ./packages/frontend exec vitest run src/api/__tests__/contract.live.test.ts`

## 风险与降级
<!-- 影响范围、回滚方案、外部依赖降级情况（ffmpeg / Tinify / Redis） -->

## 自查
- [ ] 未破坏前端契约（`packages/frontend/src/api/index.ts` + `src/types/schemas.ts`）
- [ ] 无密钥 / `.env` / 大二进制入库
- [ ] 业务删除走软删除（未物理删）
- [ ] 外部依赖缺失有降级（ffmpeg / Tinify / Redis 不可用时不影响主流程）

## 审查关注点
<!-- 提示 Reviewer 重点看哪里：架构红线 / 安全 / 性能 -->

## 目的
<!-- 这个 PR 解决什么问题 / 关联 issue（如 #123） -->

## 改动概要
<!-- 关键文件与逻辑，1-3 句 -->

## 验证方式
<!-- 写明实际执行结果；不适用项标 N/A。区分开发页、生产构建预览、线上，不将模拟测试写成浏览器验收。 -->
- [ ] `go vet ./...` + `go test ./...`（packages/server）通过
- [ ] `pnpm --filter ./packages/frontend run type-check` 通过
- [ ] 涉及前端共享行为时 `pnpm fe:test` 通过，回归用例覆盖原故障条件
- [ ] 涉及媒体/构建配置时，非空媒体基址的生产构建通过（见 `docs/frontend-media.md`）
- [ ] 涉及交互时，记录真实操作结果：打开/关闭/返回、中途取消，以及受影响的主题/移动端/减少动效路径
- [ ] 涉及接口改动时，跑过端到端验收：`pnpm accept`（需先起后端，脚本自带清理）
- [ ] 涉及响应字段改动时，跑过契约冒烟：`XQECZ_LIVE_API=http://localhost:5173/api pnpm --filter ./packages/frontend exec vitest run src/api/__tests__/contract.live.test.ts`

## 风险与降级
<!-- 影响范围、回滚方案、外部依赖降级情况（ffmpeg / Tinify / Redis） -->

## 自查
- [ ] 未破坏前端契约（`packages/frontend/src/api/index.ts` + `src/types/schemas.ts`）
- [ ] 无密钥 / `.env` / 大二进制入库
- [ ] 业务删除走物理删除（Delete() 真删，无 deleted_at；媒体交 removeOrphanMedia 按引用计数入 data/bin）
- [ ] 外部依赖缺失有降级（ffmpeg / Tinify / Redis 不可用时不影响主流程）
- [ ] 共享规则复用已有入口，新增资源有取消/释放路径，装饰失败不阻断主流程
- [ ] 按根 `AGENTS.md` 的迭代规范同步相关约束和注释；素材/媒体变更核对 `docs/frontend-media.md`

## 审查关注点
<!-- 提示 Reviewer 重点看哪里：架构红线 / 安全 / 性能 -->

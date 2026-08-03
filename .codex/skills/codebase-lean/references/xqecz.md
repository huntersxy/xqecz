# xqecz 速查（仅在本仓库改动时加载）

## 契约
- 前端契约：`packages/frontend/src/api/index.ts` 是唯一接口定义，改接口必须同步。
- gRPC：`proto/xqecz.proto`（snake_case）→ `pnpm --filter @xqecz/proto run generate` → worker stub + `packages/api/src/worker/worker.service.ts`。
- 统一响应 `{ code, message, data }`；content.type 仅 image/text；删除一律软删除（deleted_at）。

## 验证命令
- API：`pnpm --filter ./packages/api run typecheck`
- 前端：`pnpm --filter ./packages/frontend run type-check`、`pnpm --filter ./packages/frontend run build`
- Worker：`cd packages/worker && go test ./...`
- 全量（可选）：`pnpm build`

## 尺寸热点（2026-08-03 多轮瘦身后，scripts/repo_stats.py 口径）
- 全仓 132 文件 ≈ 16,617 行｜frontend 71 文件 ≈ 12,202 行（73%）｜api 50 文件 ≈ 3,092 行｜worker 8 文件 ≈ 939 行｜proto 94 行｜scripts 290 行。
- 最大手写文件：AdminApiKeysDocs.vue(1080)、App.vue(666)、content.service.ts(545)、AdminDashboard.vue(491)、api/index.ts(462)、HomeView.vue(457)。
- 已完成瘦身：api 未使用 import/DTO/常量、前端死文件 asyncComponent、webVitals 未用导出、v-viewer 依赖、api/index.ts 死方法（changePassword/vote）、admin store 包装函数与分页加载合并、AdminApiKeysDocs 端点卡片数据化、AdminDashboard 分布卡片合并、useFilePicker 提取（三处上传共用）、CSS 同体规则合并、滚动/生命周期调试日志清理、content.service 批量用户/装饰合并、content.controller 上传文件准备合并、comment.service 评论 DTO 映射合并、utils 死函数 formatRelativeTime（含 dayjs relativeTime 插件与测试）。
- 剩余候选：大文件 SCSS/CSS 语义去重（需视觉验证）、HomeView 瀑布流缓存逻辑简化、AdminContentDrawer（Arco 上传变体）。
- 优先瘦身候选：前端 admin 组件族（重复表格/表单/抽屉）、重复请求封装、超长模板组件抽子组件。

## 行为红线
- 保留软删除与降级分支（外部依赖缺失 → success=false 降级，不抛错）。
- Redis 推荐 ZSet `xqecz:recommend:hot`、分布式锁语义不变；bigint 主键与 ZSet 成员比对需 String() 归一化。

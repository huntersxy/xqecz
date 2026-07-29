# 前端改进建议清单（packages/frontend）

> 基于 `tree.md` 文件树分析后的代码库改进建议。每条附**实测证据**与建议动作——核实方式为 `git grep`、构建输出、行数统计，非猜测。
> 生成日期：2026-07-29。本文档只记录建议，未执行任何改动。

---

## 🔴 高优先级（死代码 / 冗余依赖，低风险高收益）

### 1. ✅ 移除未使用的 FormKit
- **已完成**：`main.ts` 两行注册已删除，`@formkit/vue` 依赖已卸载。type-check + 52/52 test + 构建全绿。
- **原问题**：`src/main.ts` 第 5 行 `import { plugin as FormKitPlugin, defaultConfig } from '@formkit/vue'`、第 17 行 `app.use(FormKitPlugin, defaultConfig())` 全局注册；但全项目 grep 无任何 `<FormKit>` 模板使用或二次 import——整个表单库被打进 bundle 却零使用。

### 2. ✅ 清理死资源
- **已完成**：`package-lock.json` 已删除；`src/assets/qrcode.webp` 已删除。type-check + 52/52 test + 构建全绿。
- **原问题**：
  - `package-lock.json`：`git ls-files` 确认未被 git 跟踪，是 pnpm 仓库中的 npm 残留噪音（仓库锁文件为根目录 `pnpm-lock.yaml`）。
  - `src/assets/qrcode.webp`：src 中零引用（grep `qrcode` 无结果）。

### 3. ✅ 统一 Toast 体系（当前两套并存）
- **已完成**：4 个文件（`api/index.ts`、`stores/admin.ts`、`components/admin/AdminApiKeys.vue`、`views/QuickUploadView.vue`）全部从 antd `message` 迁移到 `@/composables/useToast` 的 `toast`。type-check + 52/52 test 全绿。
- **原问题**：`src/api/index.ts` 用 antd 的 `message.error/success`；而 `App.vue`（挂 `<Toaster>`）、`composables/useToast.ts`、`components/ConfirmDialog.vue`、`views/ContentDetailView.vue` 用 vue-sonner。两套提示 UI 风格、位置、动画均不一致。

### 4. ✅ 统一确认弹窗（当前两套并存）
- **已完成**：5 个组件（AdminUserTable ×3、AdminApiKeys、AdminContentDrawer、AdminPollPanel、AdminReportTable）的 `Modal.confirm` 全部迁移到 `useConfirm` + `ConfirmDialog.vue`。`stores/admin.ts:29` 按用户选择保留。type-check + 52/52 test 全绿。
- **原问题**：项目已有 `useConfirm` + `ConfirmDialog.vue`（Teleport 全局确认）抽象，但仍有 **8 处**直接调用 antd `Modal.confirm`。

---

## 🟡 中优先级（架构一致性 / 体积）

### 5. ✅ 搜索关键字有 3 个真相源
- **已完成**：`searchKeyword` 单一真相源统一到 `home store`。`useGlobalSearch.ts` 改为读写 store 的 computed；`useSearchFilter.ts` 的 `searchKeyword` 改为 computed 读写 store；`HomeView.vue` 直接读 `homeStore.searchKeyword`。type-check + 52/52 test 全绿。
- **原问题**：`searchKeyword` 同时存在于 `stores/home.ts:6`（store 持久化）、`composables/useGlobalSearch.ts:4`（模块级单例 ref，App 写入 / HomeView 读取）、`composables/useSearchFilter.ts:16`（本地 ref，初始化时从 home store 拷贝）。三份状态靠手动同步，容易出现"搜索框、请求参数、恢复现场"三者不一致。

### 6. ✅ Antd 全量注册导致 bundle 超标
- **已完成**：`main.ts` 移除 `app.use(Antd)`，改用 `unplugin-vue-components` 的 `AntDesignVueResolver` 按需引入（`importStyle: false` + 手动引入 `ant-design-vue/dist/reset.css`）。构建产物 `antd-vendor` 从 **1.6MB → 44.80kB（gzip 14.38KB）**。type-check + 52/52 test + 构建全绿。
- **原问题**：`main.ts:19` `app.use(Antd)` 全量引入 → 构建产物 `vue-vendor` 块 **1.6MB（gzip 495KB）**，Vite 已发出 >500KB 告警。

### 7. ✅ vue-query 仅一处使用
- **已完成**：卸载 `@tanstack/vue-query` 依赖 + 移除 `main.ts` 的 `VueQueryPlugin`；`useRecommendLoader.ts` 改为裸 `async/await` + `ref(false)` 跟踪 isLoading。type-check + 52/52 test + 构建全绿。
- **原问题**：`useQuery` 仅出现在 `composables/useRecommendLoader.ts`；其余数据流均为裸 `api/index.ts` + Pinia。为一个推荐加载引入整套 @tanstack/vue-query（含全局插件注册）性价比低。

### 8. ✅ 清理 ContentType 兼容死分支
- **已完成**：`schemas.ts` 的 `contentTypeValues` 收窄为 `['image', 'text']`；`AdminContentTable.vue` 移除 video/link Tag 样式 + 视频刷新封面按钮；`HomeView.vue` 移除 video 播放按钮 + link 徽标 + 对应 CSS；`AdminClaimTable.vue` 简化 `!== 'link'` 条件；`AdminContentDrawer.vue` 保留 video/link 展示分支作为历史脏数据兜底（用 `drawerTypeStr` computed + `as string` 避免类型错误）。type-check + 52/52 test + 构建全绿。
- **原问题**：数据已收窄为 `image`/`text`（后端迁移脚本已归并 `video`/`link`），但：
  - `types/schemas.ts` 的 `ContentType` 仍含 `video`/`link`；
  - 多个组件模板残留 `item.type === 'video'` / `item.type === 'link'` 等死分支。
- **风险**：中；依赖“历史脏数据已全部迁移”这一前提。AdminContentDrawer 保留展示兜底。

---

## 🟢 低优先级（可维护性）

### 9. ✅ 拆分巨型组件
- **已完成**：从 `ContentDetailView.vue` 抽出 3 个独立组件：`CommentSections.vue`（评论列表 + 输入 + 分页 + 菜单遮罩）、`ReportModal.vue`（举报弹窗）、`ClaimModal.vue`（认领弹窗）。ContentDetailView 从 **1022 行 → 645 行**（减少 37%）。type-check + 52/52 test + 构建全绿。
- **原问题**：`views/ContentDetailView.vue` **1022 行**（详情 + 评论 + 举报弹窗 + 认领弹窗 + viewerjs 揉在一起）；`views/HomeView.vue` **615 行**（已从原 `WaterfallTheme.vue` 并入）。

### 10. ✅ 命名残留：已修复
- **已完成**：`WaterfallTheme.vue` 内容已并入 `HomeView.vue`，旧文件删除，`tree.md` 与 `useGlobalSearch.ts` 注释同步更新。
- **原问题**：多主题系统已删除，`HomeView.vue` 只是空壳包一层 `<WaterfallTheme />`，"Theme"命名成为遗留概念。

### 11. 测试覆盖偏薄
- **证据**：仅 5 个测试文件（ConfirmDialog / useSearchFilter / user store / home store / utils）。`stores/admin.ts`、`api/index.ts`（含 401 静默逻辑）、两个巨型 view 均无测试。
- **动作**：优先给关键路径补测——api 层 401 静默与错误 toast、admin store 的审核/删除流程。
- **风险**：无（只增不改）。

### 12. ✅ dist 产物累积
- **已完成**：`vite.config.ts` 恢复 `emptyOutDir: true`，构建前整体清空 dist。构建验证：21 个文件，0 个旧公告残留。
- **原问题**：`vite.config.ts:75` `emptyOutDir: false`（因平台 safe-delete 拦截批量清空）导致旧 chunk 堆积——此前"公告弹窗残留"事故的根因即为过期 chunk 未清理。

---

## 建议执行顺序

| 批次 | 条目 | 说明 |
|------|------|------|
| 第一批（立即，零风险） | #1 FormKit ✅、#2 死资源 ✅、#10 命名 ✅ | 删除/更名即可，无行为变化 |
| 第二批（需回归测试） | #3 Toast 统一 ✅、#4 确认弹窗统一 ✅、#5 搜索单一来源 ✅ | 涉及交互，改完跑 lint/test + 手动回归 |
| 第三批（需构建验证） | #6 Antd 按需 ✅、#7 vue-query 取舍 ✅、#12 dist 清理 ✅ | 改完核对 bundle 体积与产物 |
| 第四批（按需排期） | #8 类型收窄 ✅、#9 组件拆分 ✅、#11 补测 ⏸️ | 工作量较大，可拆独立任务。#11 用户选择暂缓 |

# 小泉动漫二创站 — 前端

xqecz monorepo 的前端子包（`packages/frontend`）：Vue 3 + TypeScript + Vite 单页应用。首页为瀑布流信息流，内容详情为全屏覆盖式，支持日间 / 暗色切换。

![Vue](https://img.shields.io/badge/Vue-3.5-4fc08d?logo=vuedotjs)
![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178c6?logo=typescript)
![Vite](https://img.shields.io/badge/Vite-8.0-646cff?logo=vite)
![Tailwind CSS](https://img.shields.io/badge/Tailwind-4.3-06b6d4?logo=tailwindcss)
![License](https://img.shields.io/badge/License-GPLv3-blue)

> **前端是契约** — `src/api/index.ts` 是全站唯一接口定义，后端（Go 单体进程）按此实现；`src/types/schemas.ts` 用 zod 校验响应形状。改接口先改这里。

## 页面

| 路径 | 视图 | 说明 |
| ---- | ---- | ---- |
| `/` | `HomeView` | 瀑布流首页：推荐区 + 搜索 / 标签筛选 / 分页，滚动位置与布局缓存 |
| `/content/:id` | `ContentDetailView` | 全屏覆盖式详情：媒体、正文、评论区、操作栏 |
| `/quick-upload` | `QuickUploadView` | 游客快速上传（仅图片，留邮箱 + 昵称标识身份） |
| `/login` | `LoginView` | 登录 / 注册 |
| `/admin` | `AdminView` | 后台管理：上传对登录用户开放，审核 / 用户 / 举报 / 认领 / API 密钥需管理员 |

路由为 **hash 模式**，懒加载 + 守卫 + 空闲预加载。

## 目录结构

```
src/
├── api/              # HTTP 层：全站接口定义 + 统一错误通道；__tests__ 含契约冒烟
├── components/       # 通用组件
│   ├── WaterfallCard.vue       # 瀑布流卡片（ResizeObserver 驱动重排）
│   ├── ContentMedia.vue / MediaImage.vue / ContentActions.vue / ContentSidebar.vue
│   ├── CommentItem.vue / CommentSections.vue
│   ├── ClaimModal.vue / ReportModal.vue / ConfirmDialog.vue / ErrorBoundary.vue
│   ├── MarkdownEditor.vue / QuickUploadSheet.vue / RecommendSection.vue
│   └── admin/                  # 后台专用组件
├── composables/      # 组合式函数
│   ├── useWaterfallLayout.ts   # 瀑布流布局纯算法
│   ├── useContentBrowse.ts     # 浏览链路编排
│   ├── useGlobalSearch.ts / useSearchFilter.ts / useListCache.ts
│   ├── useRecommendLoader.ts / useFilePicker.ts / useToast.ts
├── stores/           # Pinia：home（筛选 / 分页 / 滚动位置）、user、admin、theme（明暗）
├── views/            # 路由页（薄层，只组装组件）
├── router/           # 路由表 + 守卫 + 预加载
├── types/            # 类型定义 + zod schema
├── utils/            # 图片 URL / 时间 / Markdown 渲染工具、Web Vitals
└── assets/           # 背景图、Logo、Tailwind 入口 CSS
```

## 技术栈

| 技术 | 说明 |
| ---- | ---- |
| Vue 3.5 | Composition API + `<script setup>`，unplugin 自动导入 |
| TypeScript 6 | strict 模式 |
| Vite 8 | 开发 / 构建；dev/preview 代理、图片优化插件 |
| Pinia 3 | Setup Store 语法 |
| Tailwind CSS 4.3 | 原子类优先，`@theme` 自定义变量 |
| vue-router 5 | hash 模式，懒加载 + 守卫 |
| ofetch | HTTP 客户端，统一超时与错误通道 |
| zod 4 | API 响应运行时校验 |
| Arco Design Vue | UI 组件（按需自动注册） |
| marked + DOMPurify + highlight.js | Markdown 渲染与 XSS 防护 |
| vditor / viewerjs / vue-sonner | 编辑器 / 图片查看 / Toast |

## 快速开始

本包不单独克隆使用，随 monorepo 根目录统一编排（Node ≥ 20 + pnpm）：

```bash
# 在 monorepo 根目录
pnpm install                       # 首次安装依赖

pnpm dev                           # 起 Go 后端(:3000) + 前端(:5173)
pnpm dev:fe                        # 仅启动前端 Vite 开发服务器（:5173）

# 仅操作本包（--filter 包名为 xiaoquanweb）
pnpm --filter xiaoquanweb run build        # 生产构建（type-check + vite build）
pnpm --filter xiaoquanweb run type-check   # TypeScript 类型检查
pnpm --filter xiaoquanweb run test         # vitest 单测
pnpm --filter xiaoquanweb run lint         # oxlint + eslint
pnpm --filter xiaoquanweb run format       # prettier 格式化
```

开发 / 预览服务器已将 `/api`、`/uploads` 等代理到 `http://localhost:3000`（Go 后端），可用环境变量 `VITE_PROXY_TARGET` 覆盖目标。

契约冒烟（用 zod schema 校验运行中的后端响应）：

```bash
XQECZ_LIVE_API=http://localhost:5173/api pnpm --filter xiaoquanweb exec vitest run src/api/__tests__/contract.live.test.ts
```

## 环境变量

配置统一放在 **monorepo 根目录的 `.env`**（Vite 的 `envDir` 指向上两级仓库根），本包内不单独放 env 文件：

```bash
VITE_API_BASE_URL=/api    # 接口基础地址（相对路径，经 Vite 代理或同域反代）
VITE_MEDIA_BASE_URL=      # 媒体资源基础地址（留空即同源）

# 仅本地开发按需覆盖（shell 环境变量，不写入 .env）
VITE_PROXY_TARGET=http://localhost:3000   # Vite 代理目标（默认即此值）
```

## 开发规范（摘要）

- 组件统一用 `<script setup lang="ts">`，Props 用 `defineProps<Props>()`
- Tailwind 原子类优先；必要样式加 `scoped`
- 统一 `@/` 别名导入，禁止相对路径穿透
- **禁止在组件里直接 `fetch`**，一律经 `@/api`
- View 只渲染、Composable 管逻辑、Store 管全局状态

完整约定（数据流、组件职责、测试与提交要求）见 **[AGENTS.md](./AGENTS.md)**。

## 许可证

本项目采用 **GNU General Public License v3.0**，详见 [LICENSE](./LICENSE)。站内用户上传内容遵循 CC BY-NC 4.0（非商业使用）。

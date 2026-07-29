# 小泉动漫二创站 — 前端

xqecz monorepo 的前端子包（`packages/frontend`），基于 Vue 3 + TypeScript + Vite 的动漫二创内容分享平台，支持多主题运行时切换与响应式布局。

![Vue](https://img.shields.io/badge/Vue-3.5-4fc08d?logo=vuedotjs)
![TypeScript](https://img.shields.io/badge/TypeScript-6.0-3178c6?logo=typescript)
![Vite](https://img.shields.io/badge/Vite-8.0-646cff?logo=vite)
![Tailwind CSS](https://img.shields.io/badge/Tailwind-4.3-06b6d4?logo=tailwindcss)
![License](https://img.shields.io/badge/License-GPLv3-blue)

> **前端是契约** — `src/api/index.ts` 是全站唯一接口定义，后端（NestJS API）按此实现。改接口先改这里。

## 特性

- **三主题系统** — 默认 macOS 风格 / Bilibili 大屏网格 / 瀑布流，运行时动态切换
- **日间 / 暗色模式** — 每个主题内置 light / dark 两套配色，CSS 变量即时注入
- **响应式设计** — 移动端优先，适配手机 / 平板 / 桌面
- **用户认证** — 登录注册、会话持久化、管理员权限
- **内容浏览** — 支持视频、图片、文字、链接四种类型，标签筛选 + 关键词搜索
- **游客快速上传** — 免登录上传图片，留邮箱 + 昵称标识身份（顶部「快速上传」入口）
- **推荐系统** — 首页推荐位（后端 Redis ZSet + Go Worker 打分），可刷新
- **投票互动** — 内置投票组件
- **后台管理** — 内容上传 / 审核、用户管理、举报处理、认领审批、API 密钥（需相应权限）
- **性能监控** — Web Vitals 跟踪（LCP / FID / CLS / TTFB / FCP）
- **可访问性** — WCAG 2.1 AA 级 ARIA 标注、键盘导航
- **安全渲染** — DOMPurify XSS 防护、marked + DOMPurify 安全 Markdown

## 技术栈

| 技术 | 说明 |
| ---- | ---- |
| Vue 3.5 | Composition API + `<script setup>`，unplugin 自动导入 |
| TypeScript 6.0 | strict 模式，全量类型覆盖 |
| Vite 8.0 | 开发 / 构建，vue-devtools / 图片优化插件 |
| Pinia 3.0 | Setup Store 语法状态管理 |
| Tailwind CSS 4.3 | 原子类优先，`@theme` 自定义变量 |
| Vue Router 5.0 | hash 模式，懒加载 + 路由守卫 + 空闲预加载 |
| ofetch | HTTP 客户端，自动重试 / 超时 / 统一错误通道 |
| zod | API 响应运行时校验（`types/schemas.ts`） |
| marked + DOMPurify | Markdown 渲染 + XSS 防护 |
| @tanstack/vue-query | 数据请求与竞态处理 |
| motion-v | 声明式动画 |
| ant-design-vue | UI 组件（自动按需注册） |

## 快速开始

本包不独立克隆使用，随 monorepo 根目录统一编排（Node ≥ 20 + pnpm）：

```bash
# 在 monorepo 根目录（D:\xqecz）
pnpm install --shamefully-hoist   # 首次安装依赖

pnpm dev          # 一键起三端：API(:3000) + Worker(:50051) + 前端(:5173)
pnpm dev:fe       # 仅启动前端 Vite 开发服务器（:5173）

# 仅操作本包（--filter 包名为 xiaoquanweb）
pnpm --filter xiaoquanweb run build        # 生产构建（type-check + vite build）
pnpm --filter xiaoquanweb run type-check   # TypeScript 类型检查
pnpm --filter xiaoquanweb run test         # vitest 单测
pnpm --filter xiaoquanweb run lint         # oxlint + eslint
pnpm --filter xiaoquanweb run format       # prettier 格式化
```

开发 / 预览服务器已将 `/api`、`/uploads` 等代理到 `http://localhost:3000`（NestJS API），可用环境变量 `VITE_PROXY_TARGET` 覆盖目标。

## 项目结构

```
src/
├── api/              # HTTP 请求层 — 全站接口契约（authApi / contentApi / commentApi / pollApi / adminApi / apiKeyApi）
├── assets/           # 静态资源（Logo、背景图、Tailwind 入口 CSS）
├── themes/           # 主题文件，自动扫描注册
│   ├── DefaultTheme.vue       # macOS 卡片风格
│   ├── BilibiliStyleTheme.vue # 大屏网格布局
│   └── WaterfallTheme.vue     # 瀑布流布局
├── components/       # 通用 UI 组件
│   ├── admin/        # 后台管理专用组件（上传 / 审核 / 用户 / 举报 / 认领 / API 密钥）
│   ├── ErrorBoundary.vue      # 错误边界
│   ├── MarkdownModal.vue      # 公告弹窗
│   ├── PollComponent.vue      # 投票
│   ├── ConfirmDialog.vue      # 全局确认对话框
│   └── HomeContentCard.vue    # 内容卡片
├── composables/      # 组合式函数
│   ├── useThemeRegistry.ts    # 主题自动注册 + applyThemeColors
│   ├── useSearchFilter.ts     # 搜索 / 标签筛选 + localStorage 缓存
│   ├── useContentLoader.ts    # 内容加载 + @tanstack/vue-query
│   ├── useRecommendLoader.ts  # 推荐内容加载
│   ├── useHomeLogic.ts        # 首页业务编排
│   ├── useMarkdownEditor.ts   # Markdown 编辑器逻辑
│   └── useToast.ts            # Toast + Confirm 全局单例
├── router/
│   └── index.ts      # 路由表 + 导航守卫 + 空闲预加载
├── stores/           # Pinia 全局状态
│   ├── theme.ts      # 主题切换（currentTheme + mode）
│   ├── home.ts       # 首页状态缓存（搜索 / 分页 / 滚动位置）
│   ├── user.ts       # 登录态 / 用户信息
│   └── admin.ts      # 后台管理状态（上传 / 审核 / 列表）
├── types/
│   ├── index.ts      # 全局 TypeScript 类型定义（请求 / 响应契约）
│   └── schemas.ts    # zod 运行时校验 schema（Content / User / Comment 等）
├── utils/
│   ├── index.ts             # toFormData / 图片 URL / 时间 / Markdown 渲染工具
│   ├── constants.ts         # 共享常量（CC 协议文本等）
│   ├── asyncComponent.ts    # 异步组件封装（加载态 / 错误态）
│   └── webVitals.ts         # Web Vitals 性能监控
├── views/            # 页面级组件（薄路由层）
├── App.vue           # 根组件（导航 / 页脚 / Toast / Confirm 容器）
└── main.ts           # 入口（createApp → Pinia → Router → mount）
```

## 页面路由

| 路径 | 说明 | 权限 |
| ---- | ---- | ---- |
| `/` | 首页 — 推荐内容 + 搜索 + 标签筛选 | 公开 |
| `/quick-upload` | 游客快速上传（仅图片，留邮箱 + 昵称） | 公开 |
| `/content/:id` | 内容详情 | 公开 |
| `/login` | 登录 / 注册 | 公开 |
| `/easter-egg` | 彩蛋空间 | 公开 |
| `/theme` | 主题设置 | 公开 |
| `/admin` | 后台管理（我的内容 / 上传对任意登录用户开放；审核 / 用户 / 举报等仅管理员） | 登录用户 |

## 主题系统

主题系统通过 `import.meta.glob` 自动扫描 `src/themes/` 下的 `.vue` 文件，无须手动注册。

### 内置主题

| 主题 | key | 说明 |
| ---- | --- | ---- |
| 默认主题 | `default` | macOS 风格卡片布局，窗口圆点标题栏 |
| Bilibili 风格 | `bilibiliStyle` | 大屏网格布局，B 站蓝粉配色，无限滚动 |
| 瀑布流 | `waterfall` | 多列瀑布流布局 |

### 创建新主题

参考 [theme.md](./theme.md) 获取完整主题开发指南。核心步骤：

1. 在 `src/themes/` 创建 `XxxTheme.vue`
2. `<script lang="ts">` 导出 `themeMeta`（key / name / colors 等）
3. `<script setup>` 中实现布局和数据加载
4. 保存即生效，系统自动注册

### 日间 / 暗色切换

每个主题的 `themeMeta.colors` 包含 `light` 和 `dark` 两套配色。`applyThemeColors()` 运行时注入 CSS 变量，`ConfigProvider` + `darkAlgorithm` 自动处理 antd 组件暗色适配。

## 数据流

```
View（薄层，组装组件）
  └── Composable（业务逻辑，调用 API + Store）
        ├── Store（Pinia，跨组件共享状态）
        └── API（ofetch / XHR 上传，HTTP 通信）
              └── Type（types/index.ts + zod schema，请求 / 响应约束）
```

- View 只管渲染，不直接调 API
- Composable 管逻辑，拥有本地状态
- Store 管全局状态（theme / home / user / admin 四个领域）
- 后端统一响应 `{ code, message, data }`，业务错误经统一 `ApiError` 通道透出

## 开发规范

- `<script setup lang="ts">` — 所有组件统一语法（`ref` / `computed` 等由 unplugin 自动导入）
- Props 用纯类型 `defineProps<Props>()`
- Tailwind 原子类优先，非必要不写 CSS
- 样式必须加 `scoped`
- 使用 `@/` 别名导入，禁止相对路径
- API 调用必须通过 `@/api` 模块，禁止组件内直接调 `fetch`
- 所有 async 函数用 try / catch 包裹

更多约定见 [AGENTS.md](./AGENTS.md)。

## 环境变量

```bash
# .env.development / .env.production（均已提交，默认走同源代理）
VITE_API_BASE_URL=/api    # API 基础地址（相对路径，经 Vite 代理或同域反代）
VITE_MEDIA_BASE_URL=      # 媒体资源基础地址（留空即同源）

# 仅本地开发按需覆盖（shell 环境变量）
VITE_PROXY_TARGET=http://localhost:3000   # Vite 代理目标（默认即此值）
```

## 许可证

本项目采用 **GNU General Public License v3.0** 许可证，详见 [LICENSE](./LICENSE)。站内用户上传内容遵循 CC BY-NC 4.0（非商业使用）。

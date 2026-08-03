# 前端文件树分析（packages/frontend）

> Vue 3 `<script setup>` + Composition API + Pinia + vue-router（hash 模式）+ Vite。
> 本文档标注**每个文件的作用**，便于快速定位。自动生成文件与构建产物（`dist/`、`node_modules/`）不在此列。

---

## 一、完整文件树（带作用标注）

```
packages/frontend/
├── index.html                     # SPA 入口 HTML，挂载 #app、引入 main.ts
├── package.json                   # 依赖清单 + 脚本（dev/build/test/lint/type-check）
├── package-lock.json              # npm 锁文件（仓库用 pnpm，此文件为历史遗留）
├── vite.config.ts                 # Vite 配置：插件、别名 @→src、构建分块、dev/preview 代理、图片优化
├── vitest.config.ts               # 单元测试配置（jsdom 环境 + setupFiles）
├── tsconfig.json                  # TS 根配置（引用 app/node 两个子配置）
├── tsconfig.app.json              # 应用侧 TS 配置（src 源码）
├── tsconfig.node.json             # 构建脚本侧 TS 配置（vite/vitest 等 node 环境）
├── eslint.config.ts               # ESLint 扁平配置（第二道 lint）
├── .oxlintrc.json                 # oxlint 配置（第一道快速 lint）
├── .prettierrc.json               # Prettier 格式化规则
├── .editorconfig                  # 编辑器通用格式（缩进/换行/编码）
├── .gitignore / .gitattributes    # Git 忽略规则 / 属性（换行、二进制标记）
├── .eslintcache                   # ESLint 缓存（自动生成，可删）
├── env.d.ts                       # Vite 客户端环境类型声明（import.meta.env 等）
├── build.bat / run.bat            # Windows 批处理：构建 / 启动脚本
├── README.md                      # 前端说明文档
├── AGENTS.md                      # 面向 AI/协作者的前端约定
├── LICENSE                        # 许可证
│
├── public/                        # 原样拷贝到 dist 根的静态资源
│   ├── favicon.webp               # 站点图标
│   ├── manifest.json              # PWA 清单
│   └── icons/*.svg                # UI 图标（home/user/search/menu/close 等 17 个）
│
└── src/
    ├── main.ts                    # 应用入口：createApp + Pinia + Router + Antd + VueQuery + FormKit，initWebVitals
    ├── App.vue                    # 根组件：整体布局、顶部 header、移动端菜单、全局搜索框、路由出口、访客 checkAuth
    ├── auto-imports.d.ts          # 【自动生成】unplugin-auto-import：vue/vue-router API 全局类型
    ├── components.d.ts            # 【自动生成】unplugin-vue-components：组件全局类型
    ├── test-setup.ts              # Vitest 全局 setup：mock 静态 svg 资源，防 file URL 报错
    │
    ├── api/
    │   └── index.ts               # HTTP 层：ofetch 实例 + 401 统一拦截 + 6 组接口
    │                              #   authApi / contentApi / commentApi / pollApi / adminApi / apiKeyApi
    │
    ├── assets/
    │   ├── main.css               # 全局样式入口：Tailwind + :root/html.dark 主题 CSS 变量
    │   ├── logo.webp              # 站点 Logo
    │   ├── bg.webp                # 背景图
    │   └── qrcode.webp            # 二维码图片
    │
    ├── router/
    │   └── index.ts               # 路由表（hash 模式，5 条）+ beforeEach 鉴权守卫
    │                              #   /、/quick-upload、/login、/content/:id、/admin
    │
    ├── stores/                    # Pinia（Setup Store 写法）
    │   ├── user.ts                # 用户会话：login/logout/checkAuth/getMe、isLoggedIn/needsEmail
    │   ├── home.ts                # 首页状态持久化：搜索词/标签/类型/页码/滚动位置，返回时恢复
    │   ├── admin.ts               # 后台聚合状态：内容/用户/举报/认领/投票分页 + 相关 API 调用
    │   ├── theme.ts               # 日间/暗色模式：切 html.dark class + localStorage 记忆
    │   └── __tests__/
    │       ├── user.test.ts       # user store 单测
    │       └── home.test.ts       # home store 单测
    │
    ├── composables/               # 组合式函数
    │   ├── useToast.ts            # vue-sonner 封装 toast + useConfirm（配合 ConfirmDialog 的全局确认）
    │   ├── useGlobalSearch.ts     # 模块级搜索关键字单例：App 写入、WaterfallTheme 监听触发
    │   ├── useSearchFilter.ts     # 搜索/标签筛选：与 home store 同步 + 300ms 防抖 + 加载标签
    │   ├── useRecommendLoader.ts  # 推荐内容加载：@tanstack/vue-query，6 页 × 16 条循环
    │   └── __tests__/
    │       └── useSearchFilter.test.ts  # useSearchFilter 单测
    │
    ├── views/                     # 路由级页面
    │   ├── HomeView.vue           # 首页主体：精选推荐区 + 瀑布流卡片 + 搜索筛选 + 无限加载
    │   ├── ContentDetailView.vue  # 内容详情路由页（全屏）：图片 viewerjs 预览、Markdown 正文、评论、举报、认领
    │   ├── LoginView.vue          # 登录 / 注册页（含邮箱补全）
    │   ├── QuickUploadView.vue    # 游客快速上传页（CC 协议 + Markdown 工具栏 + 标签云）
    │   └── AdminView.vue          # 后台管理容器：侧边菜单编排下方各 Admin* 面板/表格
    │
    ├── components/                # 通用 / 业务组件
    │   ├── ConfirmDialog.vue      # 全局确认对话框（Teleport to body，配合 useToast 的 useConfirm）
    │   ├── ErrorBoundary.vue      # 错误边界：onErrorCaptured 捕获子树异常并兜底展示
    │   ├── CommentItem.vue        # 单条评论渲染（支持嵌套回复）
    │   ├── MarkdownToolbar.vue    # Markdown 编辑工具栏：插入语法 / 触发图片上传
    │   ├── __tests__/
    │   │   └── ConfirmDialog.test.ts   # ConfirmDialog 单测
    │   └── admin/                 # 后台专用组件
    │       ├── adminColumns.ts    # 共享表格列定义（操作/状态/时间/内容/认领人/理由列 + 次要样式）
    │       ├── AdminContentTable.vue   # 内容管理表格：审核 / 编辑 / 删除 / 同步
    │       ├── AdminContentDrawer.vue  # 内容编辑抽屉：改作者 / 标签，Markdown 预览
    │       ├── AdminUserTable.vue      # 用户管理表格：封禁 / 锁定 / 权限
    │       ├── AdminReportTable.vue    # 举报处理表格
    │       ├── AdminClaimTable.vue     # 内容认领审核表格
    │       ├── AdminPollPanel.vue      # 投票管理面板
    │       ├── AdminUploadPanel.vue    # 后台上传面板（CC/视频条款 + 标签云 + Markdown 工具栏）
    │       ├── AdminApiKeys.vue        # API 密钥管理：列表 / 新建 / 展示完整 key
    │       ├── TagCloud.vue            # 标签云选择组件（可自定义标签）
    │       └── _admin.scss             # 后台组件共享样式
    │
    ├── types/                     # 类型与校验
    │   ├── schemas.ts             # zod schema + 运行时常量：ContentType / CONTENT_TYPES、User/Content/Comment/Poll/Claim 校验
    │   └── index.ts               # 类型 re-export + ApiResponse / PaginatedResponse 等接口
    │
    └── utils/                     # 工具函数
        ├── index.ts               # 通用工具：getImageUrl、renderMarkdown(marked+DOMPurify)、formatTime(dayjs)、toFormData、renameFileToMd5(SparkMD5)
        ├── constants.ts           # 文案常量：CC 授权协议、视频转链接条款
        ├── webVitals.ts           # Web Vitals 性能监控（LCP/FID/CLS/TTFB）
        └── __tests__/
            └── index.test.ts      # utils 单测（含 renderMarkdown）
```

---

## 二、分层详解

### 1. 入口与根组件
| 文件 | 作用 |
|------|------|
| `src/main.ts` | 创建 Vue 实例，注册 Pinia、Router、Ant Design Vue、VueQuery、FormKit，引入全局 CSS 并初始化 Web Vitals。 |
| `src/App.vue` | 全站骨架：顶部导航（Logo/搜索/用户菜单/移动端汉堡菜单）、`<RouterView>` 出口。访客首次进入时触发 `checkAuth`（401 已静默，不弹提示）。 |
| `index.html` | 唯一 HTML 模板，提供 `#app` 挂载点。 |

### 2. 路由（`router/index.ts`）
- hash 模式（`createWebHashHistory`），共 **5** 条路由：`/`（首页）、`/quick-upload`（游客上传）、`/login`、`/content/:id`（详情）、`/admin`（后台，`requiresAuth`）。
- 组件通过 `createAsyncComponent` 懒加载。
- `beforeEach` 仅对 `requiresAuth` 路由调用 `checkAuth` 做登录校验。

### 3. 状态管理（`stores/`）
| Store | 职责 |
|-------|------|
| `user.ts` | 登录态核心：`login/logout/checkAuth/getMe`，暴露 `isLoggedIn`、`needsEmail`。 |
| `home.ts` | 首页浏览状态持久化（搜索/标签/类型/页码/滚动位置），从详情页返回时恢复现场。 |
| `admin.ts` | 后台数据中枢：内容、用户、举报、认领、投票的分页状态与增删改查 API 编排。 |
| `theme.ts` | 明暗模式：切换 `html.dark`/`html.light` class，localStorage 记忆。 |

### 4. 组合式函数（`composables/`）
| 文件 | 职责 |
|------|------|
| `useToast.ts` | 基于 vue-sonner 的 `toast`（success/error/warning/info）+ `useConfirm` 全局确认（与 `ConfirmDialog.vue` 联动）。 |
| `useGlobalSearch.ts` | 模块级搜索关键字单例：`App.vue` 搜索框写入，`HomeView` 监听 `searchTrigger` 触发查询。 |
| `useSearchFilter.ts` | 标签/关键字筛选逻辑，与 `home` store 同步，300ms 防抖，负责 `loadTags`。 |
| `useRecommendLoader.ts` | “精选推荐”加载，使用 vue-query 缓存，6 页 × 16 条循环换批。 |

### 5. 页面（`views/`）
| 文件 | 职责 |
|------|------|
| `HomeView.vue` | 首页主体：顶部“精选推荐”横向滚动区 + 瀑布流卡片（图片/视频/纯文字/链接）+ 搜索筛选 + 分页加载。 |
| `ContentDetailView.vue` | 内容详情全屏路由页：viewerjs 图片查看、Markdown 正文渲染、评论区、举报与认领弹窗。 |
| `LoginView.vue` | 登录/注册切换表单，含邮箱补全流程。 |
| `QuickUploadView.vue` | 游客免登录快速上传，含 CC 协议、Markdown 工具栏、标签云。 |
| `AdminView.vue` | 后台管理外壳，左侧菜单切换，编排各 `Admin*` 面板/表格。 |

### 6. 组件（`components/`）
- **通用**：`ConfirmDialog.vue`（全局确认框）、`ErrorBoundary.vue`（错误边界）、`CommentItem.vue`（评论/嵌套回复）、`MarkdownToolbar.vue`（Markdown 编辑工具栏）。
- **后台 `admin/`**：以 `adminColumns.ts` 复用表格列，包含内容表/抽屉、用户表、举报表、认领表、投票面板、上传面板、API 密钥管理、标签云，样式集中在 `_admin.scss`。

### 7. 类型与工具
| 文件 | 职责 |
|------|------|
| `types/schemas.ts` | zod 运行时校验 + `ContentType`/`CONTENT_TYPES`（新数据仅 `image`/`text`，兼容历史 `video`/`link`）。 |
| `types/index.ts` | 类型统一出口 + `ApiResponse`/`PaginatedResponse` 等通用接口。 |
| `utils/index.ts` | 图片 URL 拼接、Markdown 渲染消毒、时间格式化、FormData 构造、文件 MD5 重命名等。 |
| `utils/constants.ts` | CC 授权协议、视频转链接条款等长文案常量。 |
| `utils/webVitals.ts` | 采集 LCP/FID/CLS/TTFB 核心性能指标。 |

### 8. 测试
- 单测位于各层的 `__tests__/`：`ConfirmDialog`、`useSearchFilter`、`user`/`home` store、`utils/index`。
- `test-setup.ts` 提供 Vitest 全局 setup（mock 静态 svg），环境为 **jsdom**（见 `vitest.config.ts`）。

### 9. 自动生成 / 无需手改
`src/auto-imports.d.ts`、`src/components.d.ts`（unplugin 生成）、`env.d.ts`、`.eslintcache`。修改源码后由工具自动更新。

---

## 三、依赖流向速览

```
main.ts → App.vue → router → views → components
                        ↓         ↓         ↓
                     stores ← composables → api(index.ts) → 后端
                        ↓
                   types / utils（被各层共享）
```

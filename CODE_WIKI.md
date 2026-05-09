# 小泉动漫二创平台 - Code Wiki

## 1. 项目概述

### 1.1 项目简介

小泉动漫二创平台是一个采用 Mac 风格 UI 设计的动漫二创内容分享平台，基于 Vue 3 + TypeScript + Vite 构建。平台提供用户认证、内容管理、评论系统、举报功能和后台管理等完整功能。

### 1.2 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | ^3.5.32 | 渐进式 JavaScript 框架 |
| TypeScript | ~6.0.0 | 类型安全的 JavaScript 超集 |
| Vite | ^8.0.8 | 下一代前端构建工具 |
| Vue Router | ^5.0.4 | Vue.js 官方路由管理器 |
| Pinia | ^3.0.4 | Vue.js 轻量级状态管理 |
| Axios | ^1.16.0 | Promise HTTP 客户端 |
| Marked | ^18.0.3 | Markdown 解析器 |

## 2. 项目架构

### 2.1 目录结构

```
xiaoquanweb/
├── src/
│   ├── api/              # API 接口封装层
│   ├── assets/           # 静态资源（CSS、图片）
│   ├── components/       # 通用组件
│   ├── router/            # 路由配置
│   ├── stores/            # Pinia 状态管理
│   ├── types/             # TypeScript 类型定义
│   ├── views/             # 页面视图组件
│   ├── App.vue            # 根组件
│   └── main.ts            # 应用入口
├── public/                # 公共资源（favicon、二维码等）
├── .env.development       # 开发环境变量
├── .env.production        # 生产环境变量
├── vite.config.ts         # Vite 配置
└── package.json           # 项目依赖配置
```

### 2.2 模块职责

| 模块 | 职责 | 关键文件 |
|------|------|----------|
| API 层 | 封装所有后端 API 调用 | `src/api/index.ts` |
| 路由层 | 管理页面路由和导航守卫 | `src/router/index.ts` |
| 状态层 | 管理用户认证状态 | `src/stores/user.ts` |
| 视图层 | 页面展示和用户交互 | `src/views/*.vue` |
| 组件层 | 可复用 UI 组件 | `src/components/*.vue` |
| 类型层 | TypeScript 类型定义 | `src/types/index.ts` |

### 2.3 数据流架构

```
┌─────────────────────────────────────────────────────────┐
│                      视图层 (Views)                       │
│  HomeView | LoginView | ContentDetailView | AdminView   │
└─────────────────────┬───────────────────────────────────┘
                      │ 用户交互
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    状态层 (Stores)                        │
│                   useUserStore (Pinia)                   │
│  - user: 用户信息                                        │
│  - isLoggedIn: 登录状态                                  │
└─────────────────────┬───────────────────────────────────┘
                      │ API 调用
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    API 层 (api/index.ts)                 │
│  authApi | contentApi | commentApi | adminApi            │
└─────────────────────┬───────────────────────────────────┘
                      │ HTTP 请求
                      ▼
┌─────────────────────────────────────────────────────────┐
│                    后端 API (8080)                       │
│         https://xqapi.xiey.work (生产环境)                │
└─────────────────────────────────────────────────────────┘
```

## 3. 路由系统

### 3.1 路由配置

| 路径 | 组件 | 说明 | 权限要求 |
|------|------|------|----------|
| `/` | HomeView | 首页内容列表 | 公开 |
| `/login` | LoginView | 登录/注册页 | 公开 |
| `/content/:id` | ContentDetailView | 内容详情页 | 公开 |
| `/admin` | AdminView | 后台管理 | 需要登录 |
| `/admin/reports` | AdminReportsView | 举报管理 | 需要管理员 |
| `/easter-egg` | EasterEggView | 彩蛋空间 | 公开（隐藏入口） |

### 3.2 导航守卫

```typescript
router.beforeEach(async (to, from, next) => {
  const userStore = useUserStore()
  
  // 未登录时检查认证状态
  if (!userStore.isLoggedIn) {
    await userStore.checkAuth()
  }
  
  // 需要认证的页面检查登录状态
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else {
    next()
  }
})
```

## 4. 状态管理 (Pinia Stores)

### 4.1 useUserStore

用户状态管理，负责认证相关功能。

**状态属性：**
- `user`: User | null - 当前用户信息
- `isLoggedIn`: boolean - 登录状态
- `isLoading`: boolean - 加载状态

**核心方法：**
- `login(username, password)`: 用户登录
- `logout()`: 用户登出
- `checkAuth()`: 检查认证状态
- `setUser(userData)`: 设置用户信息

## 5. API 接口层

### 5.1 API 模块划分

| 模块 | 前缀 | 功能 |
|------|------|------|
| authApi | /auth | 用户认证 |
| contentApi | /content | 内容管理 |
| commentApi | /comment | 评论系统 |
| adminApi | /admin | 后台管理 |

### 5.2 核心 API 函数

#### authApi

| 方法 | 路径 | 说明 |
|------|------|------|
| login | POST /auth/login | 用户登录 |
| register | POST /auth/register | 用户注册 |
| logout | POST /auth/logout | 用户登出 |
| getMe | GET /auth/me | 获取当前用户 |

#### contentApi

| 方法 | 路径 | 说明 |
|------|------|------|
| getTags | GET /content/tags | 获取所有标签 |
| upload | POST /content/upload | 上传内容 |
| list | GET /content/list | 获取内容列表 |
| myList | GET /content/my | 获取我的内容 |
| detail | GET /content/:id | 获取内容详情 |
| search | GET /content/search | 搜索内容 |
| update | PUT /content/:id | 更新内容 |
| delete | DELETE /content/:id | 删除内容 |
| uploadImage | POST /content/upload-image | 上传图片 |

#### commentApi

| 方法 | 路径 | 说明 |
|------|------|------|
| add | POST /comment/add | 添加评论 |
| list | GET /comment/list/:contentId | 获取评论列表 |
| delete | DELETE /comment/:id | 删除评论 |
| count | GET /comment/count/:contentId | 获取评论数 |
| report | POST /comment/report | 举报评论 |

#### adminApi

| 方法 | 路径 | 说明 |
|------|------|------|
| audit | POST /admin/audit/:id | 审核内容 |
| pending | GET /admin/pending | 获取待审核列表 |
| getAllContent | GET /admin/content/all | 获取所有内容 |
| getUsers | GET /admin/users | 获取用户列表 |
| updateUserRole | PUT /admin/users/:id/role | 更新用户角色 |
| updateUserBan | PUT /admin/users/:id/ban | 封禁/解封用户 |
| deleteUser | DELETE /admin/users/:id | 删除用户 |
| getReports | GET /admin/comments/reports | 获取举报列表 |
| handleReport | POST /admin/comments/reports/:id/handle | 处理举报 |

## 6. 视图组件详解

### 6.1 HomeView - 首页

**功能特性：**
- 内容卡片网格展示（视频/图片/文字）
- 标签云筛选
- 内容类型筛选
- 关键词搜索
- 分页导航
- 动态随机背景加载

**核心函数：**
- `loadContents()`: 加载内容列表
- `loadTags()`: 加载标签
- `selectTag(tag)`: 标签筛选
- `selectType(type)`: 类型筛选
- `goToDetail(content)`: 跳转详情页

### 6.2 LoginView - 登录注册

**功能特性：**
- 登录/注册模式切换
- 表单验证
- 加载状态提示
- 错误消息显示

**核心函数：**
- `handleSubmit()`: 表单提交
- `toggleMode()`: 切换登录/注册模式

### 6.3 ContentDetailView - 内容详情

**功能特性：**
- Markdown 内容渲染
- 评论系统（支持回复）
- 评论举报功能
- 评论删除功能
- 动态图片 URL 处理

**核心函数：**
- `loadContent()`: 加载内容
- `loadComments()`: 加载评论
- `submitComment()`: 提交评论
- `openReport(comment)`: 打开举报弹窗

### 6.4 AdminView - 后台管理

**功能特性：**
- Tab 切换（我的内容/审核内容/所有内容/用户管理）
- Markdown 富文本编辑器
- 图片上传（支持进度显示）
- 标签管理
- 内容编辑弹窗
- 用户角色管理
- 封禁/解封用户

**核心函数：**
- `handleUpload()`: 上传内容
- `handleUpdate()`: 更新内容
- `handleAudit(id, status)`: 审核内容
- `handleUpdateUserRole/UpdateUserBan`: 用户管理
- `insertMarkdown()`: Markdown 插入

### 6.5 AdminReportsView - 举报管理

**功能特性：**
- 举报列表展示
- 举报处理
- 评论删除

**核心函数：**
- `loadReports()`: 加载举报列表
- `handleReport(reportId)`: 标记已处理
- `deleteComment()`: 删除评论

### 6.6 EasterEggView - 彩蛋空间

**功能特性：**
- 隐藏页面入口
- 群成员展示
- 社交平台链接
- QQ群二维码展示
- 模态框放大查看二维码

**核心函数：**
- `copyNumber()`: 复制群号
- `openQrModal()`: 打开二维码弹窗

## 7. TypeScript 类型定义

### 7.1 核心类型

```typescript
interface User {
  id: number
  username: string
  is_admin: boolean
  is_banned: boolean
  created_at: string
  updated_at: string
}

interface Content {
  id: number
  title: string
  type: 'video' | 'image' | 'text'
  content: string
  file_path: string
  thumb_path?: string
  user_id: number
  user: User
  tags: string[]
  audit_status: 'pending' | 'approved' | 'rejected'
  created_at: string
  updated_at: string
}

interface Comment {
  id: number
  content_id: number
  user_id: number
  text: string
  parent_id: number | null
  user?: User
  replies?: Comment[]
}

interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}
```

## 8. 依赖关系图

```
package.json
├── vue ^3.5.32
├── vue-router ^5.0.4
├── pinia ^3.0.4
├── axios ^1.16.0
├── marked ^18.0.3
└── 开发依赖
    ├── vite ^8.0.8
    ├── @vitejs/plugin-vue
    ├── typescript ~6.0.0
    ├── vue-tsc ^3.2.6
    ├── eslint ^10.2.1
    ├── prettier ^3.8.3
    └── vite-plugin-vue-devtools
```

## 9. 环境配置

### 9.1 开发环境 (.env.development)

```
VITE_API_BASE_URL=/api
```

API 请求代理到 `http://localhost:8080`

### 9.2 生产环境 (.env.production)

```
VITE_API_BASE_URL=https://xqapi.xiey.work/api
```

## 10. 项目运行

### 10.1 环境要求

- Node.js: ^20.19.0 || >=22.12.0
- npm 或 yarn

### 10.2 安装依赖

```bash
npm install
```

### 10.3 开发命令

| 命令 | 说明 |
|------|------|
| `npm run dev` | 启动开发服务器 |
| `npm run build` | 构建生产版本 |
| `npm run preview` | 预览生产版本 |
| `npm run type-check` | TypeScript 类型检查 |
| `npm run lint` | 代码检查 |
| `npm run format` | 代码格式化 |

### 10.4 Vite 配置

- 开发服务器代理 `/api` 到 `http://localhost:8080`
- `@` 别名指向 `src` 目录
- 启用 Vue DevTools 插件

## 11. 样式设计系统

### 11.1 全局样式

- Mac 风格按钮 (`.mac-btn`)
- Mac 风格输入框 (`.mac-input`)
- Mac 风格标题栏 (`.mac-title-bar`)
- 圆角设计
- 毛玻璃效果

### 11.2 响应式断点

| 断点 | 屏幕宽度 | 特点 |
|------|----------|------|
| Desktop | > 768px | 完整布局 |
| Mobile | <= 768px | 简化导航 |
| Small Mobile | <= 480px | 单列布局 |

### 11.3 移动端适配

- 固定顶部导航栏
- 移动端菜单抽屉
- 触摸友好的按钮尺寸
- 响应式卡片网格

## 12. 安全考虑

### 12.1 XSS 防护

- 内容详情页对 Markdown 内容进行脚本过滤
- 富文本编辑禁止脚本注入

### 12.2 权限控制

- 路由级别权限控制
- API 请求需要认证
- 后台功能仅管理员可用

## 13. 性能优化

### 13.1 图片处理

- 懒加载图片
- 视频封面生成
- CDN 加速（xqapi.xiey.work）

### 13.2 代码分割

- 路由组件懒加载
- 减少首屏加载体积

### 13.3 动态背景

- 随机背景图预加载
- 图片加载失败降级处理

## 14. 第三方服务

### 14.1 背景图 API

```
GET https://rimg.xiey.work/
```

随机获取背景图片

### 14.2 后端 API

```
开发环境: http://localhost:8080/api
生产环境: https://xqapi.xiey.work/api
```

## 15. 开发规范

### 15.1 代码规范

- ESLint 代码检查
- Prettier 代码格式化
- TypeScript 严格类型检查

### 15.2 Git 规范

- 组件文件使用 PascalCase
- 工具函数使用 camelCase
- 常量使用 UPPER_SNAKE_CASE

### 15.3 注释规范

- 使用中文注释
- JSDoc 风格函数注释
- 关键业务逻辑需要详细说明

---

**文档版本**: 1.0.0  
**最后更新**: 2024年  
**维护者**: huntersxy

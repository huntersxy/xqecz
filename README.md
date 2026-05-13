# 小泉动漫二创平台

一个优雅的动漫二创内容分享平台，采用 Mac 风格 UI 设计。

![Vue](https://img.shields.io/badge/Vue-3.5-blue)
![TypeScript](https://img.shields.io/badge/TypeScript-6.0-blue)
![Vite](https://img.shields.io/badge/Vite-8.0-purple)
![License](https://img.shields.io/badge/License-CC%20BY--NC%204.0-red)

## ✨ 特性

- 🎨 **Mac 风格 UI** - 毛玻璃效果、圆角卡片、流畅动画（motion-v）
- 📱 **响应式设计** - 完美适配桌面端和移动端
- 🔐 **用户认证** - 登录注册、会话管理、Cookie 跨域支持
- 📝 **内容管理** - 支持视频、图片、Markdown 图文三种类型
- 💬 **评论系统** - 嵌套回复、举报功能、删除管理
- 📊 **推荐系统** - 智能推荐内容、无限刷新
- 🔍 **搜索筛选** - 关键词搜索、标签筛选、类型筛选
- ⏳ **上传管理** - Markdown 编辑器、图片上传、文件上传、实时进度
- 🎁 **彩蛋空间** - 隐藏的惊喜页面
- ⚙️ **后台管理** - 内容审核、用户管理、举报管理、缩略图管理
- ⚡ **性能优化** - 路由懒加载、图片懒加载、组件预加载、滚动位置保持
- 🎯 **异步加载** - Suspense + defineAsyncComponent 优雅加载体验
- 🔒 **安全渲染** - DOMPurify XSS 防护、Markdown 安全解析

## 🛠️ 技术栈

| 技术 | 版本 | 说明 |
|------|------|------|
| Vue | 3.5 | 渐进式 JavaScript 框架 |
| TypeScript | 6.0 | 类型安全的 JavaScript 超集 |
| Vite | 8.0 | 下一代前端构建工具 |
| Vue Router | 5.0 | Vue.js 官方路由管理器 |
| Pinia | 3.0 | Vue.js 轻量级状态管理 |
| Axios | 1.16 | Promise HTTP 客户端 |
| Marked | 18.0 | Markdown 解析器 |
| motion-v | 2.2 | Vue 动画库 |
| DOMPurify | 3.4 | HTML 净化与 XSS 防护 |

## 🚀 快速开始

```bash
# 克隆项目
git clone https://github.com/huntersxy/xqecz.git

# 进入项目目录
cd xqecz

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 类型检查
npm run type-check

# 代码格式化
npm run format
```

## 📁 项目结构

```
xiaoquanweb/
├── src/
│   ├── api/           # API 接口封装（认证、内容、评论、管理）
│   ├── assets/        # 静态资源（CSS、Logo）
│   ├── components/    # 通用组件
│   │   └── admin/     # 后台管理组件
│   ├── router/        # 路由配置与守卫
│   ├── stores/        # Pinia 状态管理
│   │   ├── counter.ts # 计数器示例
│   │   ├── home.ts    # 首页状态（筛选、滚动位置）
│   │   └── user.ts    # 用户认证状态
│   ├── types/         # TypeScript 类型定义
│   ├── utils/         # 工具函数（Markdown、异步组件）
│   └── views/         # 页面视图
├── public/            # 公共资源
└── dist/              # 构建输出
```

## 📄 页面路由

| 路径 | 说明 | 权限 |
|------|------|------|
| `/` | 首页 - 推荐内容、搜索、筛选、分页 | 公开 |
| `/login` | 登录/注册 | 公开 |
| `/content/:id` | 内容详情 - Markdown渲染、评论 | 公开 |
| `/upload` | 上传内容 - Markdown编辑器、标签 | 需登录 |
| `/admin` | 后台管理 - 内容审核、用户管理 | 管理员 |
| `/admin/reports` | 举报管理 - 评论举报处理 | 管理员 |
| `/easter-egg` | 彩蛋空间（隐藏入口） | 公开 |

## 🎨 核心功能

### 首页
- 🔥 推荐内容网格（4列布局，无限刷新）
- 📅 最近上传列表（搜索、筛选、分页）
- 🏷️ 标签云筛选
- 📂 类型筛选（视频/图片/图文）
- 💾 滚动位置与筛选状态保持

### 内容详情
- 📝 Markdown 富文本渲染（DOMPurify XSS 防护）
- 🖼️ 图片画廊展示
- 🎬 视频在线播放
- 💬 嵌套评论系统（回复、举报、删除）
- 🔙 返回保持页面状态

### 上传发布
- ✍️ Markdown 编辑器（标题、粗体、斜体、列表、代码、图片）
- 🖼️ 图片内嵌上传
- 🏷️ 预设标签选择 + 自定义标签
- 📊 上传进度实时显示

### 后台管理
- ✅ 内容审核（通过/拒绝）
- 👥 用户管理（角色、封禁）
- 🚨 举报管理（评论举报处理）
- 🖼️ 缩略图管理（单个/批量重新生成）

## 🔧 环境变量

```bash
# 开发环境
VITE_API_BASE_URL=http://localhost:8080/api

# 生产环境
VITE_API_BASE_URL=https://xqapi.xiey.work/api
```

## 📝 许可证

**版权所有 © 2025-2026 小泉动漫**

本项目采用 **CC BY-NC 4.0（知识共享署名-非商业性使用 4.0 国际）** 许可证进行授权。

### 您可以

- ✅ 复制、分发、展示和表演本作品
- ✅ 创作衍生作品
- ✅ 用于个人学习、教育和非商业目的

### 您必须

- 署名：在使用本项目时，必须给出适当的署名，提供本许可证的链接，并标明是否对原始作品进行了修改
- 署名方式：在显眼位置标明"小泉动漫"或项目名称
- 提供许可证链接：https://creativecommons.org/licenses/by-nc/4.0/

### 您不得

- ❌ 商业使用：禁止将本项目用于任何商业目的，包括但不限于：
  - 任何形式的付费服务
  - 商业产品或软件的集成
  - 以盈利为目的的网站或应用
  - 任何需要付费才能使用的服务
- ❌ 去标识化：移除或更改本项目中的任何版权声明、署名信息或许可证链接

### 免责声明

本项目仅供学习交流使用，作者不对使用本项目造成的任何直接或间接损失负责。

---

> 如果这个项目对你有帮助，欢迎 Star ⭐

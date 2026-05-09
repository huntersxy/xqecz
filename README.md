# 小泉动漫二创平台

一个优雅的动漫二创内容分享平台，采用 Mac 风格 UI 设计。

![Vue](https://img.shields.io/badge/Vue-3.4-blue)
![TypeScript](https://img.shields.io/badge/TypeScript-5.0-blue)
![Vite](https://img.shields.io/badge/Vite-5.0-purple)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ 特性

- 🎨 **Mac 风格 UI** - 毛玻璃效果、圆角卡片、流畅动画
- 📱 **响应式设计** - 完美适配桌面端和移动端
- 🔐 **用户认证** - 登录注册、会话管理
- 📝 **内容管理** - 支持视频、图片、文字三种类型
- 💬 **评论系统** - 互动交流、举报功能
- 🎁 **彩蛋空间** - 隐藏的惊喜页面
- ⚙️ **后台管理** - 内容审核、用户管理

## 🛠️ 技术栈

| 技术 | 说明 |
|------|------|
| Vue 3 | 渐进式 JavaScript 框架 |
| TypeScript | 类型安全的 JavaScript 超集 |
| Vite | 下一代前端构建工具 |
| Vue Router | Vue.js 官方路由管理器 |
| Pinia | Vue.js 轻量级状态管理 |
| Axios | Promise HTTP 客户端 |
| Marked | Markdown 解析器 |

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
```

## 📁 项目结构

```
xiaoquanweb/
├── src/
│   ├── api/           # API 接口封装
│   ├── assets/        # 静态资源
│   ├── components/    # 通用组件
│   ├── router/        # 路由配置
│   ├── stores/        # 状态管理
│   ├── types/         # TypeScript 类型
│   └── views/         # 页面视图
│       ├── HomeView.vue          # 首页
│       ├── LoginView.vue         # 登录注册
│       ├── ContentDetailView.vue # 内容详情
│       ├── AdminView.vue         # 后台管理
│       └── EasterEggView.vue     # 彩蛋空间
├── public/            # 公共资源
└── dist/              # 构建输出
```

## 📄 页面路由

| 路径 | 说明 |
|------|------|
| `/` | 首页 - 内容列表 |
| `/login` | 登录/注册 |
| `/content/:id` | 内容详情 |
| `/admin` | 后台管理 |
| `/easter-egg` | 彩蛋空间（隐藏入口） |

## 🎨 设计亮点

- **毛玻璃效果** - 使用 `backdrop-filter` 实现优雅的模糊背景
- **响应式布局** - 移动端优先设计，支持各种屏幕尺寸
- **流畅动画** - 过渡效果、微交互增强用户体验

## 📝 License

MIT License - 详见 [LICENSE](LICENSE) 文件

---

> 如果这个项目对你有帮助，欢迎 Star ⭐

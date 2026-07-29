# 小泉动漫二创站 (xqecz)

Vue 3 前端 + NestJS 后端 + Go gRPC Worker 的 monorepo，运行在 pnpm workspace 下。

## 快速开始

```bash
# 安装依赖
pnpm install

# 复制环境变量模版并填入实际值
cp .env.example .env

# 启动全部服务（api + worker + frontend）
pnpm dev
```

## 项目结构

```
xqecz/
├── packages/
│   ├── api/          # NestJS 后端 (TypeScript)
│   ├── frontend/     # Vue 3 前端 (TypeScript + Ant Design Vue + Tailwind CSS)
│   └── worker/       # Go gRPC Worker (图片压缩/缩略图/推荐)
├── proto/            # gRPC 协议定义 (Protobuf)
├── scripts/          # 开发辅助脚本
├── data/             # 共享卷 (uploads/thumbs/images)
│   ├── uploads/      # 原文件
│   ├── thumbs/       # 缩略图
│   └── images/       # 压缩图 (WebP)
├── .env.example      # 环境变量模版
├── pnpm-workspace.yaml
└── package.json
```

## 环境变量

复制 `.env.example` 为 `.env`，填入实际值：

| 变量 | 说明 |
|------|------|
| `MYSQL_HOST` | MySQL 数据库地址 |
| `MYSQL_PORT` | MySQL 端口 (默认 3306) |
| `MYSQL_USER` | 数据库用户名 |
| `MYSQL_PASSWORD` | 数据库密码 |
| `MYSQL_DATABASE` | 数据库名 |
| `REDIS_HOST` | Redis 地址 |
| `REDIS_PORT` | Redis 端口 (默认 6379) |
| `REDIS_PASSWORD` | Redis 密码 |
| `REDIS_PREFIX` | Redis 键前缀 |
| `PORT` | API 服务端口 (默认 3000) |
| `CORS_ORIGINS` | CORS 允许的源，逗号分隔 |
| `TINIFY_API_KEY` | TinyPNG API Key（图片压缩） |
| `VITE_API_BASE_URL` | 前端 API 地址 (默认 /api) |
| `VITE_MEDIA_BASE_URL` | 媒体文件前缀 (默认空) |

## 技术栈

- **前端**: Vue 3 + Vite + TypeScript + Ant Design Vue 4 + Tailwind CSS 4
- **后端**: NestJS + TypeORM + MySQL + Redis
- **Worker**: Go + gRPC + FFmpeg + Tinify
- **构建**: pnpm workspace + Turborepo

## 命令

```bash
pnpm dev          # 启动全部开发服务
pnpm build        # 构建全��
pnpm start        # 生产模式启动
pnpm lint         # 代码检查
pnpm typecheck    # 类型检查
```

## License

MIT

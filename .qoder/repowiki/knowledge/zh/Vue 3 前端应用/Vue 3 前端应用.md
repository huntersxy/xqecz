---
kind: external_dependency
name: Vue 3 前端应用
slug: vue3-vite
category: external_dependency
category_hints:
    - framework_behavior
scope:
    - '**'
---

### Vue 3 + Vite
- 角色：用户界面层，通过 HTTP 代理访问 NestJS API。
- 集成点：`packages/frontend/`，Vite 开发服务器 :5173，生产预览 :4173；默认代理目标为 `http://localhost:3000`（NestJS），可通过 `VITE_PROXY_TARGET` 覆盖。
- 行为约束：`/api`、`/uploads`、`/thumbnails`、`/images` 等路径均代理到后端；`vite preview` 继承 `server.proxy` 配置。
- 运行方式：`pnpm dev:fe`（开发）、`pnpm build`（生产构建）、`pnpm start:services` 中的 `preview`。
- verify exact API/params against vite.config.ts
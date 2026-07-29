---
kind: external_dependency
name: NestJS 主后端框架
slug: nestjs
category: external_dependency
category_hints:
    - framework_behavior
scope:
    - '**'
---

### NestJS
- 角色：monorepo 中的主后端（API），独占 MySQL 与 Redis，通过 gRPC 调用 Go Worker。
- 集成点：`packages/api/src/`，TypeORM + ioredis + `@nestjs/microservices` 实现 gRPC client；`main.ts` 暴露静态 `/uploads` 目录。
- 行为约束：所有外部依赖（Tinify/S3/Worker）缺失即降级，gRPC 永不返回 rpc error，只返回 `success=false` + error 文本；推荐刷新由 api 主导，worker 仅做无状态打分。
- 运行方式：`pnpm dev:api`（热重载 :3000）、`pnpm start:prod`（生产模式）。
- verify exact API/params against official docs
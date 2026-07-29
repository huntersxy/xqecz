---
kind: external_dependency
name: MySQL 数据库
slug: mysql
category: external_dependency
category_hints:
    - client_constraint
scope:
    - '**'
---

### MySQL
- 角色：NestJS API 的持久化存储，8 张表（users/contents/comments/polls/poll_votes/api_keys/claims/comment_reports），含软删除字段 `deleted_at`。
- 集成点：TypeORM Entity 定义在 `packages/api/src/entities/`，连接配置来自 `packages/api/.env`。
- 部署：云端实例（非本地 Docker），通过环境变量注入凭据。
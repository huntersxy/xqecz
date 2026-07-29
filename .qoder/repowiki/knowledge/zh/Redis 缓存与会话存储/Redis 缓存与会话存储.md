---
kind: external_dependency
name: Redis 缓存与会话存储
slug: redis
category: external_dependency
category_hints:
    - client_constraint
scope:
    - '**'
---

### Redis
- 角色：NestJS 的 session 存储、缓存、浏览量统计、推荐 ZSet（`recommend:hot`）。
- 集成点：`packages/api/src/redis/redis.service.ts`，ioredis 客户端，keyPrefix 为 `xqecz:`（实际 key 如 `xqecz:recommend:hot`）。
- 约束：Worker 不直连 Redis，推荐数据由 api 写入；ZSet 成员为字符串化的 contentId。
- 部署：云端实例（非本地 Docker），通过环境变量注入凭据。
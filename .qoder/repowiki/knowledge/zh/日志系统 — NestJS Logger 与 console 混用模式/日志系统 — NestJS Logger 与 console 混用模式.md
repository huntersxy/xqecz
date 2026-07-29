---
kind: logging_system
name: 日志系统 — NestJS Logger 与 console 混用模式
category: logging_system
scope:
    - '**'
source_files:
    - packages/api/src/main.ts
    - packages/api/src/auth/auth.service.ts
    - packages/api/src/content/content.service.ts
    - packages/api/src/redis/redis.service.ts
    - packages/api/src/worker/worker.service.ts
    - packages/frontend/src/api/index.ts
---

本仓库的日志系统由两部分组成：NestJS 内置 Logger（用于 API 服务）和原生 console 输出（散落在各模块中），尚未形成统一的日志框架或集中配置。

1. 使用的系统与工具
- NestJS 内置 Logger：在 `packages/api/src/auth/auth.service.ts` 中通过 `new Logger(AuthService.name)` 实例化，使用 `warn` 级别输出管理员账号初始化、密码修改等关键事件。
- NestFactory 启动时通过 `{ logger: ['error', 'warn', 'log'] }` 指定全局日志级别过滤，仅输出 error/warn/log 三级。
- 大量业务代码直接使用 `console.log` / `console.warn` / `console.error` 进行调试与错误记录，如 Redis 连接、推荐刷新、媒体处理、gRPC 调用超时等场景。
- 前端（Vue3）同样使用 `console.error` / `console.warn` 记录 API 请求失败、未授权、标签加载失败等。
- Go Worker 端未发现专门的日志库引用，未见结构化日志输出。

2. 核心文件与位置
- `packages/api/src/main.ts`：NestJS 应用入口，配置全局 logger 级别为 `['error', 'warn', 'log']`。
- `packages/api/src/auth/auth.service.ts`：唯一使用 NestJS `Logger` 类的地方，以类名为上下文前缀输出 warn 级别日志。
- `packages/api/src/content/content.service.ts`：大量 `console.warn` / `console.log` 用于推荐刷新、缩略图生成、S3 上传、链接预览等流程。
- `packages/api/src/redis/redis.service.ts`：Redis 连接事件通过 `console.error` / `console.log` 输出。
- `packages/api/src/worker/worker.service.ts`：gRPC 超时通过 `console.warn` 记录。
- `packages/frontend/src/api/index.ts`：前端统一拦截器中使用 `console.error` / `console.warn` 输出请求失败与未授权信息。

3. 架构与约定
- 无独立日志模块或配置文件，日志策略分散在各 service 文件中。
- NestJS Logger 仅被 auth.service.ts 使用，其他模块未遵循该约定，而是直接依赖 console。
- 日志级别策略：NestJS 全局仅启用 error/warn/log 三级，debug 级别被过滤；业务代码自行决定使用 console.warn/error 而非结构化字段。
- 无统一日志格式、无结构化字段（如 requestId、userId）、无外部 sink（文件/ELK/云日志服务）集成。

4. 约定与约束
- 观察到的约定：NestJS 服务类可通过 `@nestjs/common` 的 `Logger` 注入并使用其 warn/info/error 方法，但未被强制或广泛采用。
- 实际约束：`main.ts` 中硬编码的 `logger: ['error', 'warn', 'log']` 限制了 NestJS 内部日志的输出级别，debug 信息不会出现在控制台。
- 无 lint 规则或文档强制要求使用特定日志框架，console 输出是事实上的主要方式。
- 环境变量中未发现 LOG_LEVEL、LOG_FORMAT 等日志相关配置项。
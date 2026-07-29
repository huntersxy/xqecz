---
kind: external_dependency
name: Go 无状态 gRPC Worker 微服务
slug: go-worker
category: external_dependency
category_hints:
    - framework_behavior
scope:
    - '**'
---

### Go Worker
- 角色：无状态计算服务，处理文件缩略图、图片压缩、S3 上传、链接预览、推荐打分，不直连 MySQL/Redis。
- 集成点：`packages/worker/server/` 实现 gRPC server，入口 `cmd/server/main.go`；通过 `scripts/run-worker.mjs` 启动，自动从 `packages/api/.env` 注入 `UPLOAD_DIR/THUMB_DIR/IMAGES_DIR`。
- 运行方式：`pnpm dev:worker`（go run :50051）、`pnpm worker:build`（编译二进制）。
- verify exact API/params against proto/xqecz.proto
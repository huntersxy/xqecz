---
kind: external_dependency
name: FFmpeg 多媒体处理工具
slug: ffmpeg
category: external_dependency
category_hints:
    - client_constraint
scope:
    - '**'
---

### FFmpeg
- 角色：Worker 中视频抽帧生成缩略图、图片缩放的核心工具。
- 集成点：`packages/worker/server/media/thumbnail.go`，通过系统命令调用 ffmpeg。
- 约束：可选依赖，未安装时降级为跳过缩略图生成；Windows 下需确保 PATH 中包含 ffmpeg 可执行文件。
- 版本：项目文档要求 Go 1.25+，FFmpeg 为系统级工具，无版本锁定。
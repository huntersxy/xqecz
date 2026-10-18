# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

小泉动漫二次创作者、浏览作品的观众，以及负责审核和运营的管理员。创作者在站内发布文本和可选媒体，也可以使用 AI 工作台生成、整理和继续编辑视觉内容。

## Product Purpose

小泉动漫二创站让用户发现、发布、讨论和继续创作二次创作内容。首页需要让人快速进入作品浏览，同时保留上传、搜索、筛选、点赞和进入详情的路径。

## Positioning

这是一个把二创内容社区、AI 辅助创作和社区治理放在同一条链路上的作品广场。内容以“文本 + 可选媒体”发布，不依赖固定分类，浏览体验以持续发现作品为中心。

## Operating Context

用户通常从首页浏览作品、搜索标题或作者、按标签筛选、打开详情互动，或从首页进入快速上传、AI 工作台、角色资源和创意广场。页面需要兼容桌面与手机浏览，并在列表增量加载、返回首页和图片加载时保持阅读位置。

## Capabilities and Constraints

- 前端为 Vue 3 单页应用，后端 API 与现有响应契约保持兼容。
- 首页保留瀑布流、推荐内容、搜索、标签筛选、点赞、上传入口、详情跳转和 KeepAlive 状态恢复。
- 作品可能只有文本，也可能带图片或视频；媒体地址由共享工具解析，不能在页面内另起一套拼接规则。
- 公开内容遵循现有审核、授权、评论和社区治理规则。

## Evidence on Hand

- 现有内容接口、类型契约和首页实现位于 `packages/frontend/src/api`、`packages/frontend/src/types` 与 `packages/frontend/src/views/HomeView.vue`。
- 用户指定参考画廊 `https://image.oblivionis.net/gallery`，本轮首页改造借鉴其侧栏画廊结构、密集图片流和轻量元信息呈现方式。

## Product Principles

- 先让作品被看见，再让装饰退到背景。
- 浏览、搜索、筛选、上传和详情之间的路径要短而清楚。
- 视觉可以大胆替换，但内容契约、交互状态和社区规则必须连续。
- 图片、文字和失败状态都要有可辨认的呈现。

## Accessibility & Inclusion

保留语义标题、键盘可达控件、可见焦点样式、媒体替代文本和 `prefers-reduced-motion` 降级。

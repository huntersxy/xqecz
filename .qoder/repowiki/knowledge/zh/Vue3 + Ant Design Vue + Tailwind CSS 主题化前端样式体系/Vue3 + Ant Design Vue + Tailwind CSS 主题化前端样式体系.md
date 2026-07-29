---
kind: frontend_style
name: Vue3 + Ant Design Vue + Tailwind CSS 主题化前端样式体系
category: frontend_style
scope:
    - '**'
source_files:
    - packages/frontend/src/App.vue
    - packages/frontend/src/stores/theme.ts
    - packages/frontend/src/composables/useThemeRegistry.ts
    - packages/frontend/index.html
    - packages/frontend/package.json
---

## 系统概述
xqecz 前端采用 Vue 3 + Vite + TypeScript 技术栈，以 Ant Design Vue 4.x 作为 UI 组件库基础，结合 Tailwind CSS v4 进行原子化样式构建，并通过自研的主题注册机制实现多主题、明暗模式切换。

## 核心架构
- **框架与工具链**：Vue 3.5 + Vite 8 + TypeScript 6，使用 @vitejs/plugin-vue 和 vue-tsc 进行类型检查
- **UI 组件库**：Ant Design Vue 4.2.6 + @ant-design/icons-vue 7.0.1，提供基础组件和图标
- **样式方案**：Tailwind CSS 4.3 + PostCSS + Sass，支持 CSS 变量主题系统
- **状态管理**：Pinia 3.0.4 管理主题状态和用户状态
- **路由与数据**：Vue Router 5 + TanStack Vue Query 5 处理页面路由和数据请求

## 主题系统设计
项目实现了完整的多主题架构，通过 `src/composables/useThemeRegistry.ts` 统一管理：
- **主题元数据结构**：定义 ThemeMeta 接口，包含 key、name、description、category（mac/large）、previewColor、colors.light/dark 等字段
- **动态注册机制**：使用 `registerTheme()` 函数和 `import.meta.glob('@/themes/*.vue')` 自动发现并注册主题组件
- **CSS 变量映射**：`applyThemeColors()` 将主题颜色映射到 CSS 自定义属性（--theme-primary、--theme-text、--theme-surface 等 17 个变量）
- **持久化存储**：通过 localStorage 保存用户选择的主题和明暗模式

## 样式组织模式
- **组件级样式**：使用 Vue scoped `<style>` 标签，配合 CSS 变量实现主题适配
- **响应式设计**：基于媒体查询的移动端适配（max-width: 768px），使用 Drawer 组件实现移动端导航
- **设计令牌**：通过 CSS 自定义属性统一管理颜色、背景、边框等视觉元素
- **特殊主题**：liquidGlass 主题通过 SVG filter 实现毛玻璃效果

## 关键文件结构
- `packages/frontend/src/App.vue`：主应用组件，包含主题切换 UI 和全局布局
- `packages/frontend/src/stores/theme.ts`：Pinia 主题状态管理
- `packages/frontend/src/composables/useThemeRegistry.ts`：主题注册和颜色应用逻辑
- `packages/frontend/index.html`：HTML 入口，包含 PWA 配置和 SVG 滤镜定义
- `packages/frontend/package.json`：依赖管理和构建脚本

## 约束与规范
- 所有主题必须遵循 ThemeMeta 接口规范
- 颜色变量统一通过 CSS 自定义属性访问，禁止硬编码颜色值
- 移动端优先的响应式断点为 768px
- 主题切换通过修改 document.documentElement 的 class 和 style 属性实现
- 使用 Ant Design Vue 组件时通过 :deep() 选择器覆盖默认样式
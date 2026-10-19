<script setup lang="ts">
// 后台面板容器：统一页头（标题/描述/操作位）+ 集中式内容样式
// 表格、图标按钮、移动端卡片等共享样式经 :deep 在此定义一次，子面板零样式即可复用
//
// 波次 2 只增不减（§9.1）：title / desc / flat 与默认、actions 两个插槽的渲染结构不变；
// 新增能力全部默认关闭，老页面不传即维持原样。
interface Props {
  title: string
  desc?: string
  flat?: boolean
  /** M2：面板子元素进场错峰（默认关闭，老页签不会被动获得动画） */
  stagger?: boolean
  // 波次 4：允许面板内部元素吸顶（日志工具条）。只放开裁剪，不改 padding 与页头样式
  flow?: boolean
}

withDefaults(defineProps<Props>(), { stagger: false })

const titleEl = ref<HTMLElement | null>(null)

// 页签切换后由 AdminView 调用，把焦点移到页面标题（§7.5）；标题不在 Tab 序里（tabindex=-1）
defineExpose({ focusTitle: () => titleEl.value?.focus() })
</script>

<template>
  <section class="admin-panel" :class="{ 'is-flat': flat, 'is-flow': flow }">
    <header class="admin-panel-head">
      <div class="admin-panel-head-text">
        <h2 ref="titleEl" class="admin-panel-title" tabindex="-1">{{ title }}</h2>
        <p v-if="desc" class="admin-panel-desc">{{ desc }}</p>
      </div>
      <div v-if="$slots.actions" class="admin-panel-actions">
        <slot name="actions" />
      </div>
    </header>
    <div v-if="$slots.toolbar" class="admin-panel-toolbar">
      <slot name="toolbar" />
    </div>
    <div class="admin-panel-body" :class="{ 'adm-stagger': stagger }">
      <slot />
    </div>
  </section>
</template>

<style lang="scss" scoped>
// 颜色、圆角、字号、间距一律优先取新令牌 --adm-*，取不到时回落到旧的 --admin-*，
// 这样 WP2 把令牌注入 body 之前页面也不会变色（两组值当前指向同一批 --site-* 变量）。
.admin-panel {
  background: var(--adm-surface, var(--admin-surface));
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-card, var(--admin-radius));
  overflow: hidden;
}

// 与 is-flat 解耦：只放开 overflow，保留默认页头与内边距
.admin-panel.is-flow {
  overflow: visible;
}

.admin-panel.is-flat {
  overflow: visible;
  background: transparent;
  border: 0;
  border-radius: 0;
}

.admin-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-lg, 16px);
  flex-wrap: wrap;
  padding: 20px var(--adm-gap-xl, 24px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.admin-panel.is-flat .admin-panel-head {
  align-items: flex-end;
  padding: 0 0 20px;
  border-bottom: 0;
}

.admin-panel-title {
  display: flex;
  align-items: center;
  gap: 0;
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-xl, 21px);
  font-weight: 600;
  letter-spacing: -.02em;
  color: var(--adm-ink, var(--admin-text));
}

// 页签切换后的程序化聚焦（tabindex=-1）不需要可见外框：它不是可操作控件，
// 键盘用户也不会 Tab 到它，焦点环只会让标题多出一个突兀的方框。
.admin-panel-title:focus {
  outline: none;
}

.admin-panel-desc {
  margin: 4px 0 0;
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: tabular-nums;
}

.admin-panel-actions {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
}

// 工具区（§10：计数 / 刷新 / 页面动作）：默认插槽为空时不渲染，老页签 DOM 不变
.admin-panel-toolbar {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  padding: var(--adm-gap-md, 12px) var(--adm-gap-xl, 24px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.admin-panel-body {
  min-height: 200px;

  // 确保 spin 容器撑满
  :deep(.arco-spin) {
    width: 100%;
  }

  // ── 表格现代化：去 Arco 默认外框、宽松行距、悬浮填充 ──
  :deep(.arco-table) {
    background: transparent;
    width: 100%;
  }

  :deep(.arco-table-element) {
    width: 100% !important;
  }

  // 强制表格内部容器撑满
  :deep(.arco-table-header),
  :deep(.arco-table-body) {
    width: 100% !important;
  }

  :deep(.arco-table-header table),
  :deep(.arco-table-body table) {
    width: 100% !important;
  }

  // 去 Arco 默认外边框（border:true 会给 container 的 top/left/right 框线，
  // 与 panel 自身边框叠成“框中框”，造成表格与面板宽度割裂）
  :deep(.arco-table-border .arco-table-container) {
    border: none;
    width: 100%;
  }

  :deep(.arco-table-th) {
    background: transparent;
    padding: 9px 16px;
    font-size: var(--adm-font-xs, 12px);
    font-weight: 500;
    letter-spacing: 0.02em;
    color: var(--adm-muted, var(--admin-text-3));
    border-bottom: 1px solid var(--adm-line, var(--admin-border));
  }

  :deep(.arco-table-td) {
    padding: 10px 16px;
    font-size: var(--adm-font-sm, 13px);
    color: var(--adm-ink, var(--admin-text));
    border-bottom: 1px solid var(--admin-border-soft);
  }

  // 首尾单元格与页头 24px padding 对齐，消除内容左右错位
  :deep(.arco-table-th:first-child),
  :deep(.arco-table-td:first-child) {
    padding-left: 24px;
  }

  :deep(.arco-table-th:last-child),
  :deep(.arco-table-td:last-child) {
    padding-right: 24px;
  }

  :deep(.arco-table-tr:last-child .arco-table-td) {
    border-bottom: none;
  }

  :deep(.arco-table .arco-table-tr:hover .arco-table-td),
  :deep(.arco-table .arco-table-tr-hover .arco-table-td) {
    background: var(--adm-soft, var(--admin-fill));
  }

  :deep(.arco-table-pagination) {
    padding: 16px 24px;
  }

  // ── 图标操作按钮（幽灵风格）──
  :deep(.admin-icon-btn) {
    width: 30px;
    height: 30px;
    padding: 0;
    border-radius: var(--adm-r-control, 10px);
    font-size: var(--adm-font-base, 14px);
    color: var(--adm-muted, var(--admin-text-3));
    transition: background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
      color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));
  }

  :deep(.admin-icon-btn:hover) {
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-ink, var(--admin-text));
  }

  :deep(.admin-icon-btn.is-primary:hover) {
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    color: var(--adm-accent, var(--admin-primary));
  }

  :deep(.admin-icon-btn.is-danger:hover) {
    background: var(--admin-danger-soft);
    color: var(--adm-danger, var(--admin-danger));
  }

  :deep(.admin-action-group) {
    display: inline-flex;
    align-items: center;
    gap: var(--adm-gap-xs, 4px);
  }

  // ── 单元格文本层级 ──
  :deep(.admin-cell-2) {
    font-size: var(--adm-font-sm, 13px);
    color: var(--adm-muted, var(--admin-text-2));
  }

  :deep(.admin-cell-3) {
    font-size: var(--adm-font-xs, 12px);
    color: var(--adm-muted, var(--admin-text-3));
  }

  :deep(.admin-cell-title) {
    font-size: var(--adm-font-sm, 13px);
    font-weight: 500;
    color: var(--adm-ink, var(--admin-text));
  }

  :deep(.admin-mono) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: var(--adm-font-xs, 12px);
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-muted, var(--admin-text-2));
  }

  // ── 移动端卡片 ──
  :deep(.admin-mobile-list) {
    display: flex;
    flex-direction: column;
    gap: var(--adm-gap-md, 12px);
    padding: var(--adm-gap-lg, 16px);
  }

  :deep(.admin-mobile-card) {
    border: 1px solid var(--adm-line, var(--admin-border));
    border-radius: var(--adm-r-card, var(--admin-radius));
    padding: 14px 16px;
    transition: border-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));
  }

  :deep(.admin-mobile-card:active) {
    border-color: var(--adm-accent, var(--admin-primary));
  }

  :deep(.admin-mobile-actions) {
    display: flex;
    gap: var(--adm-gap-sm, 8px);
    margin-top: var(--adm-gap-md, 12px);
    padding-top: var(--adm-gap-md, 12px);
    border-top: 1px solid var(--admin-border-soft);
  }

  :deep(.admin-mobile-pagination) {
    display: flex;
    justify-content: center;
    padding: var(--adm-gap-lg, 16px);
    border-top: 1px solid var(--admin-border-soft);
  }

  :deep(.arco-empty) {
    padding: 48px 0;
  }
}

.admin-panel.is-flat .admin-panel-body { min-height: 0; }

@media (max-width: 768px) {
  .admin-panel-head {
    padding: 16px;
  }

  .admin-panel.is-flat .admin-panel-head { padding: 0 0 16px; }

  .admin-panel-toolbar { padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px); }

  // 手机档把图标按钮放到 §6.2 要求的 44px 触控高度。
  // 桌面档保持 30px：审核行的 4 个图标按钮塞在 110px 操作列里，现在放大到 36px
  // 会直接溢出，桌面尺寸随波次 3 的表格重制一起改。
  .admin-panel-body :deep(.admin-icon-btn) {
    width: var(--adm-touch, 44px);
    height: var(--adm-touch, 44px);
    font-size: var(--adm-font-base, 14px);
  }
}
</style>

<!--
  全局动效工具（M2 错峰进场 / M8 行退场）：故意不加 scoped。
  AdminPanel 是每个后台页签的必经节点，样式随该组件模块只输出一次；
  若写进 _admin.scss，会被 9 个页签的 scoped 样式各编译一份（顶层规则会重复 9 份）。
-->
<style lang="scss">
@keyframes adm-rise {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: none; }
}

// M2：只作用前 8 个子元素（§8.4 同屏错峰上限 12，这里取更保守的 8）。
// 动画只在节点挂载时播一次：页签切换会重建内容节点（AdminView 的 :key），
// 而数据刷新只是替换列表数据，不会重建 Panel，因此不会重播。
.adm-stagger {
  > :nth-child(-n + 8) {
    animation: adm-rise var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1)) backwards;
  }

  @for $i from 1 through 8 {
    > :nth-child(#{$i}) {
      animation-delay: calc(var(--adm-stagger, 40ms) * #{$i - 1});
    }
  }
}

// M8：审核/删除后该行先淡出 280ms 再刷新数据，类名由表格的 row-class 打在 <tr> 上。
// 只动 opacity / transform，不动高度（§8.1 第 2 条）。
.admin-theme {
  .arco-table-tr.is-leaving > .arco-table-td {
    opacity: 0;
    transform: translateY(-4px);
    pointer-events: none;
    transition: opacity 280ms var(--adm-ease-in, ease-in), transform 280ms var(--adm-ease-in, ease-in);
  }

  .admin-mobile-card.is-leaving {
    opacity: 0;
    transform: translateY(-4px);
    pointer-events: none;
    transition: opacity 280ms var(--adm-ease-in, ease-in), transform 280ms var(--adm-ease-in, ease-in);
  }
}

// 命中 reduced-motion 时错峰动画直接归零，元素停在终态（不需要 JS 参与）。
@media (prefers-reduced-motion: reduce) {
  .adm-stagger > * {
    animation: none;
  }
}

// ── 卡片流（内容 / 认领 / 举报共用）：后台列表不再使用 <table> ──
// 只在这里定义一次；各列表组件只写少量 scoped 补充，避免同一套卡片样式被各页签各编译一份。
.adm-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--adm-gap-md, 12px);
}

.adm-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--adm-surface, var(--admin-surface));
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  overflow: hidden;
  transition:
    border-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
    transform var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1));
}

@media (hover: hover) and (pointer: fine) {
  .adm-card:hover {
    border-color: var(--adm-accent, var(--admin-primary));
    transform: translateY(-2px);
  }
}

// M8 全局通用的卡片离场：内容 / 认领 / 举报共用
.adm-card.is-leaving {
  opacity: 0;
  transform: translateY(-4px);
  pointer-events: none;
  transition: opacity 280ms var(--adm-ease-in, ease-in), transform 280ms var(--adm-ease-in, ease-in);
}

.adm-card-media {
  position: relative;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  background: var(--adm-soft, var(--admin-fill));
}

.adm-card-media > img,
.adm-card-media > video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.adm-card-media.is-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-sm, 13px);
}

// 认领/举报这类没有配图的卡片用矮占位条，4:3 的空块会白吃掉半屏
.adm-card-media.is-quiet {
  aspect-ratio: auto;
  min-height: 88px;
  padding: 12px;
}

@media (max-width: 768px) {
  .adm-card-media.is-quiet {
    min-height: 76px;
  }
}

.adm-card-badge {
  position: absolute;
  top: 8px;
  left: 8px;
  padding: 2px 8px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--adm-surface, var(--admin-surface)) 88%, transparent);
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}

.adm-card-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 12px 14px 14px;
}

.adm-card-title {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  line-height: 1.45;
}

.adm-card-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: tabular-nums;
}

.adm-card-actions {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  margin-top: auto;
  padding: 0 14px 14px;
}

// M2 卡片进场错峰：只在挂载时播一次，刷新列表不重播
.adm-card-grid > .adm-card:nth-child(-n + 8) {
  animation: adm-rise var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1)) backwards;
}

@for $i from 1 through 8 {
  .adm-card-grid > .adm-card:nth-child(#{$i}) {
    animation-delay: calc(var(--adm-stagger, 40ms) * #{$i - 1});
  }
}

@media (max-width: 768px) {
  .adm-card-grid {
    grid-template-columns: repeat(auto-fill, minmax(156px, 1fr));
    gap: 10px;
  }

  .adm-card-body { padding: 10px 12px 12px; }

  .adm-card-actions { padding: 0 12px 12px; }
}

@media (prefers-reduced-motion: reduce) {
  .adm-card:hover { transform: none; }

  .adm-card-grid > .adm-card { animation: none; }
}
</style>

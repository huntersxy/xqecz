<script setup lang="ts">
interface Props {
  title: string
  desc?: string
  flat?: boolean
  stagger?: boolean
  flow?: boolean
}

withDefaults(defineProps<Props>(), { stagger: false })

const titleEl = ref<HTMLElement | null>(null)

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
.admin-panel {
  background: var(--adm-surface, var(--admin-surface));
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-card, var(--admin-radius));
  overflow: hidden;
}

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

  :deep(.arco-spin) {
    width: 100%;
  }

  :deep(.arco-table) {
    background: transparent;
    width: 100%;
  }

  :deep(.arco-table-element) {
    width: 100% !important;
  }

  :deep(.arco-table-header),
  :deep(.arco-table-body) {
    width: 100% !important;
  }

  :deep(.arco-table-header table),
  :deep(.arco-table-body table) {
    width: 100% !important;
  }

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

  .admin-panel-body :deep(.admin-icon-btn) {
    width: var(--adm-touch, 44px);
    height: var(--adm-touch, 44px);
    font-size: var(--adm-font-base, 14px);
  }
}
</style>

<style lang="scss">
@keyframes adm-rise {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: none; }
}

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

@media (prefers-reduced-motion: reduce) {
  .adm-stagger > * {
    animation: none;
  }
}

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

.adm-board {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
  gap: var(--adm-gap-lg, 16px);
  align-items: start;
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-xl, 24px) var(--adm-gap-xl, 24px);
}

.adm-board-main {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-lg, 16px);
  min-width: 0;
}

.adm-board-aside {
  position: sticky;
  top: var(--adm-gap-lg, 16px);
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
}

@media (max-width: 1180px) {
  .adm-board { grid-template-columns: minmax(0, 1fr); }

  .adm-board-aside { position: static; }
}

@media (max-width: 768px) {
  .adm-board { padding: var(--adm-gap-md, 12px); }
}

.adm-stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(148px, 1fr));
  gap: var(--adm-gap-sm, 8px);
}

.adm-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
}

.adm-stat-label {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.adm-stat-value {
  color: var(--adm-ink, var(--admin-text));
  font-size: 22px;
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -.02em;
  font-variant-numeric: tabular-nums;
}

.adm-stat-hint {
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-card-grid.is-wide {
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
}

.adm-card-grid.is-dense {
  grid-template-columns: repeat(auto-fill, minmax(196px, 1fr));
}

.adm-card-content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px 16px;
}

.adm-card-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  margin-top: auto;
  padding: 10px 14px 12px;
  border-top: 1px solid var(--admin-border-soft);
}

.adm-card-foot > .is-push {
  margin-left: auto;
}

.adm-kv {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
}

.adm-kv-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
  font-size: var(--adm-font-xs, 12px);
}

.adm-kv-row dt {
  flex: 0 0 auto;
  color: var(--adm-muted, var(--admin-text-3));
}

.adm-kv-row dd {
  overflow: hidden;
  min-width: 0;
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .adm-card-grid.is-wide,
  .adm-card-grid.is-dense {
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  }
}

@media (max-width: 560px) {
  .adm-card-grid.is-wide,
  .adm-card-grid.is-dense {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

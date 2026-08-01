<script setup lang="ts">
// 后台面板容器：统一页头（标�?描述/操作位）+ 集中式内容样�?// 表格、图标按钮、移动端卡片等共享样式经 :deep 在此定义一次，子面板零样式即可复用
interface Props {
  title: string
  desc?: string
}

defineProps<Props>()
</script>

<template>
  <section class="admin-panel">
    <header class="admin-panel-head">
      <div class="admin-panel-head-text">
        <h2 class="admin-panel-title">{{ title }}</h2>
        <p v-if="desc" class="admin-panel-desc">{{ desc }}</p>
      </div>
      <div v-if="$slots.actions" class="admin-panel-actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="admin-panel-body">
      <slot />
    </div>
  </section>
</template>

<style lang="scss" scoped>
.admin-panel {
  background: var(--admin-surface);
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--admin-radius);
  box-shadow: var(--admin-shadow);
  overflow: hidden;
}

.admin-panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 20px 24px;
  border-bottom: 1px solid var(--admin-border-soft);
}

.admin-panel-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--admin-text);
}

.admin-panel-desc {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--admin-text-3);
  font-variant-numeric: tabular-nums;
}

.admin-panel-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.admin-panel-body {
  min-height: 200px;

  // ── 表格现代化：�?Arco 默认外框、宽松行距、悬浮填�?──
  :deep(.arco-table) {
    background: transparent;
    width: 100%;
  }

  :deep(.arco-table-element) {
    width: 100% !important;
  }

  // �?Arco 默认外边框（border:true 会给 container �?top/left/right 框线�?  // �?panel 自身边框叠成“框中框”，造成表格与面板宽度割裂）
  :deep(.arco-table-border .arco-table-container) {
    border: none;
    width: 100%;
  }

  :deep(.arco-table-th) {
    background: transparent;
    padding: 12px 16px;
    font-size: 12px;
    font-weight: 500;
    letter-spacing: 0.02em;
    color: var(--admin-text-3);
    border-bottom: 1px solid var(--admin-border);
  }

  :deep(.arco-table-td) {
    padding: 14px 16px;
    font-size: 13px;
    color: var(--admin-text);
    border-bottom: 1px solid var(--admin-border-soft);
  }

  // 首尾单元格与页头 24px padding 对齐，消除内容左右错�?  :deep(.arco-table-th:first-child),
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
    background: var(--admin-fill);
  }

  :deep(.arco-table-pagination) {
    padding: 16px 24px;
  }

  // ── 图标操作按钮（幽灵风格）──
  :deep(.admin-icon-btn) {
    width: 30px;
    height: 30px;
    padding: 0;
    border-radius: 8px;
    font-size: 14px;
    color: var(--admin-text-3);
    transition: background 0.15s ease, color 0.15s ease;
  }

  :deep(.admin-icon-btn:hover) {
    background: var(--admin-fill);
    color: var(--admin-text);
  }

  :deep(.admin-icon-btn.is-primary:hover) {
    background: var(--admin-primary-soft);
    color: var(--admin-primary);
  }

  :deep(.admin-icon-btn.is-danger:hover) {
    background: var(--admin-danger-soft);
    color: var(--admin-danger);
  }

  :deep(.admin-action-group) {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  // ── 单元格文本层�?──
  :deep(.admin-cell-2) {
    font-size: 13px;
    color: var(--admin-text-2);
  }

  :deep(.admin-cell-3) {
    font-size: 12px;
    color: var(--admin-text-3);
  }

  :deep(.admin-cell-title) {
    font-size: 13px;
    font-weight: 500;
    color: var(--admin-text);
  }

  :deep(.admin-mono) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px;
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--admin-fill);
    color: var(--admin-text-2);
  }

  // ── 移动端卡�?──
  :deep(.admin-mobile-list) {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
  }

  :deep(.admin-mobile-card) {
    border: 1px solid var(--admin-border);
    border-radius: var(--admin-radius);
    padding: 14px 16px;
    transition: border-color 0.2s ease;
  }

  :deep(.admin-mobile-card:active) {
    border-color: var(--admin-primary);
  }

  :deep(.admin-mobile-actions) {
    display: flex;
    gap: 8px;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid var(--admin-border-soft);
  }

  :deep(.admin-mobile-pagination) {
    display: flex;
    justify-content: center;
    padding: 16px;
    border-top: 1px solid var(--admin-border-soft);
  }

  :deep(.arco-empty) {
    padding: 48px 0;
  }
}

@media (max-width: 768px) {
  .admin-panel-head {
    padding: 16px;
  }
}
</style>








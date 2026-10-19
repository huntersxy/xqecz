<script setup lang="ts">
// 数据区工具条（§10「工具区：计数 / 刷新 / 页面动作」）：左侧摘要，右侧动作与刷新。
// 保持无状态：计数与刷新动作都由使用方提供，组件只负责版式与触控尺寸，
// 避免把各页签不同的数据源抽象进来。
interface Props {
  /** 左侧摘要文案，例如「共 128 条内容」；不传则只渲染默认插槽 */
  summary?: string
  /** 内置刷新按钮的 loading 态 */
  loading?: boolean
  /** 是否渲染内置刷新按钮；不想用内置按钮时改用 actions 插槽 */
  refreshable?: boolean
}

withDefaults(defineProps<Props>(), {
  summary: '',
  loading: false,
  refreshable: false,
})

const emit = defineEmits<{ (e: 'refresh'): void }>()
</script>

<template>
  <div class="admin-data-toolbar">
    <div class="admin-data-toolbar-info">
      <span v-if="summary" class="admin-data-toolbar-summary">{{ summary }}</span>
      <slot />
    </div>
    <div class="admin-data-toolbar-actions">
      <slot name="actions" />
      <a-button v-if="refreshable" size="small" :loading="loading" @click="emit('refresh')">刷新</a-button>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.admin-data-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  padding: var(--adm-gap-md, 12px) 0;
}

.admin-data-toolbar-info {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  flex-wrap: wrap;
  min-width: 0;
}

.admin-data-toolbar-summary {
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: tabular-nums;
}

.admin-data-toolbar-actions {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
  margin-left: auto;
}
</style>

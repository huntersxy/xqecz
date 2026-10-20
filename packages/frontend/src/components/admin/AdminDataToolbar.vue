<script setup lang="ts">
interface Props {
  summary?: string
  loading?: boolean
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

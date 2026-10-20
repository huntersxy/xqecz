<script setup lang="ts">
interface Props {
  type?: 'success' | 'warning' | 'danger' | 'info' | 'neutral'
  label: string
}

const props = withDefaults(defineProps<Props>(), { type: 'neutral' })

const colorMap: Record<NonNullable<Props['type']>, string> = {
  success: 'var(--adm-success, rgb(var(--success-6)))',
  warning: 'var(--adm-warning, rgb(var(--warning-6)))',
  danger: 'var(--adm-danger, rgb(var(--danger-6)))',
  info: 'var(--adm-accent, var(--admin-primary, rgb(var(--primary-6))))',
  neutral: 'var(--adm-muted, var(--color-text-4))',
}

const dotColor = computed(() => colorMap[props.type])
</script>

<template>
  <span class="admin-status">
    <i class="admin-status-dot" :style="{ background: dotColor }" />
    <span class="admin-status-label">{{ label }}</span>
  </span>
</template>

<style scoped>
.admin-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.4;
  color: var(--adm-muted, var(--color-text-2));
  white-space: nowrap;
}

.admin-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>

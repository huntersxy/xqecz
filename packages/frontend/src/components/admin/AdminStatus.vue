<script setup lang="ts">
// 状态点：替代高饱和度填充 Tag，用于审核/封禁/处理等状态展示
interface Props {
  type?: 'success' | 'warning' | 'danger' | 'info' | 'neutral'
  label: string
}

const props = withDefaults(defineProps<Props>(), { type: 'neutral' })

const colorMap: Record<NonNullable<Props['type']>, string> = {
  success: 'var(--success-6)',
  warning: 'var(--warning-6)',
  danger: 'var(--danger-6)',
  info: 'var(--primary-6)',
  neutral: 'var(--color-text-4)',
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
  font-size: 13px;
  line-height: 1.4;
  color: var(--color-text-2);
  white-space: nowrap;
}

.admin-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>

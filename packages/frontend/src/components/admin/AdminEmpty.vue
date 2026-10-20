<script setup lang="ts">
import { IconEmpty } from '@arco-design/web-vue/es/icon'

interface Props {
  title?: string
  desc?: string
  compact?: boolean
}

withDefaults(defineProps<Props>(), {
  title: '暂无数据',
  desc: '',
  compact: false,
})
</script>

<template>
  <div class="admin-empty" :class="{ 'is-compact': compact }">
    <div class="admin-empty-mark" aria-hidden="true">
      <slot name="icon"><IconEmpty /></slot>
    </div>
    <p class="admin-empty-title">{{ title }}</p>
    <p v-if="desc" class="admin-empty-desc">{{ desc }}</p>
    <div v-if="$slots.default" class="admin-empty-extra">
      <slot />
    </div>
    <div v-if="$slots.actions" class="admin-empty-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.admin-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--adm-gap-sm, 8px);
  padding: 48px var(--adm-gap-xl, 24px);
  text-align: center;

  &.is-compact {
    padding: var(--adm-gap-xl, 24px) var(--adm-gap-lg, 16px);
  }
}

.admin-empty-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  margin-bottom: var(--adm-gap-xs, 4px);
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-3));
  font-size: 20px;
}

.admin-empty-title {
  margin: 0;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  letter-spacing: -.01em;
  color: var(--adm-ink, var(--admin-text));
}

.admin-empty-desc {
  margin: 0;
  max-width: 46ch;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
}

.admin-empty-extra,
.admin-empty-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
  margin-top: var(--adm-gap-sm, 8px);
}
</style>

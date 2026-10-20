<script setup lang="ts">
import SiteIcon from '@/components/SiteIcon.vue'

withDefaults(
  defineProps<{
    variant?: 'primary' | 'ghost'
    size?: 'md' | 'lg'
    type?: 'button' | 'submit'
    loading?: boolean
    disabled?: boolean
  }>(),
  { variant: 'ghost', size: 'md', type: 'button' },
)
</script>

<template>
  <button
    class="site-btn"
    :class="[`is-${variant}`, `is-${size}`, { 'is-loading': loading }]"
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
  >
    <SiteIcon v-if="loading" name="refresh" spin />
    <slot />
  </button>
</template>

<style scoped>
.site-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 18px;
  border: 1px solid var(--creative-line, var(--site-line));
  border-radius: 999px;
  background: transparent;
  color: var(--creative-ink, var(--site-ink));
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color .18s ease, border-color .18s ease, color .18s ease, box-shadow .18s ease, transform .12s ease;
}
.site-btn.is-lg { min-height: 44px; padding: 0 24px; font-size: 13px; }
.site-btn.is-ghost:hover:not(:disabled) { background: var(--creative-soft, var(--site-soft)); }
.site-btn.is-primary {
  border-color: transparent;
  background: var(--creative-accent, var(--site-accent));
  color: var(--site-on-action, #fff);
}
.site-btn.is-primary:hover:not(:disabled) {
  background: var(--creative-accent-hover, var(--creative-accent, var(--site-accent)));
  box-shadow: 0 8px 20px color-mix(in srgb, var(--creative-accent, var(--site-accent)) 28%, transparent);
}
.site-btn:active:not(:disabled) { transform: scale(.97); }
.site-btn:disabled { opacity: .6; cursor: not-allowed; }
.site-btn.is-loading:disabled { cursor: progress; }
</style>

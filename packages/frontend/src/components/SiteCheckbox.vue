<script setup lang="ts">
const model = defineModel<boolean>({ default: false })
defineProps<{ disabled?: boolean }>()
</script>

<template>
  <label class="site-checkbox" :class="{ 'is-disabled': disabled }">
    <input v-model="model" class="site-checkbox-input" type="checkbox" :disabled="disabled" />
    <span class="site-checkbox-box" aria-hidden="true">
      <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3.5 8.5l3 3 6-7" /></svg>
    </span>
    <span class="site-checkbox-label"><slot /></span>
  </label>
</template>

<style scoped>
.site-checkbox { position: relative; display: inline-flex; align-items: flex-start; gap: 9px; font-size: 12px; line-height: 1.6; cursor: pointer; }
.site-checkbox.is-disabled { cursor: not-allowed; opacity: .55; }
.site-checkbox-input { position: absolute; width: 1px; height: 1px; margin: 0; opacity: 0; }
.site-checkbox-box {
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  width: 16px;
  height: 16px;
  margin-top: 2px;
  border: 1.5px solid var(--creative-line, var(--site-line));
  border-radius: 5px;
  background: var(--creative-paper, var(--site-paper));
  color: transparent;
  transition: background-color .16s ease, border-color .16s ease, color .16s ease, box-shadow .16s ease;
}
.site-checkbox-box svg { width: 11px; height: 11px; }
.site-checkbox:hover:not(.is-disabled) .site-checkbox-box { border-color: var(--creative-accent, var(--site-accent)); }
.site-checkbox-input:checked + .site-checkbox-box {
  border-color: var(--creative-accent, var(--site-accent));
  background: var(--creative-accent, var(--site-accent));
  color: var(--site-on-action, #fff);
}
.site-checkbox-input:focus-visible + .site-checkbox-box {
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent, var(--site-accent)) 24%, transparent);
}
</style>

<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import SiteIcon from '@/components/SiteIcon.vue'
import SiteOverlay from '@/components/SiteOverlay.vue'

const props = withDefaults(defineProps<{
  open: boolean
  title: string
  confirmText?: string
  cancelText?: string
  busy?: boolean
}>(), { confirmText: '确定', cancelText: '取消', busy: false })

const emit = defineEmits<{ close: []; confirm: [] }>()

const panel = ref<HTMLElement | null>(null)
const titleId = `cd-dialog-title-${Math.random().toString(36).slice(2, 8)}`

watch(() => props.open, async (open) => {
  if (!open) return
  await nextTick()
  panel.value?.querySelector<HTMLElement>('textarea, input')?.focus()
})
</script>

<template>
  <SiteOverlay
    :model-value="open"
    class="cd-dialog-shell creative-theme"
    :labelledby="titleId"
    :close-on-mask="!busy"
    :close-on-esc="!busy"
    :z-index="10000"
    @update:model-value="value => { if (!value) emit('close') }"
  >
    <div ref="panel" class="cd-dialog">
      <header class="cd-dialog-head">
        <h3 :id="titleId" class="cd-dialog-title">{{ title }}</h3>
        <button type="button" class="cd-dialog-close" aria-label="关闭" :disabled="busy" @click="emit('close')">
          <SiteIcon name="close" />
        </button>
      </header>
      <div class="cd-dialog-body"><slot /></div>
      <footer class="cd-dialog-foot">
        <button type="button" class="cd-dialog-btn" :disabled="busy" @click="emit('close')">{{ cancelText }}</button>
        <button type="button" class="cd-dialog-btn is-primary" :disabled="busy" :aria-busy="busy" @click="emit('confirm')">
          {{ busy ? '处理中…' : confirmText }}
        </button>
      </footer>
    </div>
  </SiteOverlay>
</template>

<style scoped>
.cd-dialog-shell :deep(.site-overlay-mask) {
  background: color-mix(in srgb, var(--creative-ink) 38%, transparent);
  backdrop-filter: none;
}
.cd-dialog-shell :deep(.site-overlay-panel) {
  width: min(480px, 100%);
  max-height: 100%;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  transition: none;
}
.cd-dialog-shell :deep(.site-overlay-enter-from .site-overlay-panel),
.cd-dialog-shell :deep(.site-overlay-leave-to .site-overlay-panel) { transform: none; }

.cd-dialog {
  width: 100%; max-height: 100%; display: flex; flex-direction: column;
  background: var(--creative-paper); color: var(--creative-ink);
  border: 1px solid var(--creative-line); border-radius: 20px; box-shadow: var(--creative-shadow);
  animation: cd-dialog-rise .2s ease-out;
}
.cd-dialog-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px 20px 14px 24px; border-bottom: 1px solid var(--creative-line); }
.cd-dialog-title { margin: 0; font-size: 15px; font-weight: 600; }
.cd-dialog-title::before { content: '✧'; margin-right: 9px; color: var(--creative-accent); }
.cd-dialog-close { display: grid; place-items: center; width: 30px; height: 30px; border: 0; border-radius: 50%; background: transparent; color: var(--creative-muted); font-size: 16px; cursor: pointer; }
.cd-dialog-close:hover:not(:disabled) { color: var(--creative-accent); background: var(--creative-soft); }
.cd-dialog-body { padding: 20px 24px; overflow-y: auto; line-height: 1.8; overflow-wrap: anywhere; }
.cd-dialog-foot { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 24px 18px; border-top: 1px solid var(--creative-line); }
.cd-dialog-btn { height: 34px; padding: 0 18px; border: 1px solid var(--creative-line); border-radius: 999px; background: var(--creative-canvas); color: var(--creative-ink); font: inherit; font-size: 13px; cursor: pointer; transition: background .18s, color .18s, border-color .18s; }
.cd-dialog-btn:hover:not(:disabled) { border-color: var(--creative-accent); color: var(--creative-accent); }
.cd-dialog-btn.is-primary { border-color: var(--creative-accent); background: var(--creative-accent); color: var(--color-on-primary, #fff); }
.cd-dialog-btn.is-primary:hover:not(:disabled) { background: var(--creative-accent-hover); color: var(--color-on-primary, #fff); }
.cd-dialog-btn:disabled, .cd-dialog-close:disabled { opacity: .55; cursor: not-allowed; }
@keyframes cd-dialog-rise { from { opacity: 0; transform: translateY(8px) scale(.98); } to { opacity: 1; transform: none; } }
</style>

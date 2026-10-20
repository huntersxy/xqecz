<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import CdIcon from './CdIcon.vue'

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
let opener: HTMLElement | null = null

const FOCUSABLE = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'

watch(() => props.open, async (open) => {
  if (open) {
    opener = document.activeElement as HTMLElement | null
    await nextTick()
    const target = panel.value?.querySelector<HTMLElement>('textarea, input') ?? panel.value?.querySelector<HTMLElement>(FOCUSABLE)
    target?.focus()
  } else {
    opener?.focus?.()
    opener = null
  }
})

onBeforeUnmount(() => { opener?.focus?.() })

function onKeydown(e: KeyboardEvent) {
  e.stopPropagation()
  if (e.key === 'Escape') {
    if (!props.busy) emit('close')
    return
  }
  if (e.key !== 'Tab' || !panel.value) return
  const items = Array.from(panel.value.querySelectorAll<HTMLElement>(FOCUSABLE))
  if (items.length === 0) return
  const first = items[0]
  const last = items[items.length - 1]
  if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus() }
  else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus() }
}

function onBackdrop() {
  if (!props.busy) emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="cd-dialog-mask creative-theme" @click.self="onBackdrop">
      <div
        ref="panel"
        class="cd-dialog"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @keydown="onKeydown"
      >
        <header class="cd-dialog-head">
          <h3 :id="titleId" class="cd-dialog-title">{{ title }}</h3>
          <button type="button" class="cd-dialog-close" aria-label="关闭" :disabled="busy" @click="emit('close')">
            <CdIcon name="close" />
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
    </div>
  </Teleport>
</template>

<style scoped>
.cd-dialog-mask {
  position: fixed; inset: 0; z-index: 10000; display: grid; place-items: center; padding: 16px;
  background: color-mix(in srgb, var(--creative-ink) 38%, transparent);
  animation: cd-dialog-fade .16s ease-out;
}
.cd-dialog {
  width: min(480px, 100%); max-height: calc(100dvh - 32px); display: flex; flex-direction: column;
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
@keyframes cd-dialog-fade { from { opacity: 0; } to { opacity: 1; } }
@keyframes cd-dialog-rise { from { opacity: 0; transform: translateY(8px) scale(.98); } to { opacity: 1; transform: none; } }
</style>

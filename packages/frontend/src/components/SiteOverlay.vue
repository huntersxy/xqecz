<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

/**
 * 站内统一浮层：底部抽屉（sheet）与居中弹窗（center）共用。
 * 负责挂载到 body、遮罩、Esc / 遮罩关闭、焦点收进面板与关闭后归还、Tab 焦点圈定。
 * 背景滚动锁定由 main.css 按 `.site-overlay` 是否在 DOM 里决定，这里不碰内联样式。
 */
defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    placement?: 'center' | 'sheet'
    closeOnMask?: boolean
    closeOnEsc?: boolean
    label?: string
    labelledby?: string
    zIndex?: number
  }>(),
  { placement: 'center', closeOnMask: true, closeOnEsc: true, zIndex: 2000 },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const panel = ref<HTMLElement | null>(null)
let opener: HTMLElement | null = null

// 多层浮层叠放时（例如上传抽屉里再弹确认框），Esc 只交给最上面的一层。
const stack: symbol[] = (globalThis as { __siteOverlayStack?: symbol[] }).__siteOverlayStack ??= []
const token = Symbol('overlay')

function close() {
  emit('update:modelValue', false)
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (stack[stack.length - 1] !== token) return
  if (event.key === 'Escape' && props.closeOnEsc) {
    event.preventDefault()
    close()
  }
}

const FOCUSABLE = 'a[href], button:not(:disabled), input:not(:disabled):not([type="hidden"]), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'

function onPanelKeydown(event: KeyboardEvent) {
  if (event.key !== 'Tab' || !panel.value) return
  const nodes = [...panel.value.querySelectorAll<HTMLElement>(FOCUSABLE)].filter((el) => el.offsetParent !== null || el === document.activeElement)
  if (!nodes.length) {
    event.preventDefault()
    panel.value.focus()
    return
  }
  const first = nodes[0]!
  const last = nodes[nodes.length - 1]!
  const active = document.activeElement
  if (event.shiftKey && (active === first || active === panel.value)) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

function teardown() {
  document.removeEventListener('keydown', onDocumentKeydown)
  const index = stack.indexOf(token)
  if (index >= 0) stack.splice(index, 1)
}

watch(
  () => props.modelValue,
  async (open) => {
    if (open) {
      opener = document.activeElement instanceof HTMLElement ? document.activeElement : null
      stack.push(token)
      document.addEventListener('keydown', onDocumentKeydown)
      await nextTick()
      // 面板里已有自动聚焦的控件就不抢；否则把焦点交给面板本身，读屏从标题开始。
      if (panel.value && !panel.value.contains(document.activeElement)) panel.value.focus({ preventScroll: true })
    } else {
      teardown()
      opener?.focus?.({ preventScroll: true })
      opener = null
    }
  },
  { immediate: true },
)
onBeforeUnmount(teardown)
</script>

<template>
  <Teleport to="body">
    <Transition name="site-overlay">
      <div v-if="modelValue" v-bind="$attrs" class="site-overlay" :class="`is-${placement}`" :style="{ zIndex }">
        <div class="site-overlay-mask" @click="closeOnMask && close()"></div>
        <div
          ref="panel"
          class="site-overlay-panel"
          role="dialog"
          aria-modal="true"
          tabindex="-1"
          :aria-label="labelledby ? undefined : label"
          :aria-labelledby="labelledby"
          @keydown="onPanelKeydown"
        >
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.site-overlay { position: fixed; inset: 0; display: flex; justify-content: center; }
.site-overlay.is-center { align-items: center; padding: 16px; }
.site-overlay.is-sheet { align-items: flex-end; }
.site-overlay-mask { position: absolute; inset: 0; background: rgba(47, 32, 43, .34); backdrop-filter: blur(6px); }
.site-overlay-panel {
  position: relative;
  max-width: 100%;
  max-height: 100%;
  outline: 0;
  border: 1px solid var(--creative-line, var(--site-line));
  background: var(--creative-paper, var(--site-paper));
  color: var(--creative-ink, var(--site-ink));
  box-shadow: var(--creative-shadow, 0 24px 100px rgba(51, 28, 43, .18));
}
.is-center .site-overlay-panel { width: min(440px, 100%); border-radius: 20px; }
.is-sheet .site-overlay-panel { width: min(920px, calc(100% - 48px)); margin-bottom: 24px; overflow: hidden; border-radius: 24px; }

/* 遮罩淡入淡出，面板上浮并带一点缩放；离场更快，不挡着下一步操作。 */
.site-overlay-enter-active,
.site-overlay-leave-active { transition: opacity .24s cubic-bezier(.22, 1, .36, 1); }
.site-overlay-enter-active .site-overlay-panel { transition: transform .32s cubic-bezier(.22, 1, .36, 1); }
.site-overlay-leave-active { transition-duration: .16s; }
.site-overlay-leave-active .site-overlay-panel { transition: transform .16s ease-in; }
.site-overlay-enter-from,
.site-overlay-leave-to { opacity: 0; }
.is-center.site-overlay-enter-from .site-overlay-panel,
.is-center.site-overlay-leave-to .site-overlay-panel { transform: translateY(10px) scale(.96); }
.is-sheet.site-overlay-enter-from .site-overlay-panel,
.is-sheet.site-overlay-leave-to .site-overlay-panel { transform: translateY(28px); }

@media (max-width: 768px) {
  .is-sheet .site-overlay-panel { width: 100%; margin-bottom: 0; border-bottom: 0; border-radius: 22px 22px 0 0; }
}
</style>

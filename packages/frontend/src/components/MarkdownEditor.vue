<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Vditor from 'vditor'
import 'vditor/dist/index.css'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    height?: number
    disabled?: boolean
  }>(),
  { modelValue: '', placeholder: '支持 Markdown 排版…', height: 320, disabled: false }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  blur: []
}>()

const containerRef = ref<HTMLDivElement | null>(null)
let editor: Vditor | null = null
let editorReady = false
let pendingValue: string | null = null
let themeObserver: MutationObserver | null = null

function applyValue(value: string) {
  if (!editor || editor.getValue() !== value) editor?.setValue(value)
}

function isDark() {
  return document.body.hasAttribute('arco-theme')
}

function applyTheme() {
  if (!editor) return
  const dark = isDark()
  editor.setTheme(dark ? 'dark' : 'classic', dark ? 'dark' : 'light', dark ? 'vs2015' : 'github')
}

function onInput(value: string) {
  if (value !== props.modelValue) emit('update:modelValue', value)
}

onMounted(() => {
  if (!containerRef.value) return
  const dark = isDark()
  editor = new Vditor(containerRef.value, {
    value: props.modelValue || '',
    mode: 'wysiwyg',
    theme: dark ? 'dark' : 'classic',
    lang: 'zh_CN',
    placeholder: props.placeholder,
    height: props.height,
    width: '100%',
    cdn: '/vditor',
    cache: { enable: false },
    fullscreen: { index: 2000 },
    outline: { enable: false, position: 'left' },
    toolbar: [
      'headings',
      'bold',
      'italic',
      'strike',
      'line',
      'quote',
      'list',
      'ordered-list',
      'check',
      'inline-code',
      'code',
      'link',
      'table',
      'undo',
      'redo',
      'fullscreen',
    ],
    preview: {
      theme: { current: dark ? 'dark' : 'light' },
      hljs: { enable: true, lineNumber: false, style: dark ? 'vs2015' : 'github' },
      markdown: { sanitize: true, toc: false, autoSpace: false },
    },
    input: onInput,
    blur: () => emit('blur'),
    after: () => {
      editorReady = true
      if (pendingValue !== null) {
        applyValue(pendingValue)
        pendingValue = null
      }
      if (props.disabled) editor?.disabled()
    },
  })
  themeObserver = new MutationObserver(applyTheme)
  themeObserver.observe(document.body, { attributes: true, attributeFilter: ['arco-theme'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  themeObserver = null
  if (editor) {
    if (editorReady) {
      editor.destroy()
    } else {
      // vditor 的 init 依赖脚本异步加载;未就绪时其内部状态尚未构建,
      // destroy() 会直接崩溃,只能置 isDestroyed 阻断待执行的初始化链
      ;(editor as unknown as { isDestroyed: boolean }).isDestroyed = true
    }
  }
  editor = null
  editorReady = false
  pendingValue = null
})

watch(
  () => props.modelValue,
  (value) => {
    if (!editor) return
    if (!editorReady) {
      pendingValue = value || ''
      return
    }
    applyValue(value || '')
  }
)

watch(
  () => props.disabled,
  (value) => {
    if (!editor || !editorReady) return
    if (value) editor.disabled()
    else editor.enable()
  }
)
</script>

<template>
  <div ref="containerRef" class="markdown-editor" />
</template>

<style scoped>
.markdown-editor {
  width: 100%;
  min-width: 0;
  flex: 1 1 auto;
  box-sizing: border-box;
  border-radius: 8px;
  overflow: hidden;
  border-color: var(--color-border-2);
}
</style>

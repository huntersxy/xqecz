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
let themeObserver: MutationObserver | null = null

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
      if (props.disabled) editor?.disabled()
    },
  })
  themeObserver = new MutationObserver(applyTheme)
  themeObserver.observe(document.body, { attributes: true, attributeFilter: ['arco-theme'] })
})

onBeforeUnmount(() => {
  themeObserver?.disconnect()
  themeObserver = null
  editor?.destroy()
  editor = null
})

watch(
  () => props.modelValue,
  (value) => {
    if (editor && editor.getValue() !== (value || '')) editor.setValue(value || '')
  }
)

watch(
  () => props.disabled,
  (value) => {
    if (!editor) return
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
  border-radius: 8px;
  overflow: hidden;
}

/* 与 Arco 表单卡片保持视觉一致：编辑器外框跟随圆角与描边 */
.markdown-editor :deep(.vditor) {
  border-radius: 8px;
  border-color: var(--color-border-2);
}
</style>

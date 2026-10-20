<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  modelValue: string[]
  placeholder?: string
  disabled?: boolean
  ariaLabel?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const draft = ref('')
const input = ref<HTMLInputElement | null>(null)

function commit() {
  const value = draft.value.trim()
  draft.value = ''
  if (!value || props.modelValue.includes(value)) return
  emit('update:modelValue', [...props.modelValue, value])
}

function remove(index: number) {
  if (props.disabled) return
  emit('update:modelValue', props.modelValue.filter((_, i) => i !== index))
}

function onKeydown(event: KeyboardEvent) {
  if (event.isComposing) return
  if (event.key === 'Enter') {
    event.preventDefault()
    commit()
  } else if (event.key === 'Backspace' && !draft.value && props.modelValue.length) {
    remove(props.modelValue.length - 1)
  }
}
</script>

<template>
  <div class="site-tags" :class="{ 'is-disabled': disabled }" @click="input?.focus()">
    <span v-for="(tag, index) in modelValue" :key="tag" class="site-tags-chip">
      <span>{{ tag }}</span>
      <button type="button" :aria-label="`移除标签 ${tag}`" :disabled="disabled" @click.stop="remove(index)">×</button>
    </span>
    <input
      ref="input"
      v-model="draft"
      class="site-tags-input"
      type="text"
      :aria-label="ariaLabel"
      :placeholder="modelValue.length ? '' : placeholder"
      :disabled="disabled"
      @keydown="onKeydown"
      @blur="commit"
    />
  </div>
</template>

<style scoped>
.site-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 5px 10px;
  border: 1px solid var(--creative-line, var(--site-line));
  border-radius: 9px;
  background: var(--creative-canvas, var(--site-canvas));
  cursor: text;
  transition: border-color .2s ease, box-shadow .2s ease, background-color .2s ease;
}
.site-tags:focus-within {
  border-color: var(--creative-accent, var(--site-accent));
  background: var(--creative-paper, var(--site-paper));
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent, var(--site-accent)) 9%, transparent);
}
.site-tags.is-disabled { cursor: not-allowed; opacity: .6; }
.site-tags-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  padding: 2px 4px 2px 9px;
  border-radius: 999px;
  background: var(--creative-soft, var(--site-soft));
  color: var(--creative-accent, var(--site-accent));
  font-size: 11px;
  line-height: 1.6;
}
.site-tags-chip > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.site-tags-chip button {
  display: grid;
  place-items: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: inherit;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
}
.site-tags-chip button:hover:not(:disabled) { background: color-mix(in srgb, currentColor 16%, transparent); }
.site-tags-input {
  flex: 1 1 96px;
  min-width: 0;
  padding: 3px 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--creative-ink, var(--site-ink));
  font: inherit;
  font-size: 12px;
}
</style>

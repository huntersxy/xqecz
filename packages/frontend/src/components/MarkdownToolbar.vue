<script setup lang="ts">
import {
  IconH2, IconBold, IconItalic, IconStrikethrough, IconCode,
  IconQuote, IconList, IconOrderedList, IconLink, IconImage,
} from '@arco-design/web-vue/es/icon'

defineEmits<{
  insert: [prefix: string, suffix: string]
  uploadImage: []
}>()

const tools = [
  { title: '标题', icon: IconH2, prefix: '## ', suffix: '' },
  { title: '粗体', icon: IconBold, prefix: '**', suffix: '**' },
  { title: '斜体', icon: IconItalic, prefix: '*', suffix: '*' },
  { title: '删除线', icon: IconStrikethrough, prefix: '~~', suffix: '~~' },
  { title: '行内代码', icon: IconCode, prefix: '`', suffix: '`' },
  { title: '引用', icon: IconQuote, prefix: '> ', suffix: '' },
  { title: '无序列表', icon: IconList, prefix: '- ', suffix: '' },
  { title: '有序列表', icon: IconOrderedList, prefix: '1. ', suffix: '' },
  { title: '链接', icon: IconLink, prefix: '[', suffix: '](url)' },
] as const
</script>

<template>
  <div class="md-toolbar">
    <a-space :size="2" wrap>
      <a-tooltip v-for="tool in tools" :key="tool.title" :content="tool.title">
        <a-button
          type="text"
          size="small"
          class="md-btn"
          @click="$emit('insert', tool.prefix, tool.suffix)"
        >
          <component :is="tool.icon" />
        </a-button>
      </a-tooltip>

      <a-tooltip content="上传图片">
        <a-button
          type="text"
          size="small"
          class="md-btn"
          @click="$emit('uploadImage')"
        >
          <IconImage />
        </a-button>
      </a-tooltip>
    </a-space>
  </div>
</template>

<style lang="scss" scoped>
.md-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  padding: 6px 8px;
  margin-bottom: 8px;
  border-radius: 8px;
  background: rgb(var(--color-fill-2));
}

.md-btn {
  color: var(--color-text-2);
  border-radius: 6px;

  &:hover {
    color: rgb(var(--primary-6));
    background: rgb(var(--color-fill-3));
  }
}
</style>

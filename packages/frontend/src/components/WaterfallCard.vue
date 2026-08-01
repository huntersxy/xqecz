<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { getImageUrl } from '@/utils'
import type { Content } from '@/types'

interface Props {
  item: Content
}

const props = defineProps<Props>()
const emit = defineEmits<{ click: [content: Content]; imageLoaded: [id: string | number] }>()

// Arco <Image> 不对外 emit load 事件（内部吞掉了原生 img 的 onLoad），
// 而瀑布流 masonry 依赖 imageLoaded 触发重排，故用 ResizeObserver 监听媒体区高度变化来替代。
const mediaRef = ref<HTMLElement | null>(null)
let ro: ResizeObserver | null = null
onMounted(() => {
  if (mediaRef.value) {
    ro = new ResizeObserver(() => emit('imageLoaded', props.item.id))
    ro.observe(mediaRef.value)
  }
})
onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <div class="wf-card" @click="emit('click', props.item)" @keydown.enter="emit('click', props.item)" tabindex="0">
    <template v-if="props.item.type !== 'text'">
      <div class="wf-card-media" ref="mediaRef">
        <a-image :src="getImageUrl(props.item.thumb)" :alt="props.item.title" :preview="false" loading="lazy" decoding="async" />
        <div v-if="props.item.tags?.some(t => /ai/i.test(t))" class="wf-badge-ai">AI</div>
      </div>
    </template>
    <template v-else>
      <div class="wf-card-text-body">
        <p>{{ props.item.title }}</p>
      </div>
    </template>
    <div class="wf-card-info">
      <span class="wf-card-title">{{ props.item.title }}</span>
      <div class="wf-card-meta">
        <span class="wf-card-user">{{ props.item.user?.username }}</span>
        <span class="wf-card-views">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
          {{ props.item.view_count }}
        </span>
      </div>
      <div v-if="props.item.tags?.length" class="wf-card-tags">
        <span v-for="tag in props.item.tags.slice(0, 3)" :key="tag" class="wf-mini-tag">{{ tag }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wf-card {
  margin-bottom: 10px;
  border-radius: 0.625rem;
  overflow: hidden;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
}
.wf-card:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(0,0,0,0.1); }
.wf-card:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }

.wf-card-media {
  position: relative; width: 100%; overflow: hidden; line-height: 0;
  min-height: 80px; background: var(--color-placeholder);
}
/* Arco <Image> 包裹层默认 inline-block，需改为块级填满卡片宽度；破图时 wrapper 不能塌成 0，否则 .arco-image-error 绝对定位 overlay 无高度可显示 */
.wf-card-media :deep(.arco-image) { display: block; width: 100%; min-height: 80px; border-radius: 0; }
.wf-card-media :deep(.arco-image-img) { width: 100%; height: auto; display: block; vertical-align: top; }

.wf-badge-ai {
  position: absolute; top: 0.375rem; left: 0.375rem;
  padding: 0.0625rem 0.375rem; font-size: 0.5625rem; font-weight: 700;
  letter-spacing: 0.04em; color: #fff;
  background: rgba(var(--purple-6), 0.85); border-radius: 0.25rem;
}

.wf-card-text-body {
  padding: 1rem; display: flex; align-items: center; justify-content: center;
  min-height: 80px; background: var(--color-hover);
}
.wf-card-text-body p {
  font-size: 0.8125rem; line-height: 1.5; color: var(--color-text); margin: 0;
  display: -webkit-box; -webkit-line-clamp: 5; -webkit-box-orient: vertical; overflow: hidden;
}

.wf-card-info { padding: 0.5rem 0.625rem 0.625rem; }
.wf-card-title {
  display: block; font-size: 0.8125rem; font-weight: 600; color: var(--color-text);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 0.25rem;
}
.wf-card-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.wf-card-user { font-size: 0.6875rem; color: var(--color-text-secondary); }
.wf-card-views {
  display: flex; align-items: center; gap: 0.25rem;
  font-size: 0.625rem; color: var(--color-text-secondary);
}
.wf-card-views svg { width: 0.75rem; height: 0.75rem; }
.wf-card-tags { display: flex; gap: 0.25rem; margin-top: 0.375rem; flex-wrap: wrap; }
.wf-mini-tag {
  font-size: 0.5625rem; padding: 0.0625rem 0.375rem; border-radius: 1rem;
  background: var(--color-hover); color: var(--color-text-secondary);
}
</style>

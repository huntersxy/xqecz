<script setup lang="ts">
import { getImageUrl } from '@/utils'
import type { Content } from '@/types'

interface Props {
  item: Content
}

const props = defineProps<Props>()
const emit = defineEmits<{ click: [content: Content]; imageLoaded: [id: string | number] }>()

function onImgLoad() {
  emit('imageLoaded', props.item.id)
}
</script>

<template>
  <div class="wf-card" @click="emit('click', props.item)" @keydown.enter="emit('click', props.item)" tabindex="0">
    <template v-if="props.item.type !== 'text'">
      <div class="wf-card-media">
        <img :src="getImageUrl(props.item.thumb)" :alt="props.item.title" loading="lazy" decoding="async" @load="onImgLoad" />
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
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
}
.wf-card:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(0,0,0,0.1); }
.wf-card:focus-visible { outline: 2px solid var(--theme-primary); outline-offset: 2px; }

.wf-card-media {
  position: relative; width: 100%; overflow: hidden; line-height: 0;
  min-height: 80px; background: var(--theme-placeholder-bg);
}
.wf-card-media img { width: 100%; height: auto; display: block; }

.wf-badge-ai {
  position: absolute; top: 0.375rem; left: 0.375rem;
  padding: 0.0625rem 0.375rem; font-size: 0.5625rem; font-weight: 700;
  letter-spacing: 0.04em; color: #fff;
  background: rgba(139, 92, 246, 0.85); border-radius: 0.25rem;
}

.wf-card-text-body {
  padding: 1rem; display: flex; align-items: center; justify-content: center;
  min-height: 80px; background: var(--theme-hover-bg);
}
.wf-card-text-body p {
  font-size: 0.8125rem; line-height: 1.5; color: var(--theme-text); margin: 0;
  display: -webkit-box; -webkit-line-clamp: 5; -webkit-box-orient: vertical; overflow: hidden;
}

.wf-card-info { padding: 0.5rem 0.625rem 0.625rem; }
.wf-card-title {
  display: block; font-size: 0.8125rem; font-weight: 600; color: var(--theme-text);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 0.25rem;
}
.wf-card-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.wf-card-user { font-size: 0.6875rem; color: var(--theme-text-secondary); }
.wf-card-views {
  display: flex; align-items: center; gap: 0.25rem;
  font-size: 0.625rem; color: var(--theme-text-secondary);
}
.wf-card-views svg { width: 0.75rem; height: 0.75rem; }
.wf-card-tags { display: flex; gap: 0.25rem; margin-top: 0.375rem; flex-wrap: wrap; }
.wf-mini-tag {
  font-size: 0.5625rem; padding: 0.0625rem 0.375rem; border-radius: 1rem;
  background: var(--theme-hover-bg); color: var(--theme-text-secondary);
}
</style>

<script setup lang="ts">
import { computed, ref, nextTick } from 'vue'
import { getImageUrl, renderMarkdown } from '@/utils'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'
import type { Content } from '@/types'

interface Props {
  content: Content
}

const props = defineProps<Props>()

let viewerInstance: Viewer | null = null

const mediaUrl = computed(() => {
  if (props.content.img) return getImageUrl(props.content.img)
  if (props.content.video) return getImageUrl(props.content.video)
  return ''
})
const mediaKind = computed<'image' | 'video' | 'link' | 'text'>(() => {
  if (props.content.img) return 'image'
  if (props.content.video) return 'video'
  if (props.content.url) return 'link'
  return 'text'
})

const noMediaReason = computed(() => {
  if (mediaKind.value !== 'text') return ''
  const t = props.content.type as string | undefined
  if (t === 'image' || t === 'video') {
    if (!props.content.img && !props.content.video) return '原文件丢失或未生成，等待后台处理中...'
  }
  return ''
})

const renderedText = computed(() => {
  const t = props.content.text
  return t ? renderMarkdown(t) : ''
})

function openViewerInline() {
  const img = document.querySelector('.cd-media-image img') as HTMLImageElement | null
  if (!img) return
  const originUrl = props.content.origin ? getImageUrl(props.content.origin) : ''
  if (originUrl) img.dataset.origin = originUrl
  viewerInstance = new Viewer(img, {
    navbar: false, zIndex: 10000, zIndexInline: 10000,
    url(imgEl: HTMLImageElement) { return (imgEl as HTMLImageElement).dataset.origin || imgEl.src },
    hidden() { viewerInstance?.destroy(); viewerInstance = null },
  })
  nextTick(() => { viewerInstance?.show() })
}

function openExternalLink() {
  if (mediaKind.value === 'link' && props.content.url) globalThis.open(props.content.url, '_blank', 'noopener')
}

defineExpose({ mediaKind, mediaUrl })
</script>

<template>
  <section class="cd-media-wrap">
    <div v-if="mediaKind === 'image'" class="cd-media-image" @click="openViewerInline">
      <img :src="mediaUrl" :alt="content.title" class="cd-image" draggable="false" />
    </div>
    <video v-else-if="mediaKind === 'video'" :src="mediaUrl" controls playsinline class="cd-video">
      您的浏览器不支持视频播放。
    </video>
    <div v-else-if="mediaKind === 'link'" class="cd-link-card">
      <div class="cd-link-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
      </div>
      <div class="cd-link-text">
        <div class="cd-link-label">外部链接</div>
        <div class="cd-link-url">{{ content.url }}</div>
      </div>
      <button class="cd-link-open" type="button" @click="openExternalLink">打开</button>
    </div>
    <div v-else class="cd-text-only">
      <p v-if="noMediaReason" class="cd-no-media-reason">{{ noMediaReason }}</p>
      <p v-else-if="content.text" class="cd-text-content" v-html="renderedText"></p>
      <p v-else>{{ content.title }}</p>
    </div>
  </section>
</template>

<style scoped>
.cd-media-wrap {
  flex: 1 1 60%; min-width: 0; display: flex; align-items: center; justify-content: center;
  background: var(--theme-bg-color); padding: 1rem; overflow: hidden;
}
.cd-media-image { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; cursor: zoom-in; }
.cd-image { max-width: 100%; max-height: 100%; object-fit: contain; border-radius: 6px; user-select: none; }
.cd-video { width: 100%; max-height: 100%; border-radius: 8px; background: #000; }

.cd-link-card {
  display: flex; align-items: center; gap: 1rem; width: min(100%, 560px);
  padding: 1.25rem 1.5rem; background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 12px;
}
.cd-link-icon {
  width: 48px; height: 48px; border-radius: 50%; background: var(--theme-hover-bg);
  display: flex; align-items: center; justify-content: center;
  color: var(--theme-primary); flex-shrink: 0;
}
.cd-link-icon svg { width: 22px; height: 22px; }
.cd-link-text { flex: 1; min-width: 0; }
.cd-link-label { font-size: 0.875rem; font-weight: 600; color: var(--theme-text); margin-bottom: 4px; }
.cd-link-url { font-size: 0.8125rem; color: var(--theme-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-link-open {
  padding: 0.5rem 1rem; font-size: 0.8125rem; font-weight: 600;
  color: var(--theme-on-primary); background: var(--theme-primary);
  border: none; border-radius: 6px; cursor: pointer; flex-shrink: 0;
}
.cd-link-open:hover { filter: brightness(0.92); }

.cd-text-only { padding: 2rem; text-align: center; font-size: 1.125rem; color: var(--theme-text-secondary); max-width: 600px; }
.cd-no-media-reason {
  padding: 3rem 1rem; font-size: 0.875rem; color: var(--theme-text-tertiary);
  background: var(--theme-placeholder-bg); border-radius: 0.5rem; border: 1px dashed var(--theme-card-border);
}
.cd-text-content { text-align: left; line-height: 1.7; }
</style>

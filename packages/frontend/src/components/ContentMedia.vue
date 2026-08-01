<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { getImageUrl, renderMarkdown } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
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
      <MediaImage :src="mediaUrl" :alt="content.title" class="cd-image" :preview="false" draggable="false" />
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
  background: var(--color-bg); padding: 1rem; overflow: hidden;
}
.cd-media-image { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; cursor: zoom-in; min-height: 0; }
/* Arco <Image> 的 .arco-image 包裹层：填满媒体区并居中，作为内部 .arco-image-img 的百分比高度基准 */
.cd-image {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
/* 关键：约束内部真实 .arco-image-img，等比缩放至完整可见（适应/contain），不裁切、不溢出 */
.cd-image :deep(.arco-image-img) {
  display: block;
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 6px;
  user-select: none;
}
.cd-video { width: 100%; max-height: 100%; border-radius: 8px; background: #000; }

.cd-link-card {
  display: flex; align-items: center; gap: 1rem; width: min(100%, 560px);
  padding: 1.25rem 1.5rem; background: var(--color-surface);
  border: 1px solid var(--color-border); border-radius: 12px;
}
.cd-link-icon {
  width: 48px; height: 48px; border-radius: 50%; background: var(--color-hover);
  display: flex; align-items: center; justify-content: center;
  color: var(--color-primary); flex-shrink: 0;
}
.cd-link-icon svg { width: 22px; height: 22px; }
.cd-link-text { flex: 1; min-width: 0; }
.cd-link-label { font-size: 0.875rem; font-weight: 600; color: var(--color-text); margin-bottom: 4px; }
.cd-link-url { font-size: 0.8125rem; color: var(--color-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-link-open {
  padding: 0.5rem 1rem; font-size: 0.8125rem; font-weight: 600;
  color: var(--color-on-primary); background: var(--color-primary);
  border: none; border-radius: 6px; cursor: pointer; flex-shrink: 0;
}
.cd-link-open:hover { filter: brightness(0.92); }

.cd-text-only {
  width: 100%;
  height: 100%;
  max-width: 760px;
  margin: 0 auto;
  padding: 2.25rem 2.5rem 3rem;
  overflow-y: auto;
  text-align: center;
  font-size: 1.125rem;
  color: var(--color-text-secondary);
}
.cd-no-media-reason {
  padding: 3rem 1rem; font-size: 0.875rem; color: var(--color-text-tertiary);
  background: var(--color-placeholder); border-radius: 0.5rem; border: 1px dashed var(--color-border);
}
.cd-text-content {
  text-align: left;
  font-size: 1rem;
  line-height: 1.85;
  color: var(--color-text-1);
  word-break: break-word;
}
.cd-text-content :deep(p) { margin: 0 0 1.1em; }
.cd-text-content :deep(p:last-child) { margin-bottom: 0; }
.cd-text-content :deep(h1),
.cd-text-content :deep(h2),
.cd-text-content :deep(h3),
.cd-text-content :deep(h4) {
  margin: 1.4em 0 0.6em;
  font-weight: 700;
  line-height: 1.4;
  color: var(--color-text-1);
}
.cd-text-content :deep(h1) { font-size: 1.5rem; }
.cd-text-content :deep(h2) { font-size: 1.3rem; }
.cd-text-content :deep(h3) { font-size: 1.15rem; }
.cd-text-content :deep(h4) { font-size: 1.05rem; }
.cd-text-content :deep(h1:first-child),
.cd-text-content :deep(h2:first-child) { margin-top: 0; }
.cd-text-content :deep(ul),
.cd-text-content :deep(ol) { margin: 0 0 1.1em; padding-left: 1.6em; }
.cd-text-content :deep(li) { margin: 0.3em 0; }
.cd-text-content :deep(blockquote) {
  margin: 0 0 1.1em;
  padding: 0.6em 1em;
  border-left: 3px solid rgb(var(--primary-6));
  background: var(--color-fill-1);
  border-radius: 0 8px 8px 0;
  color: var(--color-text-2);
}
.cd-text-content :deep(pre) {
  margin: 0 0 1.1em;
  padding: 0.875rem 1rem;
  background: var(--color-fill-2);
  border-radius: 8px;
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.6;
}
.cd-text-content :deep(code) {
  font-family: Consolas, 'Courier New', monospace;
  background: var(--color-fill-2);
  padding: 0.15em 0.4em;
  border-radius: 4px;
  font-size: 0.875em;
}
.cd-text-content :deep(pre code) { background: none; padding: 0; font-size: inherit; }
.cd-text-content :deep(a) {
  color: rgb(var(--primary-6));
  text-decoration: none;
  border-bottom: 1px solid rgb(var(--primary-3));
}
.cd-text-content :deep(a:hover) { border-bottom-color: rgb(var(--primary-6)); }
.cd-text-content :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 0.5em auto 1.1em;
  border-radius: 8px;
}
.cd-text-content :deep(table) {
  border-collapse: collapse;
  margin: 0 0 1.1em;
  width: 100%;
  font-size: 0.875rem;
}
.cd-text-content :deep(th),
.cd-text-content :deep(td) {
  border: 1px solid var(--color-border-2);
  padding: 0.5em 0.75em;
  text-align: left;
}
.cd-text-content :deep(th) { background: var(--color-fill-2); font-weight: 600; }
.cd-text-content :deep(hr) { border: none; border-top: 1px solid var(--color-border-2); margin: 1.5em 0; }
.cd-text-content :deep(strong) { font-weight: 600; }

@media (max-width: 768px) {
  .cd-text-only { padding: 1.25rem 1.25rem 2rem; }
  .cd-text-content { font-size: 0.9375rem; }
}
</style>

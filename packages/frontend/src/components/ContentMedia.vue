<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { getImageUrl, renderMarkdown } from '@/utils'
import { cachedImageSource, fallbackChain, markUnreachable, pickImageUrl, probeMirror, recheckNetwork, type MediaSource } from '@/utils/imageSource'
import MediaImage from '@/components/MediaImage.vue'
import SketchPlanet from '@/components/SketchPlanet.vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'
import type { Content } from '@/types'

interface Props {
  content: Content
}

const props = defineProps<Props>()

let viewerInstance: Viewer | null = null

const localUrl = computed(() => {
  if (props.content.img) return getImageUrl(props.content.img)
  if (props.content.video) return getImageUrl(props.content.video)
  return ''
})

const mirrorUrl = computed(() => {
  if (props.content.img) return props.content.mirror_img || ''
  if (props.content.video) return props.content.mirror_video || ''
  return ''
})

const mirror2Url = computed(() => {
  if (props.content.img) return props.content.mirror2_img || ''
  if (props.content.video) return props.content.mirror2_video || ''
  return ''
})

const mediaKind = computed<'image' | 'video' | 'text'>(() => {
  if (props.content.img) return 'image'
  if (props.content.video) return 'video'
  return 'text'
})

const source = ref<MediaSource | null>(cachedImageSource())
const resolved = computed(() => source.value !== null)

const mediaUrl = computed(() =>
  source.value ? pickImageUrl(localUrl.value, mirrorUrl.value, mirror2Url.value, source.value) : '',
)
const fallbackUrl = computed(() =>
  fallbackChain(localUrl.value, mirrorUrl.value, mirror2Url.value, source.value),
)

let resolveSeq = 0

async function resolveSource() {
  const r2 = mirrorUrl.value
  const qn = mirror2Url.value
  const seq = ++resolveSeq
  if (!r2 && !qn) {
    source.value = 'origin'
    return
  }
  const known = cachedImageSource()
  const usable =
    known === 'origin' ||
    (known === 'mirror' && !!r2) ||
    (known === 'mirror2' && !!qn)
  if (known && usable) {
    source.value = known
  } else {
    const v = await probeMirror(qn, r2)
    if (seq !== resolveSeq) return
    source.value = v
  }
  const changed = await recheckNetwork()
  if (!changed || seq !== resolveSeq || (!mirrorUrl.value && !mirror2Url.value)) return
  const v = await probeMirror(mirror2Url.value, mirrorUrl.value)
  if (seq !== resolveSeq) return
  source.value = v
}

function onMirrorBroken(failedUrl: string) {
  if (failedUrl && failedUrl === mirror2Url.value) {
    markUnreachable('mirror2')
    if (source.value === 'mirror2') source.value = mirrorUrl.value ? 'mirror' : 'origin'
    return
  }
  markUnreachable('mirror')
  if (source.value === 'mirror') source.value = 'origin'
}

watch(
  () => [props.content.id, localUrl.value, mirrorUrl.value, mirror2Url.value] as const,
  resolveSource,
  { immediate: true },
)

const renderedText = computed(() => {
  const t = props.content.text
  return t ? renderMarkdown(t) : ''
})

function openViewerInline() {
  const img = document.querySelector('.cd-media-image img') as HTMLImageElement | null
  if (!img || !img.complete || img.naturalWidth === 0) return
  const originUrl = pickImageUrl(
    props.content.origin ? getImageUrl(props.content.origin) : '',
    props.content.mirror_img || '',
    props.content.mirror2_img || '',
    source.value,
  )
  if (originUrl) img.dataset.origin = originUrl
  viewerInstance = new Viewer(img, {
    navbar: false, zIndex: 10000, zIndexInline: 10000,
    url(imgEl: HTMLImageElement) { return (imgEl as HTMLImageElement).dataset.origin || imgEl.src },
    hidden() { viewerInstance?.destroy(); viewerInstance = null },
  })
  nextTick(() => { viewerInstance?.show() })
}

defineExpose({ mediaKind, mediaUrl })
</script>

<template>
  <section class="cd-media-wrap" :class="{ 'has-ambient': mediaKind === 'image' && !!mediaUrl }">
    <div
      v-if="mediaKind === 'image' && mediaUrl"
      class="cd-media-ambient"
      aria-hidden="true"
      :style="{ backgroundImage: 'url(' + JSON.stringify(mediaUrl) + ')' }"
    ></div>
    <div
      v-if="(mediaKind === 'image' || mediaKind === 'video') && !resolved"
      class="cd-media-loading"
      role="status"
      aria-busy="true"
    >
      <SketchPlanet class="cd-media-loading-planet" loading />
      <p>{{ mediaKind === 'video' ? '正在加载作品视频…' : '正在加载作品图片…' }}</p>
    </div>
    <div
      v-else-if="mediaKind === 'image'"
      class="cd-media-image"
      @click="openViewerInline"
    >
      <MediaImage
        :src="mediaUrl"
        :fallback-src="fallbackUrl"
        :alt="content.title"
        class="cd-image"
        :preview="false"
        draggable="false"
        @fallback="onMirrorBroken"
      >
        <template #loader>
          <div class="cd-media-loading" role="status" aria-busy="true">
            <SketchPlanet class="cd-media-loading-planet" loading />
            <p>正在加载作品图片…</p>
          </div>
        </template>
        <template #error>
          <div class="cd-media-error" role="status">
            <IconImageClose />
            <p>图片加载失败，请稍后重试</p>
          </div>
        </template>
      </MediaImage>
    </div>
    <video v-else-if="mediaKind === 'video'" :src="mediaUrl" controls playsinline class="cd-video">
      您的浏览器不支持视频播放。
    </video>
    <div v-else class="cd-text-only">
      <div v-if="content.text" class="cd-text-content" v-html="renderedText"></div>
      <p v-else>{{ content.title }}</p>
    </div>
  </section>
</template>

<style scoped>
.cd-media-wrap {
  position: relative;
  flex: 1 1 0; min-width: 0; display: flex; align-items: center; justify-content: center;
  background: var(--creative-canvas); padding: 64px 76px 22px; overflow: hidden;
}
.cd-media-ambient {
  position: absolute; inset: -14%;
  background-size: cover;
  background-position: center;
  filter: blur(58px) saturate(1.25);
  opacity: .5;
  transform: scale(1.06);
  pointer-events: none;
}
.cd-media-wrap.has-ambient::after {
  content: '';
  position: absolute; inset: 0;
  pointer-events: none;
  background: radial-gradient(
    118% 92% at 50% 46%,
    transparent 38%,
    color-mix(in srgb, var(--creative-canvas) 62%, transparent) 100%
  );
}
.cd-media-image { position: relative; z-index: 1; width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; cursor: zoom-in; min-height: 0; }
.cd-media-loading, .cd-media-error { width: 100%; height: 100%; min-height: 12rem; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--creative-muted); cursor: default; }
.cd-media-loading p, .cd-media-error p { margin: 0; font-size: 12px; line-height: 1.6; }
.cd-media-loading-planet { width: 52px; height: 52px; color: var(--creative-accent); }
.cd-media-error > svg { width: 36px; height: 36px; color: var(--creative-muted); }
.cd-media-image :deep(.arco-image-footer) { display: none; }
.cd-image {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.cd-image :deep(.arco-image-img) {
  display: block;
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 12px;
  box-shadow: 0 18px 48px rgba(48, 26, 41, .16), 0 2px 8px rgba(48, 26, 41, .08);
  user-select: none;
}
.cd-video { width: 100%; max-height: 100%; border-radius: 8px; background: #000; }

.cd-link-card {
  display: flex; align-items: center; gap: 1rem; width: min(100%, 560px);
  padding: 1.25rem 1.5rem; background: var(--creative-paper);
  border: 1px solid var(--creative-line); border-radius: var(--radius-md);
}
.cd-link-icon {
  width: 48px; height: 48px; border-radius: 50%; background: var(--creative-soft);
  display: flex; align-items: center; justify-content: center;
  color: var(--creative-accent); flex-shrink: 0;
}
.cd-link-icon svg { width: 22px; height: 22px; }
.cd-link-text { flex: 1; min-width: 0; }
.cd-link-label { font-size: 0.875rem; font-weight: 600; color: var(--creative-ink); margin-bottom: 4px; }
.cd-link-url { font-size: 0.8125rem; color: var(--creative-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-link-open {
  padding: 0.5rem 1rem; font-size: 0.8125rem; font-weight: 600;
  color: #fff; background: var(--creative-accent);
  border: none; border-radius: var(--radius-sm); cursor: pointer; flex-shrink: 0;
}
.cd-link-open:hover { background: var(--creative-accent-hover); }

.cd-text-only {
  width: 100%;
  height: 100%;
  max-width: 760px;
  margin: 0 auto;
  padding: 2.25rem 2.5rem 3rem;
  overflow-y: auto;
  text-align: center;
  font-size: 1.125rem;
  color: var(--creative-muted);
}
.cd-no-media-reason {
  padding: 3rem 1rem; font-size: 0.875rem; color: var(--creative-muted);
  background: var(--creative-canvas); border-radius: var(--radius-sm); border: 1px dashed var(--creative-line);
}
.cd-text-content {
  text-align: left;
  padding: 28px 32px;
  border-radius: 12px;
  border: 1px solid var(--creative-line);
  background: var(--creative-paper);
  box-shadow: var(--creative-shadow);
  font-size: 1rem;
  line-height: 1.85;
  color: var(--creative-ink);
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
  color: var(--creative-ink);
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
  border-left: 3px solid var(--creative-accent);
  background: var(--creative-soft);
  border-radius: 0 8px 8px 0;
  color: var(--creative-muted);
}
.cd-text-content :deep(pre) {
  margin: 0 0 1.1em;
  padding: 0.875rem 1rem;
  background: var(--creative-canvas);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  font-size: 0.875rem;
  line-height: 1.6;
}
.cd-text-content :deep(code) {
  font-family: Consolas, 'Courier New', monospace;
  background: var(--creative-canvas);
  padding: 0.15em 0.4em;
  border-radius: 4px;
  font-size: 0.875em;
}
.cd-text-content :deep(pre code) { background: none; padding: 0; font-size: inherit; }
.cd-text-content :deep(a) {
  color: var(--creative-accent);
  text-decoration: none;
  border-bottom: 1px solid color-mix(in srgb, var(--creative-accent) 40%, transparent);
}
.cd-text-content :deep(a:hover) { border-bottom-color: var(--creative-accent); }
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
  border: 1px solid var(--creative-line);
  padding: 0.5em 0.75em;
  text-align: left;
}
.cd-text-content :deep(th) { background: var(--creative-soft); font-weight: 600; }
.cd-text-content :deep(hr) { border: none; border-top: 1px solid var(--creative-line); margin: 1.5em 0; }
.cd-text-content :deep(strong) { font-weight: 600; }

@media (max-width: 1024px) {
  .cd-media-wrap { padding: 64px 68px 18px; }
}
@media (max-width: 768px) {
  .cd-media-wrap { padding: 54px 14px 14px; }
  .cd-text-only { height: auto; overflow: visible; padding: 0; }
  .cd-text-content { font-size: 0.9375rem; padding: 20px; }
}
</style>

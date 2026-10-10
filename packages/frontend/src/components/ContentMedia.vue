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

/** 源站上的大图地址。 */
const localUrl = computed(() => {
  if (props.content.img) return getImageUrl(props.content.img)
  if (props.content.video) return getImageUrl(props.content.video)
  return ''
})

/** R2 上的同一份副本（后端给的是绝对地址）；未接入 R2 时为空。 */
const mirrorUrl = computed(() => {
  if (props.content.img) return props.content.mirror_img || ''
  if (props.content.video) return props.content.mirror_video || ''
  return ''
})

/** 首选镜像（自建 OpenList）上的同一份副本；未配公开域名时为空。 */
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

/**
 * 来源确定后才渲染。原先「先用源站渲染、测速完再换源」会让同一张图被请求两次
 * （源站一次 + 镜像一次），F12 里就是 api39 → r2 → api39 的来回跳。
 * 现在：本机已有同一网络的结论就同步渲染（零等待、零重复请求）；
 * 没有结论才等一次镜像探测——这是冷启动唯一的一次等待。
 */
const source = ref<MediaSource | null>(cachedImageSource())
const resolved = computed(() => source.value !== null)

const mediaUrl = computed(() =>
  source.value ? pickImageUrl(localUrl.value, mirrorUrl.value, mirror2Url.value, source.value) : '',
)
/** 兜底链：当前这级挂掉后按序退到下一级（OpenList → R2 → 源站）。 */
const fallbackUrl = computed(() =>
  fallbackChain(localUrl.value, mirrorUrl.value, mirror2Url.value, source.value),
)

let resolveSeq = 0

async function resolveSource() {
  const r2 = mirrorUrl.value
  const qn = mirror2Url.value
  const seq = ++resolveSeq
  if (!r2 && !qn) {
    // 两级都没地址（镜像未启用 / 该内容没有副本）：立即用源站，不必探测。
    source.value = 'origin'
    return
  }
  // 缓存里选中的那一级必须**当前真的有地址**，否则视为失效（比如首选未配公开
  // 域名后 mirror2_* 变空，而本机还留着上次「首选可达」的结论）。
  const known = cachedImageSource()
  const usable =
    known === 'origin' ||
    (known === 'mirror' && !!r2) ||
    (known === 'mirror2' && !!qn)
  if (known && usable) {
    source.value = known
  } else {
    const v = await probeMirror(qn, r2)
    if (seq !== resolveSeq) return // 期间切了内容则丢弃，避免张冠李戴
    source.value = v
  }
  // 后台校验公网 IP：网络变了（典型是代理开/关）就推翻旧结论再探一次。
  const changed = await recheckNetwork()
  if (!changed || seq !== resolveSeq || (!mirrorUrl.value && !mirror2Url.value)) return
  const v = await probeMirror(mirror2Url.value, mirrorUrl.value)
  if (seq !== resolveSeq) return
  source.value = v
}

/**
 * 渲染层发现某一级取不到：给**那一级**记内存负标记（不落盘，60s 冷却后自动
 * 恢复资格——渲染失败可能是瞬时抖动，判死交给探测），并立刻往下退一级。
 * 只标死失败的那一级——OpenList 挂了不连累 R2，R2 挂着也不影响 OpenList 的结论，
 * 它正是链上该顶上的下一级。
 */
function onMirrorBroken(failedUrl: string) {
  if (failedUrl && failedUrl === mirror2Url.value) {
    markUnreachable('mirror2')
    // 首选失败 → 退次选（R2），没有次选才回源站。
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
  // 查看原图同样跟随来源选择，避免「预览快、点开慢」的割裂。
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
  <section class="cd-media-wrap">
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
  flex: 1 1 0; min-width: 0; display: flex; align-items: center; justify-content: center;
  background: var(--creative-canvas); padding: 28px; overflow: hidden;
}
.cd-media-image { width: 100%; height: 100%; display: flex; align-items: center; justify-content: center; cursor: zoom-in; min-height: 0; }
/* 冷启动唯一一次等待：镜像可达性尚未有结论时不渲染媒体，避免「先源站后换源」的重复请求 */
.cd-media-loading, .cd-media-error { width: 100%; height: 100%; min-height: 12rem; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--creative-muted); cursor: default; }
.cd-media-loading p, .cd-media-error p { margin: 0; font-size: 12px; line-height: 1.6; }
.cd-media-loading-planet { width: 52px; height: 52px; color: var(--creative-accent); }
.cd-media-error > svg { width: 36px; height: 36px; color: var(--creative-muted); }
/* Arco <Image> 的 .arco-image 包裹层：填满媒体区并居中，作为内部 .arco-image-img 的百分比高度基准 */
/* 标题已在详情页头部展示，关闭 Arco 自动生成的空图片页脚遮罩。 */
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
/* 关键：约束内部真实 .arco-image-img，等比缩放至完整可见（适应/contain），不裁切、不溢出 */
.cd-image :deep(.arco-image-img) {
  display: block;
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 10px;
  box-shadow: 0 8px 32px rgba(48, 26, 41, .07);
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
  background: var(--creative-soft);
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
  .cd-text-only { height: auto; overflow: visible; padding: 0; }
  .cd-text-content { font-size: 0.9375rem; padding: 20px; }
}
</style>

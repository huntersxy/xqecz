<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'
import './prose.css'
import { getImageUrl, getRemoteFallbackUrl, renderMarkdown } from '@/utils'
import {
  cachedImageSource, markUnreachable, pickImageUrl, probeMirror, recheckNetwork, type MediaSource,
} from '@/utils/imageSource'
import { mediaKindOf, mediaTiers, resolveMedia, rewriteMediaInHtml } from '@/utils/contentMedia'
import SketchPlanet from '@/components/SketchPlanet.vue'
import CdIcon from './CdIcon.vue'
import type { Content } from '@/types'

const props = defineProps<{ content: Content }>()

const kind = computed(() => mediaKindOf(props.content))

/**
 * 来源确定后才发请求，避免「先源站、测速后换镜像」把同一张图拉两遍。
 * 本机已有同网络的结论就同步渲染；没有才等一次镜像探测（冷启动唯一一次等待）。
 */
const source = ref<MediaSource | null>(cachedImageSource())
const resolved = computed(() => source.value !== null)

/**
 * 候选地址，按优先级：当前来源 → 其余镜像 → 源站 → 生产服务器同路径。
 * 一个 <img> 只请求当前这一项，失败才换下一项，所以任何时刻最多一个在途请求。
 */
const candidates = computed(() => {
  if (kind.value === 'text' || !source.value) return [] as string[]
  const { url, fallbacks } = resolveMedia(props.content, source.value)
  const list = [url, ...fallbacks, getRemoteFallbackUrl(mediaTiers(props.content).local)]
  return list.filter((u, i) => u && list.indexOf(u) === i)
})

const attempt = ref(0)
const phase = ref<'loading' | 'loaded' | 'failed'>('loading')
const currentUrl = computed(() => candidates.value[attempt.value] ?? '')

watch(
  () => props.content.id + '|' + candidates.value.join('\n'),
  () => { attempt.value = 0; phase.value = 'loading' },
)

let resolveSeq = 0
async function resolveSource() {
  const mine = ++resolveSeq
  if (kind.value === 'text') return
  const { mirror, mirror2 } = mediaTiers(props.content)
  if (!mirror && !mirror2) { source.value = 'origin'; return } // 没有镜像副本，无需探测
  // 缓存里选中的那一级必须当前真的有地址，否则视为失效
  const known = cachedImageSource()
  const usable = known === 'origin' || (known === 'mirror' && !!mirror) || (known === 'mirror2' && !!mirror2)
  if (known && usable) {
    source.value = known
  } else {
    const v = await probeMirror(mirror2, mirror)
    if (mine !== resolveSeq) return // 期间切了内容，丢弃
    source.value = v
  }
  // 后台校验公网 IP：换网（典型是代理开关）才重新探测。图已经显示出来就不再换源，免得闪一下。
  const changed = await recheckNetwork()
  if (!changed || mine !== resolveSeq || phase.value === 'loaded') return
  const v = await probeMirror(mirror2, mirror)
  if (mine !== resolveSeq) return
  source.value = v
}
watch(
  () => { const t = mediaTiers(props.content); return [props.content.id, t.local, t.mirror, t.mirror2].join('|') },
  resolveSource,
  { immediate: true },
)

function onLoad() { phase.value = 'loaded' }

/** 某一级取不到：给那一级记内存负标记，换下一个候选；候选用尽才算失败。 */
function onError() {
  const failed = currentUrl.value
  const { mirror, mirror2 } = mediaTiers(props.content)
  if (failed && failed === mirror2) markUnreachable('mirror2')
  else if (failed && failed === mirror) markUnreachable('mirror')
  if (attempt.value + 1 < candidates.value.length) attempt.value++
  else phase.value = 'failed'
}

function retry() { attempt.value = 0; phase.value = 'loading' }

const renderedText = computed(() => {
  const t = props.content.text
  return t ? rewriteMediaInHtml(renderMarkdown(t)) : ''
})

// ── 查看原图 ──
const imgEl = ref<HTMLImageElement | null>(null)
let viewer: Viewer | null = null
function openViewer() {
  const img = imgEl.value
  if (!img || phase.value !== 'loaded') return
  const c = props.content
  // 原图同样跟随来源选择，避免「预览快、点开慢」的割裂
  const originUrl = pickImageUrl(c.origin ? getImageUrl(c.origin) : '', c.mirror_img || '', c.mirror2_img || '', source.value)
  if (originUrl) img.dataset.origin = originUrl
  viewer?.destroy()
  viewer = new Viewer(img, {
    navbar: false, zIndex: 10000, zIndexInline: 10000,
    url(el: HTMLImageElement) { return el.dataset.origin || el.src },
    hidden() { viewer?.destroy(); viewer = null },
  })
  nextTick(() => viewer?.show())
}
onBeforeUnmount(() => { viewer?.destroy(); viewer = null })
</script>

<template>
  <section
    class="cd-media"
    :class="{ 'is-text': kind === 'text', 'has-ambient': kind === 'image' && phase === 'loaded' }"
  >
    <!-- 虚化底：只在正图真正加载完成后才出现，且复用正图已缓存的同一地址，不额外请求 -->
    <img
      v-if="kind === 'image' && phase === 'loaded' && currentUrl"
      class="cd-ambient"
      :src="currentUrl"
      alt=""
      aria-hidden="true"
      decoding="async"
    />

    <div class="cd-media-frame">
      <template v-if="kind !== 'text'">
        <div v-if="!resolved || (kind === 'image' && phase === 'loading')" class="cd-media-state" role="status" aria-busy="true">
          <SketchPlanet class="cd-media-planet" loading />
          <p>{{ kind === 'video' ? '正在加载作品视频…' : '正在加载作品图片…' }}</p>
        </div>
        <div v-else-if="phase === 'failed'" class="cd-media-state is-error" role="status">
          <CdIcon name="image-broken" class="cd-media-broken" />
          <p>{{ kind === 'video' ? '视频加载失败，请稍后重试' : '图片加载失败，请稍后重试' }}</p>
          <button type="button" class="cd-retry" @click="retry"><CdIcon name="refresh" />重试</button>
        </div>

        <template v-if="resolved && currentUrl && phase !== 'failed'">
          <img
            v-if="kind === 'image'"
            :key="currentUrl"
            ref="imgEl"
            class="cd-image"
            :class="{ 'is-loaded': phase === 'loaded' }"
            :src="currentUrl"
            :alt="content.title"
            draggable="false"
            decoding="async"
            @load="onLoad"
            @error="onError"
            @click="openViewer"
          />
          <video
            v-else
            :key="currentUrl"
            class="cd-video"
            :src="currentUrl"
            controls
            playsinline
            @loadeddata="onLoad"
            @error="onError"
          >您的浏览器不支持视频播放。</video>
        </template>
      </template>

      <div v-else class="cd-text-only">
        <div v-if="content.text" class="cd-prose cd-text-card" v-html="renderedText"></div>
        <p v-else class="cd-text-fallback">{{ content.title }}</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.cd-media {
  position: relative; flex: 1 1 0; min-width: 0; min-height: 0; overflow: hidden;
  background: var(--creative-canvas);
}
/* 虚化底：向外扩一圈再模糊，边缘不留裁切线；只作氛围 */
.cd-ambient {
  position: absolute; inset: -14%; width: 128%; height: 128%; object-fit: cover;
  filter: blur(58px) saturate(1.25); opacity: .5; transform: scale(1.06); pointer-events: none;
  animation: cd-ambient-in .6s ease-out;
}
@keyframes cd-ambient-in { from { opacity: 0; } to { opacity: .5; } }
.cd-media.has-ambient::after {
  content: ''; position: absolute; inset: 0; pointer-events: none;
  background: radial-gradient(118% 92% at 50% 46%, transparent 38%, color-mix(in srgb, var(--creative-canvas) 62%, transparent) 100%);
}
/* 作品区：上方避开悬浮的返回键/计数，左右避开翻页键 */
.cd-media-frame {
  position: absolute; inset: 64px 76px 22px; z-index: 1;
  display: flex; align-items: center; justify-content: center; min-height: 0;
}
.cd-image {
  max-width: 100%; max-height: 100%; width: auto; height: auto; object-fit: contain;
  border-radius: 12px; box-shadow: 0 18px 48px rgba(48, 26, 41, .16), 0 2px 8px rgba(48, 26, 41, .08);
  opacity: 0; transition: opacity .25s ease; cursor: zoom-in; user-select: none;
}
.cd-image.is-loaded { opacity: 1; }
.cd-video { max-width: 100%; max-height: 100%; border-radius: 8px; background: #000; }

.cd-media-state {
  position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 14px; color: var(--creative-muted); pointer-events: none;
}
.cd-media-state.is-error { pointer-events: auto; }
.cd-media-state p { margin: 0; font-size: 12px; line-height: 1.6; }
.cd-media-planet { width: 52px; height: 52px; color: var(--creative-accent); }
.cd-media-broken { font-size: 36px; }
.cd-retry {
  display: inline-flex; align-items: center; gap: 6px; height: 32px; padding: 0 14px;
  border: 1px solid var(--creative-line); border-radius: 999px; background: var(--creative-paper);
  color: var(--creative-accent); font: inherit; font-size: 12px; cursor: pointer;
}
.cd-retry:hover { border-color: var(--creative-accent); background: var(--creative-soft); }

.cd-text-only { width: 100%; height: 100%; max-width: 760px; overflow-y: auto; scrollbar-width: thin; }
.cd-text-card {
  padding: 28px 32px; border: 1px solid var(--creative-line); border-radius: 12px;
  background: var(--creative-paper); box-shadow: var(--creative-shadow); font-size: 1rem;
}
.cd-text-fallback { margin: 0; padding: 3rem 1rem; text-align: center; color: var(--creative-muted); }

@media (max-width: 1024px) { .cd-media-frame { inset: 64px 68px 18px; } }
@media (max-width: 768px) {
  .cd-media { flex: 0 0 auto; height: 46dvh; min-height: 200px; max-height: 56dvh; }
  .cd-media-frame { inset: 54px 14px 14px; }
  /* 纯文字作品：高度随正文，整页滚动 */
  .cd-media.is-text { height: auto; max-height: none; }
  .cd-media.is-text .cd-media-frame { position: static; padding: 54px 14px 14px; }
  .cd-text-only { height: auto; overflow: visible; max-width: none; }
  .cd-text-card { padding: 20px; font-size: .9375rem; }
}
</style>

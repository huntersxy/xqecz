<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import {
  IconDelete,
  IconHeart,
  IconHeartFill,
  IconHistory,
  IconImage,
  IconLoading,
  IconPlus,
} from '@arco-design/web-vue/es/icon'
import CreativeImage from '@/components/CreativeImage.vue'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import type { AIAsset, AITaskView } from '@/types/schemas'
import {
  clockOf,
  dayOf,
  isFailedStatus,
  isPendingStatus,
  stateTileLabel,
  stateTileRatio,
  type GalleryItem,
} from '@/utils/studioTask'

/**
 * 「全部结果」瀑布流。布局算法归 useWaterfallLayout（与首页同一份），这里只负责：
 * 1. 成图卡片按资产宽高比预留高度，状态卡片按任务画幅预留高度——卡片高度由宽度确定，
 *    首次排版不依赖图片解码，也就不会出现「先叠在一起再弹开」；
 * 2. 数据集合变化、卡片高度变化后合并到一帧做一次全量重排（全量重排是幂等的，不会来回抖动）；
 * 3. 新卡片直接落在目标位置，已有卡片位置变化才用过渡滑动。
 */
const props = defineProps<{
  items: GalleryItem[]
  ready: boolean
  /** 搜索/状态筛选生效中：空列表时提示「没有匹配」而不是「还没有作品」。 */
  filtered: boolean
}>()
const emit = defineEmits<{
  open: [row: AITaskView, asset: AIAsset]
  retry: [row: AITaskView]
  remove: [row: AITaskView]
  like: [asset: AIAsset]
  create: []
  'clear-filters': []
}>()

const masonry = ref<HTMLElement | null>(null)
const layoutItems = computed(() => props.items.map((item) => ({ id: item.key })))
const waterfall = useWaterfallLayout(masonry, layoutItems)

// 已经落过位的卡片才允许位置过渡；新卡片首次出现时直接在目标位置，不从左上角飞入。
const placed = shallowRef<ReadonlySet<string>>(new Set())
let epoch = 0
let frames: number[] = []
let cellObserver: ResizeObserver | null = null
let observedKey = ''

function cancelFrames() {
  for (const id of frames) cancelAnimationFrame(id)
  frames = []
}

/** 让 ResizeObserver 盯住当前所有卡片：文字换行、字体加载导致的高度变化会触发重排。 */
function observeCells() {
  const el = masonry.value
  const key = props.items.map((item) => item.key).join('|')
  if (!el || typeof ResizeObserver === 'undefined') return
  if (cellObserver && key === observedKey) return
  observedKey = key
  cellObserver?.disconnect()
  cellObserver = new ResizeObserver(() => scheduleSettle(false))
  for (const cell of el.querySelectorAll<HTMLElement>('[data-wf-id]')) cellObserver.observe(cell)
}

/**
 * 全量重排并在下一帧复核一次：第一遍卡片可能还是兜底宽度，高度不准；
 * 第二遍宽度已落地，按真实高度重新平衡各列。两遍都发生在同一次绘制之前，用户看不到中间态。
 */
function scheduleSettle(initial: boolean) {
  const token = ++epoch
  cancelFrames()
  if (!masonry.value) return
  waterfall.relayout({ anchor: false, full: true })
  frames.push(
    requestAnimationFrame(() => {
      if (token !== epoch) return
      waterfall.relayout({ anchor: !initial, full: true })
      observeCells()
      frames.push(
        requestAnimationFrame(() => {
          if (token !== epoch) return
          placed.value = new Set(props.items.map((item) => item.key))
        }),
      )
    }),
  )
}

watch(
  () => props.items.map((item) => item.key).join('|'),
  () => void nextTick(() => scheduleSettle(false)),
)
// 容器只在有内容时存在：出现时做首次排版，消失（切到单个任务/清空）时放掉观察器与计时。
watch(masonry, (el) => {
  if (el) {
    placed.value = new Set()
    void nextTick(() => scheduleSettle(true))
  } else {
    epoch++
    cancelFrames()
    cellObserver?.disconnect()
    cellObserver = null
    observedKey = ''
  }
})
onBeforeUnmount(() => {
  epoch++
  cancelFrames()
  cellObserver?.disconnect()
  cellObserver = null
})

function cellStyle(key: string) {
  const pos = waterfall.positions.value.get(key)
  if (!pos) return { visibility: 'hidden' as const }
  return {
    width: `${pos.w}px`,
    transform: `translate3d(${pos.x}px, ${pos.y}px, 0)`,
  }
}
function mediaRatio(asset: AIAsset) {
  return asset.width && asset.height ? `${asset.width} / ${asset.height}` : '3 / 4'
}
const skeletonRatios = ['3 / 4', '1 / 1', '4 / 5', '2 / 3', '1 / 1', '3 / 4', '4 / 3', '4 / 5']
</script>

<template>
  <div class="ws-gallery" aria-label="全部结果">
    <div v-if="!ready" class="ws-gallery-skeleton" role="status">
      <span class="sr-only">正在恢复创作…</span>
      <span
        v-for="(ratio, i) in skeletonRatios"
        :key="i"
        class="ws-skeleton-tile"
        :style="{ aspectRatio: ratio }"
        aria-hidden="true"
      />
    </div>

    <div v-else-if="!items.length" class="ws-gallery-empty">
      <span class="ws-gallery-empty-mark"><IconImage /></span>
      <template v-if="filtered">
        <strong>没有匹配的任务</strong>
        <p>当前筛选下还没有生成结果。</p>
        <button type="button" class="ws-pill-btn" @click="emit('clear-filters')">清除筛选</button>
      </template>
      <template v-else>
        <strong>这里会收纳你所有的生成结果</strong>
        <p>写下画面，或添加一张参考图。每次尝试都会留在这里。</p>
        <button type="button" class="ws-pill-btn is-primary" @click="emit('create')">
          <IconPlus />开始创作
        </button>
      </template>
    </div>

    <div
      v-else
      ref="masonry"
      class="ws-masonry"
      :style="{ height: `${waterfall.containerHeight.value}px` }"
    >
      <div
        v-for="(item, order) in items"
        :key="item.key"
        class="ws-cell"
        :class="{ 'is-placed': placed.has(item.key) }"
        :data-wf-id="item.key"
        :style="cellStyle(item.key)"
      >
        <article
          v-if="item.asset"
          class="ws-gallery-card"
          :style="{ '--enter-delay': `${Math.min(order, 12) * 28}ms` }"
        >
          <button
            type="button"
            class="ws-gallery-open"
            :aria-label="`打开图片 ${item.asset.id}`"
            :title="item.row.task.prompt"
            @click="emit('open', item.row, item.asset)"
          >
            <span class="ws-gallery-media" :style="{ aspectRatio: mediaRatio(item.asset) }">
              <CreativeImage :id="item.asset.id" />
            </span>
            <span v-if="item.total > 1" class="ws-gallery-index" aria-hidden="true">
              {{ item.index + 1 }}/{{ item.total }}
            </span>
            <span class="ws-gallery-veil">
              <span class="ws-gallery-veil-prompt">{{ item.row.task.prompt || '（无提示词）' }}</span>
              <span class="ws-gallery-veil-meta">
                {{ dayOf(item.row.task.created_at) }} · {{ clockOf(item.row.task.created_at) }}
              </span>
            </span>
          </button>
          <button
            type="button"
            class="ws-gallery-like"
            :class="{ 'is-liked': item.asset.liked }"
            :aria-pressed="!!item.asset.liked"
            :aria-label="item.asset.liked ? `取消喜欢图片 ${item.asset.id}` : `喜欢图片 ${item.asset.id}`"
            @click="emit('like', item.asset)"
          >
            <IconHeartFill v-if="item.asset.liked" />
            <IconHeart v-else />
          </button>
        </article>

        <article
          v-else
          class="ws-gallery-card is-state"
          :class="{
            'is-error': isFailedStatus(item.row.task.status),
            'is-pending': isPendingStatus(item.row.task.status),
          }"
          :style="{
            aspectRatio: stateTileRatio(item.row.task.size),
            '--enter-delay': `${Math.min(order, 12) * 28}ms`,
          }"
        >
          <span class="ws-state-icon">
            <IconLoading v-if="isPendingStatus(item.row.task.status)" class="ws-spin" />
            <IconImage v-else />
          </span>
          <strong>{{ stateTileLabel(item.row.task.status) }}</strong>
          <p class="ws-gallery-prompt">{{ item.row.task.prompt }}</p>
          <p v-if="item.row.task.error" class="ws-gallery-error">{{ item.row.task.error }}</p>
          <div v-if="isFailedStatus(item.row.task.status)" class="ws-gallery-note-actions">
            <button
              type="button"
              aria-label="复用提示词重试"
              title="复用提示词重试"
              @click="emit('retry', item.row)"
            >
              <IconHistory /><span>重试</span>
            </button>
            <button
              type="button"
              aria-label="删除任务记录"
              title="删除任务记录"
              @click="emit('remove', item.row)"
            >
              <IconDelete /><span>删除</span>
            </button>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 滚动容器自带上下羽化：卡片滑出视口时淡出，而不是被硬边切掉；顶部留白给羽化让位。 */
.ws-gallery {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 6px 120px;
  scrollbar-width: thin;
  scrollbar-color: var(--site-line) transparent;
  overscroll-behavior: contain;
  mask-image: linear-gradient(to bottom, transparent 0, #000 14px, #000 calc(100% - 56px), transparent 100%);
}
.ws-masonry {
  position: relative;
  width: 100%;
}
.ws-cell {
  position: absolute;
  top: 0;
  left: 0;
  /* 兜底宽度：布局算出位置之前卡片已经按一列宽度排出真实高度，visibility 隐藏避免闪现。 */
  width: 240px;
  will-change: transform;
}
.ws-cell.is-placed {
  transition:
    transform 0.42s cubic-bezier(0.22, 1, 0.36, 1),
    width 0.42s cubic-bezier(0.22, 1, 0.36, 1);
}

.ws-gallery-card {
  position: relative;
  overflow: hidden;
  border-radius: 16px;
  background: var(--site-soft);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--site-ink) 6%, transparent),
    0 8px 22px color-mix(in srgb, var(--site-ink) 8%, transparent);
  animation: ws-tile-in 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: var(--enter-delay, 0ms);
  transition: box-shadow 0.28s ease, transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}
@keyframes ws-tile-in {
  from {
    opacity: 0;
    transform: translateY(10px) scale(0.97);
  }
}
.ws-gallery-card:not(.is-state):hover,
.ws-gallery-card:not(.is-state):focus-within {
  box-shadow: 0 2px 4px color-mix(in srgb, var(--site-ink) 8%, transparent),
    0 18px 38px color-mix(in srgb, var(--site-ink) 18%, transparent);
  transform: translateY(-2px);
}
.ws-gallery-open {
  position: relative;
  display: block;
  width: 100%;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: zoom-in;
}
.ws-gallery-media {
  display: block;
  width: 100%;
  overflow: hidden;
  background: var(--site-soft);
}
.ws-gallery-media .creative-image {
  min-height: 0;
}
.ws-gallery-media :deep(.creative-image img) {
  object-fit: cover;
  transition: transform 0.6s cubic-bezier(0.22, 1, 0.36, 1);
}
.ws-gallery-card:hover .ws-gallery-media :deep(.creative-image img),
.ws-gallery-card:focus-within .ws-gallery-media :deep(.creative-image img) {
  transform: scale(1.045);
}
.ws-gallery-index {
  position: absolute;
  top: 9px;
  left: 9px;
  padding: 2px 8px;
  border-radius: 999px;
  background: rgba(24, 18, 20, 0.5);
  color: #fff;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.04em;
  backdrop-filter: blur(8px);
}
/* 提示词与时间浮在图片底部的渐变上；固定深色渐变保证日夜主题下文字都可读。 */
.ws-gallery-veil {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 36px 12px 11px;
  background: linear-gradient(to top, rgba(20, 14, 16, 0.78), rgba(20, 14, 16, 0));
  color: #fff;
  opacity: 0;
  transform: translateY(6px);
  transition: opacity 0.28s ease, transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
  pointer-events: none;
}
.ws-gallery-card:hover .ws-gallery-veil,
.ws-gallery-card:focus-within .ws-gallery-veil {
  opacity: 1;
  transform: none;
}
.ws-gallery-veil-prompt {
  display: -webkit-box;
  overflow: hidden;
  font-size: 12px;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.ws-gallery-veil-meta {
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  opacity: 0.72;
}
.ws-gallery-like {
  position: absolute;
  top: 8px;
  right: 8px;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: rgba(24, 18, 20, 0.46);
  color: #fff;
  opacity: 0;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: opacity 0.2s ease, transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1), background 0.2s ease;
}
.ws-gallery-like svg {
  width: 15px;
  height: 15px;
}
.ws-gallery-card:hover .ws-gallery-like,
.ws-gallery-card:focus-within .ws-gallery-like,
.ws-gallery-like.is-liked {
  opacity: 1;
}
.ws-gallery-like:hover {
  background: rgba(24, 18, 20, 0.66);
  transform: scale(1.1);
}
.ws-gallery-like.is-liked {
  color: #ff8fa8;
}
@media (hover: none) {
  .ws-gallery-like {
    opacity: 1;
  }
  .ws-gallery-veil {
    padding-top: 28px;
    opacity: 1;
    transform: none;
  }
  .ws-gallery-veil-prompt {
    -webkit-line-clamp: 1;
  }
  .ws-gallery-veil-meta {
    display: none;
  }
}

/* 无图任务：排队/生成中用流光占位，失败用淡红底，二者共用画幅预留的高度。 */
.ws-gallery-card.is-state {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-height: 176px;
  padding: 16px;
  border: 1px solid var(--site-line);
  background: var(--site-paper);
}
.ws-gallery-card.is-state.is-pending {
  border-color: color-mix(in srgb, var(--site-accent) 22%, var(--site-line));
  background: linear-gradient(
      105deg,
      transparent 30%,
      color-mix(in srgb, var(--site-accent) 11%, transparent) 50%,
      transparent 70%
    )
    0 0 / 220% 100% no-repeat,
    var(--site-paper);
  animation:
    ws-tile-in 0.5s cubic-bezier(0.22, 1, 0.36, 1) both,
    ws-sheen 1.9s linear infinite;
  animation-delay: var(--enter-delay, 0ms), 0ms;
}
@keyframes ws-sheen {
  from {
    background-position: 160% 0, 0 0;
  }
  to {
    background-position: -60% 0, 0 0;
  }
}
.ws-gallery-card.is-state.is-error {
  border-color: color-mix(in srgb, rgb(var(--danger-6)) 26%, var(--site-line));
  background: color-mix(in srgb, rgb(var(--danger-6)) 5%, var(--site-paper));
}
.ws-state-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 11px;
  background: var(--site-soft);
  color: var(--site-muted);
}
.ws-state-icon svg {
  width: 17px;
  height: 17px;
}
.is-pending .ws-state-icon {
  background: var(--site-accent-soft);
  color: var(--site-accent);
}
.is-error .ws-state-icon {
  background: color-mix(in srgb, rgb(var(--danger-6)) 12%, transparent);
  color: rgb(var(--danger-6));
}
.ws-gallery-card.is-state > strong {
  color: var(--site-ink);
  font-size: 13px;
}
.ws-gallery-prompt,
.ws-gallery-error {
  margin: 0;
  font-size: 11px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.ws-gallery-prompt {
  display: -webkit-box;
  overflow: hidden;
  color: var(--site-muted);
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}
.ws-gallery-error {
  display: -webkit-box;
  overflow: hidden;
  color: rgb(var(--danger-6));
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
.ws-gallery-note-actions {
  display: flex;
  gap: 6px;
  margin-top: auto;
}
.ws-gallery-note-actions button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 11px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  color: var(--site-muted);
  font-size: 11px;
  cursor: pointer;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
}
.ws-gallery-note-actions button:hover {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
  color: var(--site-accent);
}
.ws-gallery-note-actions svg {
  width: 13px;
  height: 13px;
}
.ws-spin {
  animation: ws-spin 1s linear infinite;
}
@keyframes ws-spin {
  to {
    transform: rotate(360deg);
  }
}

/* 加载骨架与空状态 */
.ws-gallery-skeleton {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
}
.ws-skeleton-tile {
  display: block;
  border-radius: 16px;
  background: linear-gradient(
      105deg,
      transparent 30%,
      color-mix(in srgb, var(--site-paper) 70%, transparent) 50%,
      transparent 70%
    )
    0 0 / 220% 100% no-repeat,
    var(--site-soft);
  animation: ws-sheen 1.6s linear infinite;
}
.ws-gallery-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  max-width: 360px;
  margin: 12vh auto 0;
  text-align: center;
}
.ws-gallery-empty-mark {
  display: grid;
  place-items: center;
  width: 56px;
  height: 56px;
  margin-bottom: 6px;
  border: 1px dashed var(--site-line);
  border-radius: 18px;
  background: var(--site-paper);
  color: var(--site-muted);
}
.ws-gallery-empty-mark svg {
  width: 24px;
  height: 24px;
}
.ws-gallery-empty strong {
  color: var(--site-ink);
  font: 600 17px/1.3 var(--site-title-font);
}
.ws-gallery-empty p {
  margin: 0;
  color: var(--site-muted);
  font-size: 12px;
  line-height: 1.7;
}
.ws-pill-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  height: 34px;
  padding: 0 16px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  color: var(--site-ink);
  font-size: 12px;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, transform 0.15s ease;
}
.ws-pill-btn:hover {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
}
.ws-pill-btn:active {
  transform: translateY(1px);
}
.ws-pill-btn.is-primary {
  border-color: transparent;
  background: var(--site-accent);
  color: var(--site-paper);
}
.ws-pill-btn.is-primary:hover {
  background: color-mix(in srgb, var(--site-accent) 86%, var(--site-ink));
}
.ws-pill-btn svg {
  width: 14px;
  height: 14px;
}

.ws-gallery button:focus-visible {
  outline: 2px solid var(--site-accent);
  outline-offset: 2px;
}

/* 手机：页面整体滚动，画廊不再自己滚动，也不需要羽化遮罩。 */
@media (max-width: 820px) {
  .ws-gallery {
    flex: none;
    overflow: visible;
    padding: 4px 0 8px;
    mask-image: none;
  }
  .ws-gallery-card.is-state {
    min-height: 150px;
    padding: 12px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .ws-gallery-card {
    animation: none;
  }
  .ws-gallery-card.is-state.is-pending {
    animation: ws-sheen 1.9s linear infinite;
  }
}
</style>

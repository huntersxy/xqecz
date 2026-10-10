<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import MediaImage from '@/components/MediaImage.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import { getPreviewText } from '@/utils'
import type { Content } from '@/types'
import { contentApi } from '@/api'
import { toast } from '@/composables/useToast'

interface Props {
  item: Content
}

const props = defineProps<Props>()
const emit = defineEmits<{
  click: [content: Content]
  imageLoaded: [id: string | number]
  /** 点赞结果回传列表：父级持有 allContents，不更新它的话返回首页时缓存会把数量复原。 */
  liked: [payload: { id: string | number; liked: boolean; like_count: number }]
}>()
const likeCount = ref(props.item.like_count || 0)
const isLiked = ref(false)
const isLikePending = ref(false)

async function toggleLike(event: MouseEvent) {
  event.stopPropagation()
  if (isLikePending.value) return
  isLikePending.value = true
  try {
    const res = await contentApi.toggleLike(Number(props.item.id))
    if (res.code === 200) {
      isLiked.value = res.data.liked
      likeCount.value = res.data.like_count
      emit('liked', { id: props.item.id, liked: res.data.liked, like_count: res.data.like_count })
    }
  } catch (err) {
    // request() 已对非 401 失败弹过 toast；只有未登录这一种需要额外引导，
    // 其余情况静默即可，不能在事件回调里把异常放出去（会冒泡到 ErrorBoundary 顶掉整页）。
    if ((err as { status?: number })?.status === 401) {
      toast.info('登录后即可点亮')
    }
  } finally {
    isLikePending.value = false
  }
}

// 纯文本卡：从正文提炼一段摘要（去 Markdown），无正文时回退到标题
const previewText = computed(() => {
  const source = props.item.text || props.item.title || ''
  return getPreviewText(source, 96)
})

// 瀑布流依赖卡片高度变化触发重排：用 ResizeObserver 监听整卡高度，
// 只在高度发生有意义变化时 emit（初始观察回调不重复触发），图片加载/失败、
// 字体变化等都会反映到高度上。
const cardRef = ref<HTMLElement | null>(null)
let ro: ResizeObserver | null = null
let lastHeight = 0

function onCardResize(entry: ResizeObserverEntry) {
  const h = entry.contentRect.height
  if (h > 0 && Math.abs(h - lastHeight) > 0.5) {
    lastHeight = h
    emit('imageLoaded', props.item.id)
  }
}

onMounted(() => {
  if (cardRef.value) {
    ro = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) onCardResize(entry)
    })
    ro.observe(cardRef.value)
  }
})
onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <div ref="cardRef" class="wf-card" @click="emit('click', props.item)" @keydown.enter="emit('click', props.item)" tabindex="0">
    <template v-if="props.item.thumb">
      <div class="wf-card-media">
        <MediaImage :src="props.item.thumb" :alt="props.item.title" :preview="false" loading="lazy" decoding="async" />
        <div v-if="props.item.tags?.some(t => /ai/i.test(t))" class="wf-badge-ai">AI</div>
      </div>
    </template>
    <template v-else>
      <div class="wf-card-text-body">
        <span class="wf-card-text-mark" aria-hidden="true">&ldquo;</span>
        <p class="wf-card-text-excerpt">{{ previewText }}</p>
        <span class="wf-card-text-more">阅读全文</span>
      </div>
    </template>
    <div class="wf-card-info">
      <span class="wf-card-title">{{ props.item.title }}</span>
      <div class="wf-card-meta">
        <span class="wf-card-user">
          <UserAvatar class="wf-card-user-mark" :src="props.item.avatar_url" :email="props.item.user?.email" :username="props.item.user?.username" />
          <span class="wf-card-user-name">{{ props.item.user?.username }}</span>
        </span>
        <button
          class="wf-card-like"
          :class="{ liked: isLiked, pending: isLikePending }"
          type="button"
          :aria-label="isLiked ? '取消点赞' : '点赞'"
          :aria-pressed="isLiked"
          :title="isLiked ? '取消点赞' : '点赞'"
          @click="toggleLike"
        >
          <svg viewBox="0 0 24 24" :fill="isLiked ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="1.8"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
          <span>{{ likeCount }}</span>
        </button>
      </div>
      <div v-if="props.item.tags?.length" class="wf-card-tags">
        <span v-for="tag in props.item.tags.slice(0, 3)" :key="tag" class="wf-mini-tag">{{ tag }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wf-card {
  --card-paper: var(--creative-paper);
  --card-ink: var(--creative-ink);
  --card-muted: var(--creative-muted);
  --card-line: var(--creative-line);
  --card-soft: var(--creative-soft);
  --card-accent: var(--creative-accent);
  margin-bottom: 12px;
  padding: 6px;
  border-radius: 16px;
  background: var(--card-paper);
  border: 1px solid var(--card-line);
  box-shadow: 0 2px 5px rgba(61, 29, 46, .025);
  cursor: pointer;
  box-sizing: border-box;
  /* Preserve content-driven block height for waterfall measurement. */
  container: wfc / inline-size;
  transition: border-color .18s ease, box-shadow .18s ease;
}
.wf-card:focus-visible { outline: 2px solid var(--card-accent); outline-offset: 3px; }
@media (hover: hover) and (pointer: fine) {
  .wf-card:hover { border-color: color-mix(in srgb, var(--card-accent) 45%, var(--card-line)); box-shadow: var(--creative-shadow); }
  .wf-card:hover .wf-card-title { color: var(--card-accent); }
}
.wf-card-media { position: relative; overflow: hidden; border-radius: 11px; min-height: 80px; background: var(--color-placeholder); line-height: 0; }
.wf-card-media :deep(.arco-image) { display: block; width: 100%; min-height: 80px; border-radius: 0; }
.wf-card-media :deep(.arco-image-img) { display: block; width: 100%; height: auto; vertical-align: top; }
.wf-card-media :deep(.arco-image-footer) { display: none !important; }
:global(body[arco-theme='dark'] .wf-card-media::after) { content: ''; position: absolute; inset: 0; background: rgba(0,0,0,.18); pointer-events: none; }
.wf-badge-ai { position: absolute; top: 9px; left: 9px; z-index: 2; padding: 5px 7px; border: 1px solid rgba(255,255,255,.75); border-radius: 6px; background: #fff4f8; color: #a43664; font-size: 9px; font-weight: 700; line-height: 1; letter-spacing: .04em; }
.wf-card-info { padding: 13px 9px 9px; }
.wf-card-title { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; margin-bottom: 10px; font-size: 14px; font-weight: 600; line-height: 1.5; color: var(--card-ink); transition: color .18s ease; }
.wf-card-meta { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.wf-card-user { display: flex; align-items: center; gap: 6px; min-width: 0; font-size: 11px; color: var(--card-muted); }
.wf-card-user-mark { --avatar-size: 22px; }
.wf-card-user-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.wf-card-like { display: inline-flex; align-items: center; gap: 4px; flex-shrink: 0; padding: 3px 0; border: 0; background: transparent; color: var(--card-muted); cursor: pointer; font: inherit; font-size: 11px; font-variant-numeric: tabular-nums; transition: color .18s ease, opacity .18s ease; }
.wf-card-like svg { width: 13px; height: 13px; color: var(--card-accent); transition: transform .18s ease; }
.wf-card-like:hover { color: var(--card-accent); }
.wf-card-like:hover svg { transform: scale(1.12); }
.wf-card-like.liked { color: var(--card-accent); }
.wf-card-like.pending { opacity: .55; cursor: wait; }
.wf-card-tags { display: flex; flex-wrap: wrap; gap: 5px; margin-top: 11px; padding-top: 10px; border-top: 1px solid var(--card-line); }
.wf-mini-tag { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 3px 7px; border-radius: 5px; font-size: 10px; line-height: 1.4; color: var(--card-muted); background: var(--card-soft); }
.wf-card-text-body { position: relative; display: flex; flex-direction: column; min-height: 185px; padding: 20px 16px 15px; border-radius: 11px; background: linear-gradient(145deg, var(--card-soft), var(--card-paper)); }
.wf-card-text-mark { font: 42px/1 Georgia, serif; color: var(--card-accent); opacity: .45; height: 28px; pointer-events: none; }
.wf-card-text-excerpt { display: -webkit-box; -webkit-line-clamp: 5; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; margin: 10px 0 18px; color: var(--card-ink); font-size: 13px; line-height: 1.85; }
.wf-card-text-more { margin-top: auto; align-self: flex-end; font-size: 11px; color: var(--card-accent); }
@container wfc (max-width: 190px) {
  .wf-card-info { padding: 10px 5px 6px; }
  .wf-card-title { font-size: 12px; margin-bottom: 8px; }
  .wf-card-user { gap: 4px; font-size: 10px; }
  .wf-card-user-mark { --avatar-size: 18px; }
  .wf-card-like { font-size: 10px; gap: 3px; }
  .wf-mini-tag { font-size: 9px; padding: 2px 5px; }
  .wf-card-tags { gap: 4px; margin-top: 8px; padding-top: 8px; }
  .wf-card-text-body { padding: 15px 10px 12px; min-height: 150px; }
  .wf-card-text-excerpt { font-size: 12px; -webkit-line-clamp: 4; }
}
@container wfc (min-width: 260px) {
  .wf-card-info { padding: 15px 11px 11px; }
  .wf-card-title { font-size: 15px; }
  .wf-card-text-body { padding: 23px 19px 18px; min-height: 210px; }
  .wf-card-text-excerpt { font-size: 14px; -webkit-line-clamp: 6; }
}
</style>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { IconHeart, IconStar, IconShareAlt, IconDownload } from '@arco-design/web-vue/es/icon'
import { formatFileSize, getImageUrl, getUrlExtension, withDownloadFlag } from '@/utils'
import type { Content } from '@/types'

const props = defineProps<{ content: Content }>()
const userStore = useUserStore()

const likeCount = ref(props.content.like_count || 0)
const isLiked = ref(false)
const isFavorited = ref(false)

async function loadInteractionStatus() {
  if (!userStore.isLoggedIn) return
  try {
    const res = await contentApi.likeStatus(props.content.id)
    if (res.code === 200) {
      isLiked.value = res.data.liked
      isFavorited.value = res.data.favorited
      likeCount.value = res.data.like_count
    }
  } catch { /* 未登录或网络错误，保持默认值 */ }
}

watch(
  () => props.content.id,
  () => {
    isLiked.value = false
    isFavorited.value = false
    likeCount.value = props.content.like_count || 0
    loadInteractionStatus()
  },
  { immediate: true },
)

async function toggleLike() {
  if (!userStore.isLoggedIn) return
  try {
    const res = await contentApi.toggleLike(props.content.id)
    if (res.code === 200) {
      isLiked.value = res.data.liked
      likeCount.value = res.data.like_count
    }
  } catch { /* 忽略 */ }
}

async function toggleFavorite() {
  if (!userStore.isLoggedIn) return
  try {
    const res = await contentApi.toggleFavorite(props.content.id)
    if (res.code === 200) isFavorited.value = res.data.favorited
  } catch { /* 忽略 */ }
}

async function shareContent() {
  const url = globalThis.location.href
  try {
    if (navigator.share) await navigator.share({ title: props.content.title || '', url })
    else if (navigator.clipboard) await navigator.clipboard.writeText(url)
  } catch { /* 用户取消分享，忽略 */ }
}

// ── 下载 ──
// origin 是作品文件；兼容旧数据时只回退到视频或非缩略图的图片。
const originalFile = computed(() => {
  const { origin, video, img, thumb } = props.content
  return origin || video || (img !== thumb ? img : '')
})
const downloadUrl = computed(() => withDownloadFlag(getImageUrl(originalFile.value)))
const isVideo = computed(() => !!props.content.video)
// 大小直接使用详情接口，不为展示元信息额外请求媒体。
const sizeText = computed(() => formatFileSize(props.content.file_size))
const originExt = computed(() => getUrlExtension(originalFile.value))

</script>

<template>
  <footer class="cd-bottombar" aria-label="作品操作">
    <span class="cd-action-note"><span aria-hidden="true">✧</span> 点亮，或收进星图</span>
    <div class="cd-action-tools">
      <div class="cd-interaction-group" aria-label="喜欢与收藏">
        <button
          type="button" class="cd-action cd-like-action" :class="{ active: isLiked }"
          :disabled="!userStore.isLoggedIn" :aria-pressed="isLiked"
          :aria-label="isLiked ? '取消点赞' : '点赞'" :title="userStore.isLoggedIn ? '点亮这颗星球' : '登录后即可点亮'"
          @click="toggleLike"
        >
          <span class="cd-action-icon"><IconHeart :fill="isLiked ? 'currentColor' : 'none'" /></span>
          <span>{{ isLiked ? '已喜欢' : '喜欢' }}</span>
          <span class="cd-like-count">{{ likeCount }}</span>
        </button>
        <span class="cd-action-divider" aria-hidden="true"></span>
        <button
          type="button" class="cd-action" :class="{ active: isFavorited }"
          :disabled="!userStore.isLoggedIn" :aria-pressed="isFavorited"
          :aria-label="isFavorited ? '取消收藏' : '收藏'" :title="userStore.isLoggedIn ? '收进你的星图' : '登录后即可收进星图'"
          @click="toggleFavorite"
        >
          <span class="cd-action-icon"><IconStar :fill="isFavorited ? 'currentColor' : 'none'" /></span>
          <span>{{ isFavorited ? '已收藏' : '收藏' }}</span>
        </button>
      </div>
      <button type="button" class="cd-action cd-share-action" @click="shareContent">
        <IconShareAlt /> <span>分享</span>
      </button>
      <a
        v-if="downloadUrl"
        :href="downloadUrl"
        rel="noopener"
        class="cd-action cd-download-action"
        aria-label="下载作品"
      >
        <span class="cd-download-icon"><IconDownload /></span>
        <span class="cd-download-label"><span>下载作品</span><small>{{ originExt || (isVideo ? 'VIDEO' : '作品文件') }}<template v-if="sizeText"> · {{ sizeText }}</template></small></span>
      </a>
    </div>
  </footer>
</template>

<style scoped>
.cd-bottombar { flex-shrink: 0; position: relative; display: flex; align-items: center; justify-content: center; gap: 24px; padding: 14px 28px; border-top: 1px solid var(--creative-line); background: var(--creative-paper); }
.cd-action-note { position: absolute; left: 28px; display: flex; align-items: center; gap: 9px; color: var(--creative-muted); font-size: 11px; letter-spacing: .06em; }
.cd-action-note > span { color: var(--creative-accent); font-size: 22px; }
.cd-action-tools { display: flex; align-items: center; gap: 14px; min-width: 0; }
.cd-interaction-group { display: flex; align-items: center; padding: 4px; border: 1px solid var(--creative-line); border-radius: 15px; background: var(--creative-canvas); }
.cd-action { display: inline-flex; align-items: center; justify-content: center; gap: 8px; height: 42px; padding: 0 13px; border: 0; border-radius: 11px; background: transparent; color: var(--creative-ink); font: inherit; font-size: 12px; font-weight: 500; white-space: nowrap; cursor: pointer; transition: background .2s, color .2s, box-shadow .2s, transform .2s; }
.cd-action > svg, .cd-action-icon > svg { width: 17px; height: 17px; }
.cd-action-icon { display: grid; place-items: center; color: var(--creative-accent); }
.cd-like-count { min-width: 18px; padding: 2px 5px; border-radius: 5px; background: var(--creative-paper); color: var(--creative-muted); font-size: 10px; font-variant-numeric: tabular-nums; }
.cd-action-divider { width: 1px; height: 18px; margin-inline: 2px; background: var(--creative-line); }
.cd-action:hover:not(:disabled) { color: var(--creative-accent); background: var(--creative-soft); }
.cd-action:active:not(:disabled) { transform: translateY(1px); }
.cd-action.active { color: var(--creative-accent); background: var(--creative-soft); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--creative-accent) 12%, transparent); }
.cd-action:disabled { cursor: not-allowed; color: var(--creative-muted); }
.cd-action:disabled .cd-action-icon { color: color-mix(in srgb, var(--creative-accent) 70%, var(--creative-muted)); }
.cd-share-action { color: var(--creative-muted); }
.cd-download-action { text-decoration: none; height: 50px; padding: 0 16px 0 10px; gap: 11px; border: 1px solid color-mix(in srgb, var(--creative-accent) 22%, var(--creative-line)); border-radius: 15px; background: var(--creative-soft); color: var(--creative-accent); box-shadow: 0 3px 9px color-mix(in srgb, var(--creative-accent) 5%, transparent); }
.cd-download-action:hover { box-shadow: 0 5px 16px color-mix(in srgb, var(--creative-accent) 12%, transparent); }
.cd-download-icon { display: grid; place-items: center; width: 32px; height: 32px; background: var(--creative-paper); border-radius: 10px; }
.cd-download-icon svg { width: 17px; height: 17px; }
.cd-download-label { display: flex; flex-direction: column; align-items: flex-start; gap: 3px; line-height: 1.25; }
.cd-download-label small { font-size: 9px; font-weight: 400; color: var(--creative-muted); letter-spacing: .03em; }
@media (max-width: 1100px) { .cd-action-note { display: none; } }
@media (max-width: 768px) {
  .cd-bottombar { padding: 10px 14px calc(10px + env(safe-area-inset-bottom)); }
  .cd-action-tools { width: 100%; justify-content: center; gap: 8px; }
  .cd-action { height: 40px; padding-inline: 10px; gap: 6px; font-size: 11px; }
  .cd-interaction-group { padding: 3px; border-radius: 13px; }
  .cd-download-action { height: 48px; padding-inline: 8px; gap: 8px; border-radius: 13px; }
  .cd-download-icon { width: 27px; height: 30px; border-radius: 8px; }
}
@media (max-width: 380px) {
  .cd-bottombar { padding-inline: 10px; }
  .cd-action-tools { gap: 5px; }
  .cd-action { padding-inline: 7px; gap: 5px; }
  .cd-share-action { flex-direction: column; gap: 3px; font-size: 10px; }
  .cd-like-count { display: none; }
  .cd-download-icon { display: none; }
  .cd-download-action { padding-inline: 10px; }
}
</style>

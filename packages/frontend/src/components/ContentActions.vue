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
  } catch { }
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
  } catch { }
}

async function toggleFavorite() {
  if (!userStore.isLoggedIn) return
  try {
    const res = await contentApi.toggleFavorite(props.content.id)
    if (res.code === 200) isFavorited.value = res.data.favorited
  } catch { }
}

async function shareContent() {
  const url = globalThis.location.href
  try {
    if (navigator.share) await navigator.share({ title: props.content.title || '', url })
    else if (navigator.clipboard) await navigator.clipboard.writeText(url)
  } catch { }
}

const originalFile = computed(() => {
  const { origin, video, img, thumb } = props.content
  return origin || video || (img !== thumb ? img : '')
})
const downloadUrl = computed(() => withDownloadFlag(getImageUrl(originalFile.value)))
const isVideo = computed(() => !!props.content.video)
const sizeText = computed(() => formatFileSize(props.content.file_size))
const originExt = computed(() => getUrlExtension(originalFile.value))

</script>

<template>
  <section class="cd-actions" aria-label="作品操作">
    <div class="cd-action-row">
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
      <button
        type="button" class="cd-action" :class="{ active: isFavorited }"
        :disabled="!userStore.isLoggedIn" :aria-pressed="isFavorited"
        :aria-label="isFavorited ? '取消收藏' : '收藏'" :title="userStore.isLoggedIn ? '收进你的星图' : '登录后即可收进星图'"
        @click="toggleFavorite"
      >
        <span class="cd-action-icon"><IconStar :fill="isFavorited ? 'currentColor' : 'none'" /></span>
        <span>{{ isFavorited ? '已收藏' : '收藏' }}</span>
      </button>
      <button type="button" class="cd-action cd-share-action" @click="shareContent">
        <IconShareAlt /> <span>分享</span>
      </button>
    </div>
    <a
      v-if="downloadUrl"
      :href="downloadUrl"
      rel="noopener"
      class="cd-download-action"
      aria-label="下载作品"
    >
      <span class="cd-download-icon"><IconDownload /></span>
      <span class="cd-download-label"><span>下载作品</span><small>{{ originExt || (isVideo ? 'VIDEO' : '作品文件') }}<template v-if="sizeText"> · {{ sizeText }}</template></small></span>
    </a>
    <p v-if="!userStore.isLoggedIn" class="cd-action-hint">登录后即可点亮、收藏作品</p>
  </section>
</template>

<style scoped>
.cd-actions { display: flex; flex-direction: column; flex-shrink: 0; gap: 10px; }
.cd-action-row { display: flex; gap: 8px; }
.cd-action {
  flex: 1 1 0; min-width: 0; display: inline-flex; align-items: center; justify-content: center; gap: 7px; height: 42px; padding: 0 10px;
  border: 1px solid var(--creative-line); border-radius: 12px; background: var(--creative-canvas);
  color: var(--creative-ink); font: inherit; font-size: 12px; font-weight: 500; white-space: nowrap; cursor: pointer;
  transition: background .2s, color .2s, border-color .2s, box-shadow .2s, transform .2s;
}
.cd-share-action { flex: 0 0 auto; color: var(--creative-muted); }
.cd-action > svg, .cd-action-icon > svg { width: 16px; height: 16px; }
.cd-action-icon { display: grid; place-items: center; color: var(--creative-accent); }
.cd-like-count { min-width: 18px; padding: 2px 5px; border-radius: 5px; background: var(--creative-paper); color: var(--creative-muted); font-size: 10px; font-variant-numeric: tabular-nums; }
.cd-action:hover:not(:disabled) { color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 35%, var(--creative-line)); background: var(--creative-soft); }
.cd-action:active:not(:disabled) { transform: translateY(1px); }
.cd-action.active { color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 30%, var(--creative-line)); background: var(--creative-soft); }
.cd-action:disabled { cursor: not-allowed; color: var(--creative-muted); }
.cd-action:disabled .cd-action-icon { color: color-mix(in srgb, var(--creative-accent) 70%, var(--creative-muted)); }

.cd-download-action {
  display: flex; align-items: center; gap: 12px; padding: 9px 14px 9px 10px; text-decoration: none;
  border: 1px solid color-mix(in srgb, var(--creative-accent) 22%, var(--creative-line)); border-radius: 14px;
  background: var(--creative-soft); color: var(--creative-accent); font-size: 13px; font-weight: 500;
  transition: box-shadow .2s, transform .2s;
}
.cd-download-action:hover { box-shadow: 0 6px 18px color-mix(in srgb, var(--creative-accent) 14%, transparent); transform: translateY(-1px); }
.cd-download-action:active { transform: translateY(0); }
.cd-download-icon { display: grid; place-items: center; width: 34px; height: 34px; background: var(--creative-paper); border-radius: 10px; flex-shrink: 0; }
.cd-download-icon svg { width: 17px; height: 17px; }
.cd-download-label { display: flex; flex-direction: column; align-items: flex-start; gap: 3px; line-height: 1.25; }
.cd-download-label small { font-size: 10px; font-weight: 400; color: var(--creative-muted); letter-spacing: .03em; }
.cd-action-hint { margin: 0; color: var(--creative-muted); font-size: 11px; }
@media (max-width: 380px) {
  .cd-action { padding-inline: 7px; gap: 5px; }
  .cd-share-action { flex-direction: column; gap: 3px; font-size: 10px; }
  .cd-like-count { display: none; }
}
</style>

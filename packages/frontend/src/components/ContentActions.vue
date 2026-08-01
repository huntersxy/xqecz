<script setup lang="ts">
import { ref, watch } from 'vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { IconHeart, IconStar, IconShareAlt, IconDownload } from '@arco-design/web-vue/es/icon'
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

function downloadMedia() {
  const url = props.content.img || props.content.video
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = props.content.title || 'download'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}
</script>

<template>
  <footer class="cd-bottombar">
    <a-button
      class="cd-action"
      shape="round"
      :class="{ active: isLiked }"
      :disabled="!userStore.isLoggedIn"
      @click="toggleLike"
    >
      <IconHeart :fill="isLiked ? 'currentColor' : 'none'" />
      <span>{{ likeCount || '点赞' }}</span>
    </a-button>

    <a-button
      class="cd-action"
      shape="round"
      :class="{ active: isFavorited }"
      :disabled="!userStore.isLoggedIn"
      @click="toggleFavorite"
    >
      <IconStar :fill="isFavorited ? 'currentColor' : 'none'" />
      <span>收藏</span>
    </a-button>

    <a-button class="cd-action" shape="round" @click="shareContent">
      <IconShareAlt />
      <span>分享</span>
    </a-button>

    <a-button class="cd-action" shape="round" @click="downloadMedia">
      <IconDownload />
      <span>下载</span>
    </a-button>
  </footer>
</template>

<style scoped>
.cd-bottombar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-header-bg);
}

.cd-action {
  color: var(--color-text-secondary);
  border-color: var(--color-border);
  background: var(--color-surface);
}

.cd-action:hover {
  color: var(--color-primary);
  border-color: var(--color-primary);
}

.cd-action.active {
  color: var(--color-primary);
  border-color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 14%, transparent);
}

@media (max-width: 768px) {
  .cd-bottombar {
    padding: 0.5rem 0.75rem;
    gap: 0.375rem;
  }
}
</style>

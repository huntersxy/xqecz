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
// 原文件优先用 origin（未生成缩略图时 img 才可能指向缩略图）；缩略图单独提供入口。
const originUrl = computed(() =>
  getImageUrl(props.content.origin || props.content.img || props.content.video || ''),
)
const thumbUrl = computed(() => getImageUrl(props.content.thumb || ''))
const isVideo = computed(() => !!props.content.video)
const sizeText = computed(() => formatFileSize(props.content.file_size))
const canDownload = computed(() => !!originUrl.value)
const showThumbOption = computed(() => !!thumbUrl.value && thumbUrl.value !== originUrl.value)

/** 主文件格式：优先取 URL 后缀，视频无后缀时按 video 标记兜底。 */
const originExt = computed(() => getUrlExtension(props.content.origin || props.content.img || props.content.video))
/** 缩略图格式（接口不返回其体积，只展示后缀便于区分）。 */
const thumbExt = computed(() => getUrlExtension(props.content.thumb))

/** 触发下载：链接尾附 ?download=1，由服务端回 Content-Disposition（文件名取内容标题）。 */
function triggerDownload(url: string) {
  if (!url) return
  const a = document.createElement('a')
  a.href = withDownloadFlag(url)
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function onDownloadSelect(value: string | number | Record<string, unknown> | undefined) {
  if (value === 'thumb') triggerDownload(thumbUrl.value)
  else triggerDownload(originUrl.value)
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

    <a-dropdown v-if="canDownload" trigger="click" position="tr" @select="onDownloadSelect">
      <a-button class="cd-action" shape="round" aria-label="下载">
        <IconDownload />
        <span>下载{{ sizeText ? ` ${sizeText}` : '' }}</span>
      </a-button>
      <template #content>
        <a-doption value="origin">
          <span class="flex items-center justify-between gap-4 min-w-[9rem]">
            <span>原文件{{ isVideo ? '（视频）' : originExt ? ` · ${originExt}` : '' }}</span>
            <span class="text-xs opacity-60">{{ sizeText || '未知大小' }}</span>
          </span>
        </a-doption>
        <a-doption v-if="showThumbOption" value="thumb">
          <span class="flex items-center justify-between gap-4 min-w-[9rem]">
            <span>缩略图{{ thumbExt ? ` · ${thumbExt}` : '' }}</span>
            <span class="text-xs opacity-60">预览用图</span>
          </span>
        </a-doption>
      </template>
    </a-dropdown>
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

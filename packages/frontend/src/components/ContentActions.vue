<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { IconHeart, IconStar, IconShareAlt, IconDownload, IconDown, IconImage, IconFile } from '@arco-design/web-vue/es/icon'
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
      <a-dropdown v-if="canDownload" trigger="click" position="tr" popup-container=".cd-root" @select="onDownloadSelect">
        <button type="button" class="cd-action cd-download-action" aria-label="下载" aria-haspopup="true">
          <span class="cd-download-icon"><IconDownload /></span>
          <span class="cd-download-label"><span>下载作品</span><small>{{ originExt || (isVideo ? 'VIDEO' : '原文件') }}<template v-if="sizeText"> · {{ sizeText }}</template></small></span>
          <IconDown class="cd-download-chevron" />
        </button>
        <template #content>
          <div class="cd-download-menu">
            <div class="cd-download-menu-head">保存这份灵感 <span aria-hidden="true">✦</span></div>
            <a-doption value="origin" class="cd-download-option">
              <span class="cd-download-option-icon"><IconFile /></span>
              <span class="cd-download-option-copy"><strong>原文件<span v-if="originExt"> · {{ originExt }}</span></strong><small>{{ isVideo ? '原始视频' : '保留原始画质' }} · {{ sizeText || '未知大小' }}</small></span>
              <IconDownload class="cd-option-arrow" />
            </a-doption>
            <a-doption v-if="showThumbOption" value="thumb" class="cd-download-option">
              <span class="cd-download-option-icon"><IconImage /></span>
              <span class="cd-download-option-copy"><strong>缩略图<span v-if="thumbExt"> · {{ thumbExt }}</span></strong><small>轻量预览，方便分享</small></span>
              <IconDownload class="cd-option-arrow" />
            </a-doption>
          </div>
        </template>
      </a-dropdown>
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
.cd-download-action { height: 50px; padding: 0 16px 0 10px; gap: 11px; border: 1px solid color-mix(in srgb, var(--creative-accent) 22%, var(--creative-line)); border-radius: 15px; background: var(--creative-soft); color: var(--creative-accent); box-shadow: 0 3px 9px color-mix(in srgb, var(--creative-accent) 5%, transparent); }
.cd-download-action:hover { box-shadow: 0 5px 16px color-mix(in srgb, var(--creative-accent) 12%, transparent); }
.cd-download-icon { display: grid; place-items: center; width: 32px; height: 32px; background: var(--creative-paper); border-radius: 10px; }
.cd-download-icon svg { width: 17px; height: 17px; }
.cd-download-label { display: flex; flex-direction: column; align-items: flex-start; gap: 3px; line-height: 1.25; }
.cd-download-label small { font-size: 9px; font-weight: 400; color: var(--creative-muted); letter-spacing: .03em; }
.cd-download-chevron { width: 11px !important; height: 11px !important; margin-left: 4px; }
.cd-download-menu { width: 250px; padding: 8px; background: var(--creative-paper); }
.cd-download-menu-head { display: flex; align-items: center; justify-content: space-between; padding: 5px 8px 12px; color: var(--creative-muted); font-size: 10px; letter-spacing: .08em; }
.cd-download-menu-head > span { color: var(--creative-accent); }
.cd-download-option { padding: 11px 8px; border-radius: 9px; color: var(--creative-ink); }
.cd-download-option :deep(.arco-dropdown-option-content) { display: flex; align-items: center; gap: 10px; }
.cd-download-option-icon { display: grid; place-items: center; flex-shrink: 0; width: 32px; height: 36px; border: 1px solid var(--creative-line); border-radius: 7px; background: var(--creative-canvas); color: var(--creative-accent); font-size: 16px; }
.cd-download-option-copy { display: flex; flex: 1; flex-direction: column; gap: 4px; }
.cd-download-option-copy strong { font-size: 12px; font-weight: 500; }
.cd-download-option-copy small { font-size: 10px; color: var(--creative-muted); }
.cd-option-arrow { color: var(--creative-muted); font-size: 13px; }
:global(.cd-root .arco-dropdown:has(.cd-download-menu)) { padding: 0; border: 1px solid var(--creative-line); border-radius: 14px; overflow: hidden; background: var(--creative-paper); box-shadow: 0 12px 40px color-mix(in srgb, var(--creative-ink) 12%, transparent); }
@media (max-width: 1100px) { .cd-action-note { display: none; } }
@media (max-width: 768px) {
  .cd-bottombar { padding: 10px 14px calc(10px + env(safe-area-inset-bottom)); }
  .cd-action-tools { width: 100%; justify-content: center; gap: 8px; }
  .cd-action { height: 40px; padding-inline: 10px; gap: 6px; font-size: 11px; }
  .cd-interaction-group { padding: 3px; border-radius: 13px; }
  .cd-download-action { height: 48px; padding-inline: 8px; gap: 8px; border-radius: 13px; }
  .cd-download-icon { width: 27px; height: 30px; border-radius: 8px; }
  .cd-download-chevron { display: none; }
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

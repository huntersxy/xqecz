<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { contentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import { useListCache } from '@/composables/useListCache'
import { getImageUrl, formatTime } from '@/utils'
import SafeImage from '@/components/SafeImage.vue'
import ContentMedia from '@/components/ContentMedia.vue'
import ContentSidebar from '@/components/ContentSidebar.vue'
import ReportModal from '@/components/ReportModal.vue'
import ClaimModal from '@/components/ClaimModal.vue'
import type { Content, Comment } from '@/types'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const listCache = useListCache()

const content = ref<Content | null>(null)
const reportTarget = ref<Comment | null>(null)
const showClaimModal = ref(false)

// ── 列表上下文 ──
const cachedList = ref<Content[]>([])
const currentId = ref(Number(route.params.id))
const currentIndex = computed(() => cachedList.value.findIndex((c) => c.id === currentId.value))
const hasPrev = computed(() => currentIndex.value > 0)
const hasNext = computed(() => currentIndex.value >= 0)
const isFetchingMore = ref(false)
const listPage = ref(1)
const listTotalPages = ref(1)

const THUMB_H = 64
const THUMB_GAP = 6
const previewCount = ref(5)
const previewRef = ref<HTMLElement | null>(null)

function calcPreviewCount() {
  const h = window.innerHeight
  previewCount.value = Math.max(3, Math.floor((h - 120) / (THUMB_H + THUMB_GAP)))
}

const previewItems = computed(() => {
  if (currentIndex.value < 0) return []
  const total = cachedList.value.length
  const count = previewCount.value
  const half = Math.floor(count / 2)
  let start = currentIndex.value - half
  let end = start + count
  if (start < 0) { start = 0; end = Math.min(count, total) }
  if (end > total) { end = total; start = Math.max(0, end - count) }
  return cachedList.value.slice(start, end).map((item) => ({
    ...item,
    isCurrent: item.id === currentId.value,
  }))
})

function scrollThumbIntoView() {
  nextTick(() => {
    const el = previewRef.value?.querySelector('.cd-thumb-current')
    el?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
  })
}

async function fetchMore(): Promise<boolean> {
  if (isFetchingMore.value || listPage.value >= listTotalPages.value) return false
  isFetchingMore.value = true
  try {
    const nextPage = listPage.value + 1
    const res = await contentApi.list({ page: nextPage, page_size: 20, sort_by: 'created_at', order: 'desc' })
    if (res.code === 200) {
      const newItems = res.data.list as Content[]
      const existingIds = new Set(cachedList.value.map((c) => c.id))
      const fresh = newItems.filter((item) => !existingIds.has(item.id))
      cachedList.value.push(...fresh)
      listPage.value = nextPage
      listTotalPages.value = res.data.total_page
      listCache.save(cachedList.value, res.data.total, listTotalPages.value)
      return fresh.length > 0
    }
  } catch (e) {
    console.warn('加载更多失败:', e)
  } finally {
    isFetchingMore.value = false
  }
  return false
}

async function navigateTo(id: number) {
  currentId.value = id
  history.replaceState(null, '', `/#/content/${id}`)
  loadContent()
  scrollThumbIntoView()
  // 接近末尾时预加载更多
  const idx = currentIndex.value
  if (idx >= 0 && idx >= cachedList.value.length - 3) {
    await fetchMore()
  }
}

function goToPrev() {
  if (!hasPrev.value) return
  navigateTo(cachedList.value[currentIndex.value - 1].id)
}

async function goToNext() {
  const idx = currentIndex.value
  if (idx < 0) return
  const next = cachedList.value[idx + 1]
  if (next) await navigateTo(next.id)
}

// ── 交互 ──
const likeCount = ref(0)
const isLiked = ref(false)
const isFavorited = ref(false)

async function loadInteractionStatus() {
  if (!content.value || !userStore.isLoggedIn) return
  try {
    const res = await contentApi.likeStatus(content.value.id)
    if (res.code === 200) {
      isLiked.value = res.data.liked
      isFavorited.value = res.data.favorited
      likeCount.value = res.data.like_count
    }
  } catch { /* 未登录或网络错误，保持默认值 */ }
}

async function toggleLike() {
  if (!content.value || !userStore.isLoggedIn) return
  try {
    const res = await contentApi.toggleLike(content.value.id)
    if (res.code === 200) {
      isLiked.value = res.data.liked
      likeCount.value += isLiked.value ? 1 : -1
    }
  } catch { /* 忽略 */ }
}

async function toggleFavorite() {
  if (!content.value || !userStore.isLoggedIn) return
  try {
    const res = await contentApi.toggleFavorite(content.value.id)
    if (res.code === 200) {
      isFavorited.value = res.data.favorited
    }
  } catch { /* 忽略 */ }
}

async function shareContent() {
  const url = globalThis.location.href
  try {
    if (navigator.share) await navigator.share({ title: content.value?.title || '', url })
    else if (navigator.clipboard) await navigator.clipboard.writeText(url)
  } catch { /* 用户取消分享，忽略 */ }
}

function downloadMedia() {
  if (!content.value) return
  const url = content.value.img || content.value.video
  if (!url) return
  const a = document.createElement('a')
  a.href = url
  a.download = content.value.title || 'download'
  a.target = '_blank'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

// ── 加载内容 ──
async function loadContent() {
  try {
    const res = await contentApi.detail(currentId.value)
    if (res.code === 200) {
      content.value = res.data
      loadInteractionStatus()
    } else {
      message.error(res.message)
    }
  } catch {
    message.error('加载内容失败')
  }
}

// ── 关闭 ──
function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/')
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Escape') goBack()
  if (e.key === 'ArrowLeft') goToPrev()
  if (e.key === 'ArrowRight') void goToNext()
}

// ── 生命周期 ──
onMounted(() => {
  userStore.checkAuth()
  loadContent()
  const cached = listCache.load()
  cachedList.value = cached?.list || []
  // 根据缓存数据量估算当前页数
  listPage.value = Math.ceil(cachedList.value.length / 20) || 1
  listTotalPages.value = cached?.totalPages || 1
  calcPreviewCount()
  document.addEventListener('keydown', onKeyDown)
  window.addEventListener('resize', calcPreviewCount)
  const scrollBarW = window.innerWidth - document.documentElement.clientWidth
  const prevOverflow = document.body.style.overflow
  const prevPadding = document.body.style.paddingRight
  document.body.style.overflow = 'hidden'
  if (scrollBarW > 0) document.body.style.paddingRight = `${scrollBarW}px`
  onBeforeUnmount(() => {
    document.body.style.overflow = prevOverflow
    if (scrollBarW > 0) document.body.style.paddingRight = prevPadding
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', calcPreviewCount)
})

watch(() => route.params.id, (newId) => {
  if (newId && Number(newId) !== currentId.value) {
    currentId.value = Number(newId)
    loadContent()
    scrollThumbIntoView()
  }
})
</script>

<template>
  <div class="cd-root" role="dialog" aria-modal="true">
    <!-- 遮罩 -->
    <div class="cd-backdrop" @click="goBack"></div>

    <div class="cd-shell" @click.self="goBack">
      <!-- 顶部条 -->
      <header class="cd-topbar">
        <button class="cd-back-btn" type="button" aria-label="返回" @click="goBack">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
        <div class="cd-topbar-title">
          <h2 class="cd-title">{{ content?.title || '加载中...' }}</h2>
          <div v-if="content" class="cd-meta">
            <span class="cd-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>
              {{ formatTime(content.created_at) }}
            </span>
            <span class="cd-meta-item">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path><circle cx="12" cy="12" r="3"></circle></svg>
              {{ content.view_count }} 浏览
            </span>
            <span v-if="currentIndex >= 0" class="cd-meta-item cd-meta-index">
              {{ currentIndex + 1 }} / {{ cachedList.length }}
            </span>
          </div>
        </div>
      </header>

      <!-- 主体 -->
      <div v-if="content" class="cd-body">
        <!-- 中间主体 -->
        <div class="cd-main">
          <ContentMedia :content="content" />
          <ContentSidebar :content="content" @open-claim="showClaimModal = true" @report-comment="reportTarget = $event" />
        </div>
      </div>

      <!-- 底部预览走马灯 -->
      <div v-if="content && previewItems.length > 0" ref="previewRef" class="cd-carousel">
        <button
          v-for="item in previewItems"
          :key="item.id"
          :class="['cd-carousel-thumb', { 'cd-thumb-current': item.isCurrent }]"
          @click="navigateTo(item.id)"
        >
          <SafeImage :src="getImageUrl(item.thumb)" :alt="item.title" loading="lazy" />
        </button>
      </div>

      <!-- 加载中 -->
      <div v-else class="cd-loading">
        <div class="cd-spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- 左右切换按钮 -->
      <button
        v-if="hasPrev"
        class="cd-nav-btn cd-nav-prev"
        type="button"
        aria-label="上一个"
        @click="goToPrev"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
          <polyline points="15 18 9 12 15 6"></polyline>
        </svg>
      </button>
      <button
        v-if="hasNext"
        class="cd-nav-btn cd-nav-next"
        type="button"
        aria-label="下一个"
        @click="goToNext"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
          <polyline points="9 6 15 12 9 18"></polyline>
        </svg>
      </button>

      <!-- 底部交互栏 -->
      <footer v-if="content" class="cd-bottombar">
        <button :class="['cd-action', { active: isLiked }]" type="button" @click="toggleLike">
          <svg viewBox="0 0 24 24" :fill="isLiked ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"></path></svg>
          <span>{{ likeCount || '点赞' }}</span>
        </button>
        <button :class="['cd-action', { active: isFavorited }]" type="button" @click="toggleFavorite">
          <svg viewBox="0 0 24 24" :fill="isFavorited ? 'currentColor' : 'none'" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon></svg>
          <span>收藏</span>
        </button>
        <button class="cd-action" type="button" @click="shareContent">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>
          <span>分享</span>
        </button>
        <button class="cd-action" type="button" @click="downloadMedia">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
          <span>下载</span>
        </button>
      </footer>
    </div>

    <!-- 举报弹窗 -->
    <ReportModal :target="reportTarget" @close="reportTarget = null" />
    <!-- 认领弹窗 -->
    <ClaimModal :open="showClaimModal" :content-id="Number(content?.id) || 0" @close="showClaimModal = false" />
  </div>
</template>

<style scoped>
.cd-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--theme-text);
  animation: cd-fade-in 0.2s ease-out;
}

.cd-backdrop {
  position: absolute;
  inset: 0;
  background: rgba(8, 4, 18, 0.92);
  backdrop-filter: blur(8px);
}

.cd-shell {
  position: relative;
  width: 100vw;
  height: 100vh;
  height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--theme-surface);
  overflow: hidden;
}

@keyframes cd-fade-in {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}

/* ── 顶部条 ── */
.cd-topbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--theme-card-border);
  background: var(--theme-header-bg);
}

.cd-back-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--theme-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.2s, color 0.2s;
}
.cd-back-btn:hover { background: var(--theme-hover-bg); color: var(--theme-text); }
.cd-back-btn svg { width: 18px; height: 18px; }

.cd-topbar-title { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cd-title {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.3;
  margin: 0;
  color: var(--theme-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cd-meta {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  font-size: 0.75rem;
  color: var(--theme-text-secondary);
  margin-top: 2px;
}
.cd-meta-item { display: inline-flex; align-items: center; gap: 4px; }
.cd-meta-item svg { width: 13px; height: 13px; flex-shrink: 0; }
.cd-meta-index { color: var(--theme-primary); font-weight: 600; }

/* ── 主体 ── */
.cd-body { flex: 1; min-height: 0; display: flex; gap: 0; }

/* ── 底部预览走马灯 ── */
.cd-carousel {
  flex-shrink: 0;
  display: flex;
  gap: 6px;
  padding: 8px 12px;
  overflow-x: auto;
  overflow-y: hidden;
  background: var(--theme-header-bg);
  border-top: 1px solid var(--theme-card-border);
  scroll-behavior: smooth;
  scrollbar-width: none;
}
.cd-carousel::-webkit-scrollbar { display: none; }
.cd-carousel-thumb {
  flex-shrink: 0;
  width: 56px;
  height: 56px;
  border-radius: 6px;
  overflow: hidden;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
  background: var(--theme-card-border);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.cd-carousel-thumb:hover { border-color: var(--theme-primary); }
.cd-thumb-current {
  border-color: var(--theme-primary);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--theme-primary) 30%, transparent);
}
.cd-carousel-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }

/* ── 中间主体 ── */
.cd-main { flex: 1; min-width: 0; display: flex; }

/* ── 左右切换按钮 ── */
.cd-nav-btn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10;
  width: 40px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: rgba(0, 0, 0, 0.4);
  color: #fff;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s;
}
.cd-root:hover .cd-nav-btn { opacity: 1; }
.cd-nav-btn:hover { background: rgba(0, 0, 0, 0.6); }
.cd-nav-btn svg { width: 20px; height: 20px; }
.cd-nav-prev { left: 0; border-radius: 0 6px 6px 0; }
.cd-nav-next { right: 76px; border-radius: 6px 0 0 6px; }

/* ── 加载中 ── */
.cd-loading {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center; gap: 1rem;
  color: var(--theme-text-secondary);
}
.cd-spinner {
  width: 2rem; height: 2rem;
  border: 3px solid var(--theme-card-border);
  border-top-color: var(--theme-primary);
  border-radius: 50%; animation: cd-spin 0.7s linear infinite;
}
@keyframes cd-spin { to { transform: rotate(360deg); } }

/* ── 底部交互栏 ── */
.cd-bottombar {
  flex-shrink: 0; display: flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 0.625rem 1rem;
  border-top: 1px solid var(--theme-card-border); background: var(--theme-header-bg);
}
.cd-action {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 0.5rem 0.875rem; font-size: 0.8125rem;
  color: var(--theme-text-secondary); background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 999px;
  cursor: pointer; transition: all 0.15s;
}
.cd-action svg { width: 16px; height: 16px; }
.cd-action:hover {
  color: var(--theme-primary); border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 8%, transparent);
}
.cd-action.active {
  color: var(--theme-primary); border-color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 14%, transparent);
}

/* ── 窄屏 ── */
@media (max-width: 768px) {
  .cd-body { flex-direction: column; overflow-y: auto; }
  .cd-main { flex-direction: column; }
  .cd-nav-btn { display: none; }
  .cd-bottombar { padding: 0.5rem 0.75rem; gap: 0.375rem; }
  .cd-action { padding: 0.4375rem 0.625rem; font-size: 0.75rem; }
  :deep(.cd-media-wrap) { flex: 1 1 100%; max-height: 50vh; }
  :deep(.cd-side) { flex: 1 1 100%; max-width: 100%; border-left: none; border-top: 1px solid var(--theme-card-border); }
}
</style>

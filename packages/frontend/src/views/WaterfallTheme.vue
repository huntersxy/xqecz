<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import { useThemeStore } from '@/stores/theme'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { useGlobalSearch, watchGlobalSearch } from '@/composables/useGlobalSearch'
import { getImageUrl } from '@/utils'
import { ContentSchema } from '@/types/schemas'
import type { Content, ListParams } from '@/types'
import ContentOverlay from '@/components/ContentOverlay.vue'

const homeStore = useHomeStore()
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.mode === 'dark')

const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const { searchKeyword } = useGlobalSearch()

const swapSections = computed(
  () =>
    searchFilter.selectedTags.value.length > 0 ||
    searchFilter.selectedTypes.value.length > 0 ||
    !!searchKeyword.value.trim(),
)

// ── 数据状态 ──
const allContents = ref<Content[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const totalPages = ref(1)
const isLoading = ref(false)
const isLoadingMore = ref(false)
const hasMore = computed(() => currentPage.value <= totalPages.value)

// ── 覆盖层状态 ──
const overlayContent = ref<Content | null>(null)

// ── 哨兵 ──
const sentinelRef = ref<HTMLElement | null>(null)

const masonryRef = ref<HTMLElement | null>(null)

// ── 数据加载 ──
// 打开全页面覆盖层（不再是直接 viewer / 路由跳转）
function openContent(content: Content) {
  // 推荐区是 RecommendContent，缺 text/img/video/url 等字段；先尝试 detail 拉一次（silent 模式不计浏览量）
  if (!content.text && !content.img && !content.video && !content.url && content.id) {
    contentApi.detail(content.id, { silent: true }).then((res) => {
      if (res.code === 200) overlayContent.value = res.data
    }).catch(() => {
      overlayContent.value = content
    })
    return
  }
  overlayContent.value = content
}

// 兼容旧 openViewer 命名（推荐区卡片 onClick 沿用此名）
const openViewer = openContent

async function fetchPage(page: number, append = false) {
  if (isLoading.value || isLoadingMore.value) return

  if (append) {
    isLoadingMore.value = true
  } else {
    isLoading.value = true
  }

  try {
    let res
    if (searchKeyword.value) {
      res = await contentApi.search(searchKeyword.value, {
        page,
        page_size: pageSize.value,
        tag: searchFilter.selectedTags.value.length > 0 ? searchFilter.selectedTags.value.join(',') : undefined,
        type: searchFilter.selectedTypes.value.length > 0 ? searchFilter.selectedTypes.value.join(',') : undefined,
      })
    } else {
      const params: ListParams = {
        page,
        page_size: pageSize.value,
        sort_by: 'created_at',
        order: 'desc',
      }
      if (searchFilter.selectedTags.value.length > 0) params.tag = searchFilter.selectedTags.value.join(',')
      if (searchFilter.selectedTypes.value.length > 0) params.type = searchFilter.selectedTypes.value.join(',')
      res = await contentApi.list(params)
    }

    if (res.code === 200) {
      const parsed = res.data.list.map((item: unknown) => ContentSchema.parse(item))
      total.value = res.data.total
      totalPages.value = res.data.total_page
      currentPage.value = page

      if (append) {
        allContents.value.push(...parsed)
      } else {
        allContents.value = parsed
      }
    }
  } catch (e) {
    console.error('加载失败:', e)
  } finally {
    isLoading.value = false
    isLoadingMore.value = false
  }
}

function resetAndLoad() {
  currentPage.value = 1
  allContents.value = []
  fetchPage(1)
}

function clearFilters() {
  searchFilter.selectedTags.value = []
  searchFilter.selectedTypes.value = []
  searchKeyword.value = ''
  resetAndLoad()
}

// 监听全局搜索触发器（App.vue 搜索框回车/点按钮时 +1）
watchGlobalSearch(() => resetAndLoad())

// 监听全局搜索触发器（App.vue 搜索框回车/点按钮时 +1）
watchGlobalSearch(() => resetAndLoad())
useIntersectionObserver(
  sentinelRef,
  ([{ isIntersecting }]) => {
    if (isIntersecting && hasMore.value && !isLoading.value && !isLoadingMore.value) {
      fetchPage(currentPage.value + 1, true)
    }
  },
  { rootMargin: '200px' },
)

function onMountedTheme() {  const navEntries = performance.getEntriesByType('navigation') as PerformanceNavigationTiming[]
  if (navEntries.length > 0 && navEntries[0].type === 'reload') {
    homeStore.clearState()
  }

  if (homeStore.hasLoaded) {
    homeStore.restoreScroll()
    recommendLoader.recommendPage.value = homeStore.recommendPage
  }

  fetchPage(1)
  searchFilter.loadTags()
  recommendLoader.loadRecommendContents(recommendLoader.recommendPage.value)
}

onMounted(() => {
  onMountedTheme()
})
</script>

<template>
  <div class="wf-root">
    <!-- 推荐区 -->
    <section v-if="!swapSections && recommendLoader.recommendContents.value.length > 0" class="wf-recommend">
      <div class="wf-recommend-head">
        <h2>✨ 精选推荐</h2>
        <button
          class="wf-reload"
          :disabled="recommendLoader.isRecommendLoading.value"
          @click="recommendLoader.refreshRecommend()"
        >
          {{ recommendLoader.isRecommendLoading.value ? '...' : '换一批' }}
        </button>
      </div>
      <div class="wf-recommend-scroll">
        <div
          v-for="item in recommendLoader.recommendContents.value"
          :key="item.id"
          class="wf-recommend-card"
          @click="openViewer(item as unknown as Content)"
        >
          <img
            v-if="item.type !== 'text'"
                :src="getImageUrl(item.thumb)"
            :alt="item.title"
            loading="lazy"
          />
          <div v-else class="wf-rec-text">{{ item.title }}</div>
          <div class="wf-rec-overlay">
            <span>{{ item.title }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- 瀑布流 -->
    <div class="wf-masonry-wrap">
      <!-- 首次加载 -->
      <div v-if="isLoading && allContents.length === 0" class="wf-center-state">
        <div class="wf-spinner-lg"></div>
        <p>加载中...</p>
      </div>

      <!-- 空状态 -->
      <div v-else-if="!isLoading && allContents.length === 0" class="wf-center-state">
        <p>暂无内容</p>
      </div>

      <!-- 瀑布流容器 -->
      <div
        v-else
        ref="masonryRef"
        class="wf-masonry"
      >
        <div
          v-for="item in allContents"
          :key="item.id"
          class="wf-card"
          @click="openViewer(item)"
          @keydown.enter="openViewer(item)"
          tabindex="0"
        >
          <!-- 图片/视频封面 -->
          <template v-if="item.type !== 'text'">
            <div class="wf-card-media">
              <img
                :src="getImageUrl(item.thumb)"
                :alt="item.title"
                loading="lazy"
                decoding="async"
              />
              <div v-if="item.type === 'video'" class="wf-play-btn">
                <svg viewBox="0 0 24 24" fill="white"><path d="M8 5v14l11-7z"/></svg>
              </div>
              <div v-if="item.tags?.some(t => /ai/i.test(t))" class="wf-badge-ai">AI</div>
              <div v-if="item.type === 'link'" class="wf-badge-link">
                <svg viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
              </div>
            </div>
          </template>

          <!-- 纯文字卡片 -->
          <template v-else>
            <div class="wf-card-text-body">
              <p>{{ item.title }}</p>
            </div>
          </template>

          <!-- 底部信息 -->
          <div class="wf-card-info">
            <span class="wf-card-title">{{ item.title }}</span>
            <div class="wf-card-meta">
              <span class="wf-card-user">{{ item.user?.username }}</span>
              <span class="wf-card-views">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                {{ item.view_count }}
              </span>
            </div>
            <div v-if="item.tags?.length" class="wf-card-tags">
              <span v-for="tag in item.tags.slice(0, 3)" :key="tag" class="wf-mini-tag">{{ tag }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 加载更多 -->
      <div v-if="isLoadingMore" class="wf-loadmore">
        <div class="wf-spinner-sm"></div>
        <span>加载更多...</span>
      </div>

      <!-- 哨兵 -->
      <div ref="sentinelRef" class="wf-sentinel"></div>

      <!-- 到底了 -->
      <div v-if="!hasMore && allContents.length > 0" class="wf-end">
        <span>— 到底啦 —</span>
      </div>
    </div>

    <!-- 全页面覆盖层（点击卡片触发，不改路由） -->
    <ContentOverlay
      v-if="overlayContent"
      :content="overlayContent"
      @close="overlayContent = null"
    />
  </div>
</template>

<style lang="scss" scoped>
.wf-root {
  min-height: 100vh;
  background: transparent;
  color: var(--theme-text);
}

/* ===== 推荐横滚 ===== */
/* ===== 推荐横滚 ===== */
.wf-recommend {
  max-width: 1600px;
  margin: 0 auto;
  padding: 1rem 1rem 0;
}

.wf-recommend-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.625rem;

  h2 {
    font-size: 0.9375rem;
    font-weight: 700;
    margin: 0;
    color: var(--theme-text);
  }
}

.wf-reload {
  font-size: 0.75rem;
  padding: 0.25rem 0.75rem;
  border-radius: 2rem;
  border: 1px solid var(--theme-card-border);
  background: var(--theme-surface);
  color: var(--theme-text-secondary);
  cursor: pointer;
  transition: all 0.15s;

  &:hover:not(:disabled) {
    border-color: var(--theme-primary);
    color: var(--theme-primary);
  }

  &:disabled { opacity: 0.4; cursor: not-allowed; }
}

.wf-recommend-scroll {
  display: flex;
  gap: 0.625rem;
  overflow-x: auto;
  padding-bottom: 0.75rem;
  scrollbar-width: none;

  &::-webkit-scrollbar { display: none; }
}

.wf-recommend-card {
  flex-shrink: 0;
  width: 140px;
  height: 180px;
  border-radius: 0.75rem;
  overflow: hidden;
  position: relative;
  cursor: pointer;
  background: var(--theme-placeholder-bg);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.3s;
  }

  &:hover img { transform: scale(1.08); }

  .wf-rec-text {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: 0.75rem;
    font-size: 0.75rem;
    color: var(--theme-text-secondary);
    text-align: center;
  }
}

.wf-rec-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 1.5rem 0.5rem 0.5rem;
  background: linear-gradient(transparent, rgba(0,0,0,0.7));
  color: #fff;
  font-size: 0.6875rem;
  font-weight: 500;
  line-height: 1.3;

  span {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}

.wf-masonry-wrap {
  max-width: 1600px;
  margin: 0 auto;
  padding: 0.75rem 0.75rem 3rem;

  @media (min-width: 640px) { padding: 1rem 1rem 3rem; }
}

.wf-masonry {
  columns: 2;
  column-gap: 10px;
}

@media (min-width: 640px) {
  .wf-masonry { columns: 3; }
}
@media (min-width: 1024px) {
  .wf-masonry { columns: 4; }
}
@media (min-width: 1400px) {
  .wf-masonry { columns: 5; }
}

.wf-card {
  break-inside: avoid;
  margin-bottom: 10px;
  border-radius: 0.625rem;
  overflow: hidden;
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
  /* 隔离每个卡片的渲染影响范围：滚动/加载新卡片时，layout 与 paint 不向其它卡片传播。
     不含 size —— columns 瀑布流依赖子元素实际高度来分列，加 size 会破坏分列计算。 */
  contain: layout style paint;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 24px rgba(0,0,0,0.1);
  }

  &:focus-visible {
    outline: 2px solid var(--theme-primary);
    outline-offset: 2px;
  }
}

.wf-card-media {
  position: relative;
  width: 100%;
  overflow: hidden;
  line-height: 0;
  /* 最小高度占位，防止卡片塌缩为0 */
  min-height: 80px;
  background: var(--theme-placeholder-bg);

  img {
    width: 100%;
    height: auto;
    display: block;
  }
}

.wf-play-btn {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 2.5rem;
  height: 2.5rem;
  background: rgba(0,0,0,0.5);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  opacity: 0.85;
  transition: opacity 0.2s;

  svg { width: 1.125rem; height: 1.125rem; margin-left: 2px; }

  .wf-card:hover & { opacity: 1; }
}

.wf-badge-ai {
  position: absolute;
  top: 0.375rem;
  left: 0.375rem;
  padding: 0.0625rem 0.375rem;
  font-size: 0.5625rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #fff;
  background: rgba(139, 92, 246, 0.85);
  border-radius: 0.25rem;
}

.wf-badge-link {
  position: absolute;
  top: 0.375rem;
  right: 0.375rem;
  width: 1.5rem;
  height: 1.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0,0,0,0.55);
  border-radius: 50%;

  svg { width: 0.875rem; height: 0.875rem; }
}

.wf-card-text-body {
  padding: 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80px;
  background: var(--theme-hover-bg);

  p {
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--theme-text);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 5;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
}

.wf-card-info {
  padding: 0.5rem 0.625rem 0.625rem;
}

.wf-card-title {
  display: block;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--theme-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 0.25rem;
}

.wf-card-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.wf-card-user {
  font-size: 0.6875rem;
  color: var(--theme-text-secondary);
}

.wf-card-views {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.625rem;
  color: var(--theme-text-secondary);

  svg { width: 0.75rem; height: 0.75rem; }
}

.wf-card-tags {
  display: flex;
  gap: 0.25rem;
  margin-top: 0.375rem;
  flex-wrap: wrap;
}

.wf-mini-tag {
  font-size: 0.5625rem;
  padding: 0.0625rem 0.375rem;
  border-radius: 1rem;
  background: var(--theme-hover-bg);
  color: var(--theme-text-secondary);
}

/* ===== 状态 ===== */
.wf-center-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 1rem;
  color: var(--theme-text-secondary);

  p { font-size: 0.875rem; margin-top: 1rem; }
}

.wf-spinner-lg {
  width: 2rem;
  height: 2rem;
  border: 3px solid var(--theme-card-border);
  border-top-color: var(--theme-primary);
  border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}

.wf-spinner-sm {
  width: 1.25rem;
  height: 1.25rem;
  border: 2px solid var(--theme-card-border);
  border-top-color: var(--theme-primary);
  border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}

.wf-loadmore {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1.5rem;
  font-size: 0.8125rem;
  color: var(--theme-text-secondary);
}

.wf-sentinel {
  height: 1px;
}

.wf-end {
  text-align: center;
  padding: 2rem;
  font-size: 0.75rem;
  color: var(--theme-text-secondary);
  opacity: 0.6;
}

@keyframes wf-spin {
  to { transform: rotate(360deg); }
}
</style>

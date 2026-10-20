<script lang="ts">
export default { name: 'HomeView' }
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, onDeactivated, onBeforeUnmount, nextTick, watch } from 'vue'
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore, type FeedMode } from '@/stores/home'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import { useListCache, diffLists } from '@/composables/useListCache'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import SiteImage from '@/components/SiteImage.vue'
import SiteIcon from '@/components/SiteIcon.vue'
import QuickUploadSheet from '@/components/QuickUploadSheet.vue'
import type { Content, ListParams, RecommendContent } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const listCache = useListCache()
const showUploadSheet = ref(false)
const feedMode = computed<FeedMode>({
  get: () => homeStore.feedMode,
  set: (mode) => { homeStore.feedMode = mode },
})

const feedSort = computed(() => (feedMode.value === 'popular' ? 'view_count' : 'created_at'))
const featuredWorks = computed(() => {
  const pool = recommendLoader.recommendContents.value.filter((item) => item.thumb)
  if (!pool.length) return []
  const start = recommendLoader.displayStart.value % pool.length
  return pool.slice(start, start + 8)
})

function selectFeedMode(mode: FeedMode) {
  if (feedMode.value === mode) return
  feedMode.value = mode
  if (mode === 'recommended') searchFilter.selectedTags.value = []
  if (homeStore.searchKeyword) homeStore.searchKeyword = ''
  resetAndLoad(false)
}

function goToUpload() {
  showUploadSheet.value = true
}

const swapSections = computed(
  () =>
    searchFilter.selectedTags.value.length > 0 ||
    !!homeStore.searchKeyword.trim(),
)

const allContents = ref<Content[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const totalPages = ref(1)
const isLoading = ref(false)
const isLoadingMore = ref(false)
const hasMore = computed(
  () => currentPage.value <= totalPages.value && allContents.value.length < total.value,
)
const sentinelRef = ref<HTMLElement | null>(null)
const masonryRef = ref<HTMLElement | null>(null)

const waterfall = useWaterfallLayout(masonryRef, allContents)

const pendingScrollRestore = ref(false)

let loadSeq = 0

let isFirstActivation = true
let isHomeActive = false
let restoreTimer: ReturnType<typeof setTimeout> | undefined
function cancelScrollRestore() {
  isHomeActive = false
  clearTimeout(restoreTimer)
  restoreTimer = undefined
  pendingScrollRestore.value = false
}
onDeactivated(cancelScrollRestore)
onBeforeUnmount(cancelScrollRestore)

watch(
  () => allContents.value.length,
  (newLen, oldLen) => {
    if (newLen === 0) {
      waterfall.reset()
      return
    }
    if (oldLen === 0 || newLen < oldLen) {
      nextTick(() => waterfall.relayout({ full: true }))
    }
  },
)

function onCardResized(id: string | number) {
  waterfall.onCardResized(id)
  if (pendingScrollRestore.value) {
    requestAnimationFrame(() => {
      if (!isHomeActive || !pendingScrollRestore.value) return
      const target = homeStore.scrollPosition
      if (document.documentElement.scrollHeight > target + 100) {
        homeStore.restoreScroll()
        pendingScrollRestore.value = false
      }
    })
  }
}

function openContent(content: Content | RecommendContent) {
  router.push(`/content/${content.id}`)
}

function onCardLiked(payload: { id: string | number; liked: boolean; like_count: number }) {
  const target = allContents.value.find((item) => item.id === payload.id)
  if (target) target.like_count = payload.like_count
}

onBeforeRouteLeave(() => {
  const pos = globalThis.scrollY
  homeStore.saveState({
    searchKeyword: homeStore.searchKeyword,
    selectedTags: searchFilter.selectedTags.value,
    page: currentPage.value,
    recommendPage: recommendLoader.loadedPage.value,
    scrollPosition: pos,
    contents: allContents.value,
    total: total.value,
    totalPages: totalPages.value,
    positions: new Map(waterfall.positions.value),
    containerHeight: waterfall.containerHeight.value,
  })
  if (!homeStore.searchKeyword && feedMode.value === 'latest') {
    listCache.save(allContents.value, total.value, totalPages.value)
  }
})

interface FeedPageData {
  list: unknown[]
  total: number
  totalPages: number
}

async function requestFeedPage(page: number): Promise<FeedPageData | null> {
  if (homeStore.searchKeyword) {
    const res = await contentApi.search(homeStore.searchKeyword, {
      page, page_size: pageSize.value,
      tag: searchFilter.selectedTags.value.length > 0 ? searchFilter.selectedTags.value.join(',') : undefined,
    })
    if (res.code !== 200) return null
    return { list: res.data.list, total: res.data.total, totalPages: res.data.total_page }
  }
  if (feedMode.value === 'recommended') {
    const res = await contentApi.recommend(pageSize.value, page)
    if (res.code !== 200) return null
    const totalPages = Math.max(1, Math.ceil(res.data.count / pageSize.value))
    return { list: res.data.list, total: res.data.count, totalPages }
  }
  const params: ListParams = { page, page_size: pageSize.value, sort_by: feedSort.value, order: 'desc' }
  if (searchFilter.selectedTags.value.length > 0) params.tag = searchFilter.selectedTags.value.join(',')
  const res = await contentApi.list(params)
  if (res.code !== 200) return null
  return { list: res.data.list, total: res.data.total, totalPages: res.data.total_page }
}

async function fetchPage(page: number, append = false, force = false) {
  if (!force && (isLoading.value || isLoadingMore.value)) return
  const seq = ++loadSeq
  if (append) isLoadingMore.value = true
  else isLoading.value = true

  try {
    const pageData = await requestFeedPage(page)
    if (seq !== loadSeq || !pageData) return
    const parsed = pageData.list.map((item: unknown) => ContentSchema.parse(item))
    total.value = pageData.total
    totalPages.value = pageData.totalPages
    if (!homeStore.searchKeyword && searchFilter.selectedTags.value.length === 0) {
      if (feedMode.value !== 'recommended' || homeStore.catalogTotal === 0) {
        homeStore.catalogTotal = pageData.total
      }
    }
    if (append) {
      const existing = new Set(allContents.value.map((item) => item.id))
      const fresh = parsed.filter((item) => !existing.has(item.id))
      const oldLen = allContents.value.length
      allContents.value.push(...fresh)
      if (fresh.length > 0) {
        currentPage.value = page
        nextTick(() => waterfall.appendNewItems(allContents.value.slice(oldLen)))
      }
    } else {
      currentPage.value = page
      allContents.value = parsed
      nextTick(() => waterfall.relayout({ full: true }))
    }
  } catch (e) { console.error('加载失败:', e) }
  finally {
    if (seq === loadSeq) { isLoading.value = false; isLoadingMore.value = false }
  }
}

function onTagSelect(tag: string) {
  if (feedMode.value === 'recommended') feedMode.value = 'latest'
  searchFilter.selectTag(tag, () => resetAndLoad(false))
}

function clearFilters() {
  searchFilter.selectedTags.value = []
  homeStore.searchKeyword = ''
  resetAndLoad()
}

function resetAndLoad(clearTags = true) {
  if (clearTags) searchFilter.selectedTags.value = []
  loadSeq++
  currentPage.value = 1
  allContents.value = []
  isLoading.value = false
  isLoadingMore.value = false
  listCache.clear()
  void fetchPage(1, false, true)
}

watchGlobalSearch(() => {
  feedMode.value = 'latest'
  resetAndLoad(true)
})

async function loadAllPages() {
  if (isLoading.value || feedMode.value === 'recommended') return
  isLoading.value = true
  const seq = ++loadSeq
  try {
    const loaded = new Set<number | string>(allContents.value.map((i) => i.id))
    const all = [...allContents.value]
    const batchSize = 100
    for (let page = 1; ; page++) {
      const res = await contentApi.list({ page, page_size: batchSize, sort_by: feedSort.value, order: 'desc' })
      if (seq !== loadSeq) return
      if (res.code !== 200) break
      const parsed = res.data.list.map((item: unknown) => ContentSchema.parse(item))
      total.value = res.data.total
      homeStore.catalogTotal = res.data.total
      totalPages.value = Math.ceil(res.data.total / pageSize.value)
      const fresh = parsed.filter((item) => !loaded.has(item.id))
      for (const item of fresh) { loaded.add(item.id); all.push(item) }
      if (parsed.length === 0 || all.length >= res.data.total) break
    }
    allContents.value = all
    currentPage.value = Math.max(1, Math.ceil(allContents.value.length / pageSize.value))
    nextTick(() => waterfall.relayout({ full: true }))
  } catch (e) { console.error('加载失败:', e) }
  finally {
    if (seq === loadSeq) { isLoading.value = false; isLoadingMore.value = false }
  }
}

async function fetchMore() {
  if (!hasMore.value || isLoading.value || isLoadingMore.value) return
  const nextPage = Math.floor(allContents.value.length / pageSize.value) + 1
  await fetchPage(nextPage, true)
}


async function fetchFreshList(): Promise<Content[] | null> {
  const out: Content[] = []
  for (let p = 1; ; p++) {
    const res = await contentApi.list({ page: p, page_size: 100, sort_by: feedSort.value, order: 'desc' })
    if (res.code !== 200) return null
    out.push(...res.data.list.map((item: unknown) => ContentSchema.parse(item)))
    if (out.length >= res.data.total) break
  }
  return out
}

function captureViewAnchor(): { id: string | number; y: number; scrollY: number } | null {
  const scrollY = globalThis.scrollY
  let anchorId: string | number | null = null
  let anchorTop = Number.POSITIVE_INFINITY
  for (const [id, pos] of waterfall.positions.value) {
    if (pos.y >= scrollY - 4 && pos.y < anchorTop) {
      anchorId = id
      anchorTop = pos.y
    }
  }
  return anchorId != null ? { id: anchorId, y: anchorTop, scrollY } : null
}

async function syncLatestOnActivated() {
  if (feedMode.value === 'recommended') return
  if (swapSections.value || allContents.value.length === 0) return
  try {
    const probe = await contentApi.list({ page: 1, page_size: 100, sort_by: feedSort.value, order: 'desc' })
    if (probe.code !== 200) return
    const head = probe.data.list.map((item: unknown) => ContentSchema.parse(item))
    const newTotal = probe.data.total

    const localHead = allContents.value.slice(0, head.length)
    const headSame =
      localHead.length === head.length &&
      localHead.every((item, i) => item.id === head[i].id)
    const totalSame = newTotal === total.value

    if (headSame && totalSame) {
      const headMap = new Map(head.map((i) => [i.id, i]))
      for (const item of allContents.value) {
        const fresh = headMap.get(item.id)
        if (fresh && fresh.like_count !== item.like_count) {
          item.like_count = fresh.like_count
        }
      }
      return
    }

    const freshList = await fetchFreshList()
    if (!freshList) return
    const { merged, removed } = diffLists(allContents.value, freshList)

    const anchor = captureViewAnchor()
    allContents.value = merged.filter((item) => !removed.has(item.id))
    total.value = freshList.length
    totalPages.value = Math.ceil(total.value / pageSize.value)
    currentPage.value = Math.max(1, Math.ceil(allContents.value.length / pageSize.value))

    nextTick(() => {
      waterfall.relayout({ anchor: false, full: true })
      if (anchor) {
        const newPos = waterfall.positions.value.get(anchor.id)
        if (newPos) {
          const delta = newPos.y - anchor.y
          globalThis.scrollTo({ top: Math.max(0, anchor.scrollY + delta) })
        }
      }
    })
  } catch (e) {
    console.warn('返回首页同步最新列表失败:', e)
  }
}

function onUploaded() {
  if (
    feedMode.value !== 'latest' ||
    homeStore.searchKeyword ||
    searchFilter.selectedTags.value.length > 0
  ) {
    feedMode.value = 'latest'
    homeStore.searchKeyword = ''
    searchFilter.selectedTags.value = []
    resetAndLoad(false)
    return
  }
  void syncLatestOnActivated()
}

useIntersectionObserver(sentinelRef, ([{ isIntersecting }]) => {
  if (isIntersecting && hasMore.value && !isLoading.value && !isLoadingMore.value) {
    fetchMore()
  }
}, { rootMargin: '1000px' })

onMounted(() => {
  const navEntries = performance.getEntriesByType('navigation') as PerformanceNavigationTiming[]
  if (navEntries.length > 0 && navEntries[0].type === 'reload') {
    homeStore.clearState()
    listCache.clear()
  }

  void searchFilter.loadTags()
  void recommendLoader.loadRecommendContents()

  if (homeStore.searchKeyword) {
    fetchPage(1)
    return
  }

  if (homeStore.hasLoaded && homeStore.cachedContents.length > 0) {
    allContents.value = homeStore.cachedContents
    currentPage.value = homeStore.page
    total.value = homeStore.cachedTotal
    totalPages.value = homeStore.cachedTotalPages
    if (homeStore.cachedPositions.size > 0) {
      waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
    }
    nextTick(() => waterfall.relayout())
    nextTick(() => requestAnimationFrame(() => homeStore.restoreScroll()))
    return
  }

  const cached = feedMode.value === 'latest' ? listCache.load() : null
  if (cached && cached.list.length > 0) {
    allContents.value = cached.list
    total.value = cached.total
    totalPages.value = cached.totalPages
    homeStore.catalogTotal = cached.total
    currentPage.value = 1
    void loadAllPages()
  } else if (feedMode.value === 'recommended') {
    void fetchPage(1, false, true)
  } else {
    void loadAllPages()
  }
})

onActivated(() => {
  isHomeActive = true
  if (homeStore.cachedPositions.size > 0) {
    waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
  }
  nextTick(() => {
    if (!isHomeActive) return
    waterfall.relayout()
    void nextTick(() => {
      if (!isHomeActive) return
      homeStore.restoreScroll()
      pendingScrollRestore.value = Math.abs(globalThis.scrollY - homeStore.scrollPosition) > 1
    })
  })

  pendingScrollRestore.value = true

  const firstActivation = isFirstActivation
  isFirstActivation = false
  restoreTimer = setTimeout(() => {
    restoreTimer = undefined
    if (pendingScrollRestore.value) homeStore.restoreScroll()
    pendingScrollRestore.value = false
    if (firstActivation) return
    void syncLatestOnActivated()
  }, 300)
})
</script>

<template>
  <div class="gallery-home-content">
      <div class="gallery-content">
        <h1 class="gallery-visually-hidden">作品广场</h1>

        <section class="gallery-featured" aria-labelledby="featured-title">
          <div class="gallery-section-heading">
            <div>
              <span class="gallery-section-label">CURATED WORKS</span>
              <h2 id="featured-title">正在被看见</h2>
            </div>
            <button class="gallery-icon-action" type="button" :disabled="recommendLoader.isRecommendLoading.value" aria-label="换一批推荐作品" title="换一批" @click="recommendLoader.refreshRecommend(8)">
              <SiteIcon name="refresh" :spin="recommendLoader.isRecommendLoading.value" />
            </button>
          </div>
          <TransitionGroup v-if="featuredWorks.length" tag="div" name="gallery-featured" class="gallery-featured-strip">
            <button v-for="item in featuredWorks" :key="item.id" class="gallery-featured-item" type="button" :data-preload-route="`/content/${item.id}`" @click="openContent(item)">
              <SiteImage class="gallery-featured-thumb" :src="item.thumb" :alt="item.title" />
              <span>{{ item.title }}</span>
            </button>
          </TransitionGroup>
          <div v-else-if="recommendLoader.isRecommendLoading.value" class="gallery-featured-strip is-skeleton" aria-hidden="true">
            <div v-for="n in 6" :key="n" class="gallery-featured-item"><span class="gallery-skeleton-thumb"></span><span class="gallery-skeleton-line"></span></div>
          </div>
          <div v-else class="gallery-featured-empty">推荐作品正在整理中</div>
        </section>

        <section class="gallery-feed-toolbar" aria-label="作品筛选">
          <div class="gallery-filter-tabs" role="tablist" aria-label="作品排序">
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'recommended' }" type="button" role="tab" :aria-selected="feedMode === 'recommended'" @click="selectFeedMode('recommended')">推荐</button>
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'latest' }" type="button" role="tab" :aria-selected="feedMode === 'latest'" @click="selectFeedMode('latest')">最新</button>
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'popular' }" type="button" role="tab" :aria-selected="feedMode === 'popular'" @click="selectFeedMode('popular')">热门</button>
          </div>
          <div class="gallery-tag-row" role="group" aria-label="标签筛选">
            <button class="gallery-tag" :class="{ 'is-active': searchFilter.selectedTags.value.length === 0 }" type="button" @click="resetAndLoad()">全部</button>
            <button v-for="tag in searchFilter.shuffledTags.value.slice(0, 7)" :key="tag" class="gallery-tag" :class="{ 'is-active': searchFilter.selectedTags.value.includes(tag) }" type="button" :title="tag" @click="onTagSelect(tag)">{{ tag }}</button>
            <button v-if="searchFilter.shuffledTags.value.length > 1" class="gallery-tag gallery-tag-refresh" type="button" aria-label="换一组标签" title="换一组标签" @click="searchFilter.reshuffleTags()"><SiteIcon name="refresh" /></button>
          </div>
        </section>

        <section class="gallery-feed" aria-labelledby="gallery-feed-title">
          <div class="gallery-feed-heading">
            <div>
              <h2 id="gallery-feed-title">{{ homeStore.searchKeyword ? '搜索结果' : searchFilter.selectedTags.value[0] || (feedMode === 'recommended' ? '推荐作品' : feedMode === 'popular' ? '热门作品' : '最新作品') }}</h2>
              <p v-if="homeStore.searchKeyword || searchFilter.selectedTags.value.length">{{ homeStore.searchKeyword ? `正在查找“${homeStore.searchKeyword}”` : '按标签筛选' }} · {{ total }} 件作品</p>
              <p v-else-if="feedMode === 'recommended'">{{ total > 0 ? `为你挑选 ${total} 件作品` : '推荐作品正在整理' }}</p>
              <p v-else>{{ total > 0 ? `${total} 件作品，持续更新中` : '作品正在加载' }}</p>
            </div>
            <button v-if="swapSections" class="gallery-clear-filter" type="button" @click="clearFilters">清除筛选</button>
          </div>

          <div v-if="isLoading && allContents.length === 0" class="gallery-skeleton" role="status" aria-label="正在加载作品">
            <div v-for="n in 8" :key="n" class="gallery-skeleton-card" :style="{ '--sk-h': 140 + ((n * 37) % 90) + 'px' }">
              <span class="gallery-skeleton-thumb"></span>
              <span class="gallery-skeleton-line"></span>
              <span class="gallery-skeleton-line is-short"></span>
            </div>
          </div>
          <div v-else-if="!isLoading && allContents.length === 0" class="gallery-state">
            <div class="gallery-state-mark"><SiteIcon name="image" /></div>
            <p>{{ homeStore.searchKeyword || searchFilter.selectedTags.value.length ? '没有找到符合条件的作品' : '还没有作品出现在这里' }}</p>
            <button type="button" @click="goToUpload">发布第一件作品 <SiteIcon name="arrow-right" /></button>
          </div>
          <div v-else ref="masonryRef" class="gallery-masonry" :style="{ position: 'relative', height: waterfall.containerHeight.value + 'px' }">
            <WaterfallCard
              v-for="item in allContents"
              :key="item.id"
              :item="item"
              :data-wf-id="item.id"
              :style="{ position: 'absolute', left: (waterfall.positions.value.get(item.id)?.x ?? 0) + 'px', top: (waterfall.positions.value.get(item.id)?.y ?? 0) + 'px', width: (waterfall.positions.value.get(item.id)?.w ?? 0) + 'px' }"
              @click="openContent"
              @image-loaded="onCardResized"
              @liked="onCardLiked"
            />
          </div>
          <div v-if="isLoadingMore" class="gallery-loadmore"><div class="gallery-state-spinner"></div><span>加载更多作品</span></div>
          <div ref="sentinelRef" class="gallery-sentinel"></div>
          <div v-if="!hasMore && allContents.length > 0" class="gallery-end">已经看到这里了</div>
        </section>
      </div>
    <button class="gallery-mobile-publish" type="button" aria-label="发布作品" @click="goToUpload"><SiteIcon name="plus" /></button>
    <QuickUploadSheet :open="showUploadSheet" @close="showUploadSheet = false" @uploaded="onUploaded" />
  </div>
</template>

<style scoped>
.gallery-home-content {
  --gallery-title-font: var(--site-title-font);
  --gallery-canvas: var(--site-canvas);
  --gallery-paper: var(--site-paper);
  --gallery-ink: var(--site-ink);
  --gallery-muted: var(--site-muted);
  --gallery-line: var(--site-line);
  --gallery-soft: var(--site-soft);
  --gallery-accent: var(--site-accent);
  --gallery-accent-soft: var(--site-accent-soft);
  min-width: 0;
  color: var(--gallery-ink);
  font-family: var(--font-sans);
}
@media (min-width: 821px) {
  :global(.app-layout.home-gallery-mode .public-shell-content) {
    max-width: min(2240px, 100%);
    padding-left: clamp(20px, 2.2vw, 40px);
    padding-right: clamp(20px, 2.2vw, 40px);
  }
}
.gallery-mobile-publish { display: none; }
.gallery-content {
  width: 100%;
  margin: 0;
  padding: 0 0 76px;
}
.gallery-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}
.gallery-featured {
  padding: 24px 0 20px;
  border-bottom: 1px solid var(--gallery-line);
  scroll-margin-top: 16px;
}
.gallery-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 13px;
}
.gallery-section-label {
  color: var(--gallery-muted);
  font-size: 9px;
  letter-spacing: .16em;
}
.gallery-section-heading h2,
.gallery-feed-heading h2 {
  margin: 4px 0 0;
  color: var(--gallery-ink);
  font: 600 21px/1.2 var(--gallery-title-font);
}
.gallery-icon-action {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid var(--gallery-line);
  border-radius: 50%;
  background: var(--gallery-paper);
  color: var(--gallery-muted);
  cursor: pointer;
  transition: color .18s ease, border-color .18s ease, transform .18s ease;
}
.gallery-icon-action:hover:not(:disabled) {
  border-color: var(--gallery-accent);
  color: var(--gallery-accent);
  transform: rotate(25deg);
}
.gallery-icon-action:disabled { cursor: wait; opacity: .5; }
.gallery-icon-action :deep(svg) { width: 15px; height: 15px; }
.gallery-featured-strip {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(108px, 1fr);
  gap: 12px;
  overflow-x: auto;
  padding: 2px 2px 3px;
  scrollbar-width: thin;
  scrollbar-color: var(--gallery-line) transparent;
}
.gallery-featured-item {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--gallery-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gallery-featured-item :deep(.site-image) {
  aspect-ratio: 1.15;
  min-height: 0;
  border-radius: 10px;
  box-shadow: 0 1px 0 var(--gallery-line);
  transition: box-shadow .28s ease, transform .28s cubic-bezier(.22, 1, .36, 1);
}
.gallery-featured-item :deep(.site-image-img) {
  height: 100%;
  object-fit: cover;
  transition: opacity .32s ease, transform .4s cubic-bezier(.22, 1, .36, 1);
}
.gallery-featured-item:hover :deep(.site-image) {
  transform: translateY(-3px);
  box-shadow: 0 12px 24px color-mix(in srgb, var(--gallery-ink) 14%, transparent);
}
.gallery-featured-item:hover :deep(.site-image-img) { transform: scale(1.06); }
.gallery-featured-item > span {
  overflow: hidden;
  color: var(--gallery-ink);
  font-size: 11px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color .18s ease;
}
.gallery-featured-item:hover > span { color: var(--gallery-accent); }
.gallery-featured-empty {
  min-height: 76px;
  padding: 26px 0;
  color: var(--gallery-muted);
  font-size: 11px;
}
.gallery-feed-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 22px 0 17px;
}
.gallery-filter-tabs {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex: 0 0 auto;
  padding: 3px;
  border: 1px solid var(--gallery-line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--gallery-paper) 70%, transparent);
}
.gallery-filter-tab {
  position: relative;
  padding: 6px 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
  transition: color .18s ease, background-color .22s ease, box-shadow .22s ease, transform .18s ease;
}
.gallery-filter-tab:hover { color: var(--gallery-ink); }
.gallery-filter-tab:active { transform: scale(.96); }
.gallery-filter-tab.is-active {
  background: var(--gallery-paper);
  color: var(--gallery-ink);
  font-weight: 600;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--gallery-ink) 14%, transparent), 0 0 0 1px var(--gallery-line);
}
.gallery-tag-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.gallery-tag-row::-webkit-scrollbar { display: none; }
.gallery-tag {
  flex: 0 0 auto;
  min-height: 28px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: background-color .18s ease, color .18s ease, border-color .18s ease, transform .15s ease;
}
.gallery-tag:hover,
.gallery-tag.is-active {
  border-color: var(--gallery-line);
  background: var(--gallery-paper);
  color: var(--gallery-accent);
}
.gallery-tag.is-active { border-color: color-mix(in srgb, var(--gallery-accent) 38%, var(--gallery-line)); font-weight: 600; }
.gallery-tag:active { transform: scale(.95); }
.gallery-tag-refresh { padding: 0 7px; }
.gallery-tag-refresh :deep(svg) { width: 13px; height: 13px; }
.gallery-feed {
  scroll-margin-top: 18px;
}
.gallery-feed-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 20px;
}
.gallery-feed-heading p {
  margin: 7px 0 0;
  color: var(--gallery-muted);
  font-size: 11px;
  line-height: 1.5;
}
.gallery-clear-filter {
  padding: 0 0 4px;
  border: 0;
  background: transparent;
  color: var(--gallery-accent);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: color .18s ease, transform .18s ease;
}
.gallery-clear-filter:hover { color: var(--gallery-ink); }
.gallery-clear-filter:active { transform: scale(.98); }
.gallery-masonry {
  width: 100%;
}
.gallery-state {
  display: grid;
  place-items: center;
  min-height: 320px;
  gap: 12px;
  color: var(--gallery-muted);
  text-align: center;
}
.gallery-state p {
  margin: 0;
  font-size: 12px;
}
.gallery-state button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: var(--gallery-accent);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: color .18s ease, transform .18s ease;
}
.gallery-state button:hover { color: var(--gallery-ink); }
.gallery-state button:active { transform: scale(.98); }
.gallery-state button :deep(svg) { width: 14px; height: 14px; }
.gallery-state-mark {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: var(--gallery-soft);
  color: var(--gallery-accent);
}
.gallery-state-mark :deep(svg) { width: 20px; height: 20px; }
.gallery-state-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--gallery-line);
  border-top-color: var(--gallery-accent);
  border-radius: 50%;
  animation: gallery-spin .8s linear infinite;
}
.gallery-loadmore {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  padding: 28px 0;
  color: var(--gallery-muted);
  font-size: 11px;
}
.gallery-loadmore .gallery-state-spinner { width: 14px; height: 14px; }
.gallery-sentinel { height: 1px; }

.gallery-featured-enter-active { transition: opacity .3s ease, transform .36s cubic-bezier(.22, 1, .36, 1); }
.gallery-featured-enter-from { opacity: 0; transform: translateY(8px); }

.gallery-skeleton {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 18px;
}
.gallery-skeleton-card { display: grid; gap: 9px; }
.gallery-skeleton-thumb,
.gallery-skeleton-line {
  position: relative;
  display: block;
  overflow: hidden;
  border-radius: 10px;
  background: var(--gallery-soft);
}
.gallery-skeleton-thumb { height: var(--sk-h, 150px); }
.gallery-featured-item > .gallery-skeleton-thumb { height: auto; aspect-ratio: 1.15; }
.gallery-skeleton-line { height: 11px; border-radius: 6px; }
.gallery-skeleton-line.is-short { width: 58%; }
.gallery-skeleton-thumb::after,
.gallery-skeleton-line::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 20%, color-mix(in srgb, var(--gallery-paper) 60%, transparent) 50%, transparent 80%);
  background-size: 220% 100%;
  animation: gallery-shimmer 1.4s ease-in-out infinite;
}
@keyframes gallery-shimmer {
  from { background-position: 120% 0; }
  to { background-position: -120% 0; }
}
.gallery-end {
  padding: 34px 0 18px;
  color: var(--gallery-muted);
  font-size: 10px;
  text-align: center;
  letter-spacing: .06em;
}

.gallery-home-content button:focus-visible,
.gallery-home-content a:focus-visible,
.gallery-home-content input:focus-visible {
  outline: 2px solid var(--gallery-accent);
  outline-offset: 3px;
}
@keyframes gallery-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  .gallery-home-content *,
  .gallery-home-content *::before,
  .gallery-home-content *::after {
    scroll-behavior: auto !important;
    transition-duration: .01ms !important;
    animation-duration: .01ms !important;
  }
}
@media (max-width: 1100px) {
  .gallery-content { padding-inline: 0; }
}
@media (max-width: 820px) {
  .gallery-content { padding: 0 0 60px; }
  .gallery-featured-strip { grid-auto-columns: minmax(105px, 30vw); }
  .gallery-feed-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
  .gallery-tag-row { flex-wrap: nowrap; justify-content: flex-start; width: 100%; }
  .gallery-feed-heading { align-items: flex-start; }
  .gallery-mobile-publish {
    position: fixed;
    right: 18px;
    bottom: 20px;
    z-index: 18;
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    border: 0;
    border-radius: 50%;
    background: var(--gallery-ink);
    color: var(--gallery-paper);
    box-shadow: 0 10px 22px rgba(31, 24, 27, .22);
    cursor: pointer;
    transition: transform .18s ease, box-shadow .18s ease;
  }
  .gallery-mobile-publish:hover { transform: translateY(-2px); box-shadow: 0 12px 26px rgba(31, 24, 27, .26); }
  .gallery-mobile-publish:active { transform: scale(.94); }
  .gallery-mobile-publish :deep(svg) { width: 20px; height: 20px; }
}
@media (max-width: 520px) {
  .gallery-search kbd { display: none; }
  .gallery-section-heading h2,
  .gallery-feed-heading h2 { font-size: 19px; }
  .gallery-featured-strip { gap: 9px; grid-auto-columns: minmax(102px, 31vw); }
  .gallery-featured-item > span { font-size: 10px; }
  .gallery-filter-tabs { gap: 18px; }
}
</style>

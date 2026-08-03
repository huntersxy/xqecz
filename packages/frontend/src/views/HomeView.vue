<script lang="ts">
export default { name: 'HomeView' }
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, nextTick, watch } from 'vue'
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import { useListCache, diffLists } from '@/composables/useListCache'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import RecommendSection from '@/components/RecommendSection.vue'
import QuickUploadSheet from '@/components/QuickUploadSheet.vue'
import { IconUpload } from '@arco-design/web-vue/es/icon'
import type { Content, ListParams, RecommendContent } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const listCache = useListCache()
const showUploadSheet = ref(false)

const swapSections = computed(
  () =>
    searchFilter.selectedTags.value.length > 0 ||
    searchFilter.selectedTypes.value.length > 0 ||
    !!homeStore.searchKeyword.trim(),
)

const allContents = ref<Content[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const totalPages = ref(1)
const isLoading = ref(false)
const isLoadingMore = ref(false)
const hasMore = computed(() => currentPage.value <= totalPages.value)
const sentinelRef = ref<HTMLElement | null>(null)
const masonryRef = ref<HTMLElement | null>(null)

const waterfall = useWaterfallLayout(masonryRef, allContents)

// 是否需要在布局完成后恢复滚动位置
const pendingScrollRestore = ref(false)

// 首次加载或重置时全量布局
watch(
  () => allContents.value.length,
  (newLen, oldLen) => {
    if (newLen === 0) {
      waterfall.positions.value.clear()
      waterfall.containerHeight.value = 0
      return
    }
    if (oldLen === 0 || newLen < oldLen) {
      nextTick(() => waterfall.relayout())
    }
    // 数据变化后检查是否需要加载更多
    checkAndLoadMore()
  },
)

function onImageLoaded(id: string | number) {
  waterfall.onImageLoaded(id)
  // 如果有待恢复的滚动位置，在图片加载后尝试恢复
  if (pendingScrollRestore.value) {
    requestAnimationFrame(() => {
      const target = homeStore.scrollPosition
      // 只有当容器高度足够时才恢复
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

// 离开首页（详情/后台/登录等任意跳转）时统一保存滚动位置与列表状态，
// 返回时由 onActivated 恢复（keep-alive）。
onBeforeRouteLeave(() => {
  const pos = globalThis.scrollY
  homeStore.saveState({
    searchKeyword: homeStore.searchKeyword,
    selectedTags: searchFilter.selectedTags.value,
    selectedTypes: searchFilter.selectedTypes.value,
    page: currentPage.value,
    recommendPage: recommendLoader.loadedPage.value,
    scrollPosition: pos,
    contents: allContents.value,
    total: total.value,
    totalPages: totalPages.value,
    positions: new Map(waterfall.positions.value),
    containerHeight: waterfall.containerHeight.value,
  })
})

async function fetchPage(page: number, append = false) {
  if (isLoading.value || isLoadingMore.value) return
  if (append) isLoadingMore.value = true
  else isLoading.value = true

  try {
    let res
    if (homeStore.searchKeyword) {
      res = await contentApi.search(homeStore.searchKeyword, {
        page, page_size: pageSize.value,
        tag: searchFilter.selectedTags.value.length > 0 ? searchFilter.selectedTags.value.join(',') : undefined,
        type: searchFilter.selectedTypes.value.length > 0 ? searchFilter.selectedTypes.value.join(',') : undefined,
      })
    } else {
      const params: ListParams = { page, page_size: pageSize.value, sort_by: 'created_at', order: 'desc' }
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
        const oldLen = allContents.value.length
        allContents.value.push(...parsed)
        nextTick(() => waterfall.appendNewItems(allContents.value.slice(oldLen)))
      } else {
        allContents.value = parsed
      }
      // 更新缓存
      if (!homeStore.searchKeyword) {
        listCache.save(allContents.value, total.value, totalPages.value)
      }
    }
  } catch (e) { console.error('加载失败:', e) }
  finally { isLoading.value = false; isLoadingMore.value = false }
}

function resetAndLoad() {
  currentPage.value = 1
  allContents.value = []
  listCache.clear()
  fetchPage(1)
}

watchGlobalSearch(() => resetAndLoad())

// 增量加载更多时也更新缓存
async function fetchMore() {
  if (!hasMore.value || isLoading.value || isLoadingMore.value) return
  await fetchPage(currentPage.value + 1, true)
}

// 备用触发：列表高度不够时自动加载更多
function checkAndLoadMore() {
  nextTick(() => {
    if (!hasMore.value || isLoading.value || isLoadingMore.value) return
    const el = masonryRef.value
    if (!el) return
    // 列表高度小于视口高度的 1.5 倍时，自动加载更多
    if (el.scrollHeight < window.innerHeight * 1.5) {
      fetchMore()
    }
  })
}

// 异步 diff：拿最新数据与缓存对比，更新列表
async function diffAndUpdate() {
  try {
    const res = await contentApi.list({ page: 1, page_size: 100, sort_by: 'created_at', order: 'desc' })
    if (res.code !== 200) return

    const freshList = res.data.list.map((item: unknown) => ContentSchema.parse(item))
    const { merged, removed } = diffLists(allContents.value, freshList)

    allContents.value = merged.filter((item) => !removed.has(item.id))
    total.value = res.data.total
    totalPages.value = res.data.total_page

    listCache.save(allContents.value, total.value, totalPages.value)
    nextTick(() => waterfall.relayout())
  } catch (e) {
    console.warn('diff 更新失败:', e)
  }
}

// 返回首页（上传 / 详情页等跳转后）时：拉取最新列表与当前列表 diff，
// 有新增/删除时重排瀑布流并锚定当前可见内容；仅字段变化时原位更新（不打断滚动）。
async function syncLatestOnActivated() {
  if (homeStore.searchKeyword || allContents.value.length === 0) return
  try {
    const res = await contentApi.list({ page: 1, page_size: 100, sort_by: 'created_at', order: 'desc' })
    if (res.code !== 200) return
    const freshList = res.data.list.map((item: unknown) => ContentSchema.parse(item))
    const { merged, added, removed } = diffLists(allContents.value, freshList)

    // 无结构变化：仅原位同步字段（点赞数等），不重排、不打断滚动
    if (added.size === 0 && removed.size === 0) {
      const freshMap = new Map(freshList.map((i) => [i.id, i]))
      let changed = false
      for (const item of allContents.value) {
        const fresh = freshMap.get(item.id)
        if (fresh && fresh.like_count !== item.like_count) {
          item.like_count = fresh.like_count
          changed = true
        }
      }
      if (changed) listCache.save(allContents.value, total.value, totalPages.value)
      return
    }

    // 记录锚点：当前视口顶部第一张卡片（插入新卡片后保持同一内容的阅读位置）
    const viewTop = globalThis.scrollY
    let anchorId: string | number | null = null
    let anchorTop = Number.POSITIVE_INFINITY
    for (const [id, pos] of waterfall.positions.value) {
      if (pos.y >= viewTop - 4 && pos.y < anchorTop) {
        anchorId = id
        anchorTop = pos.y
      }
    }

    allContents.value = merged.filter((item) => !removed.has(item.id))
    total.value = res.data.total
    totalPages.value = res.data.total_page
    listCache.save(allContents.value, total.value, totalPages.value)

    nextTick(() => {
      waterfall.relayout()
      if (anchorId != null) {
        const newPos = waterfall.positions.value.get(anchorId)
        if (newPos) {
          const delta = newPos.y - anchorTop
          globalThis.scrollTo({ top: Math.max(0, viewTop + delta) })
        }
      }
    })
  } catch (e) {
    console.warn('返回首页同步最新列表失败:', e)
  }
}

// 首页悬浮按钮上传成功后：同样拉取最新列表 diff（新作品插入顶部，锚定当前阅读位置）
function onUploaded() {
  void syncLatestOnActivated()
}

// 监听 sentinel 进入视口
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

  // 搜索模式不使用缓存
  if (homeStore.searchKeyword) {
    fetchPage(1)
    return
  }

  // 恢复路由缓存（keep-alive）
  if (homeStore.hasLoaded && homeStore.cachedContents.length > 0) {
    allContents.value = homeStore.cachedContents
    currentPage.value = homeStore.page
    total.value = homeStore.cachedTotal
    totalPages.value = homeStore.cachedTotalPages
    recommendLoader.loadedPage.value = homeStore.recommendPage
    // 恢复瀑布流布局缓存
    if (homeStore.cachedPositions.size > 0) {
      waterfall.positions.value = new Map(homeStore.cachedPositions)
      waterfall.containerHeight.value = homeStore.cachedContainerHeight
      waterfall.isLayoutReady.value = true
    }
    nextTick(() => requestAnimationFrame(() => homeStore.restoreScroll()))
    return
  }

  // 尝试从 localStorage 加载缓存
  const cached = listCache.load()
  if (cached && cached.list.length > 0) {
    allContents.value = cached.list
    total.value = cached.total
    totalPages.value = cached.totalPages
    currentPage.value = 1
    // 异步 diff，不阻塞渲染
    diffAndUpdate()
  } else {
    fetchPage(1)
  }

  recommendLoader.loadRecommendContents()
  searchFilter.loadTags()
})

// keep-alive 激活时恢复滚动位置
onActivated(() => {

  // 恢复瀑布流布局缓存
  if (homeStore.cachedPositions.size > 0) {
    waterfall.positions.value = new Map(homeStore.cachedPositions)
    waterfall.containerHeight.value = homeStore.cachedContainerHeight
    waterfall.isLayoutReady.value = true
  }

  // 标记需要在图片加载后恢复滚动
  pendingScrollRestore.value = true

  // 延迟恢复滚动，再拉取最新列表 diff（有新增时锚定当前可见内容，不丢阅读位置）
  setTimeout(() => {
    pendingScrollRestore.value = false
    homeStore.restoreScroll()
    void syncLatestOnActivated()
  }, 300)
})
</script>

<template>
  <div class="wf-root">
    <RecommendSection
      v-if="!swapSections && recommendLoader.recommendContents.value.length > 0"
      :loader="recommendLoader"
      @click="openContent"
    />

    <div class="wf-masonry-wrap">
      <div v-if="isLoading && allContents.length === 0" class="wf-center-state">
        <div class="wf-spinner-lg"></div><p>加载中...</p>
      </div>
      <div v-else-if="!isLoading && allContents.length === 0" class="wf-center-state">
        <p>暂无内容</p>
      </div>
      <div
        v-else
        ref="masonryRef"
        class="wf-masonry"
        :style="{ position: 'relative', height: waterfall.containerHeight.value + 'px' }"
      >
        <WaterfallCard
          v-for="item in allContents"
          :key="item.id"
          :item="item"
          :data-wf-id="item.id"
          :style="{
            position: 'absolute',
            left: (waterfall.positions.value.get(item.id)?.x ?? 0) + 'px',
            top: (waterfall.positions.value.get(item.id)?.y ?? 0) + 'px',
            width: (waterfall.positions.value.get(item.id)?.w ?? 0) + 'px',
          }"
          @click="openContent"
          @image-loaded="onImageLoaded"
        />
      </div>
      <div v-if="isLoadingMore" class="wf-loadmore"><div class="wf-spinner-sm"></div><span>加载更多...</span></div>
      <div ref="sentinelRef" class="wf-sentinel"></div>
      <div v-if="!hasMore && allContents.length > 0" class="wf-end"><span>— 到底啦 —</span></div>
    </div>

    <!-- 移动端悬浮上传按钮 -->
    <button class="wf-fab" @click="showUploadSheet = true">
      <IconUpload />
    </button>

    <!-- 快速上传弹窗 -->
    <QuickUploadSheet :open="showUploadSheet" @close="showUploadSheet = false" @uploaded="onUploaded" />
  </div>
</template>

<style lang="scss" scoped>
.wf-root { min-height: 100vh; background: transparent; color: var(--color-text-1); }

.wf-masonry-wrap {
  max-width: 1600px; margin: 0 auto; padding: 0.75rem 0.75rem 3rem;
  @media (min-width: 640px) { padding: 1rem 1rem 3rem; }
}
.wf-masonry { width: 100%; }

.wf-center-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 4rem 1rem; color: var(--color-text-2);
}
.wf-center-state p { font-size: 0.875rem; margin-top: 1rem; }

.wf-spinner-lg {
  width: 2rem; height: 2rem; border: 3px solid var(--color-border-2);
  border-top-color: rgb(var(--primary-6)); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-spinner-sm {
  width: 1.25rem; height: 1.25rem; border: 2px solid var(--color-border-2);
  border-top-color: rgb(var(--primary-6)); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-loadmore {
  display: flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 1.5rem; font-size: 0.8125rem; color: var(--color-text-2);
}
.wf-sentinel { height: 1px; }
.wf-end { text-align: center; padding: 2rem; font-size: 0.75rem; color: var(--color-text-2); opacity: 0.6; }

/* 悬浮上传按钮 */
.wf-fab {
  display: none;
}

@media (max-width: 768px) {
  .wf-fab {
    display: flex;
    align-items: center;
    justify-content: center;
    position: fixed;
    bottom: 24px;
    right: 24px;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgb(var(--primary-6));
    color: var(--color-white);
    border: none;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    cursor: pointer;
    z-index: 100;
    font-size: 22px;
    transition: transform 0.2s, box-shadow 0.2s;
  }
  .wf-fab:hover {
    transform: scale(1.05);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.25);
  }
  .wf-fab:active {
    transform: scale(0.95);
  }
}

@keyframes wf-spin { to { transform: rotate(360deg); } }
</style>

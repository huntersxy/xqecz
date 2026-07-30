<script setup lang="ts">
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import RecommendSection from '@/components/RecommendSection.vue'
import type { Content, ListParams } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()

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

function openContent(content: Content) {
  homeStore.saveState({
    searchKeyword: homeStore.searchKeyword,
    selectedTags: searchFilter.selectedTags.value,
    selectedTypes: searchFilter.selectedTypes.value,
    page: currentPage.value,
    recommendPage: recommendLoader.recommendPage.value,
    scrollPosition: globalThis.scrollY,
  })
  router.push(`/content/${content.id}`)
}

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
      if (append) allContents.value.push(...parsed)
      else allContents.value = parsed
    }
  } catch (e) { console.error('加载失败:', e) }
  finally { isLoading.value = false; isLoadingMore.value = false }
}

function resetAndLoad() { currentPage.value = 1; allContents.value = []; fetchPage(1) }

watchGlobalSearch(() => resetAndLoad())

useIntersectionObserver(sentinelRef, ([{ isIntersecting }]) => {
  if (isIntersecting && hasMore.value && !isLoading.value && !isLoadingMore.value) fetchPage(currentPage.value + 1, true)
}, { rootMargin: '200px' })

onMounted(() => {
  const navEntries = performance.getEntriesByType('navigation') as PerformanceNavigationTiming[]
  if (navEntries.length > 0 && navEntries[0].type === 'reload') homeStore.clearState()
  if (homeStore.hasLoaded) {
    homeStore.restoreScroll()
    recommendLoader.recommendPage.value = homeStore.recommendPage
  }
  fetchPage(1)
  searchFilter.loadTags()
  recommendLoader.loadRecommendContents(recommendLoader.recommendPage.value)
})
</script>

<template>
  <div class="wf-root">
    <RecommendSection
      v-if="!swapSections && recommendLoader.recommendContents.value.length > 0"
      :contents="recommendLoader.recommendContents.value"
      @click="openContent"
    />

    <div class="wf-masonry-wrap">
      <div v-if="isLoading && allContents.length === 0" class="wf-center-state">
        <div class="wf-spinner-lg"></div><p>加载中...</p>
      </div>
      <div v-else-if="!isLoading && allContents.length === 0" class="wf-center-state">
        <p>暂无内容</p>
      </div>
      <div v-else class="wf-masonry">
        <WaterfallCard v-for="item in allContents" :key="item.id" :item="item" @click="openContent" />
      </div>
      <div v-if="isLoadingMore" class="wf-loadmore"><div class="wf-spinner-sm"></div><span>加载更多...</span></div>
      <div ref="sentinelRef" class="wf-sentinel"></div>
      <div v-if="!hasMore && allContents.length > 0" class="wf-end"><span>— 到底啦 —</span></div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.wf-root { min-height: 100vh; background: transparent; color: var(--theme-text); }

.wf-masonry-wrap {
  max-width: 1600px; margin: 0 auto; padding: 0.75rem 0.75rem 3rem;
  @media (min-width: 640px) { padding: 1rem 1rem 3rem; }
}
.wf-masonry { columns: 2; column-gap: 10px; }
@media (min-width: 640px) { .wf-masonry { columns: 3; } }
@media (min-width: 1024px) { .wf-masonry { columns: 4; } }
@media (min-width: 1400px) { .wf-masonry { columns: 5; } }

.wf-center-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 4rem 1rem; color: var(--theme-text-secondary);
}
.wf-center-state p { font-size: 0.875rem; margin-top: 1rem; }

.wf-spinner-lg {
  width: 2rem; height: 2rem; border: 3px solid var(--theme-card-border);
  border-top-color: var(--theme-primary); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-spinner-sm {
  width: 1.25rem; height: 1.25rem; border: 2px solid var(--theme-card-border);
  border-top-color: var(--theme-primary); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-loadmore {
  display: flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 1.5rem; font-size: 0.8125rem; color: var(--theme-text-secondary);
}
.wf-sentinel { height: 1px; }
.wf-end { text-align: center; padding: 2rem; font-size: 0.75rem; color: var(--theme-text-secondary); opacity: 0.6; }

@keyframes wf-spin { to { transform: rotate(360deg); } }
</style>

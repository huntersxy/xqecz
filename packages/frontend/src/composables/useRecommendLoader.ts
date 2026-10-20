import { ref } from 'vue'
import { contentApi } from '@/api'
import { RecommendContentSchema, type RecommendContent } from '@/types'

const TOTAL_PAGES = 6
const PER_PAGE = 16

export function useRecommendLoader() {
  const recommendContents = ref<RecommendContent[]>([])
  const loadedPage = ref(0)
  const poolGeneration = ref(0)
  const isRecommendLoading = ref(false)
  const displayStart = ref(0)

  function parse(list: unknown[]): RecommendContent[] {
    return list.map((item) => RecommendContentSchema.parse(item))
  }

  async function loadRecommendContents(page = 1) {
    if (isRecommendLoading.value) return
    isRecommendLoading.value = true
    try {
      const res = await contentApi.recommend(PER_PAGE, page)
      if (res.code !== 200) throw new Error(res.message || '加载推荐失败')
      recommendContents.value = parse(res.data.list)
      loadedPage.value = page
      poolGeneration.value++
    } finally {
      isRecommendLoading.value = false
    }
  }

  async function loadNextPageIntoPool(): Promise<boolean> {
    if (isRecommendLoading.value) return false
    if (loadedPage.value >= TOTAL_PAGES) return false
    const next = loadedPage.value + 1
    isRecommendLoading.value = true
    try {
      const res = await contentApi.recommend(PER_PAGE, next)
      if (res.code !== 200) throw new Error(res.message || '加载推荐失败')
      const incoming = parse(res.data.list)
      const seen = new Set(recommendContents.value.map((c) => c.id))
      const fresh = incoming.filter((c) => !seen.has(c.id))
      if (fresh.length > 0) {
        recommendContents.value = [...recommendContents.value, ...fresh]
      }
      loadedPage.value = next
      return fresh.length > 0
    } catch {
      return false
    } finally {
      isRecommendLoading.value = false
    }
  }

  async function refreshRecommend(batch = PER_PAGE) {
    if (isRecommendLoading.value) return
    const visibleCount = () =>
      recommendContents.value.filter((item) => item.thumb).length
    const next = displayStart.value + batch
    if (next + batch > visibleCount() && loadedPage.value < TOTAL_PAGES) {
      await loadNextPageIntoPool()
    }
    const count = visibleCount()
    displayStart.value = count === 0 || next >= count ? 0 : next
  }

  return {
    recommendContents,
    loadedPage,
    poolGeneration,
    isRecommendLoading,
    maxRecommendPages: TOTAL_PAGES,
    pageSize: PER_PAGE,
    displayStart,
    loadRecommendContents,
    loadNextPageIntoPool,
    refreshRecommend,
  }
}

export type RecommendLoader = ReturnType<typeof useRecommendLoader>

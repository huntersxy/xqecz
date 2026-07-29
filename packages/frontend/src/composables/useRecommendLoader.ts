import { ref, nextTick } from 'vue'
import { contentApi } from '@/api'
import { RecommendContentSchema, type RecommendContent } from '@/types'

const TOTAL_PAGES = 6
const PER_PAGE = 16

export function useRecommendLoader() {
  const recommendContents = ref<RecommendContent[]>([])
  const recommendPage = ref(1)
  const recommendHint = ref('')
  const isRecommendLoading = ref(false)

  async function loadRecommendContents(page?: number) {
    if (isRecommendLoading.value) return
    if (page !== undefined) {
      recommendPage.value = page
    }
    isRecommendLoading.value = true
    try {
      const res = await contentApi.recommend(PER_PAGE, recommendPage.value)
      if (res.code !== 200) {
        throw new Error(res.message || '加载推荐失败')
      }
      recommendContents.value = res.data.list.map((item) => RecommendContentSchema.parse(item))
    } finally {
      isRecommendLoading.value = false
    }
  }

  async function refreshRecommend() {
    if (isRecommendLoading.value) return

    const page = recommendPage.value >= TOTAL_PAGES ? 1 : recommendPage.value + 1
    recommendPage.value = page

    await loadRecommendContents()
    await nextTick()
    const el =
      document.getElementById('recommend-section-liquid') ??
      document.getElementById('recommend-section-default')
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  return {
    recommendContents,
    isRecommendLoading,
    recommendPage,
    maxRecommendPages: TOTAL_PAGES,
    recommendHint,
    loadRecommendContents,
    refreshRecommend,
  }
}

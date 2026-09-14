import { ref } from 'vue'
import { contentApi } from '@/api'
import { RecommendContentSchema, type RecommendContent } from '@/types'

const TOTAL_PAGES = 6
const PER_PAGE = 16

export function useRecommendLoader() {
  const recommendContents = ref<RecommendContent[]>([])
  const loadedPage = ref(0) // 已并入池中的最大页码（0 = 尚未加载）
  const poolGeneration = ref(0) // 整池替换次数（刷新 / 初始加载 +1），供组件重置破图记录
  const isRecommendLoading = ref(false)
  const displayStart = ref(0) // 展示窗口起点（按可见好图计数）

  function parse(list: unknown[]): RecommendContent[] {
    return list.map((item) => RecommendContentSchema.parse(item))
  }

  // 用指定页替换整个推荐池（初始加载 / 刷新）
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

  // 从下一页拉取并入池（用于破图 / 文字类导致数量不足时垫补）
  // 返回 true 表示成功并入新内容；false 表示无更多页或无新内容
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

  // 刷新：展示窗口前进一整批（16 张好图）。
  // 窗口进入"剩余的"池内容；剩余不足时由 RecommendSection 的 ensureFilled
  // 自动拉"更下一页"垫补；池耗尽且窗口越界时由 ensureFilled 回卷到 0。
  // 不滚动页面。
  async function refreshRecommend() {
    if (isRecommendLoading.value) return
    displayStart.value += PER_PAGE
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

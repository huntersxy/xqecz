import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Content } from '@/types'

export interface CachedPosition {
  x: number
  y: number
  w: number
}

export const useHomeStore = defineStore('home', () => {
  // 搜索和筛选状态
  const searchKeyword = ref('')
  const selectedTags = ref<string[]>([])
  const selectedTypes = ref<string[]>([])
  const page = ref(1)
  
  // 推荐内容页码
  const recommendPage = ref(1)
  
  // 滚动位置
  const scrollPosition = ref(0)
  
  // 是否已经加载过数据（用于判断是否需要恢复状态）
  const hasLoaded = ref(false)

  // 缓存列表数据（避免返回时重新请求）
  const cachedContents = ref<Content[]>([])
  const cachedTotal = ref(0)
  const cachedTotalPages = ref(1)

  // 缓存瀑布流布局（避免返回时重新计算）
  const cachedPositions = ref<Map<string | number, CachedPosition>>(new Map())
  const cachedContainerHeight = ref(0)

  // 搜索触发器（每次 +1 通知订阅者重新查询）
  const searchTrigger = ref(0)
  function triggerSearch() {
    searchTrigger.value++
  }

  // 保存状态
  function saveState(params: {
    searchKeyword: string
    selectedTags: string[]
    selectedTypes: string[]
    page: number
    recommendPage: number
    scrollPosition: number
    contents?: Content[]
    total?: number
    totalPages?: number
    positions?: Map<string | number, CachedPosition>
    containerHeight?: number
  }) {
    searchKeyword.value = params.searchKeyword
    selectedTags.value = params.selectedTags
    selectedTypes.value = params.selectedTypes
    page.value = params.page
    recommendPage.value = params.recommendPage
    scrollPosition.value = params.scrollPosition
    if (params.contents) cachedContents.value = params.contents
    if (params.total !== undefined) cachedTotal.value = params.total
    if (params.totalPages !== undefined) cachedTotalPages.value = params.totalPages
    if (params.positions) cachedPositions.value = params.positions
    if (params.containerHeight !== undefined) cachedContainerHeight.value = params.containerHeight
    hasLoaded.value = true
  }

  // 清除状态
  function clearState() {
    searchKeyword.value = ''
    selectedTags.value = []
    selectedTypes.value = []
    page.value = 1
    recommendPage.value = 1
    scrollPosition.value = 0
    hasLoaded.value = false
    cachedContents.value = []
    cachedTotal.value = 0
    cachedTotalPages.value = 1
    cachedPositions.value = new Map()
    cachedContainerHeight.value = 0
  }

  // 恢复滚动位置（需在 DOM 渲染后调用）
  function restoreScroll() {
    if (scrollPosition.value > 0) {
      console.log('[Scroll] restoreScroll 被调用, 目标位置:', scrollPosition.value)
      window.scrollTo({ top: scrollPosition.value, behavior: 'instant' })
    }
  }

  return {
    searchKeyword,
    selectedTags,
    selectedTypes,
    page,
    recommendPage,
    scrollPosition,
    hasLoaded,
    cachedContents,
    cachedTotal,
    cachedTotalPages,
    cachedPositions,
    cachedContainerHeight,
    searchTrigger,
    triggerSearch,
    saveState,
    clearState,
    restoreScroll
  }
})

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Content } from '@/types'

interface CachedPosition {
  x: number
  y: number
  w: number
  h?: number
  col?: number
}

export type FeedMode = 'recommended' | 'latest' | 'popular'

export const useHomeStore = defineStore('home', () => {
  const searchKeyword = ref('')
  const selectedTags = ref<string[]>([])
  const feedMode = ref<FeedMode>('latest')
  const page = ref(1)
  
  const recommendPage = ref(1)
  
  const scrollPosition = ref(0)
  
  const hasLoaded = ref(false)

  const cachedContents = ref<Content[]>([])
  const cachedTotal = ref(0)
  const cachedTotalPages = ref(1)

  const catalogTotal = ref(0)

  const cachedPositions = ref<Map<string | number, CachedPosition>>(new Map())
  const cachedContainerHeight = ref(0)

  const searchTrigger = ref(0)
  function triggerSearch() {
    searchTrigger.value++
  }

  function saveState(params: {
    searchKeyword: string
    selectedTags: string[]
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

  function clearState() {
    searchKeyword.value = ''
    selectedTags.value = []
    feedMode.value = 'latest'
    page.value = 1
    recommendPage.value = 1
    scrollPosition.value = 0
    hasLoaded.value = false
    cachedContents.value = []
    cachedTotal.value = 0
    cachedTotalPages.value = 1
    catalogTotal.value = 0
    cachedPositions.value = new Map()
    cachedContainerHeight.value = 0
  }

  function restoreScroll() {
    if (scrollPosition.value > 0) {
      window.scrollTo({ top: scrollPosition.value, behavior: 'instant' })
    }
  }

  return {
    searchKeyword,
    selectedTags,
    feedMode,
    page,
    recommendPage,
    scrollPosition,
    hasLoaded,
    cachedContents,
    cachedTotal,
    cachedTotalPages,
    catalogTotal,
    cachedPositions,
    cachedContainerHeight,
    searchTrigger,
    triggerSearch,
    saveState,
    clearState,
    restoreScroll
  }
})

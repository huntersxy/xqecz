import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import type { Content, ListParams, RecommendContent } from '@/types'

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

export function useHomeLogic() {
  const router = useRouter()
  const homeStore = useHomeStore()

  const contents = ref<Content[]>([])
  const recommendContents = ref<RecommendContent[]>([])
  const allTags = ref<string[]>([])

  const selectedTags = ref<string[]>(homeStore.selectedTags)
  const searchKeyword = ref(homeStore.searchKeyword)
  const selectedTypes = ref<string[]>(homeStore.selectedTypes)
  const page = ref(homeStore.page)
  const recommendPage = ref(homeStore.recommendPage)

  const isLoading = ref(false)
  const isRecommendLoading = ref(false)
  const contentKey = ref(0)
  const recommendKey = ref(0)

  const pageSize = ref(12)
  const total = ref(0)
  const totalPages = ref(1)
  const recommendHint = ref('')
  const recommendPerPage = 16
  const recommendTotal = 100
  const maxRecommendPages = Math.floor(recommendTotal / recommendPerPage)
  const swapSections = ref(false)

  const sortedTags = computed(() => {
    return [...allTags.value].sort()
  })

  const contentTypes = ['video', 'image', 'text', 'link']
  const contentTypeLabels: Record<string, string> = {
    video: '视频',
    image: '图片',
    text: '图文',
    link: '链接',
  }

  function normalizeContent(content: Content | Record<string, unknown>): Content {
    const rawUser = (content as Record<string, unknown>).user as Record<string, unknown> | undefined

    return {
      id: Number(content.id) || 0,
      title: String(content.title || ''),
      type: (String(content.type) as 'video' | 'image' | 'text' | 'link') || 'text',
      text: String(content.text || ''),
      url: String(content.url || ''),
      thumb: String(content.thumb || ''),
      video: String(content.video || ''),
      img: String((content as Record<string, unknown>).img || ''),
      file_size: Number(content.file_size) || 0,
      user: rawUser
        ? {
            id: Number(rawUser.id) || 0,
            username: String(rawUser.username || ''),
            is_admin: Boolean(rawUser.is_admin),
            is_banned: Boolean(rawUser.is_banned),
            created_at: Number(rawUser.created_at) || 0,
            updated_at: Number(rawUser.updated_at) || 0,
          }
        : {
            id: 0,
            username: '',
            is_admin: false,
            is_banned: false,
            created_at: 0,
            updated_at: 0,
          },
      tags: Array.isArray(content.tags) ? (content.tags as string[]) : [],
      created_at: Number(content.created_at) || 0,
      updated_at: Number(content.updated_at) || 0,
      view_count: Number(content.view_count) || 0,
    }
  }

  async function loadContents() {
    if (isLoading.value) return
    
    isLoading.value = true
    
    try {
      let res
      if (searchKeyword.value) {
        res = await contentApi.search(searchKeyword.value, {
          page: page.value,
          page_size: pageSize.value,
        })
      } else {
        const params: ListParams = {
          page: page.value,
          page_size: pageSize.value,
          sort_by: 'created_at',
          order: 'desc',
        }
        if (selectedTags.value.length > 0) params.tag = selectedTags.value.join(',')
        if (selectedTypes.value.length > 0) params.type = selectedTypes.value.join(',')
        res = await contentApi.list(params)
      }
      if (res.code === 200) {
        contents.value = res.data.list.map((item: Record<string, unknown>) => normalizeContent(item))
        total.value = res.data.total
        totalPages.value = res.data.total_page
        swapSections.value =
          selectedTags.value.length > 0 ||
          selectedTypes.value.length > 0 ||
          !!searchKeyword.value.trim()
        contentKey.value++
      }
    } catch (error) {
      console.error('加载内容失败', error)
    } finally {
      isLoading.value = false
    }
  }

  async function loadTags() {
    try {
      const res = await contentApi.getTags()
      if (res.code === 200) {
        const tags = res.data as string[]
        const shuffled = [...tags].sort(() => Math.random() - 0.5)
        allTags.value = shuffled.slice(0, 15)
      }
    } catch (error) {
      console.error('加载标签失败', error)
    }
  }

  function selectTag(tag: string) {
    const index = selectedTags.value.indexOf(tag)
    if (index > -1) {
      selectedTags.value = []
    } else {
      selectedTags.value = [tag]
    }
    page.value = 1
    loadContents().then(() => {
      document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  function selectType(type: string) {
    const index = selectedTypes.value.indexOf(type)
    if (index > -1) {
      selectedTypes.value = []
    } else {
      selectedTypes.value = [type]
    }
    page.value = 1
    loadContents().then(() => {
      document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  function handleSearch() {
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer)
    }
    searchDebounceTimer = setTimeout(() => {
      page.value = 1
      loadContents().then(() => {
        document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })
    }, 300)
  }

  function goToDetail(content: Content) {
    const linkUrl = content.type === 'link' && content.url
    if (linkUrl) {
      window.open(linkUrl, '_blank')
      return
    }
    if (content.id) {
      homeStore.saveState({
        searchKeyword: searchKeyword.value,
        selectedTags: selectedTags.value,
        selectedTypes: selectedTypes.value,
        page: page.value,
        recommendPage: recommendPage.value,
        scrollPosition: window.scrollY,
      })
      router.push(`/content/${content.id}`)
    }
  }

  function goToPage(p: number) {
    if (p >= 1 && p <= totalPages.value && !isLoading.value) {
      page.value = p
      loadContents().then(() => {
        document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })
    }
  }

  function goToEasterEgg() {
    router.push('/easter-egg')
  }

  function normalizeRecommendContent(content: RecommendContent): RecommendContent {
    return {
      id: Number(content.id) || 0,
      title: content.title || '',
      type: (content.type as 'video' | 'image' | 'text' | 'link') || 'image',
      url: content.url || '',
      thumb: content.thumb || '',
      tags: Array.isArray(content.tags) ? content.tags : [],
      view_count: content.view_count || 0,
      user: {
        id: Number(content.user?.id) || 0,
        username: content.user?.username || '',
      },
      created_at: content.created_at || 0,
    }
  }

  async function loadRecommendContents() {
    if (isRecommendLoading.value) return
    
    isRecommendLoading.value = true
    
    try {
      const res = await contentApi.recommend(recommendPerPage, recommendPage.value)
      if (res.code === 200) {
        recommendContents.value = res.data.list.map((item: RecommendContent) =>
          normalizeRecommendContent(item),
        )
        recommendHint.value = ''
        recommendKey.value++
      }
    } catch (error) {
      console.error('加载推荐内容失败', error)
    } finally {
      isRecommendLoading.value = false
    }
  }

  function refreshRecommend() {
    if (isRecommendLoading.value) return
    
    recommendPage.value++
    if (recommendPage.value > maxRecommendPages) {
      recommendPage.value = 1
    }
    loadRecommendContents().then(() => {
      document.getElementById('recommend-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  function onMountedHome() {
    const navigationEntries = performance.getEntriesByType(
      'navigation',
    ) as PerformanceNavigationTiming[]
    const isReload = navigationEntries.length > 0 && navigationEntries[0].type === 'reload'

    if (isReload) {
      homeStore.clearState()
    }

    if (homeStore.hasLoaded) {
      homeStore.restoreScroll()
    }
    loadContents()
    loadTags()
    loadRecommendContents()
  }

  function getImageUrl(image?: string): string {
    if (image) {
      let url = image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
      if (url.startsWith('/')) {
        url = `https://xqapi.xiey.work${url}`
      }
      return url
    }
    return ''
  }

  function onImageLoad(e: Event) {
    const img = e.target as HTMLImageElement
    if (img.naturalHeight > img.naturalWidth * 1.2) {
      img.style.objectPosition = '50% 8%'
    }
  }

  function formatTime(ts: number): string {
    if (!ts) return ''
    return new Date(ts * 1000).toLocaleDateString('zh-CN')
  }

  function getPreviewText(text: string, maxLength: number = 120): string {
    if (!text) return ''
    const plainText = text
      .replace(/#{1,6}\s/g, '')
      .replace(/\*\*([^*]+)\*\*/g, '$1')
      .replace(/\*([^*]+)\*/g, '$1')
      .replace(/`([^`]+)`/g, '$1')
      .replace(/```[\s\S]*?```/g, '[代码块]')
      .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
      .replace(/[-*+]\s/g, '')
      .replace(/\n/g, ' ')
      .trim()
    return plainText.length > maxLength ? plainText.substring(0, maxLength) + '...' : plainText
  }

  return {
    contents,
    recommendContents,
    allTags,
    selectedTags,
    searchKeyword,
    selectedTypes,
    page,
    recommendPage,
    isLoading,
    isRecommendLoading,
    contentKey,
    recommendKey,
    pageSize,
    total,
    totalPages,
    recommendHint,
    maxRecommendPages,
    swapSections,
    sortedTags,
    contentTypes,
    contentTypeLabels,
    loadContents,
    loadTags,
    selectTag,
    selectType,
    handleSearch,
    goToDetail,
    goToPage,
    goToEasterEgg,
    loadRecommendContents,
    refreshRecommend,
    onMountedHome,
    getImageUrl,
    onImageLoad,
    formatTime,
    getPreviewText,
  }
}

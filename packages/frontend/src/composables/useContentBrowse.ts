import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { contentApi } from '@/api'
import { useListCache } from '@/composables/useListCache'
import type { Content } from '@/types'

const THUMB_H = 64
const THUMB_GAP = 6
const PAGE_SIZE = 20

/**
 * 内容详情页的"沉浸式浏览"上下文：
 * 列表缓存 + 上一个/下一个 + 底部缩略图走马灯 + 键盘导航。
 * 与视图解耦，便于复用与测试。
 */
export function useContentBrowse() {
  const route = useRoute()
  const router = useRouter()
  const listCache = useListCache()

  const cachedList = ref<Content[]>([])
  const isFetchingMore = ref(false)
  const listPage = ref(1)
  const listTotalPages = ref(1)
  const previewCount = ref(5)
  const previewRef = ref<HTMLElement | null>(null)

  const currentId = computed(() => Number(route.params.id))
  const currentIndex = computed(() => cachedList.value.findIndex((c) => c.id === currentId.value))
  const hasPrev = computed(() => currentIndex.value > 0)
  const hasNext = computed(() => currentIndex.value >= 0)

  function calcPreviewCount() {
    const h = window.innerHeight
    previewCount.value = Math.max(3, Math.floor((h - 120) / (THUMB_H + THUMB_GAP)))
  }

  const previewItems = computed(() => {
    if (currentIndex.value < 0) return []
    const total = cachedList.value.length
    const count = previewCount.value
    const half = Math.floor(count / 2)
    let start = currentIndex.value - half
    let end = start + count
    if (start < 0) { start = 0; end = Math.min(count, total) }
    if (end > total) { end = total; start = Math.max(0, end - count) }
    return cachedList.value.slice(start, end).map((item) => ({
      ...item,
      isCurrent: item.id === currentId.value,
    }))
  })

  function scrollThumbIntoView() {
    nextTick(() => {
      const el = previewRef.value?.querySelector('.cd-thumb-current')
      el?.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' })
    })
  }

  async function fetchMore(): Promise<boolean> {
    if (isFetchingMore.value || listPage.value >= listTotalPages.value) return false
    isFetchingMore.value = true
    try {
      const nextPage = listPage.value + 1
      const res = await contentApi.list({ page: nextPage, page_size: PAGE_SIZE, sort_by: 'created_at', order: 'desc' })
      if (res.code === 200) {
        const newItems = res.data.list as Content[]
        const existingIds = new Set(cachedList.value.map((c) => c.id))
        const fresh = newItems.filter((item) => !existingIds.has(item.id))
        cachedList.value.push(...fresh)
        listPage.value = nextPage
        listTotalPages.value = res.data.total_page
        listCache.save(cachedList.value, res.data.total, listTotalPages.value)
        return fresh.length > 0
      }
    } catch (e) {
      console.warn('加载更多失败:', e)
    } finally {
      isFetchingMore.value = false
    }
    return false
  }

  async function navigateTo(id: number) {
    if (id === currentId.value) return
    await router.replace({ name: 'content-detail', params: { id } })
    scrollThumbIntoView()
  }

  async function goToPrev() {
    if (!hasPrev.value) return
    await navigateTo(cachedList.value[currentIndex.value - 1].id)
  }

  async function goToNext() {
    const idx = currentIndex.value
    if (idx < 0) return
    const next = cachedList.value[idx + 1]
    if (!next) return
    await navigateTo(next.id)
    // 接近末尾时预加载更多
    if (currentIndex.value >= cachedList.value.length - 3) await fetchMore()
  }

  function goBack() {
    if (window.history.length > 1) router.back()
    else router.push('/')
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') goBack()
    if (e.key === 'ArrowLeft') void goToPrev()
    if (e.key === 'ArrowRight') void goToNext()
  }

  onMounted(() => {
    const cached = listCache.load()
    cachedList.value = cached?.list || []
    listPage.value = Math.ceil(cachedList.value.length / PAGE_SIZE) || 1
    listTotalPages.value = cached?.totalPages || 1
    calcPreviewCount()
    document.addEventListener('keydown', handleKeydown)
    window.addEventListener('resize', calcPreviewCount)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleKeydown)
    window.removeEventListener('resize', calcPreviewCount)
  })

  return {
    cachedList,
    currentId,
    currentIndex,
    hasPrev,
    hasNext,
    previewItems,
    previewRef,
    calcPreviewCount,
    scrollThumbIntoView,
    fetchMore,
    navigateTo,
    goToPrev,
    goToNext,
    goBack,
  }
}

import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { contentApi } from '@/api'
import { useListCache } from '@/composables/useListCache'
import { scrollBehavior } from '@/utils'
import type { Content } from '@/types'

const PAGE_SIZE = 20
/** 缩略图条一次最多渲染多少张（以当前作品为中心取窗口）。 */
const PREVIEW_COUNT = 9

/** 焦点在可输入元素里时，方向键/Esc 属于输入本身，不能拿来翻页或返回。 */
function isTypingTarget(t: EventTarget | null): boolean {
  if (!(t instanceof HTMLElement)) return false
  return t.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName)
}

/**
 * 内容详情页的"沉浸式浏览"上下文：
 * 列表缓存 + 上一个/下一个 + 底部缩略图条 + 键盘导航。
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
  /** 缩略图条根节点：由视图绑定，用于把当前项滚到中间。 */
  const previewRef = ref<HTMLElement | null>(null)

  const currentId = computed(() => Number(route.params.id))
  const currentIndex = computed(() => cachedList.value.findIndex((c) => c.id === currentId.value))
  const hasMorePages = computed(() => listPage.value < listTotalPages.value)
  const hasPrev = computed(() => currentIndex.value > 0)
  // 已加载列表里还有下一个，或者后面还有没拉的页
  const hasNext = computed(() => {
    const idx = currentIndex.value
    return idx >= 0 && (idx < cachedList.value.length - 1 || hasMorePages.value)
  })

  const previewItems = computed(() => {
    if (currentIndex.value < 0) return []
    const total = cachedList.value.length
    const half = Math.floor(PREVIEW_COUNT / 2)
    let start = currentIndex.value - half
    let end = start + PREVIEW_COUNT
    if (start < 0) { start = 0; end = Math.min(PREVIEW_COUNT, total) }
    if (end > total) { end = total; start = Math.max(0, end - PREVIEW_COUNT) }
    return cachedList.value.slice(start, end).map((item) => ({
      ...item,
      isCurrent: item.id === currentId.value,
    }))
  })

  function scrollThumbIntoView() {
    nextTick(() => {
      const el = previewRef.value?.querySelector('.cd-thumb-current')
      el?.scrollIntoView({ behavior: scrollBehavior(), inline: 'center', block: 'nearest' })
    })
  }

  async function fetchMore(): Promise<boolean> {
    if (isFetchingMore.value || !hasMorePages.value) return false
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
    // 已经在已加载列表的末尾：先拉下一页，拉不到就停
    if (idx >= cachedList.value.length - 1 && !(await fetchMore())) return
    const next = cachedList.value[currentIndex.value + 1]
    if (!next) return
    await navigateTo(next.id)
    // 接近末尾时预加载更多
    if (currentIndex.value >= cachedList.value.length - 3) void fetchMore()
  }

  function goBack() {
    if (window.history.length > 1) router.back()
    else router.push('/')
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return
    if (isTypingTarget(e.target)) return
    // 看图器（viewerjs）打开时，Esc/方向键归它所有
    if (document.body.classList.contains('viewer-open')) return
    if (e.key === 'Escape') goBack()
    else if (e.key === 'ArrowLeft') void goToPrev()
    else if (e.key === 'ArrowRight') void goToNext()
  }

  onMounted(() => {
    const cached = listCache.load()
    cachedList.value = cached?.list || []
    listPage.value = Math.ceil(cachedList.value.length / PAGE_SIZE) || 1
    listTotalPages.value = cached?.totalPages || 1
    document.addEventListener('keydown', handleKeydown)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return {
    cachedList,
    currentId,
    currentIndex,
    hasPrev,
    hasNext,
    previewItems,
    previewRef,
    scrollThumbIntoView,
    fetchMore,
    navigateTo,
    goToPrev,
    goToNext,
    goBack,
  }
}

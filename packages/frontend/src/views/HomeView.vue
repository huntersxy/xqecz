<script lang="ts">
export default { name: 'HomeView' }
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, onDeactivated, onBeforeUnmount, nextTick, watch } from 'vue'
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore, type FeedMode } from '@/stores/home'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import { useListCache, diffLists } from '@/composables/useListCache'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import SiteImage from '@/components/SiteImage.vue'
import SiteIcon from '@/components/SiteIcon.vue'
import QuickUploadSheet from '@/components/QuickUploadSheet.vue'
import type { Content, ListParams, RecommendContent } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const listCache = useListCache()
const showUploadSheet = ref(false)
// 列表模式放 store：KeepAlive 往返时 tab 不能和列表内容脱节。
const feedMode = computed<FeedMode>({
  get: () => homeStore.feedMode,
  set: (mode) => { homeStore.feedMode = mode },
})

const feedSort = computed(() => (feedMode.value === 'popular' ? 'view_count' : 'created_at'))
const featuredWorks = computed(() => {
  const pool = recommendLoader.recommendContents.value.filter((item) => item.thumb)
  if (!pool.length) return []
  const start = recommendLoader.displayStart.value % pool.length
  return pool.slice(start, start + 8)
})

function selectFeedMode(mode: FeedMode) {
  if (feedMode.value === mode) return
  feedMode.value = mode
  // 推荐流走 /content/recommend，不接受标签；搜索也只在普通列表里成立。
  // 切换模式时清掉这些筛选，避免 tab 显示推荐、列表却还是搜索/标签结果。
  if (mode === 'recommended') searchFilter.selectedTags.value = []
  if (homeStore.searchKeyword) homeStore.searchKeyword = ''
  resetAndLoad(false)
}

function goToUpload() {
  showUploadSheet.value = true
}

const swapSections = computed(
  () =>
    searchFilter.selectedTags.value.length > 0 ||
    !!homeStore.searchKeyword.trim(),
)

const allContents = ref<Content[]>([])
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const totalPages = ref(1)
const isLoading = ref(false)
const isLoadingMore = ref(false)
// 到底条件：已加载数量达到总数，且页码未越过 total_page（双重兜底，避免边界多拉空页/死循环）
const hasMore = computed(
  () => currentPage.value <= totalPages.value && allContents.value.length < total.value,
)
const sentinelRef = ref<HTMLElement | null>(null)
const masonryRef = ref<HTMLElement | null>(null)

const waterfall = useWaterfallLayout(masonryRef, allContents)

// 是否需要在布局完成后恢复滚动位置
const pendingScrollRestore = ref(false)

// 请求序号：每次发起新请求 +1，旧响应返回时若序号已过期则直接丢弃，
// 避免快速切换搜索/标签时旧列表覆盖新列表（竞态）。
let loadSeq = 0

// keep-alive 首次挂载时 onActivated 也会触发一次；此时数据已由 onMounted
// （loadAllPages 或 homeStore/localStorage 缓存恢复）提供，不必再 sync。
// 首次激活后复位，后续激活（详情页返回等）正常做增量同步。
let isFirstActivation = true
let isHomeActive = false
let restoreTimer: ReturnType<typeof setTimeout> | undefined
function cancelScrollRestore() {
  isHomeActive = false
  clearTimeout(restoreTimer)
  restoreTimer = undefined
  pendingScrollRestore.value = false
}
onDeactivated(cancelScrollRestore)
onBeforeUnmount(cancelScrollRestore)

// 列表长度变化：清空时重置布局；首次加载/列表变短（diff 删除）时全量重排重新平衡。
// 布局由各写路径显式触发（relayout / appendNewItems），避免同一帧重复量高。
watch(
  () => allContents.value.length,
  (newLen, oldLen) => {
    if (newLen === 0) {
      waterfall.reset()
      return
    }
    if (oldLen === 0 || newLen < oldLen) {
      nextTick(() => waterfall.relayout({ full: true }))
    }
    // 全量加载模式下无需"高度不够自动补拉"；diff 追加时布局由写路径显式触发
  },
)

function onCardResized(id: string | number) {
  waterfall.onCardResized(id)
  // 如果有待恢复的滚动位置，在卡片尺寸稳定后尝试恢复
  if (pendingScrollRestore.value) {
    requestAnimationFrame(() => {
      if (!isHomeActive || !pendingScrollRestore.value) return
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

// 卡片内点赞成功后同步回列表：留在此页不跳转，但 allContents 是缓存与
// onBeforeRouteLeave 写入 localStorage 的唯一来源，不回写就会在返回时被旧值复原。
function onCardLiked(payload: { id: string | number; liked: boolean; like_count: number }) {
  const target = allContents.value.find((item) => item.id === payload.id)
  if (target) target.like_count = payload.like_count
}

// 离开首页（详情/后台/登录等任意跳转）时统一保存滚动位置与列表状态，
// 返回时由 onActivated 恢复（keep-alive）。
onBeforeRouteLeave(() => {
  const pos = globalThis.scrollY
  homeStore.saveState({
    searchKeyword: homeStore.searchKeyword,
    selectedTags: searchFilter.selectedTags.value,
    page: currentPage.value,
    recommendPage: recommendLoader.loadedPage.value,
    scrollPosition: pos,
    contents: allContents.value,
    total: total.value,
    totalPages: totalPages.value,
    positions: new Map(waterfall.positions.value),
    containerHeight: waterfall.containerHeight.value,
  })
  // localStorage 只缓存"最新"顺序：推荐/热门列表写入同一份缓存后，
  // 下次以最新模式打开会把别人的顺序当成最新列表恢复。
  if (!homeStore.searchKeyword && feedMode.value === 'latest') {
    listCache.save(allContents.value, total.value, totalPages.value)
  }
})

interface FeedPageData {
  list: unknown[]
  total: number
  totalPages: number
}

// 三种列表模式的统一取数入口：搜索、推荐、普通列表只在请求与字段名上不同，
// 分页、去重、布局仍走下面的 fetchPage，避免三套分页各自漂移。
async function requestFeedPage(page: number): Promise<FeedPageData | null> {
  if (homeStore.searchKeyword) {
    const res = await contentApi.search(homeStore.searchKeyword, {
      page, page_size: pageSize.value,
      tag: searchFilter.selectedTags.value.length > 0 ? searchFilter.selectedTags.value.join(',') : undefined,
    })
    if (res.code !== 200) return null
    return { list: res.data.list, total: res.data.total, totalPages: res.data.total_page }
  }
  if (feedMode.value === 'recommended') {
    const res = await contentApi.recommend(pageSize.value, page)
    if (res.code !== 200) return null
    // 推荐接口用 count 而不是 total/total_page，按同一 pageSize 折算分页，
    // 保证 hasMore 与滚动加载和普通列表一致。
    const totalPages = Math.max(1, Math.ceil(res.data.count / pageSize.value))
    return { list: res.data.list, total: res.data.count, totalPages }
  }
  const params: ListParams = { page, page_size: pageSize.value, sort_by: feedSort.value, order: 'desc' }
  if (searchFilter.selectedTags.value.length > 0) params.tag = searchFilter.selectedTags.value.join(',')
  const res = await contentApi.list(params)
  if (res.code !== 200) return null
  return { list: res.data.list, total: res.data.total, totalPages: res.data.total_page }
}

async function fetchPage(page: number, append = false, force = false) {
  // 非强制请求沿用"一次只发一个"防抖；强制请求（筛选/搜索重置）可打断在途请求
  if (!force && (isLoading.value || isLoadingMore.value)) return
  const seq = ++loadSeq
  if (append) isLoadingMore.value = true
  else isLoading.value = true

  try {
    const pageData = await requestFeedPage(page)
    // 过期响应：已被更新的请求取代，丢弃，避免旧列表覆盖新筛选结果
    if (seq !== loadSeq || !pageData) return
    const parsed = pageData.list.map((item: unknown) => ContentSchema.parse(item))
    total.value = pageData.total
    totalPages.value = pageData.totalPages
    // 标题区展示的是站点总量，不能被标签/搜索的当前结果数覆盖。
    // 推荐接口只统计已通过内容，普通列表还包含审核中；已有列表口径时
    // 不要用推荐流的 count 覆盖它，避免切换 tab 时总数跳动。
    if (!homeStore.searchKeyword && searchFilter.selectedTags.value.length === 0) {
      if (feedMode.value !== 'recommended' || homeStore.catalogTotal === 0) {
        homeStore.catalogTotal = pageData.total
      }
    }
    if (append) {
      // 合并去重：分页数据与现有列表可能重叠（diff / 并发刷新场景）
      const existing = new Set(allContents.value.map((item) => item.id))
      const fresh = parsed.filter((item) => !existing.has(item.id))
      const oldLen = allContents.value.length
      allContents.value.push(...fresh)
      // 仅当实际追加了新卡片才推进页码；若本页全部重叠（fresh 空），
      // 保持 currentPage 不变，避免 hasMore 的 currentPage<=totalPages 提前耗尽。
      if (fresh.length > 0) {
        currentPage.value = page
        nextTick(() => waterfall.appendNewItems(allContents.value.slice(oldLen)))
      }
    } else {
      currentPage.value = page
      allContents.value = parsed
      // 列表整体替换（同长度/变多时 length watch 不会触发）也要重排，
      // 否则新卡片没有位置、旧位置残留，产生空白/重叠。full 重排重新平衡各列。
      nextTick(() => waterfall.relayout({ full: true }))
    }
    // 缓存统一在 onBeforeRouteLeave 离开时写入，运行时不再频繁写 localStorage
  } catch (e) { console.error('加载失败:', e) }
  finally {
    // 仅最新请求复位加载态；过期请求的复位交给更新的一批
    if (seq === loadSeq) { isLoading.value = false; isLoadingMore.value = false }
  }
}

function onTagSelect(tag: string) {
  // 推荐接口没有标签参数；在推荐流里点标签等于改用普通列表。
  if (feedMode.value === 'recommended') feedMode.value = 'latest'
  searchFilter.selectTag(tag, () => resetAndLoad(false))
}

function clearFilters() {
  searchFilter.selectedTags.value = []
  homeStore.searchKeyword = ''
  resetAndLoad()
}

function resetAndLoad(clearTags = true) {
  if (clearTags) searchFilter.selectedTags.value = []
  // 作废所有在途请求，防止旧响应把新筛选结果覆盖掉
  loadSeq++
  currentPage.value = 1
  allContents.value = []
  isLoading.value = false
  isLoadingMore.value = false
  listCache.clear()
  void fetchPage(1, false, true)
}

watchGlobalSearch(() => {
  // 搜索是普通列表的一种；从推荐流发起搜索时先把 tab 切回最新，
  // 否则会在"推荐"高亮下展示搜索结果。顶栏是全局搜索，新搜索不继承
  // 上一次的标签筛选，否则会出现"搜到了但 0 条"的错觉；搜索结果上
  // 仍可继续点标签缩小范围。
  feedMode.value = 'latest'
  resetAndLoad(true)
})

// 首次加载：自动连续拉取所有页（1-100 → 101-200 → …），不依赖滚动触发。
// 图片仍由卡片自身的 loading="lazy" 懒加载，数据量小（每页 100 条）可一次拉满。
async function loadAllPages() {
  // 推荐流分页字段与列表不同，由 fetchPage 的推荐分支负责；这里只处理列表排序。
  if (isLoading.value || feedMode.value === 'recommended') return
  isLoading.value = true
  const seq = ++loadSeq
  try {
    // 基于现有列表去重追加（keep-alive/localStorage 恢复后补齐剩余页）
    const loaded = new Set<number | string>(allContents.value.map((i) => i.id))
    const all = [...allContents.value]
    const batchSize = 100 // API 上限，减少请求次数
    for (let page = 1; ; page++) {
      const res = await contentApi.list({ page, page_size: batchSize, sort_by: feedSort.value, order: 'desc' })
      // 过期响应（被搜索/筛选重置打断）直接丢弃
      if (seq !== loadSeq) return
      if (res.code !== 200) break
      const parsed = res.data.list.map((item: unknown) => ContentSchema.parse(item))
      total.value = res.data.total
      homeStore.catalogTotal = res.data.total
      // totalPages 统一按滚动分页的 page_size（pageSize.value=20）计算，
      // 与 hasMore / fetchMore 的页码基准一致（本函数每页要 100 条，但不能改基准）。
      totalPages.value = Math.ceil(res.data.total / pageSize.value)
      const fresh = parsed.filter((item) => !loaded.has(item.id))
      for (const item of fresh) { loaded.add(item.id); all.push(item) }
      // 拉满即停：本页为空或已拿到全部
      if (parsed.length === 0 || all.length >= res.data.total) break
    }
    allContents.value = all
    currentPage.value = Math.max(1, Math.ceil(allContents.value.length / pageSize.value))
    nextTick(() => waterfall.relayout({ full: true }))
  } catch (e) { console.error('加载失败:', e) }
  finally {
    if (seq === loadSeq) { isLoading.value = false; isLoadingMore.value = false }
  }
}

// 滚动到底部的增量加载：按已加载条数推导下一页，只追加不写缓存
//（缓存唯一写入点在 onBeforeRouteLeave）。
async function fetchMore() {
  if (!hasMore.value || isLoading.value || isLoadingMore.value) return
  // 页码由已加载条数推导，而非 currentPage+1：列表可能被 diff 替换，
  // 按条数推导保证请求的是「已加载范围之后」的页，不会重复请求已存在的页。
  const nextPage = Math.floor(allContents.value.length / pageSize.value) + 1
  await fetchPage(nextPage, true)
}


// 拉取最新全量列表（API page_size 上限 100，超过分页拉取），返回解析后的数组。
// 仅在增量同步检测到"头部有变化"时才需要全量对账（新增/删除可能发生在任意位置）。
async function fetchFreshList(): Promise<Content[] | null> {
  const out: Content[] = []
  for (let p = 1; ; p++) {
    const res = await contentApi.list({ page: p, page_size: 100, sort_by: feedSort.value, order: 'desc' })
    if (res.code !== 200) return null
    out.push(...res.data.list.map((item: unknown) => ContentSchema.parse(item)))
    if (out.length >= res.data.total) break
  }
  return out
}

/** 记录锚点：当前视口顶部第一张卡片（插入新卡片后保持同一内容的阅读位置） */
function captureViewAnchor(): { id: string | number; y: number; scrollY: number } | null {
  const scrollY = globalThis.scrollY
  let anchorId: string | number | null = null
  let anchorTop = Number.POSITIVE_INFINITY
  for (const [id, pos] of waterfall.positions.value) {
    if (pos.y >= scrollY - 4 && pos.y < anchorTop) {
      anchorId = id
      anchorTop = pos.y
    }
  }
  return anchorId != null ? { id: anchorId, y: anchorTop, scrollY } : null
}

/**
 * 返回首页（上传 / 详情页等跳转后）时增量同步最新列表：
 * 1. 先拉最新一页（100 条）与本地头部对比 —— 头部 id 一致且 total 未变 → 无结构
 *    变化，仅原位同步字段（点赞数等），不重排、不打断滚动（1 次请求，O(100)）。
 * 2. 头部有变化（新增/删除/排序）→ 才全量拉取做 diff，插入/移除并锚定当前可见内容。
 * 3. 首次激活（keep-alive 首次挂载）由 onActivated 的 isFirstActivation 跳过。
 */
async function syncLatestOnActivated() {
  if (feedMode.value === 'recommended') return
  if (swapSections.value || allContents.value.length === 0) return
  try {
    // 第一步：增量探测 —— 只拉最新一页，对比本地头部
    const probe = await contentApi.list({ page: 1, page_size: 100, sort_by: feedSort.value, order: 'desc' })
    if (probe.code !== 200) return
    const head = probe.data.list.map((item: unknown) => ContentSchema.parse(item))
    const newTotal = probe.data.total

    // 本地头部同样取前 head.length 条，逐 id 对比（顺序即 created_at 倒序，可比）
    const localHead = allContents.value.slice(0, head.length)
    const headSame =
      localHead.length === head.length &&
      localHead.every((item, i) => item.id === head[i].id)
    const totalSame = newTotal === total.value

    if (headSame && totalSame) {
      // 无结构变化：仅原位同步字段（点赞数等），不重排、不打断滚动
      const headMap = new Map(head.map((i) => [i.id, i]))
      for (const item of allContents.value) {
        const fresh = headMap.get(item.id)
        if (fresh && fresh.like_count !== item.like_count) {
          item.like_count = fresh.like_count
        }
      }
      return
    }

    // 第二步：头部有变化 → 全量对账（新增/删除可能发生在任意位置）
    const freshList = await fetchFreshList()
    if (!freshList) return
    const { merged, removed } = diffLists(allContents.value, freshList)

    const anchor = captureViewAnchor()
    allContents.value = merged.filter((item) => !removed.has(item.id))
    total.value = freshList.length // fetchFreshList 已拉全量，total 即其长度
    totalPages.value = Math.ceil(total.value / pageSize.value)
    currentPage.value = Math.max(1, Math.ceil(allContents.value.length / pageSize.value))

    nextTick(() => {
      // 这里自行锚定滚动，禁用布局层二次锚定，避免双重补偿；
      // full 重排重新平衡各列（diff 增删后避免列底空缺）。
      waterfall.relayout({ anchor: false, full: true })
      if (anchor) {
        const newPos = waterfall.positions.value.get(anchor.id)
        if (newPos) {
          const delta = newPos.y - anchor.y
          globalThis.scrollTo({ top: Math.max(0, anchor.scrollY + delta) })
        }
      }
    })
  } catch (e) {
    console.warn('返回首页同步最新列表失败:', e)
  }
}

// 首页悬浮按钮上传成功后：新作品只保证出现在"最新"顺序里。
// 推荐流不接收 pending 新作、热门按浏览量排序、筛选态也可能把它过滤掉，
// 所以先回到未筛选的最新列表，而不是在原模式里做一次看不见的 sync。
function onUploaded() {
  if (
    feedMode.value !== 'latest' ||
    homeStore.searchKeyword ||
    searchFilter.selectedTags.value.length > 0
  ) {
    feedMode.value = 'latest'
    homeStore.searchKeyword = ''
    searchFilter.selectedTags.value = []
    resetAndLoad(false)
    return
  }
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

  // 缓存恢复与搜索入口也需准备标签；展示顺序在本次浏览中保持稳定。
  void searchFilter.loadTags()
  // 精选条不依赖列表缓存/搜索状态，任何分支都应加载；
  // 否则从其他路由重新挂载时列表恢复了、推荐条却只剩空壳。
  void recommendLoader.loadRecommendContents()

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
    // 恢复瀑布流布局缓存
    if (homeStore.cachedPositions.size > 0) {
      waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
    }
    // 缓存的位置是基于"图片已加载"的高度算的；用当前实际高度重排一次，避免占位高度造成空白。
    nextTick(() => waterfall.relayout())
    nextTick(() => requestAnimationFrame(() => homeStore.restoreScroll()))
    return
  }

  // 尝试从 localStorage 加载缓存（先渲染缓存再异步补齐最新数据，不阻塞首屏）。
  // 缓存只有"最新"顺序，热门/推荐模式不能拿它冒充自己的列表。
  const cached = feedMode.value === 'latest' ? listCache.load() : null
  if (cached && cached.list.length > 0) {
    allContents.value = cached.list
    total.value = cached.total
    totalPages.value = cached.totalPages
    homeStore.catalogTotal = cached.total
    currentPage.value = 1
    // 全量加载：基于缓存补齐剩余页（图片仍懒加载），不依赖滚动触发。
    // 首次 onActivated 由 isFirstActivation 跳过重复 sync。
    void loadAllPages()
  } else if (feedMode.value === 'recommended') {
    // 推荐流第一页由统一入口取，后续页交给底部 sentinel。
    void fetchPage(1, false, true)
  } else {
    // 无缓存：直接自动拉取全部页（1-100 → 101-200 → …）
    void loadAllPages()
  }
})

// keep-alive 激活时恢复滚动位置
onActivated(() => {
  isHomeActive = true
  // 恢复瀑布流布局缓存
  if (homeStore.cachedPositions.size > 0) {
    waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
  }
  // 同上：回到首页后先按当前实际高度重排，再恢复滚动。
  nextTick(() => {
    if (!isHomeActive) return
    waterfall.relayout()
    // relayout 会更新容器高度，等 DOM 落稳后恢复，避免被旧高度截断。
    void nextTick(() => {
      if (!isHomeActive) return
      homeStore.restoreScroll()
      pendingScrollRestore.value = Math.abs(globalThis.scrollY - homeStore.scrollPosition) > 1
    })
  })

  // 标记需要在图片加载后恢复滚动
  pendingScrollRestore.value = true

  // 首帧恢复缓存位置；仅把增量同步留到入场之后，避免先显示顶部再跳到原位置。
  // 首次挂载的激活不 sync（数据已由 onMounted 提供），复位标志后后续激活正常增量同步。
  const firstActivation = isFirstActivation
  isFirstActivation = false
  restoreTimer = setTimeout(() => {
    restoreTimer = undefined
    if (pendingScrollRestore.value) homeStore.restoreScroll()
    pendingScrollRestore.value = false
    if (firstActivation) return
    void syncLatestOnActivated()
  }, 300)
})
</script>

<template>
  <div class="gallery-home-content">
      <div class="gallery-content">
        <!-- 页头视觉块已移除（顶栏已标明当前页）；保留隐藏 h1 维持文档标题层级。 -->
        <h1 class="gallery-visually-hidden">作品广场</h1>

        <section class="gallery-featured" aria-labelledby="featured-title">
          <div class="gallery-section-heading">
            <div>
              <span class="gallery-section-label">CURATED WORKS</span>
              <h2 id="featured-title">正在被看见</h2>
            </div>
            <button class="gallery-icon-action" type="button" :disabled="recommendLoader.isRecommendLoading.value" aria-label="换一批推荐作品" title="换一批" @click="recommendLoader.refreshRecommend(8)">
              <SiteIcon name="refresh" :spin="recommendLoader.isRecommendLoading.value" />
            </button>
          </div>
          <TransitionGroup v-if="featuredWorks.length" tag="div" name="gallery-featured" class="gallery-featured-strip">
            <button v-for="item in featuredWorks" :key="item.id" class="gallery-featured-item" type="button" :data-preload-route="`/content/${item.id}`" @click="openContent(item)">
              <SiteImage class="gallery-featured-thumb" :src="item.thumb" :alt="item.title" />
              <span>{{ item.title }}</span>
            </button>
          </TransitionGroup>
          <div v-else-if="recommendLoader.isRecommendLoading.value" class="gallery-featured-strip is-skeleton" aria-hidden="true">
            <div v-for="n in 6" :key="n" class="gallery-featured-item"><span class="gallery-skeleton-thumb"></span><span class="gallery-skeleton-line"></span></div>
          </div>
          <div v-else class="gallery-featured-empty">推荐作品正在整理中</div>
        </section>

        <section class="gallery-feed-toolbar" aria-label="作品筛选">
          <div class="gallery-filter-tabs" role="tablist" aria-label="作品排序">
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'recommended' }" type="button" role="tab" :aria-selected="feedMode === 'recommended'" @click="selectFeedMode('recommended')">推荐</button>
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'latest' }" type="button" role="tab" :aria-selected="feedMode === 'latest'" @click="selectFeedMode('latest')">最新</button>
            <button class="gallery-filter-tab" :class="{ 'is-active': feedMode === 'popular' }" type="button" role="tab" :aria-selected="feedMode === 'popular'" @click="selectFeedMode('popular')">热门</button>
          </div>
          <div class="gallery-tag-row" role="group" aria-label="标签筛选">
            <button class="gallery-tag" :class="{ 'is-active': searchFilter.selectedTags.value.length === 0 }" type="button" @click="resetAndLoad()">全部</button>
            <button v-for="tag in searchFilter.shuffledTags.value.slice(0, 7)" :key="tag" class="gallery-tag" :class="{ 'is-active': searchFilter.selectedTags.value.includes(tag) }" type="button" :title="tag" @click="onTagSelect(tag)">{{ tag }}</button>
            <button v-if="searchFilter.shuffledTags.value.length > 1" class="gallery-tag gallery-tag-refresh" type="button" aria-label="换一组标签" title="换一组标签" @click="searchFilter.reshuffleTags()"><SiteIcon name="refresh" /></button>
          </div>
        </section>

        <section class="gallery-feed" aria-labelledby="gallery-feed-title">
          <div class="gallery-feed-heading">
            <div>
              <h2 id="gallery-feed-title">{{ homeStore.searchKeyword ? '搜索结果' : searchFilter.selectedTags.value[0] || (feedMode === 'recommended' ? '推荐作品' : feedMode === 'popular' ? '热门作品' : '最新作品') }}</h2>
              <p v-if="homeStore.searchKeyword || searchFilter.selectedTags.value.length">{{ homeStore.searchKeyword ? `正在查找“${homeStore.searchKeyword}”` : '按标签筛选' }} · {{ total }} 件作品</p>
              <p v-else-if="feedMode === 'recommended'">{{ total > 0 ? `为你挑选 ${total} 件作品` : '推荐作品正在整理' }}</p>
              <p v-else>{{ total > 0 ? `${total} 件作品，持续更新中` : '作品正在加载' }}</p>
            </div>
            <button v-if="swapSections" class="gallery-clear-filter" type="button" @click="clearFilters">清除筛选</button>
          </div>

          <div v-if="isLoading && allContents.length === 0" class="gallery-skeleton" role="status" aria-label="正在加载作品">
            <div v-for="n in 8" :key="n" class="gallery-skeleton-card" :style="{ '--sk-h': 140 + ((n * 37) % 90) + 'px' }">
              <span class="gallery-skeleton-thumb"></span>
              <span class="gallery-skeleton-line"></span>
              <span class="gallery-skeleton-line is-short"></span>
            </div>
          </div>
          <div v-else-if="!isLoading && allContents.length === 0" class="gallery-state">
            <div class="gallery-state-mark"><SiteIcon name="image" /></div>
            <p>{{ homeStore.searchKeyword || searchFilter.selectedTags.value.length ? '没有找到符合条件的作品' : '还没有作品出现在这里' }}</p>
            <button type="button" @click="goToUpload">发布第一件作品 <SiteIcon name="arrow-right" /></button>
          </div>
          <div v-else ref="masonryRef" class="gallery-masonry" :style="{ position: 'relative', height: waterfall.containerHeight.value + 'px' }">
            <WaterfallCard
              v-for="item in allContents"
              :key="item.id"
              :item="item"
              :data-wf-id="item.id"
              :style="{ position: 'absolute', left: (waterfall.positions.value.get(item.id)?.x ?? 0) + 'px', top: (waterfall.positions.value.get(item.id)?.y ?? 0) + 'px', width: (waterfall.positions.value.get(item.id)?.w ?? 0) + 'px' }"
              @click="openContent"
              @image-loaded="onCardResized"
              @liked="onCardLiked"
            />
          </div>
          <div v-if="isLoadingMore" class="gallery-loadmore"><div class="gallery-state-spinner"></div><span>加载更多作品</span></div>
          <div ref="sentinelRef" class="gallery-sentinel"></div>
          <div v-if="!hasMore && allContents.length > 0" class="gallery-end">已经看到这里了</div>
        </section>
      </div>
    <button class="gallery-mobile-publish" type="button" aria-label="发布作品" @click="goToUpload"><SiteIcon name="plus" /></button>
    <QuickUploadSheet :open="showUploadSheet" @close="showUploadSheet = false" @uploaded="onUploaded" />
  </div>
</template>

<style scoped>
.gallery-home-content {
  --gallery-title-font: var(--site-title-font);
  --gallery-canvas: var(--site-canvas);
  --gallery-paper: var(--site-paper);
  --gallery-ink: var(--site-ink);
  --gallery-muted: var(--site-muted);
  --gallery-line: var(--site-line);
  --gallery-soft: var(--site-soft);
  --gallery-accent: var(--site-accent);
  --gallery-accent-soft: var(--site-accent-soft);
  min-width: 0;
  color: var(--gallery-ink);
  font-family: var(--font-sans);
}
/* 作品广场用满侧栏之外的内容区：收窄左右留白，让瀑布流在大屏上铺开。
   列数与卡片最小宽度由 useWaterfallLayout.computeColumnCount 统一控制（最多 6 列）。
   选择器带 .app-layout.home-gallery-mode 才有足够特异性覆盖 PublicShell 的 scoped 规则；
   手机（≤820）沿用侧栏抽屉的既有留白。 */
@media (min-width: 821px) {
  :global(.app-layout.home-gallery-mode .public-shell-content) {
    max-width: min(2240px, 100%);
    padding-left: clamp(20px, 2.2vw, 40px);
    padding-right: clamp(20px, 2.2vw, 40px);
  }
}
.gallery-mobile-publish { display: none; }
.gallery-content {
  width: 100%;
  margin: 0;
  padding: 0 0 76px;
}
/* 页头视觉块已移除；隐藏 h1 只保留文档标题层级，不占布局。 */
.gallery-visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}
.gallery-featured {
  padding: 24px 0 20px;
  border-bottom: 1px solid var(--gallery-line);
  scroll-margin-top: 16px;
}
.gallery-section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 13px;
}
.gallery-section-label {
  color: var(--gallery-muted);
  font-size: 9px;
  letter-spacing: .16em;
}
.gallery-section-heading h2,
.gallery-feed-heading h2 {
  margin: 4px 0 0;
  color: var(--gallery-ink);
  font: 600 21px/1.2 var(--gallery-title-font);
}
.gallery-icon-action {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid var(--gallery-line);
  border-radius: 50%;
  background: var(--gallery-paper);
  color: var(--gallery-muted);
  cursor: pointer;
  transition: color .18s ease, border-color .18s ease, transform .18s ease;
}
.gallery-icon-action:hover:not(:disabled) {
  border-color: var(--gallery-accent);
  color: var(--gallery-accent);
  transform: rotate(25deg);
}
.gallery-icon-action:disabled { cursor: wait; opacity: .5; }
.gallery-icon-action :deep(svg) { width: 15px; height: 15px; }
.gallery-featured-strip {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(108px, 1fr);
  gap: 12px;
  overflow-x: auto;
  padding: 2px 2px 3px;
  scrollbar-width: thin;
  scrollbar-color: var(--gallery-line) transparent;
}
.gallery-featured-item {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--gallery-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gallery-featured-item :deep(.site-image) {
  aspect-ratio: 1.15;
  min-height: 0;
  border-radius: 10px;
  box-shadow: 0 1px 0 var(--gallery-line);
  transition: box-shadow .28s ease, transform .28s cubic-bezier(.22, 1, .36, 1);
}
.gallery-featured-item :deep(.site-image-img) {
  height: 100%;
  object-fit: cover;
  transition: opacity .32s ease, transform .4s cubic-bezier(.22, 1, .36, 1);
}
.gallery-featured-item:hover :deep(.site-image) {
  transform: translateY(-3px);
  box-shadow: 0 12px 24px color-mix(in srgb, var(--gallery-ink) 14%, transparent);
}
.gallery-featured-item:hover :deep(.site-image-img) { transform: scale(1.06); }
.gallery-featured-item > span {
  overflow: hidden;
  color: var(--gallery-ink);
  font-size: 11px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color .18s ease;
}
.gallery-featured-item:hover > span { color: var(--gallery-accent); }
.gallery-featured-empty {
  min-height: 76px;
  padding: 26px 0;
  color: var(--gallery-muted);
  font-size: 11px;
}
.gallery-feed-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 22px 0 17px;
}
.gallery-filter-tabs {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex: 0 0 auto;
  padding: 3px;
  border: 1px solid var(--gallery-line);
  border-radius: 999px;
  background: color-mix(in srgb, var(--gallery-paper) 70%, transparent);
}
.gallery-filter-tab {
  position: relative;
  padding: 6px 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
  transition: color .18s ease, background-color .22s ease, box-shadow .22s ease, transform .18s ease;
}
.gallery-filter-tab:hover { color: var(--gallery-ink); }
.gallery-filter-tab:active { transform: scale(.96); }
.gallery-filter-tab.is-active {
  background: var(--gallery-paper);
  color: var(--gallery-ink);
  font-weight: 600;
  box-shadow: 0 1px 2px color-mix(in srgb, var(--gallery-ink) 14%, transparent), 0 0 0 1px var(--gallery-line);
}
.gallery-tag-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.gallery-tag-row::-webkit-scrollbar { display: none; }
.gallery-tag {
  flex: 0 0 auto;
  min-height: 28px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: background-color .18s ease, color .18s ease, border-color .18s ease, transform .15s ease;
}
.gallery-tag:hover,
.gallery-tag.is-active {
  border-color: var(--gallery-line);
  background: var(--gallery-paper);
  color: var(--gallery-accent);
}
.gallery-tag.is-active { border-color: color-mix(in srgb, var(--gallery-accent) 38%, var(--gallery-line)); font-weight: 600; }
.gallery-tag:active { transform: scale(.95); }
.gallery-tag-refresh { padding: 0 7px; }
.gallery-tag-refresh :deep(svg) { width: 13px; height: 13px; }
.gallery-feed {
  scroll-margin-top: 18px;
}
.gallery-feed-heading {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 18px;
  margin-bottom: 20px;
}
.gallery-feed-heading p {
  margin: 7px 0 0;
  color: var(--gallery-muted);
  font-size: 11px;
  line-height: 1.5;
}
.gallery-clear-filter {
  padding: 0 0 4px;
  border: 0;
  background: transparent;
  color: var(--gallery-accent);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: color .18s ease, transform .18s ease;
}
.gallery-clear-filter:hover { color: var(--gallery-ink); }
.gallery-clear-filter:active { transform: scale(.98); }
.gallery-masonry {
  width: 100%;
}
.gallery-state {
  display: grid;
  place-items: center;
  min-height: 320px;
  gap: 12px;
  color: var(--gallery-muted);
  text-align: center;
}
.gallery-state p {
  margin: 0;
  font-size: 12px;
}
.gallery-state button {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: var(--gallery-accent);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: color .18s ease, transform .18s ease;
}
.gallery-state button:hover { color: var(--gallery-ink); }
.gallery-state button:active { transform: scale(.98); }
.gallery-state button :deep(svg) { width: 14px; height: 14px; }
.gallery-state-mark {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: var(--gallery-soft);
  color: var(--gallery-accent);
}
.gallery-state-mark :deep(svg) { width: 20px; height: 20px; }
.gallery-state-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--gallery-line);
  border-top-color: var(--gallery-accent);
  border-radius: 50%;
  animation: gallery-spin .8s linear infinite;
}
.gallery-loadmore {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  padding: 28px 0;
  color: var(--gallery-muted);
  font-size: 11px;
}
.gallery-loadmore .gallery-state-spinner { width: 14px; height: 14px; }
.gallery-sentinel { height: 1px; }

/* 精选条换一批：只在数据真的变化时淡入上浮，KeepAlive 往返激活不会重播。 */
.gallery-featured-enter-active { transition: opacity .3s ease, transform .36s cubic-bezier(.22, 1, .36, 1); }
.gallery-featured-enter-from { opacity: 0; transform: translateY(8px); }

/* 首次加载骨架：占位形状接近真实卡片，避免数据到达时整页跳动。 */
.gallery-skeleton {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 18px;
}
.gallery-skeleton-card { display: grid; gap: 9px; }
.gallery-skeleton-thumb,
.gallery-skeleton-line {
  position: relative;
  display: block;
  overflow: hidden;
  border-radius: 10px;
  background: var(--gallery-soft);
}
.gallery-skeleton-thumb { height: var(--sk-h, 150px); }
.gallery-featured-item > .gallery-skeleton-thumb { height: auto; aspect-ratio: 1.15; }
.gallery-skeleton-line { height: 11px; border-radius: 6px; }
.gallery-skeleton-line.is-short { width: 58%; }
.gallery-skeleton-thumb::after,
.gallery-skeleton-line::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(100deg, transparent 20%, color-mix(in srgb, var(--gallery-paper) 60%, transparent) 50%, transparent 80%);
  background-size: 220% 100%;
  animation: gallery-shimmer 1.4s ease-in-out infinite;
}
@keyframes gallery-shimmer {
  from { background-position: 120% 0; }
  to { background-position: -120% 0; }
}
.gallery-end {
  padding: 34px 0 18px;
  color: var(--gallery-muted);
  font-size: 10px;
  text-align: center;
  letter-spacing: .06em;
}

.gallery-home-content button:focus-visible,
.gallery-home-content a:focus-visible,
.gallery-home-content input:focus-visible {
  outline: 2px solid var(--gallery-accent);
  outline-offset: 3px;
}
@keyframes gallery-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  .gallery-home-content *,
  .gallery-home-content *::before,
  .gallery-home-content *::after {
    scroll-behavior: auto !important;
    transition-duration: .01ms !important;
    animation-duration: .01ms !important;
  }
}
@media (max-width: 1100px) {
  .gallery-content { padding-inline: 0; }
}
@media (max-width: 820px) {
  .gallery-content { padding: 0 0 60px; }
  .gallery-featured-strip { grid-auto-columns: minmax(105px, 30vw); }
  .gallery-feed-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
  /* 手机上标签保持单行横向滚动，换行会把筛选区撑成一堵墙。 */
  .gallery-tag-row { flex-wrap: nowrap; justify-content: flex-start; width: 100%; }
  .gallery-feed-heading { align-items: flex-start; }
  .gallery-mobile-publish {
    position: fixed;
    right: 18px;
    bottom: 20px;
    z-index: 18;
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    border: 0;
    border-radius: 50%;
    background: var(--gallery-ink);
    color: var(--gallery-paper);
    box-shadow: 0 10px 22px rgba(31, 24, 27, .22);
    cursor: pointer;
    transition: transform .18s ease, box-shadow .18s ease;
  }
  .gallery-mobile-publish:hover { transform: translateY(-2px); box-shadow: 0 12px 26px rgba(31, 24, 27, .26); }
  .gallery-mobile-publish:active { transform: scale(.94); }
  .gallery-mobile-publish :deep(svg) { width: 20px; height: 20px; }
}
@media (max-width: 520px) {
  .gallery-search kbd { display: none; }
  .gallery-section-heading h2,
  .gallery-feed-heading h2 { font-size: 19px; }
  .gallery-featured-strip { gap: 9px; grid-auto-columns: minmax(102px, 31vw); }
  .gallery-featured-item > span { font-size: 10px; }
  .gallery-filter-tabs { gap: 18px; }
}
</style>

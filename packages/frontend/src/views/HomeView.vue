<script lang="ts">
export default { name: 'HomeView' }
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, onDeactivated, onBeforeUnmount, nextTick, watch } from 'vue'
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import { useUserStore } from '@/stores/user'
import { useThemeStore } from '@/stores/theme'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import { useListCache, diffLists } from '@/composables/useListCache'
import { scrollBehavior } from '@/utils'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import MediaImage from '@/components/MediaImage.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import QuickUploadSheet from '@/components/QuickUploadSheet.vue'
import {
  IconArrowRight,
  IconBarChart,
  IconHome,
  IconImage,
  IconMenu,
  IconMoon,
  IconPlus,
  IconRefresh,
  IconSearch,
  IconSettings,
  IconSun,
  IconUpload,
} from '@arco-design/web-vue/es/icon'
import type { Content, ListParams, RecommendContent } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const userStore = useUserStore()
const themeStore = useThemeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const listCache = useListCache()
const showUploadSheet = ref(false)
const mobileSidebarOpen = ref(false)
const searchDraft = ref('')
const feedMode = ref<'recommended' | 'latest' | 'popular'>('recommended')
const feedRef = ref<HTMLElement | null>(null)
const featuredRef = ref<HTMLElement | null>(null)

const feedSort = computed(() => (feedMode.value === 'popular' ? 'view_count' : 'created_at'))
const featuredWorks = computed(() => {
  const start = recommendLoader.displayStart.value
  return recommendLoader.recommendContents.value.slice(start, start + 8)
})

function closeMobileSidebar() {
  mobileSidebarOpen.value = false
}

function submitSearch() {
  homeStore.searchKeyword = searchDraft.value.trim()
  closeMobileSidebar()
  resetAndLoad(false)
}

function selectFeedMode(mode: 'recommended' | 'latest' | 'popular') {
  const previousMode = feedMode.value
  feedMode.value = mode
  if (mode === 'recommended') {
    if (previousMode === 'popular') resetAndLoad(false)
    closeMobileSidebar()
    featuredRef.value?.scrollIntoView({ behavior: scrollBehavior(), block: 'start' })
    return
  }
  closeMobileSidebar()
  resetAndLoad(false)
  nextTick(() => scrollToFeed())
}

function goToUpload() {
  closeMobileSidebar()
  showUploadSheet.value = true
}

const swapSections = computed(
  () =>
    searchFilter.selectedTags.value.length > 0 ||
    !!homeStore.searchKeyword.trim(),
)

const allContents = ref<Content[]>([])
function scrollToFeed() {
  feedRef.value?.scrollIntoView({ behavior: scrollBehavior(), block: 'start' })
}
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
let restoreTimer: ReturnType<typeof setTimeout> | undefined
function cancelScrollRestore() {
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
  // localStorage 缓存统一在离开时写一次（搜索模式不缓存），
  // 运行时加载/diff 不再频繁写，避免每轮全量拉取都序列化几百 KB。
  if (!homeStore.searchKeyword) {
    listCache.save(allContents.value, total.value, totalPages.value)
  }
})

async function fetchPage(page: number, append = false, force = false) {
  // 非强制请求沿用"一次只发一个"防抖；强制请求（筛选/搜索重置）可打断在途请求
  if (!force && (isLoading.value || isLoadingMore.value)) return
  const seq = ++loadSeq
  if (append) isLoadingMore.value = true
  else isLoading.value = true

  try {
    let res
    if (homeStore.searchKeyword) {
      res = await contentApi.search(homeStore.searchKeyword, {
        page, page_size: pageSize.value,
        tag: searchFilter.selectedTags.value.length > 0 ? searchFilter.selectedTags.value.join(',') : undefined,
      })
    } else {
      const params: ListParams = { page, page_size: pageSize.value, sort_by: feedSort.value, order: 'desc' }
      if (searchFilter.selectedTags.value.length > 0) params.tag = searchFilter.selectedTags.value.join(',')
      res = await contentApi.list(params)
    }
    // 过期响应：已被更新的请求取代，丢弃，避免旧列表覆盖新筛选结果
    if (seq !== loadSeq) return
    if (res.code === 200) {
      const parsed = res.data.list.map((item: unknown) => ContentSchema.parse(item))
      total.value = res.data.total
      totalPages.value = res.data.total_page
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
    }
  } catch (e) { console.error('加载失败:', e) }
  finally {
    // 仅最新请求复位加载态；过期请求的复位交给更新的一批
    if (seq === loadSeq) { isLoading.value = false; isLoadingMore.value = false }
  }
}

function onTagSelect(tag: string) {
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

watchGlobalSearch(() => resetAndLoad(false))

watch(
  () => homeStore.searchKeyword,
  (keyword) => {
    if (keyword !== searchDraft.value) searchDraft.value = keyword
  },
  { immediate: true },
)

// 首次加载：自动连续拉取所有页（1-100 → 101-200 → …），不依赖滚动触发。
// 图片仍由卡片自身的 loading="lazy" 懒加载，数据量小（每页 100 条）可一次拉满。
async function loadAllPages() {
  if (isLoading.value) return
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

// 首页悬浮按钮上传成功后：同样拉取最新列表 diff（新作品插入顶部，锚定当前阅读位置）
function onUploaded() {
  void syncLatestOnActivated()
}

// 监听 sentinel 进入视口
useIntersectionObserver(sentinelRef, ([{ isIntersecting }]) => {
  if (isIntersecting && hasMore.value && !isLoading.value && !isLoadingMore.value) {
    fetchMore()
  }
}, { rootMargin: '1000px' })

onMounted(() => {
  searchDraft.value = homeStore.searchKeyword
  const navEntries = performance.getEntriesByType('navigation') as PerformanceNavigationTiming[]
  if (navEntries.length > 0 && navEntries[0].type === 'reload') {
    homeStore.clearState()
    listCache.clear()
  }

  // 缓存恢复与搜索入口也需准备标签；展示顺序在本次浏览中保持稳定。
  void searchFilter.loadTags()

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
    recommendLoader.loadedPage.value = homeStore.recommendPage
    // 恢复瀑布流布局缓存
    if (homeStore.cachedPositions.size > 0) {
      waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
    }
    // 缓存的位置是基于"图片已加载"的高度算的；用当前实际高度重排一次，避免占位高度造成空白。
    nextTick(() => waterfall.relayout())
    nextTick(() => requestAnimationFrame(() => homeStore.restoreScroll()))
    return
  }

  // 尝试从 localStorage 加载缓存（先渲染缓存再异步补齐最新数据，不阻塞首屏）
  const cached = listCache.load()
  if (cached && cached.list.length > 0) {
    allContents.value = cached.list
    total.value = cached.total
    totalPages.value = cached.totalPages
    currentPage.value = 1
    // 全量加载：基于缓存补齐剩余页（图片仍懒加载），不依赖滚动触发。
    // 首次 onActivated 由 isFirstActivation 跳过重复 sync。
    void loadAllPages()
  } else {
    // 无缓存：直接自动拉取全部页（1-100 → 101-200 → …）
    void loadAllPages()
  }

  recommendLoader.loadRecommendContents()
})

// keep-alive 激活时恢复滚动位置
onActivated(() => {
  // 恢复瀑布流布局缓存
  if (homeStore.cachedPositions.size > 0) {
    waterfall.restore(homeStore.cachedPositions, homeStore.cachedContainerHeight)
  }
  // 同上：回到首页后先按当前实际高度重排，再恢复滚动。
  nextTick(() => waterfall.relayout())

  // 标记需要在图片加载后恢复滚动
  pendingScrollRestore.value = true

  // 延迟恢复滚动，再拉取最新列表 diff（有新增时锚定当前可见内容，不丢阅读位置）。
  // 首次挂载的激活不 sync（数据已由 onMounted 提供），复位标志后后续激活正常增量同步。
  const firstActivation = isFirstActivation
  isFirstActivation = false
  restoreTimer = setTimeout(() => {
    restoreTimer = undefined
    pendingScrollRestore.value = false
    homeStore.restoreScroll()
    if (firstActivation) return
    void syncLatestOnActivated()
  }, 300)
})
</script>

<template>
  <div class="gallery-home creative-theme" :class="{ 'is-mobile-sidebar-open': mobileSidebarOpen }">
    <div class="gallery-mobile-topbar">
      <button class="gallery-mobile-menu" type="button" aria-label="打开导航" @click="mobileSidebarOpen = true">
        <IconMenu />
      </button>
      <RouterLink to="/" class="gallery-mobile-brand">小泉宇宙</RouterLink>
      <button class="gallery-mobile-upload" type="button" aria-label="发布作品" @click="goToUpload">
        <IconPlus />
      </button>
    </div>

    <aside class="gallery-sidebar" aria-label="作品广场导航">
      <div class="gallery-brand-row">
        <button
          class="gallery-theme-toggle"
          type="button"
          :aria-label="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到暗色模式'"
          @click="themeStore.toggleMode()"
        >
          <IconSun v-if="themeStore.mode === 'dark'" />
          <IconMoon v-else />
        </button>
        <RouterLink to="/" class="gallery-brand-link" @click="closeMobileSidebar">
          <strong>小泉宇宙</strong>
          <span>二创画廊</span>
        </RouterLink>
      </div>

      <div class="gallery-primary-switch" aria-label="主要入口">
        <RouterLink to="/studio" class="gallery-primary-link" @click="closeMobileSidebar">
          <IconUpload />
          <span>工作台</span>
        </RouterLink>
        <RouterLink to="/" class="gallery-primary-link is-active" @click="closeMobileSidebar">
          <IconHome />
          <span>作品广场</span>
        </RouterLink>
      </div>

      <nav class="gallery-side-nav" aria-label="发现">
        <p class="gallery-side-label">发现</p>
        <button class="gallery-side-link" :class="{ 'is-active': feedMode === 'recommended' }" type="button" @click="selectFeedMode('recommended')">
          <IconHome />
          <span>推荐</span>
        </button>
        <button class="gallery-side-link" :class="{ 'is-active': feedMode === 'latest' }" type="button" @click="selectFeedMode('latest')">
          <IconArrowRight />
          <span>最新发布</span>
        </button>
        <button class="gallery-side-link" :class="{ 'is-active': feedMode === 'popular' }" type="button" @click="selectFeedMode('popular')">
          <IconBarChart />
          <span>热门作品</span>
        </button>
        <RouterLink to="/ideas" class="gallery-side-link" @click="closeMobileSidebar">
          <IconPlus />
          <span>创意广场</span>
        </RouterLink>
      </nav>

      <div class="gallery-side-library">
        <p class="gallery-side-label">我的库</p>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin" class="gallery-side-link" @click="closeMobileSidebar">
          <IconSettings />
          <span>我的内容</span>
        </RouterLink>
        <RouterLink v-else to="/login" class="gallery-library-note" @click="closeMobileSidebar">
          登录后查看收藏和发布
          <IconArrowRight />
        </RouterLink>
      </div>

      <div class="gallery-sidebar-bottom">
        <button class="gallery-publish-button" type="button" @click="goToUpload">
          <IconUpload />
          <span>发布作品</span>
        </button>
        <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="gallery-user-row" @click="closeMobileSidebar">
          <span class="gallery-user-avatar"><IconSettings /></span>
          <span><strong>登录 / 注册</strong><small>保存你的创作</small></span>
        </RouterLink>
        <div v-else class="gallery-user-row">
          <UserAvatar :size="32" :email="userStore.user?.email" :username="userStore.user?.username" />
          <span><strong>{{ userStore.user?.username }}</strong><small>创作者</small></span>
        </div>
      </div>
    </aside>

    <button v-if="mobileSidebarOpen" class="gallery-sidebar-scrim" type="button" aria-label="关闭导航" @click="closeMobileSidebar"></button>

    <main class="gallery-main" role="main">
      <header class="gallery-topbar">
        <div class="gallery-page-tabs" role="tablist" aria-label="页面视图">
          <button class="gallery-page-tab is-active" type="button" role="tab" aria-selected="true" @click="scrollToFeed">首页</button>
          <button class="gallery-page-tab" type="button" role="tab" aria-selected="false" @click="selectFeedMode('popular')">发现</button>
        </div>
        <div class="gallery-top-actions">
          <label class="gallery-search">
            <IconSearch aria-hidden="true" />
            <input v-model="searchDraft" type="search" placeholder="搜索作品、提示词、标签或用户" @keydown.enter.prevent="submitSearch" />
            <kbd>/</kbd>
          </label>
          <button class="gallery-theme-toggle gallery-theme-toggle-main" type="button" :aria-label="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到暗色模式'" @click="themeStore.toggleMode()">
            <IconSun v-if="themeStore.mode === 'dark'" />
            <IconMoon v-else />
          </button>
          <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="gallery-login-link">登录</RouterLink>
          <RouterLink v-else to="/admin" class="gallery-top-avatar" aria-label="打开我的内容">
            <UserAvatar :size="32" :email="userStore.user?.email" :username="userStore.user?.username" />
          </RouterLink>
        </div>
      </header>

      <div class="gallery-content">
        <section class="gallery-intro" aria-labelledby="home-title">
          <div>
            <h1 id="home-title">作品广场</h1>
            <p>把喜欢的角色、画面和灵感，留在一个可以被再次发现的地方。</p>
          </div>
          <div class="gallery-intro-stats" aria-label="作品统计">
            <span><strong>{{ total > 0 ? total : '—' }}</strong> 作品</span>
            <span><strong>{{ searchFilter.allTags.value.length || '—' }}</strong> 标签</span>
            <span><strong>{{ homeStore.searchKeyword ? '筛选中' : '持续更新' }}</strong></span>
          </div>
        </section>

        <section ref="featuredRef" class="gallery-featured" aria-labelledby="featured-title">
          <div class="gallery-section-heading">
            <div>
              <span class="gallery-section-label">CURATED WORKS</span>
              <h2 id="featured-title">正在被看见</h2>
            </div>
            <button class="gallery-icon-action" type="button" :disabled="recommendLoader.isRecommendLoading.value" aria-label="换一批推荐作品" title="换一批" @click="recommendLoader.refreshRecommend()">
              <IconRefresh :spin="recommendLoader.isRecommendLoading.value" />
            </button>
          </div>
          <div v-if="featuredWorks.length" class="gallery-featured-strip">
            <button v-for="item in featuredWorks" :key="item.id" class="gallery-featured-item" type="button" @click="openContent(item)">
              <MediaImage :src="item.thumb" :alt="item.title" :preview="false" />
              <span>{{ item.title }}</span>
            </button>
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
            <button v-if="searchFilter.shuffledTags.value.length > 1" class="gallery-tag gallery-tag-refresh" type="button" aria-label="换一组标签" title="换一组标签" @click="searchFilter.reshuffleTags()"><IconRefresh /></button>
          </div>
        </section>

        <section ref="feedRef" class="gallery-feed" aria-labelledby="gallery-feed-title">
          <div class="gallery-feed-heading">
            <div>
              <h2 id="gallery-feed-title">{{ homeStore.searchKeyword ? '搜索结果' : searchFilter.selectedTags.value[0] || (feedMode === 'popular' ? '热门作品' : '最新作品') }}</h2>
              <p v-if="homeStore.searchKeyword || searchFilter.selectedTags.value.length">{{ homeStore.searchKeyword ? `正在查找“${homeStore.searchKeyword}”` : '按标签筛选' }} · {{ total }} 件作品</p>
              <p v-else>{{ total > 0 ? `${total} 件作品，持续更新中` : '作品正在加载' }}</p>
            </div>
            <button v-if="swapSections" class="gallery-clear-filter" type="button" @click="clearFilters">清除筛选</button>
          </div>

          <div v-if="isLoading && allContents.length === 0" class="gallery-state" role="status">
            <div class="gallery-state-spinner"></div>
            <p>正在加载作品</p>
          </div>
          <div v-else-if="!isLoading && allContents.length === 0" class="gallery-state">
            <div class="gallery-state-mark"><IconImage /></div>
            <p>{{ homeStore.searchKeyword || searchFilter.selectedTags.value.length ? '没有找到符合条件的作品' : '还没有作品出现在这里' }}</p>
            <button type="button" @click="goToUpload">发布第一件作品 <IconArrowRight /></button>
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
    </main>

    <button class="gallery-mobile-publish" type="button" aria-label="发布作品" @click="goToUpload"><IconPlus /></button>
    <QuickUploadSheet :open="showUploadSheet" @close="showUploadSheet = false" @uploaded="onUploaded" />
  </div>
</template>

<style scoped>
.gallery-home {
  --gallery-title-font: Georgia, 'Songti SC', 'SimSun', serif;
  --gallery-canvas: #ebe4e1;
  --gallery-paper: #fffdf9;
  --gallery-ink: #201b1c;
  --gallery-muted: #817779;
  --gallery-line: rgba(52, 40, 42, .14);
  --gallery-soft: #f1e9e6;
  --gallery-accent: #6b173b;
  --gallery-accent-soft: #efe1e5;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  min-height: 100vh;
  background: var(--gallery-canvas);
  color: var(--gallery-ink);
  font-family: var(--font-sans);
}
body[arco-theme='dark'] .gallery-home {
  --gallery-canvas: #211b1e;
  --gallery-paper: #2b2427;
  --gallery-ink: #f8f0ed;
  --gallery-muted: #c0b1b4;
  --gallery-line: rgba(250, 240, 237, .16);
  --gallery-soft: #342b2f;
  --gallery-accent: #f0a8bb;
  --gallery-accent-soft: #4a303a;
}
.gallery-sidebar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  flex-direction: column;
  min-width: 0;
  height: 100vh;
  padding: 28px 20px 20px;
  background: var(--gallery-paper);
  border-right: 1px solid var(--gallery-line);
}
.gallery-brand-row {
  display: flex;
  align-items: flex-start;
  gap: 13px;
  min-width: 0;
}
.gallery-brand-link {
  display: grid;
  gap: 4px;
  min-width: 0;
  color: var(--gallery-ink);
  text-decoration: none;
}
.gallery-brand-link strong {
  font: 600 26px/1.05 var(--gallery-title-font);
  letter-spacing: -.02em;
}
.gallery-brand-link span {
  color: var(--gallery-muted);
  font-size: 10px;
  letter-spacing: .14em;
}
.gallery-theme-toggle {
  display: inline-grid;
  place-items: center;
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--gallery-accent);
  cursor: pointer;
  transition: background .18s ease, transform .18s ease;
}
.gallery-theme-toggle:hover {
  background: var(--gallery-accent-soft);
  transform: rotate(-12deg);
}
.gallery-theme-toggle :deep(svg) { width: 17px; height: 17px; }
.gallery-primary-switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 4px;
  margin-top: 34px;
  padding: 4px;
  border-radius: 14px;
  background: var(--gallery-soft);
}
.gallery-primary-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-width: 0;
  min-height: 40px;
  padding: 0 8px;
  border-radius: 10px;
  color: var(--gallery-muted);
  text-decoration: none;
  font-size: 11px;
  transition: background .18s ease, color .18s ease, transform .18s ease;
}
.gallery-primary-link :deep(svg) { width: 15px; height: 15px; }
.gallery-primary-link:hover { color: var(--gallery-ink); }
.gallery-primary-link.is-active {
  color: #fff;
  background: var(--gallery-ink);
  box-shadow: 0 5px 12px rgba(32, 27, 28, .12);
}
body[arco-theme='dark'] .gallery-primary-link.is-active {
  color: #2b2023;
  background: var(--gallery-ink);
}
.gallery-side-nav,
.gallery-side-library {
  display: grid;
  gap: 4px;
  margin-top: 42px;
}
.gallery-side-library {
  margin-top: 28px;
}
.gallery-side-label {
  margin: 0 0 7px 11px;
  color: var(--gallery-muted);
  font-size: 10px;
  letter-spacing: .14em;
  text-transform: uppercase;
}
.gallery-side-link {
  display: flex;
  align-items: center;
  gap: 11px;
  min-height: 38px;
  padding: 0 11px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 12px;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
  transition: background .18s ease, color .18s ease, transform .18s ease;
}
.gallery-side-link :deep(svg) {
  width: 16px;
  height: 16px;
  flex: 0 0 auto;
}
.gallery-side-link:hover {
  color: var(--gallery-ink);
  background: var(--gallery-soft);
  transform: translateX(2px);
}
.gallery-side-link.is-active {
  color: var(--gallery-accent);
  background: var(--gallery-accent-soft);
  font-weight: 600;
}
.gallery-library-note {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 11px;
  color: var(--gallery-muted);
  font-size: 11px;
  line-height: 1.6;
  text-decoration: none;
}
.gallery-library-note:hover { color: var(--gallery-accent); }
.gallery-library-note :deep(svg) { width: 14px; height: 14px; flex: 0 0 auto; }
.gallery-sidebar-bottom {
  display: grid;
  gap: 16px;
  margin-top: auto;
}
.gallery-publish-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  border: 0;
  border-radius: 12px;
  background: var(--gallery-ink);
  color: var(--gallery-paper);
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  transition: transform .18s ease, box-shadow .18s ease, background .18s ease;
}
.gallery-publish-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 9px 22px rgba(32, 27, 28, .16);
}
.gallery-publish-button :deep(svg) { width: 16px; height: 16px; }
.gallery-user-row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--gallery-ink);
  text-decoration: none;
}
.gallery-user-row:hover { color: var(--gallery-accent); }
.gallery-user-avatar {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--gallery-soft);
  color: var(--gallery-muted);
}
.gallery-user-avatar :deep(svg) { width: 15px; height: 15px; }
.gallery-user-row > span:last-child {
  display: grid;
  gap: 2px;
  min-width: 0;
}
.gallery-user-row strong {
  overflow: hidden;
  color: inherit;
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gallery-user-row small {
  color: var(--gallery-muted);
  font-size: 10px;
}
.gallery-main {
  min-width: 0;
  background: var(--gallery-canvas);
}
.gallery-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  min-height: 84px;
  padding: 0 clamp(24px, 4vw, 64px);
}
.gallery-page-tabs {
  display: flex;
  align-items: center;
  gap: 24px;
}
.gallery-page-tab {
  position: relative;
  padding: 8px 0 11px;
  border: 0;
  background: transparent;
  color: var(--gallery-muted);
  font: 600 25px/1 var(--gallery-title-font);
  cursor: pointer;
}
.gallery-page-tab.is-active { color: var(--gallery-ink); }
.gallery-page-tab.is-active::after {
  content: '';
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--gallery-accent);
}
.gallery-top-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.gallery-search {
  display: flex;
  align-items: center;
  gap: 8px;
  width: min(320px, 28vw);
  min-width: 180px;
  height: 42px;
  padding: 0 10px 0 14px;
  border: 1px solid transparent;
  border-radius: 14px;
  background: var(--gallery-paper);
  color: var(--gallery-muted);
  transition: border-color .18s ease, box-shadow .18s ease;
}
.gallery-search:focus-within {
  border-color: color-mix(in srgb, var(--gallery-accent) 48%, transparent);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--gallery-accent) 9%, transparent);
}
.gallery-search :deep(svg) { width: 16px; height: 16px; flex: 0 0 auto; }
.gallery-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--gallery-ink);
  font: inherit;
  font-size: 11px;
}
.gallery-search input::placeholder { color: var(--gallery-muted); opacity: .9; }
.gallery-search kbd {
  flex: 0 0 auto;
  padding: 2px 6px;
  border: 1px solid var(--gallery-line);
  border-radius: 5px;
  color: var(--gallery-muted);
  font: 10px/1.3 var(--font-sans);
}
.gallery-theme-toggle-main {
  background: var(--gallery-paper);
}
.gallery-login-link,
.gallery-top-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 38px;
  color: var(--gallery-ink);
  text-decoration: none;
}
.gallery-login-link {
  padding: 0 13px;
  border: 1px solid var(--gallery-line);
  border-radius: 10px;
  font-size: 11px;
}
.gallery-login-link:hover {
  border-color: var(--gallery-accent);
  color: var(--gallery-accent);
}
.gallery-top-avatar {
  border-radius: 50%;
  transition: transform .18s ease;
}
.gallery-top-avatar:hover { transform: translateY(-2px); }
.gallery-content {
  width: min(100%, 1680px);
  margin: 0 auto;
  padding: 0 clamp(24px, 4vw, 64px) 76px;
}
.gallery-intro {
  display: flex;
  align-items: end;
  justify-content: space-between;
  gap: 24px;
  padding: 12px 0 25px;
  border-bottom: 1px solid var(--gallery-line);
}
.gallery-intro h1 {
  margin: 0;
  font: 600 clamp(38px, 4.1vw, 58px)/1.05 var(--gallery-title-font);
  letter-spacing: -.035em;
}
.gallery-intro p {
  max-width: 42em;
  margin: 13px 0 0;
  color: var(--gallery-muted);
  font-size: 12px;
  line-height: 1.75;
}
.gallery-intro-stats {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px 18px;
  padding-bottom: 4px;
  color: var(--gallery-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.gallery-intro-stats strong {
  color: var(--gallery-ink);
  font-weight: 600;
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
.gallery-featured-item :deep(.arco-image) {
  display: block;
  width: 100%;
  aspect-ratio: 1.15;
  overflow: hidden;
  border-radius: 8px;
  background: var(--gallery-soft);
}
.gallery-featured-item :deep(.arco-image-img) {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform .3s ease;
}
.gallery-featured-item:hover :deep(.arco-image-img) { transform: scale(1.06); }
.gallery-featured-item > span {
  overflow: hidden;
  color: var(--gallery-ink);
  font-size: 11px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  display: flex;
  align-items: center;
  gap: 22px;
  flex: 0 0 auto;
}
.gallery-filter-tab {
  position: relative;
  padding: 8px 0 10px;
  border: 0;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
.gallery-filter-tab.is-active {
  color: var(--gallery-ink);
  font-weight: 600;
}
.gallery-filter-tab.is-active::after {
  content: '';
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  border-radius: 2px;
  background: var(--gallery-accent);
}
.gallery-tag-row {
  display: flex;
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
  padding: 0 9px;
  border: 1px solid transparent;
  border-radius: 7px;
  background: transparent;
  color: var(--gallery-muted);
  font: inherit;
  font-size: 10px;
  cursor: pointer;
  transition: background .18s ease, color .18s ease, border-color .18s ease;
}
.gallery-tag:hover,
.gallery-tag.is-active {
  border-color: var(--gallery-line);
  background: var(--gallery-paper);
  color: var(--gallery-accent);
}
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
}
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
}
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
.gallery-end {
  padding: 34px 0 18px;
  color: var(--gallery-muted);
  font-size: 10px;
  text-align: center;
  letter-spacing: .06em;
}
.gallery-mobile-topbar,
.gallery-mobile-publish,
.gallery-sidebar-scrim {
  display: none;
}
.gallery-home button:focus-visible,
.gallery-home a:focus-visible,
.gallery-home input:focus-visible {
  outline: 2px solid var(--gallery-accent);
  outline-offset: 3px;
}
@keyframes gallery-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  .gallery-home *,
  .gallery-home *::before,
  .gallery-home *::after {
    scroll-behavior: auto !important;
    transition-duration: .01ms !important;
    animation-duration: .01ms !important;
  }
}
@media (max-width: 1100px) {
  .gallery-home { grid-template-columns: 232px minmax(0, 1fr); }
  .gallery-sidebar { padding-inline: 15px; }
  .gallery-primary-link { font-size: 10px; }
  .gallery-content { padding-inline: 32px; }
  .gallery-topbar { padding-inline: 32px; }
  .gallery-intro { align-items: flex-start; flex-direction: column; }
  .gallery-intro-stats { justify-content: flex-start; }
}
@media (max-width: 820px) {
  .gallery-home {
    display: block;
    min-height: 100dvh;
  }
  .gallery-sidebar {
    position: fixed;
    inset: 0 auto 0 0;
    width: min(82vw, 300px);
    transform: translateX(-104%);
    transition: transform .24s cubic-bezier(.22, 1, .36, 1);
    box-shadow: 12px 0 32px rgba(31, 24, 27, .14);
  }
  .gallery-home.is-mobile-sidebar-open .gallery-sidebar { transform: translateX(0); }
  .gallery-sidebar-scrim {
    position: fixed;
    inset: 0;
    z-index: 25;
    display: block;
    border: 0;
    background: rgba(32, 25, 27, .32);
    cursor: pointer;
  }
  .gallery-mobile-topbar {
    position: sticky;
    top: 0;
    z-index: 20;
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 58px;
    padding: 0 16px;
    background: color-mix(in srgb, var(--gallery-paper) 92%, transparent);
    border-bottom: 1px solid var(--gallery-line);
    backdrop-filter: blur(14px);
  }
  .gallery-mobile-brand {
    color: var(--gallery-ink);
    font: 600 20px/1 var(--gallery-title-font);
    text-decoration: none;
  }
  .gallery-mobile-menu,
  .gallery-mobile-upload {
    display: inline-grid;
    place-items: center;
    width: 34px;
    height: 34px;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: transparent;
    color: var(--gallery-accent);
    cursor: pointer;
  }
  .gallery-mobile-menu :deep(svg),
  .gallery-mobile-upload :deep(svg) { width: 18px; height: 18px; }
  .gallery-topbar {
    min-height: 68px;
    padding: 0 16px;
  }
  .gallery-topbar .gallery-theme-toggle-main,
  .gallery-topbar .gallery-login-link,
  .gallery-topbar .gallery-top-avatar {
    display: none;
  }
  .gallery-top-actions { width: 100%; }
  .gallery-search {
    width: 100%;
    min-width: 0;
    height: 40px;
  }
  .gallery-page-tabs { display: none; }
  .gallery-content { padding: 0 16px 60px; }
  .gallery-intro { padding-top: 9px; }
  .gallery-intro h1 { font-size: 39px; }
  .gallery-intro p { font-size: 11px; }
  .gallery-intro-stats { gap: 8px 13px; font-size: 10px; }
  .gallery-featured-strip { grid-auto-columns: minmax(105px, 30vw); }
  .gallery-feed-toolbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
  .gallery-tag-row { justify-content: flex-start; width: 100%; }
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
  }
  .gallery-mobile-publish :deep(svg) { width: 20px; height: 20px; }
}
@media (max-width: 520px) {
  .gallery-topbar { min-height: 60px; }
  .gallery-search kbd { display: none; }
  .gallery-page-tab { font-size: 22px; }
  .gallery-section-heading h2,
  .gallery-feed-heading h2 { font-size: 19px; }
  .gallery-featured-strip { gap: 9px; grid-auto-columns: minmax(102px, 31vw); }
  .gallery-featured-item > span { font-size: 10px; }
  .gallery-filter-tabs { gap: 18px; }
}
</style>

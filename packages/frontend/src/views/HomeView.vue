<script lang="ts">
export default { name: 'HomeView' }
</script>

<script setup lang="ts">
import { ref, computed, onMounted, onActivated, nextTick, watch } from 'vue'
import { useIntersectionObserver } from '@vueuse/core'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import { useRecommendLoader } from '@/composables/useRecommendLoader'
import { useSearchFilter } from '@/composables/useSearchFilter'
import { watchGlobalSearch } from '@/composables/useGlobalSearch'
import { useWaterfallLayout } from '@/composables/useWaterfallLayout'
import { useListCache, diffLists } from '@/composables/useListCache'
import { ContentSchema } from '@/types/schemas'
import WaterfallCard from '@/components/WaterfallCard.vue'
import RecommendSection from '@/components/RecommendSection.vue'
import QuickUploadSheet from '@/components/QuickUploadSheet.vue'
import { IconUpload } from '@arco-design/web-vue/es/icon'
import type { Content, ListParams, RecommendContent } from '@/types'

const router = useRouter()
const homeStore = useHomeStore()
const recommendLoader = useRecommendLoader()
const searchFilter = useSearchFilter()
const listCache = useListCache()
const showUploadSheet = ref(false)

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
      const params: ListParams = { page, page_size: pageSize.value, sort_by: 'created_at', order: 'desc' }
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

function resetAndLoad() {
  // 作废所有在途请求，防止旧响应把新筛选结果覆盖掉
  loadSeq++
  currentPage.value = 1
  allContents.value = []
  isLoading.value = false
  isLoadingMore.value = false
  listCache.clear()
  void fetchPage(1, false, true)
}

watchGlobalSearch(() => resetAndLoad())

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
      const res = await contentApi.list({ page, page_size: batchSize, sort_by: 'created_at', order: 'desc' })
      // 过期响应（被搜索/筛选重置打断）直接丢弃
      if (seq !== loadSeq) return
      if (res.code !== 200) break
      const parsed = res.data.list.map((item: unknown) => ContentSchema.parse(item))
      total.value = res.data.total
      // totalPages 统一按滚动分页的 page_size（pageSize.value=20）计算，
      // 与 hasMore / fetchMore / fetchFreshList 的页码基准一致。
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

// 增量加载更多时也更新缓存（全量加载后的兜底：diff 期间有新内容时补拉）
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
    const res = await contentApi.list({ page: p, page_size: 100, sort_by: 'created_at', order: 'desc' })
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
  if (homeStore.searchKeyword || allContents.value.length === 0) return
  try {
    // 第一步：增量探测 —— 只拉最新一页，对比本地头部
    const probe = await contentApi.list({ page: 1, page_size: 100, sort_by: 'created_at', order: 'desc' })
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
  const navEntries = performance.getEntriesByType('navigation') as PerformanceNavigationTiming[]
  if (navEntries.length > 0 && navEntries[0].type === 'reload') {
    homeStore.clearState()
    listCache.clear()
  }

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
  searchFilter.loadTags()
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
  setTimeout(() => {
    pendingScrollRestore.value = false
    homeStore.restoreScroll()
    if (isFirstActivation) {
      isFirstActivation = false
      return
    }
    void syncLatestOnActivated()
  }, 300)
})
</script>

<template>
  <div class="wf-root">
    <RecommendSection
      v-if="!swapSections && recommendLoader.recommendContents.value.length > 0"
      :loader="recommendLoader"
      @click="openContent"
    />

    <div class="wf-masonry-wrap">
      <div v-if="isLoading && allContents.length === 0" class="wf-center-state">
        <div class="wf-spinner-lg"></div><p>加载中...</p>
      </div>
      <div v-else-if="!isLoading && allContents.length === 0" class="wf-center-state">
        <p>暂无内容</p>
      </div>
      <div
        v-else
        ref="masonryRef"
        class="wf-masonry"
        :style="{ position: 'relative', height: waterfall.containerHeight.value + 'px' }"
      >
        <WaterfallCard
          v-for="item in allContents"
          :key="item.id"
          :item="item"
          :data-wf-id="item.id"
          :style="{
            position: 'absolute',
            left: (waterfall.positions.value.get(item.id)?.x ?? 0) + 'px',
            top: (waterfall.positions.value.get(item.id)?.y ?? 0) + 'px',
            width: (waterfall.positions.value.get(item.id)?.w ?? 0) + 'px',
          }"
          @click="openContent"
          @image-loaded="onCardResized"
        />
      </div>
      <div v-if="isLoadingMore" class="wf-loadmore"><div class="wf-spinner-sm"></div><span>加载更多...</span></div>
      <div ref="sentinelRef" class="wf-sentinel"></div>
      <div v-if="!hasMore && allContents.length > 0" class="wf-end"><span>— 到底啦 —</span></div>
    </div>

    <!-- 移动端悬浮上传按钮 -->
    <button class="wf-fab" @click="showUploadSheet = true">
      <IconUpload />
    </button>

    <!-- 快速上传弹窗 -->
    <QuickUploadSheet :open="showUploadSheet" @close="showUploadSheet = false" @uploaded="onUploaded" />
  </div>
</template>

<style lang="scss" scoped>
.wf-root { min-height: 100vh; background: transparent; color: var(--color-text-1); }

.wf-masonry-wrap {
  max-width: 1600px; margin: 0 auto; padding: 0.75rem 0.75rem 3rem;
  @media (min-width: 640px) { padding: 1rem 1rem 3rem; }
}
.wf-masonry { width: 100%; }

.wf-center-state {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 4rem 1rem; color: var(--color-text-2);
}
.wf-center-state p { font-size: 0.875rem; margin-top: 1rem; }

.wf-spinner-lg {
  width: 2rem; height: 2rem; border: 3px solid var(--color-border-2);
  border-top-color: rgb(var(--primary-6)); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-spinner-sm {
  width: 1.25rem; height: 1.25rem; border: 2px solid var(--color-border-2);
  border-top-color: rgb(var(--primary-6)); border-radius: 50%;
  animation: wf-spin 0.7s linear infinite;
}
.wf-loadmore {
  display: flex; align-items: center; justify-content: center;
  gap: 0.5rem; padding: 1.5rem; font-size: 0.8125rem; color: var(--color-text-2);
}
.wf-sentinel { height: 1px; }
.wf-end { text-align: center; padding: 2rem; font-size: 0.75rem; color: var(--color-text-2); opacity: 0.6; }

/* 悬浮上传按钮 */
.wf-fab {
  display: none;
}

@media (max-width: 768px) {
  .wf-fab {
    display: flex;
    align-items: center;
    justify-content: center;
    position: fixed;
    bottom: 24px;
    right: 24px;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: rgb(var(--primary-6));
    color: var(--color-white);
    border: none;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    cursor: pointer;
    z-index: 100;
    font-size: 22px;
    transition: transform 0.2s, box-shadow 0.2s;
  }
  .wf-fab:hover {
    transform: scale(1.05);
    box-shadow: 0 6px 16px rgba(0, 0, 0, 0.25);
  }
  .wf-fab:active {
    transform: scale(0.95);
  }
}

@keyframes wf-spin { to { transform: rotate(360deg); } }
</style>

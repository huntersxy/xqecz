import { nextTick, onUnmounted, ref, watch, type Ref } from 'vue'

interface WaterfallItem {
  id: string | number
  [key: string]: unknown
}

/** 一张卡片的稳定布局信息：所在列在生命周期内保持不变，高度变化只影响本列下方卡片 */
export interface Position {
  x: number
  y: number
  w: number
  h: number
  col: number
}

export interface LayoutItem {
  id: string | number
  height: number
}

export interface LayoutResult {
  positions: Map<string | number, Position>
  colHeights: number[]
  height: number
}

/**
 * 纯布局算法（与 DOM 解耦，便于单测）：
 * - 默认保留每张卡片已有的列（`preserveColumns=true`），因此某张卡片高度变化
 *   只会顺移本列下方卡片，不会让整列卡片跨列乱跳（这是旧实现"图片加载后瀑布流
 *   到处跳/换列"的根因）。
 * - `preserveColumns=false`（列数/列宽变化）时按"最短列"重新分配所有卡片。
 * - 卡片高度为 0（尚未量到 DOM）时回退到上次已知高度，避免把卡片叠在顶部。
 */
export function computeLayout(
  items: LayoutItem[],
  cols: number,
  colW: number,
  gap: number,
  previous?: Map<string | number, Position>,
  preserveColumns = true,
): LayoutResult {
  const colHeights = Array.from({ length: cols }, () => 0)
  const positions = new Map<string | number, Position>()

  function shortestCol(): number {
    let min = 0
    for (let i = 1; i < cols; i++) {
      if (colHeights[i] < colHeights[min]) min = i
    }
    return min
  }

  for (const item of items) {
    const prev = previous?.get(item.id)
    const col =
      preserveColumns && prev && prev.col >= 0 && prev.col < cols
        ? prev.col
        : shortestCol()
    const h = item.height > 0 ? item.height : (prev?.h ?? 0)
    positions.set(item.id, { x: col * (colW + gap), y: colHeights[col], w: colW, h, col })
    colHeights[col] += h + gap
  }

  return { positions, colHeights, height: Math.max(...colHeights, 0) }
}

const GAP = 10

// 列底失衡阈值：增量重排后若最长列与最短列高度差超过此值（约为 2~3 张卡片高度），
// 触发一次全量重排重新平衡。图片高度天然有差异，过小的阈值会导致频繁全量重排（卡片换列跳动）。
const IMBALANCE_THRESHOLD = 1000

/**
 * 瀑布流布局（首页）。
 *
 * 设计要点（相对旧实现）：
 * - 稳定列：卡片一旦落列就不再换列，图片懒加载导致的高度变化只顺移本列下方卡片，
 *   配合滚动锚定，滚动/加载图片时不再出现整屏乱跳。
 * - 单一调度：所有触发源（图片加载、卡片尺寸变化、容器宽度变化）合并到一帧内只跑
 *   一次布局，避免旧实现的重复/竞态布局（同帧多次量高得到不一致结果导致重叠/空白）。
 * - 卡片高度统一由 `[data-wf-id]` 批量量取，不再逐卡 querySelector。
 * - 滚动锚定：布局重算前后把视口顶部第一张可见卡片固定在屏幕原位，图片加载把内容
 *   撑高时阅读位置不漂移。
 */
export function useWaterfallLayout(
  containerRef: Ref<HTMLElement | null>,
  items: Ref<WaterfallItem[]>,
) {
  const positions = ref<Map<string | number, Position>>(new Map())
  const containerHeight = ref(0)
  const isLayoutReady = ref(false)

  // 内部布局状态
  let cols = 0
  let colW = 0
  let lastContainerWidth = 0
  let rebalanceScheduled = false

  function getColumnCount(width: number) {
    if (width >= 1400) return 5
    if (width >= 1024) return 4
    if (width >= 640) return 3
    return 2
  }

  function getColWidth(el: HTMLElement, cols: number) {
    return (el.clientWidth - GAP * (cols - 1)) / cols
  }

  /** 一次量取所有已渲染卡片的高度（按数字 id 与字符串 id 双键登记，兼容两种来源） */
  function measureHeights(el: HTMLElement): Map<string | number, number> {
    const heights = new Map<string | number, number>()
    for (const card of el.querySelectorAll<HTMLElement>('[data-wf-id]')) {
      const raw = card.getAttribute('data-wf-id')
      if (raw === null) continue
      const h = card.offsetHeight
      const num = Number(raw)
      if (Number.isFinite(num)) heights.set(num, h)
      heights.set(raw, h)
    }
    return heights
  }

  /** 同步执行一次布局：列数/列宽变化或 `full` → 全量重排（重新平衡所有列）；否则增量（保留列） */
  function relayout(opts: { anchor?: boolean; full?: boolean } = {}) {
    const el = containerRef.value
    if (!el || items.value.length === 0) {
      if (positions.value.size > 0 || containerHeight.value !== 0) reset()
      return
    }

    const heights = measureHeights(el)
    const prevCols = cols
    const nextCols = getColumnCount(el.clientWidth)
    const nextColW = getColWidth(el, nextCols)
    const colsChanged = nextCols !== prevCols || Math.abs(nextColW - colW) > 0.5
    cols = nextCols
    colW = nextColW

    // 沿用已有位置：新卡片（分页追加/顶部插入）自动落到最短列，旧卡片保持所在列，
    // 位置集合被显式清空（reset）后自然回到全量最短列分配。
    // `full`（数据集合变化：分页追加/diff 更新/列表替换）时强制全量重排：
    // 增量分配只保证「新卡片去当前最短列」，但图片懒加载导致列高测量失真时，
    // 新卡片会集中涌入被低估的列，且稳定列不再调整 → 列底永久失衡（底部大片空缺）。
    // previous 始终传入（仅作高度回退），full 时通过 preserveColumns=false 忽略其 col。
    const previous = positions.value.size > 0 ? positions.value : undefined
    const layoutItems = items.value.map((it) => ({
      id: it.id,
      height: heights.get(it.id) ?? 0,
    }))

    const anchor = opts.anchor === false ? null : captureAnchor()
    const result = computeLayout(layoutItems, cols, colW, GAP, previous, !colsChanged && !opts.full)
    positions.value = result.positions
    containerHeight.value = result.height
    isLayoutReady.value = true
    if (anchor) restoreAnchor(anchor)

    // 增量重排后收敛列底：图片加载等只顺移本列下方卡片，若列间高度差被拉得过大
    // （例如某列图片集体加载完成而其他列未加载），短列底部会出现大片空缺。
    // 超过阈值时安排一次全量重排重新平衡（带锚定，视觉上不跳动）。
    if (!opts.full && !colsChanged && !rebalanceScheduled) {
      const maxH = Math.max(...result.colHeights, 0)
      const minH = Math.min(...result.colHeights, 0)
      if (maxH - minH > IMBALANCE_THRESHOLD) {
        rebalanceScheduled = true
        requestAnimationFrame(() => {
          rebalanceScheduled = false
          relayout({ anchor: true, full: true })
        })
      }
    }
  }

  /** 滚动锚定：记录视口顶部第一张可见卡片的旧坐标，布局后把该卡片移回屏幕原位 */
  function captureAnchor(): { id: string | number; y: number; scrollY: number } | null {
    if (typeof window === 'undefined') return null
    const scrollY = window.scrollY
    if (scrollY <= 0) return null
    let anchorId: string | number | null = null
    let bestY = Number.POSITIVE_INFINITY
    for (const [id, pos] of positions.value) {
      if (pos.y >= scrollY - 8 && pos.y < bestY) {
        anchorId = id
        bestY = pos.y
      }
    }
    if (anchorId === null) return null
    return { id: anchorId, y: bestY, scrollY }
  }

  function restoreAnchor(anchor: { id: string | number; y: number; scrollY: number }) {
    const pos = positions.value.get(anchor.id)
    if (!pos) return
    const delta = pos.y - anchor.y
    if (Math.abs(delta) > 0.5) {
      window.scrollTo({ top: Math.max(0, anchor.scrollY + delta), behavior: 'instant' })
    }
  }

  // ── 调度：同一帧内只跑一次布局，图片加载等高频事件合并处理 ──
  let rafId: number | null = null
  let anchorOnNext = false

  function scheduleRelayout(anchor = false) {
    if (rafId !== null) {
      anchorOnNext = anchorOnNext || anchor
      return
    }
    anchorOnNext = anchor
    rafId = requestAnimationFrame(() => {
      rafId = null
      relayout({ anchor: anchorOnNext })
      anchorOnNext = false
    })
  }

  // ── 容器观察：仅宽度变化触发重排（避免"自己改高度 → 自己触发"的反馈循环）──
  let containerObserver: ResizeObserver | null = null

  function setupContainerObserver(el: HTMLElement) {
    containerObserver?.disconnect()
    containerObserver = new ResizeObserver((entries) => {
      const w = entries[0]?.contentRect.width ?? 0
      if (Math.abs(w - lastContainerWidth) > 0.5) {
        lastContainerWidth = w
        scheduleRelayout(false)
      }
    })
    containerObserver.observe(el)
  }

  /** 追加分页数据：数据已并入 items，这里做一次带锚定的全量重排（重新平衡各列，防首帧叠位/列底空缺） */
  function appendNewItems(_newItems: WaterfallItem[]) {
    relayout({ anchor: true, full: true })
  }

  /** 卡片尺寸变化（图片加载/失败/字体变化等）：合并到下一帧做带锚定的增量重排 */
  function onCardResized(_id: string | number) {
    scheduleRelayout(true)
  }

  /** 恢复 keep-alive 缓存的位置/高度，并重建内部列状态供后续增量布局使用 */
  function restore(restored: Map<string | number, Partial<Position>>, height: number) {
    if (restored.size === 0) return
    const next = new Map<string | number, Position>()
    let maxCol = 0
    let knownW = 0
    for (const [id, p] of restored) {
      const x = p.x ?? 0
      const y = p.y ?? 0
      const w = p.w ?? colW
      if (w > 0) knownW = w
      const col = p.col ?? (w > 0 ? Math.max(0, Math.round(x / (w + GAP))) : 0)
      next.set(id, { x, y, w, h: p.h ?? 0, col })
      maxCol = Math.max(maxCol, col)
    }
    positions.value = next
    containerHeight.value = height
    isLayoutReady.value = true
    cols = maxCol + 1
    if (knownW > 0) colW = knownW
  }

  /** 清空布局状态（列表清空 / 重置时调用） */
  function reset() {
    positions.value.clear()
    containerHeight.value = 0
    isLayoutReady.value = false
    cols = 0
    colW = 0
    lastContainerWidth = 0
    rebalanceScheduled = false
  }

  // 容器可能在 v-else 上，初始不存在，watch 等它出现后挂观察器并布局
  watch(
    containerRef,
    (el) => {
      if (el) {
        lastContainerWidth = el.clientWidth
        setupContainerObserver(el)
        nextTick(() => relayout())
      }
    },
    { immediate: true },
  )

  onUnmounted(() => {
    containerObserver?.disconnect()
    if (rafId !== null) cancelAnimationFrame(rafId)
  })

  return {
    positions,
    containerHeight,
    isLayoutReady,
    relayout,
    appendNewItems,
    onCardResized,
    restore,
    reset,
  }
}

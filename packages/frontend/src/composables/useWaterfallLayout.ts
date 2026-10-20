import { nextTick, onActivated, onDeactivated, onUnmounted, ref, watch, type Ref } from 'vue'

interface WaterfallItem {
  id: string | number
  [key: string]: unknown
}

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

export const MAX_COLUMNS = 6
export const MIN_COLUMN_WIDTH = 240
const MOBILE_BREAKPOINT = 640

export function computeColumnCount(width: number, gap = GAP) {
  if (width < MOBILE_BREAKPOINT) return 2
  const fit = Math.floor((width + gap) / (MIN_COLUMN_WIDTH + gap))
  return Math.max(2, Math.min(MAX_COLUMNS, fit))
}

const IMBALANCE_THRESHOLD = 1000

export function useWaterfallLayout(
  containerRef: Ref<HTMLElement | null>,
  items: Ref<WaterfallItem[]>,
) {
  const positions = ref<Map<string | number, Position>>(new Map())
  const containerHeight = ref(0)
  const isLayoutReady = ref(false)

  let cols = 0
  let colW = 0
  let lastContainerWidth = 0
  let rebalanceScheduled = false
  let active = true

  function getColWidth(el: HTMLElement, cols: number) {
    return (el.clientWidth - GAP * (cols - 1)) / cols
  }

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

  function relayout(opts: { anchor?: boolean; full?: boolean } = {}) {
    if (!active) return
    const el = containerRef.value
    if (!el || items.value.length === 0) {
      if (positions.value.size > 0 || containerHeight.value !== 0) reset()
      return
    }

    const heights = measureHeights(el)
    const prevCols = cols
    const nextCols = computeColumnCount(el.clientWidth)
    const nextColW = getColWidth(el, nextCols)
    const colsChanged = nextCols !== prevCols || Math.abs(nextColW - colW) > 0.5
    cols = nextCols
    colW = nextColW

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

  let rafId: number | null = null
  let anchorOnNext = false

  function scheduleRelayout(anchor = false) {
    if (!active) return
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

  function appendNewItems(_newItems: WaterfallItem[]) {
    relayout({ anchor: true, full: true })
  }

  function onCardResized(_id: string | number) {
    scheduleRelayout(true)
  }

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

  function reset() {
    positions.value.clear()
    containerHeight.value = 0
    isLayoutReady.value = false
    cols = 0
    colW = 0
    lastContainerWidth = 0
    rebalanceScheduled = false
  }

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
  onDeactivated(() => {
    active = false
    containerObserver?.disconnect()
    if (rafId !== null) cancelAnimationFrame(rafId)
    rafId = null
    anchorOnNext = false
  })
  onActivated(() => {
    active = true
    const el = containerRef.value
    if (el) setupContainerObserver(el)
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

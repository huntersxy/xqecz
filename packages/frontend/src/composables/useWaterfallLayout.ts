import { ref, watch, onMounted, onUnmounted, nextTick, type Ref } from 'vue'

export interface WaterfallItem {
  id: string | number
  [key: string]: unknown
}

export function useWaterfallLayout(
  containerRef: Ref<HTMLElement | null>,
  items: Ref<WaterfallItem[]>,
) {
  const gap = 10
  const positions = ref<Map<string | number, { x: number; y: number; w: number }>>(new Map())
  const containerHeight = ref(0)
  const isLayoutReady = ref(false)

  function getColumnCount(width: number) {
    if (width >= 1400) return 5
    if (width >= 1024) return 4
    if (width >= 640) return 3
    return 2
  }

  function getColWidth(el: HTMLElement, cols: number) {
    return (el.clientWidth - gap * (cols - 1)) / cols
  }

  function relayout() {
    const el = containerRef.value
    if (!el) return

    const cols = getColumnCount(el.clientWidth)
    const colW = getColWidth(el, cols)
    const colHeights = Array.from({ length: cols }, () => 0)
    const newPositions = new Map<string | number, { x: number; y: number; w: number }>()

    for (const item of items.value) {
      let minIdx = 0
      for (let i = 1; i < cols; i++) {
        if (colHeights[i] < colHeights[minIdx]) minIdx = i
      }

      const x = minIdx * (colW + gap)
      const y = colHeights[minIdx]

      const cardEl = el.querySelector(`[data-wf-id="${item.id}"]`) as HTMLElement | null
      const h = cardEl ? cardEl.offsetHeight : 0

      newPositions.set(item.id, { x, y, w: colW })
      colHeights[minIdx] = y + h + gap
    }

    positions.value = newPositions
    containerHeight.value = Math.max(...colHeights, 0)
    isLayoutReady.value = true

    // 宽度变化后图片高度会变，等图片加载完再算一次
    const imgs = el.querySelectorAll('img')
    let pending = 0
    const onDone = () => {
      if (--pending <= 0) relayout()
    }
    for (const img of imgs) {
      if (!img.complete) {
        pending++
        img.addEventListener('load', onDone, { once: true })
        img.addEventListener('error', onDone, { once: true })
      }
    }
  }

  function appendNewItems(newItems: WaterfallItem[]) {
    const el = containerRef.value
    if (!el) return

    const cols = getColumnCount(el.clientWidth)
    const colW = getColWidth(el, cols)

    const colHeights = Array.from({ length: cols }, () => 0)
    for (const [id, pos] of positions.value.entries()) {
      const cardEl = el.querySelector(`[data-wf-id="${id}"]`) as HTMLElement | null
      const h = cardEl ? cardEl.offsetHeight : 0
      const colIdx = Math.round(pos.x / (colW + gap))
      if (colIdx >= 0 && colIdx < cols) {
        colHeights[colIdx] = Math.max(colHeights[colIdx], pos.y + h + gap)
      }
    }

    for (const item of newItems) {
      if (positions.value.has(item.id)) continue

      let minIdx = 0
      for (let i = 1; i < cols; i++) {
        if (colHeights[i] < colHeights[minIdx]) minIdx = i
      }

      const x = minIdx * (colW + gap)
      const y = colHeights[minIdx]
      positions.value.set(item.id, { x, y, w: colW })
      colHeights[minIdx] = y + gap
    }

    containerHeight.value = Math.max(...colHeights, 0)
  }

  function onImageLoaded(_id: string | number) {
    nextTick(() => relayout())
  }

  let resizeRaf: number | null = null
  let resizeObserver: ResizeObserver | null = null

  function scheduleRelayout() {
    if (resizeRaf) cancelAnimationFrame(resizeRaf)
    resizeRaf = requestAnimationFrame(() => {
      relayout()
      resizeRaf = null
    })
  }

  function setupResizeObserver(el: HTMLElement) {
    resizeObserver?.disconnect()
    resizeObserver = new ResizeObserver(scheduleRelayout)
    resizeObserver.observe(el)
  }

  // 容器可能在 v-else 上，初始时不存在，需要 watch 等它出现
  watch(containerRef, (el) => {
    if (el) {
      setupResizeObserver(el)
      nextTick(() => relayout())
    }
  }, { immediate: true })

  onMounted(() => {
    // 如果 watch immediate 已经挂上就不需要再处理
    if (containerRef.value && !resizeObserver) {
      setupResizeObserver(containerRef.value)
    }
  })

  onUnmounted(() => {
    resizeObserver?.disconnect()
    if (resizeRaf) cancelAnimationFrame(resizeRaf)
  })

  return {
    positions,
    containerHeight,
    isLayoutReady,
    relayout,
    appendNewItems,
    onImageLoaded,
  }
}

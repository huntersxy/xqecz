import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

export function useStudioDock(composer: Ref<HTMLElement | undefined>) {
  const height = ref(0)
  const viewportHeight = ref(0)
  const bottom = ref(0)
  let observer: ResizeObserver | undefined
  let viewport: VisualViewport | null = null
  let frame = 0
  function measure() {
    frame = 0
    height.value = Math.ceil(composer.value?.getBoundingClientRect().height ?? 0)
    const visible = viewport?.scale === 1 ? viewport : null
    viewportHeight.value = visible?.height ?? window.innerHeight
    bottom.value = visible
      ? Math.max(0, window.innerHeight - visible.height - visible.offsetTop)
      : 0
  }
  function schedule() {
    if (!frame) frame = requestAnimationFrame(measure)
  }
  onMounted(() => {
    viewport = window.visualViewport
    observer = new ResizeObserver(schedule)
    if (composer.value) observer.observe(composer.value)
    window.addEventListener('resize', schedule)
    viewport?.addEventListener('resize', schedule)
    viewport?.addEventListener('scroll', schedule)
    measure()
  })
  onBeforeUnmount(() => {
    observer?.disconnect()
    window.removeEventListener('resize', schedule)
    viewport?.removeEventListener('resize', schedule)
    viewport?.removeEventListener('scroll', schedule)
    if (frame) cancelAnimationFrame(frame)
  })
  return computed(() => ({
    '--studio-dock-height': `${height.value}px`,
    '--studio-visible-height': viewportHeight.value ? `${viewportHeight.value}px` : '100dvh',
    '--studio-dock-bottom': `${bottom.value}px`,
  }))
}

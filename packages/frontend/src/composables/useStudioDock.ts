import { computed, onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

/** The phone dock owns its measured footprint and follows the visible viewport.
 * Keep the reservation in sync with references/settings; keyboard resizing must
 * never hide the submit row or strand the last task underneath the fixed panel. */
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
    // Pinch zoom retains the native fixed-position behavior; only a viewport at
    // its normal scale represents browser chrome or the on-screen keyboard.
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

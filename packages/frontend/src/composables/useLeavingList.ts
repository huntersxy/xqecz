import { onBeforeUnmount, ref } from 'vue'
import { prefersReducedMotion } from '@/utils'

const LEAVE_MS = 280

export function useLeavingList() {
  const leavingIds = ref<number[]>([])
  const timers = new Set<ReturnType<typeof setTimeout>>()
  let disposed = false

  function isLeaving(id: number) {
    return leavingIds.value.includes(id)
  }

  function scheduleLeave(id: number, refresh: () => unknown) {
    if (disposed) return
    if (prefersReducedMotion()) {
      void refresh()
      return
    }
    if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
    timers.add(
      setTimeout(() => {
        void (async () => {
          try {
            await refresh()
          } catch {
          } finally {
            leavingIds.value = leavingIds.value.filter(item => item !== id)
          }
        })()
      }, LEAVE_MS),
    )
  }

  onBeforeUnmount(() => {
    disposed = true
    timers.forEach(timer => clearTimeout(timer))
    timers.clear()
  })

  return { isLeaving, scheduleLeave }
}

import { onScopeDispose, ref, toValue, watch } from 'vue'
import type { MaybeRefOrGetter, Ref } from 'vue'
import { prefersReducedMotion } from '@/utils'

const MAX_DURATION = 600

export interface CountUpOptions {
  duration?: number
}

export function useCountUp(
  source: MaybeRefOrGetter<number>,
  options: CountUpOptions = {},
): Ref<number> {
  const duration = Math.min(Math.max(options.duration ?? MAX_DURATION, 0), MAX_DURATION)
  const value = ref(0)
  let frame = 0
  let from = 0
  let to = 0
  let startedAt = 0

  function cancel() {
    if (frame) cancelAnimationFrame(frame)
    frame = 0
  }

  function easeOut(progress: number) {
    return 1 - Math.pow(1 - progress, 3)
  }

  function tick(now: number) {
    frame = 0
    const progress = Math.min(1, (now - startedAt) / duration)
    if (progress >= 1) {
      value.value = to
      return
    }
    value.value = Math.round(from + (to - from) * easeOut(progress))
    frame = requestAnimationFrame(tick)
  }

  function animate(target: number) {
    cancel()
    if (prefersReducedMotion() || duration === 0 || value.value === target) {
      value.value = target
      return
    }
    from = value.value
    to = target
    startedAt = performance.now()
    frame = requestAnimationFrame(tick)
  }

  watch(
    () => {
      const target = toValue(source)
      return Number.isFinite(target) ? target : 0
    },
    animate,
    { immediate: true },
  )

  onScopeDispose(cancel)

  return value
}

import { onScopeDispose, ref, toValue, watch } from 'vue'
import type { MaybeRefOrGetter, Ref } from 'vue'
import { prefersReducedMotion } from '@/utils'

/** §8.4：单个动画 ≤480ms，M6 的 KPI 滚动是唯一例外，上限 600ms。 */
const MAX_DURATION = 600

export interface CountUpOptions {
  /** 动画时长（ms）；默认 600，超出 §8.4 预算时截断到 600。 */
  duration?: number
}

/**
 * KPI 数字滚动（M6）：把数值从「当前显示值」缓动到目标值，返回整数 ref。
 *
 * - 首次从 0 滚到目标；数据刷新时从旧值过渡到新值，不从 0 重来（M6）。
 * - 系统开启「减少动态效果」时直接给终值，不创建 rAF；rAF 不受 main.css 的
 *   CSS 媒体查询约束，必须在此自判（§8.4）。
 * - 作用域销毁（组件卸载）时取消未完成的帧，避免回调写已卸载组件。
 * - 用 rAF 回调时间戳而非 setTimeout，掉帧时按真实时间推进，不会拖长动画。
 *
 * KPI 计数均为整数，这里统一四舍五入，调用方可直接交给 Intl.NumberFormat。
 */
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

  // ease-out cubic：与 --adm-ease-out 的减速观感一致，且终点导数为 0，不会猛停。
  function easeOut(progress: number) {
    return 1 - Math.pow(1 - progress, 3)
  }

  function tick(now: number) {
    frame = 0
    const progress = Math.min(1, (now - startedAt) / duration)
    if (progress >= 1) {
      // 精确落位，避免浮点误差让终值差 1
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
    // 起点取「当前显示值」而不是上次的目标值，快速连点/刷新时不会跳变
    from = value.value
    to = target
    startedAt = performance.now()
    frame = requestAnimationFrame(tick)
  }

  watch(
    () => {
      const target = toValue(source)
      // 数据未就绪（NaN/Infinity）时按 0 处理，避免把 NaN 写进 DOM
      return Number.isFinite(target) ? target : 0
    },
    animate,
    { immediate: true },
  )

  onScopeDispose(cancel)

  return value
}

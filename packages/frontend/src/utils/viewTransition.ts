import { nextTick } from 'vue'
import { prefersReducedMotion } from './index'

export interface RevealPoint {
  x: number
  y: number
}

/** 圆形揭示的时长。比 UA 默认的交叉淡出略长，因为要铺满整屏而不是换一张脸。 */
const REVEAL_MS = 420

/**
 * 圆形揭示所需的最小半径：圆心到视口最远角的距离。
 * 比它小，动画收尾时视口的角上还留着旧状态，看着像「没铺开」。
 */
export function revealRadius(x: number, y: number, width: number, height: number): number {
  return Math.hypot(Math.max(x, width - x), Math.max(y, height - y))
}

function viewportCenter(): RevealPoint {
  return { x: (globalThis.innerWidth || 0) / 2, y: (globalThis.innerHeight || 0) / 2 }
}

/**
 * 动画圆心取指针位置。键盘触发（Enter/Space 产生的 click 坐标恒为 0,0）时退回
 * 触发元素的中心，再退回视口中心——不退回的话圆是从屏幕左上角长出来的。
 */
export function revealOriginFrom(event: { clientX?: number; clientY?: number }, el?: Element | null): RevealPoint {
  const x = event.clientX ?? 0
  const y = event.clientY ?? 0
  if (x || y) return { x, y }
  const rect = el?.getBoundingClientRect?.()
  if (rect && (rect.width || rect.height)) {
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  }
  return viewportCenter()
}

/**
 * 把一次「整页观感同时改变」的更新包进 View Transition，新状态从给定圆心圆形铺开。
 * 目前只有日/夜切换用它：主题翻转改的是 body 上的属性，全站几十处颜色一起变，
 * 没有过渡时是「啪」地翻黑白，眼睛要重新找内容。
 *
 * 三层降级，任一不满足就退回改动前的瞬时切换（功能不受影响）：
 *  1. 浏览器没有 `document.startViewTransition`；
 *  2. 系统要求减少动效。注意这一条**只能在这里判**：main.css 里压时长的媒体查询用
 *     的是 `*`，而 `::view-transition-*` 是挂在 UA 快照树上的伪元素、不在本文档的
 *     DOM 里，通配选择器匹配不到它，压不住 UA 默认的那次交叉淡出；
 *  3. 驱动动画本身失败（拿不到 ready、元素不支持伪元素动画）——catch 掉不上抛，
 *     绝不让「切主题」这件事因为动画而失败。
 */
export function withRevealTransition(update: () => void | Promise<void>, origin?: RevealPoint): void {
  const doc = globalThis.document
  const start = doc?.startViewTransition?.bind(doc)
  if (!start || prefersReducedMotion()) {
    void update()
    return
  }

  const transition = start(async () => {
    await update()
    // update 走的是 Vue 响应式（主题的 applyMode 挂在 watch 上，默认 flush: 'pre'，
    // 改 DOM 要等一次调度微任务）。回调一返回就拍新快照会拍到旧状态，圆形揭示
    // 于是「铺了个寂寞」。nextTick 等的正是这次 flush。
    await nextTick()
  })

  const { x, y } = origin ?? viewportCenter()
  transition.ready
    .then(() => {
      const radius = revealRadius(x, y, globalThis.innerWidth || 0, globalThis.innerHeight || 0)
      const animation = doc.documentElement.animate(
        {
          clipPath: [
            `circle(0px at ${x}px ${y}px)`,
            `circle(${radius}px at ${x}px ${y}px)`,
          ],
        },
        {
          duration: REVEAL_MS,
          easing: 'ease-in-out',
          // 结束后释放快照效果；forwards/both 会让过渡层继续覆盖页面、拦截点击。
          fill: 'backwards',
          pseudoElement: '::view-transition-new(root)',
        },
      )
      void animation.finished.then(
        () => animation.cancel(),
        () => animation.cancel(),
      )
    })
    // ready 被拒（页面切到后台、同页只允许一个过渡等在途）时，DOM 更新在回调里已经
    // 落完了，只是没有动画。什么都不用做，静默降级成瞬时切换。
    .catch(() => {})
}

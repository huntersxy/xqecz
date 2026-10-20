import { nextTick } from 'vue'
import { prefersReducedMotion } from './index'

export interface RevealPoint {
  x: number
  y: number
}

const REVEAL_MS = 420

export function revealRadius(x: number, y: number, width: number, height: number): number {
  return Math.hypot(Math.max(x, width - x), Math.max(y, height - y))
}

function viewportCenter(): RevealPoint {
  return { x: (globalThis.innerWidth || 0) / 2, y: (globalThis.innerHeight || 0) / 2 }
}

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

export function withRevealTransition(update: () => void | Promise<void>, origin?: RevealPoint): void {
  const doc = globalThis.document
  const start = doc?.startViewTransition?.bind(doc)
  if (!start || prefersReducedMotion()) {
    void update()
    return
  }

  let transition: ViewTransition
  try {
    transition = start(async () => {
      await update()
      await nextTick()
    })
  } catch {
    void update()
    return
  }

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
          fill: 'backwards',
          pseudoElement: '::view-transition-new(root)',
        },
      )
      void animation.finished.then(
        () => animation.cancel(),
        () => animation.cancel(),
      )
    })
    .catch(() => {})
}

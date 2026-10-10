import { describe, it, expect, vi, afterEach } from 'vitest'
import { nextTick, ref, watch } from 'vue'
import { revealOriginFrom, revealRadius, withRevealTransition } from '@/utils/viewTransition'

/**
 * jsdom 既不实现 startViewTransition 也不实现 Element.animate，
 * 两个都从外部按用例定义；过渡回调通过 holder 取回，避免被 TS 的
 * 控制流收窄成「永远是 null」。
 */
type TransitionInit = (fn: () => Promise<void>) => { ready: Promise<unknown> }

function stubStartViewTransition(init: TransitionInit) {
  Object.defineProperty(document, 'startViewTransition', { value: init, configurable: true })
}

function stubAnimate(impl?: () => Animation) {
  const animate = vi.fn(
    impl ?? (() => ({ finished: Promise.resolve(), cancel: vi.fn() }) as unknown as Animation),
  )
  Object.defineProperty(document.documentElement, 'animate', { value: animate, configurable: true })
  return animate
}

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  delete (document as { startViewTransition?: unknown }).startViewTransition
  delete (document.documentElement as unknown as { animate?: unknown }).animate
})

describe('revealRadius', () => {
  it('圆心在正中时只需半条对角线', () => {
    expect(revealRadius(50, 50, 100, 100)).toBeCloseTo(Math.hypot(50, 50))
  })

  it('圆心贴着角上要铺满整条对角线', () => {
    expect(revealRadius(0, 0, 100, 100)).toBeCloseTo(Math.hypot(100, 100))
  })

  it('圆心落在视口外也按最远角算，不会算出偏小的半径留旧图', () => {
    expect(revealRadius(-20, 60, 100, 100)).toBeCloseTo(Math.hypot(120, 60))
  })
})

describe('revealOriginFrom', () => {
  it('鼠标点击取指针位置', () => {
    expect(revealOriginFrom({ clientX: 320, clientY: 44 })).toEqual({ x: 320, y: 44 })
  })

  it('键盘触发（click 坐标恒为 0,0）退回触发元素中心', () => {
    const el = {
      getBoundingClientRect: () => ({ left: 10, top: 20, width: 100, height: 40 }),
    } as unknown as Element
    expect(revealOriginFrom({ clientX: 0, clientY: 0 }, el)).toEqual({ x: 60, y: 40 })
  })

  it('连元素都没有时退回视口中心，而不是左上角', () => {
    vi.stubGlobal('innerWidth', 1200)
    vi.stubGlobal('innerHeight', 800)
    expect(revealOriginFrom({ clientX: 0, clientY: 0 })).toEqual({ x: 600, y: 400 })
  })
})

describe('withRevealTransition', () => {
  it('浏览器不支持 startViewTransition 时直接把更新执行掉', () => {
    const update = vi.fn()
    withRevealTransition(update)
    expect(update).toHaveBeenCalledTimes(1)
  })

  it('系统要求减少动效时不发起过渡（UA 的淡出压不住，只能从源头不发起）', () => {
    vi.stubGlobal('matchMedia', (q: string) => ({ matches: q.includes('reduce') }))
    const start = vi.fn()
    Object.defineProperty(document, 'startViewTransition', { value: start, configurable: true })
    const update = vi.fn()

    withRevealTransition(update)

    expect(start).not.toHaveBeenCalled()
    expect(update).toHaveBeenCalledTimes(1)
  })

  it('过渡回调返回时 DOM 必须已改完，动画排在 new(root) 伪元素上', async () => {
    const holder: { run?: () => Promise<void> } = {}
    stubStartViewTransition((fn) => {
      holder.run = fn
      return { ready: Promise.resolve() }
    })
    const animate = stubAnimate()

    // 与 theme store 同构：更新走响应式，改 DOM 落在调度微任务里
    const mode = ref('light')
    // applied 代表「此刻 DOM 上的主题」，watcher 只在变化时写，初值即改动前的状态
    let applied = 'light'
    watch(mode, (v) => { applied = v }, { flush: 'pre' })

    withRevealTransition(() => { mode.value = 'dark' }, { x: 10, y: 20 })
    expect(applied).toBe('light')

    await holder.run?.()
    expect(applied).toBe('dark')

    await nextTick()
    expect(animate).toHaveBeenCalledTimes(1)
    const [keyframes, options] = animate.mock.calls[0] as [
      { clipPath: string[] },
      KeyframeAnimationOptions,
    ]
    expect(options.pseudoElement).toBe('::view-transition-new(root)')
    expect(keyframes.clipPath[0]).toBe('circle(0px at 10px 20px)')
    expect(keyframes.clipPath[1]).toMatch(/^circle\(\d+(\.\d+)?px at 10px 20px\)$/)
  })

  it('揭示动画结束后取消动画，释放过渡快照', async () => {
    const holder: { run?: () => Promise<void> } = {}
    stubStartViewTransition((fn) => {
      holder.run = fn
      return { ready: Promise.resolve() }
    })
    const cancel = vi.fn()
    stubAnimate(() => ({ finished: Promise.resolve(), cancel }) as unknown as Animation)

    withRevealTransition(vi.fn(), { x: 10, y: 20 })
    await holder.run?.()
    await nextTick()
    await Promise.resolve()

    expect(cancel).toHaveBeenCalledTimes(1)
  })

  it('ready 被拒（页面在后台）时静默降级，不影响更新', async () => {
    const holder: { run?: () => Promise<void> } = {}
    stubStartViewTransition((fn) => {
      holder.run = fn
      return { ready: Promise.reject(new Error('document hidden')) }
    })
    const animate = stubAnimate()
    const update = vi.fn()

    withRevealTransition(update)
    await holder.run?.()

    expect(update).toHaveBeenCalledTimes(1)
    expect(animate).not.toHaveBeenCalled()
  })

  it('排动画本身抛错也不冒泡（切主题不能因为动画失败而失败）', async () => {
    const holder: { run?: () => Promise<void> } = {}
    stubStartViewTransition((fn) => {
      holder.run = fn
      return { ready: Promise.resolve() }
    })
    stubAnimate(() => {
      throw new Error('pseudo element animation unsupported')
    })
    const update = vi.fn()

    withRevealTransition(update)
    await holder.run?.()
    await nextTick()

    expect(update).toHaveBeenCalledTimes(1)
  })
})

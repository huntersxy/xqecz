import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import type { Ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useCountUp } from '../useCountUp'

interface RafHarness {
  request: ReturnType<typeof vi.fn>
  cancel: ReturnType<typeof vi.fn>
  callbacks: Map<number, FrameRequestCallback>
  runFrame: (time: number) => void
}

function installRaf(): RafHarness {
  const callbacks = new Map<number, FrameRequestCallback>()
  let nextId = 1
  const request = vi.fn((callback: FrameRequestCallback) => {
    const id = nextId++
    callbacks.set(id, callback)
    return id
  })
  const cancel = vi.fn((id: number) => {
    callbacks.delete(id)
  })
  vi.stubGlobal('requestAnimationFrame', request)
  vi.stubGlobal('cancelAnimationFrame', cancel)
  return {
    request,
    cancel,
    callbacks,
    runFrame(time: number) {
      const pending = [...callbacks.entries()]
      callbacks.clear()
      for (const [, callback] of pending) callback(time)
    },
  }
}

function mountCountUp(source: Ref<number>, options?: { duration?: number }) {
  let counted!: Ref<number>
  const wrapper = mount(
    defineComponent({
      setup() {
        counted = useCountUp(source, options)
        return () => h('span', String(counted.value))
      },
    }),
  )
  return { wrapper, counted }
}

function useFakeTime() {
  let now = 0
  const spy = vi.spyOn(performance, 'now').mockImplementation(() => now)
  return {
    get now() {
      return now
    },
    set now(value: number) {
      now = value
    },
    restore: () => spy.mockRestore(),
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('useCountUp（M6 KPI 数字滚动）', () => {
  it('从 0 用 ease-out 滚到目标值，结束时精确落位且无残留帧', () => {
    const raf = installRaf()
    const time = useFakeTime()
    const source = ref(120)
    const { counted } = mountCountUp(source)

    expect(counted.value).toBe(0)
    expect(raf.request).toHaveBeenCalledTimes(1)

    raf.runFrame(0)
    expect(counted.value).toBe(0)

    time.now = 300
    raf.runFrame(300)
    expect(counted.value).toBe(105)

    time.now = 600
    raf.runFrame(600)
    expect(counted.value).toBe(120)
    expect(raf.callbacks.size).toBe(0)
  })

  it('刷新时从当前显示值过渡到新值，不从 0 重来', async () => {
    const raf = installRaf()
    const time = useFakeTime()
    const source = ref(100)
    const { counted } = mountCountUp(source)

    raf.runFrame(0)
    raf.runFrame(600)
    expect(counted.value).toBe(100)

    time.now = 600
    source.value = 300
    await nextTick()

    raf.runFrame(600)
    expect(counted.value).toBe(100)

    raf.runFrame(900)
    expect(counted.value).toBe(275)

    raf.runFrame(1200)
    expect(counted.value).toBe(300)
  })

  it('系统要求减少动态效果时直接给终值，不创建 rAF', async () => {
    const raf = installRaf()
    useFakeTime()
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(prefers-reduced-motion: reduce)',
    }))
    const source = ref(42)
    const { counted } = mountCountUp(source)

    expect(counted.value).toBe(42)
    expect(raf.request).not.toHaveBeenCalled()

    source.value = 7
    await nextTick()
    expect(counted.value).toBe(7)
    expect(raf.request).not.toHaveBeenCalled()
  })

  it('时长超过 600ms 预算时截断到 600ms', () => {
    const raf = installRaf()
    const time = useFakeTime()
    const { counted } = mountCountUp(ref(10), { duration: 5000 })

    raf.runFrame(time.now)
    expect(counted.value).toBe(0)

    raf.runFrame(600)
    expect(counted.value).toBe(10)
  })

  it('duration 为 0 时直接落位', () => {
    const raf = installRaf()
    useFakeTime()
    const { counted } = mountCountUp(ref(88), { duration: 0 })

    expect(counted.value).toBe(88)
    expect(raf.request).not.toHaveBeenCalled()
  })

  it('组件卸载时取消未完成的帧', () => {
    const raf = installRaf()
    useFakeTime()
    const { wrapper } = mountCountUp(ref(500))

    expect(raf.callbacks.size).toBe(1)
    wrapper.unmount()
    expect(raf.cancel).toHaveBeenCalledWith(1)
    expect(raf.callbacks.size).toBe(0)
  })

  it('目标值非有限数时按 0 处理，不把 NaN 写进 DOM', () => {
    const raf = installRaf()
    useFakeTime()
    const source = ref(Number.NaN)
    const { counted } = mountCountUp(source)

    expect(counted.value).toBe(0)
    expect(raf.request).not.toHaveBeenCalled()
  })
})

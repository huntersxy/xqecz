import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import { useStudioDock } from '../useStudioDock'

let resize: ResizeObserverCallback
const disconnect = vi.fn()
let measuredHeight = 280
let viewport: EventTarget & { height: number; offsetTop: number; scale: number }
let wrapper: ReturnType<typeof mount>
let style: ReturnType<typeof useStudioDock>

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  measuredHeight = 280
  viewport = Object.assign(new EventTarget(), { height: 844, offsetTop: 0, scale: 1 })
  vi.stubGlobal('innerHeight', 844)
  vi.stubGlobal('visualViewport', viewport)
  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(callback: ResizeObserverCallback) {
        resize = callback
      }
      observe() {}
      disconnect = disconnect
    },
  )
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
    () => ({ height: measuredHeight }) as DOMRect,
  )
  wrapper = mount(
    defineComponent({
      setup() {
        const composer = ref<HTMLElement>()
        style = useStudioDock(composer)
        return () => h('section', { ref: composer })
      },
    }),
  )
})
afterEach(() => {
  wrapper.unmount()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('phone composer footprint', () => {
  it('reserves actual panel height and updates it after references/settings or collapse', () => {
    expect(style.value['--studio-dock-height']).toBe('280px')
    measuredHeight = 420
    resize([], {} as ResizeObserver)
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-dock-height']).toBe('420px')
    measuredHeight = 108
    resize([], {} as ResizeObserver)
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-dock-height']).toBe('108px')
  })
  it('follows a keyboard-reduced viewport including browser pan, then restores on close', () => {
    viewport.height = 380
    viewport.offsetTop = 40
    viewport.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-visible-height']).toBe('380px')
    expect(style.value['--studio-dock-bottom']).toBe('424px')
    viewport.offsetTop = 80
    viewport.dispatchEvent(new Event('scroll'))
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-dock-bottom']).toBe('384px')
    viewport.height = 844
    viewport.offsetTop = 0
    viewport.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-dock-bottom']).toBe('0px')
  })
  it('leaves pinch zoom native and cancels queued observations and viewport listeners on unmount', () => {
    viewport.scale = 2
    viewport.height = 422
    viewport.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(20)
    expect(style.value['--studio-dock-bottom']).toBe('0px')
    expect(style.value['--studio-visible-height']).toBe('844px')
    measuredHeight = 500
    resize([], {} as ResizeObserver)
    wrapper.unmount()
    viewport.dispatchEvent(new Event('scroll'))
    window.dispatchEvent(new Event('resize'))
    vi.advanceTimersByTime(20)
    expect(disconnect).toHaveBeenCalledOnce()
    expect(style.value['--studio-dock-height']).toBe('280px')
  })
})

import { afterEach, expect, it, vi } from 'vitest'
import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useWaterfallLayout } from '../useWaterfallLayout'

afterEach(() => vi.unstubAllGlobals())

it('does not remeasure or change scroll while the cached homepage is inactive', async () => {
  const disconnect = vi.fn()
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect = disconnect })
  const frame = vi.fn(() => 1)
  vi.stubGlobal('requestAnimationFrame', frame)
  vi.stubGlobal('cancelAnimationFrame', vi.fn())
  const visible = ref(true)
  const width = vi.fn(() => 800)
  let layout!: ReturnType<typeof useWaterfallLayout>
  const page = defineComponent({
    setup() {
      const el = ref<HTMLElement | null>(null)
      layout = useWaterfallLayout(el, ref([{ id: 1 }]))
      return () => h('div', { ref: (node) => {
        if (node instanceof HTMLElement) {
          Object.defineProperty(node, 'clientWidth', { configurable: true, get: width })
          el.value = node
        }
      } }, [h('div', { 'data-wf-id': 1 })])
    },
  })
  const wrapper = mount(defineComponent({ setup: () => () => h(KeepAlive, null, () => visible.value ? h(page) : null) }))
  await nextTick()
  layout.relayout()
  const initial = layout.positions.value
  visible.value = false
  await nextTick()
  width.mockClear()
  frame.mockClear()
  layout.onCardResized(1)
  layout.relayout()
  expect(disconnect).toHaveBeenCalled()
  expect(width).not.toHaveBeenCalled()
  expect(frame).not.toHaveBeenCalled()
  expect(layout.positions.value).toBe(initial)
  visible.value = true
  await nextTick()
  layout.relayout()
  expect(width).toHaveBeenCalled()
  wrapper.unmount()
})

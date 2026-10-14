import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import StudioReferences from '../StudioReferences.vue'

const wrappers: ReturnType<typeof mount>[] = []
const frames = new Map<number, FrameRequestCallback>()
let nextFrame = 0
function rect(left: number, width: number) {
  return {
    left,
    right: left + width,
    top: 0,
    bottom: 100,
    width,
    height: 100,
    x: left,
    y: 0,
    toJSON: () => ({}),
  }
}
function render(width = 600) {
  const w = mount(StudioReferences, {
    props: { references: [7, 8, 9].map((asset_id) => ({ asset_id, note: '' })), disabled: false },
    global: { stubs: { CreativeImage: true } },
  })
  wrappers.push(w)
  const rail = w.get('.composer-references').element as HTMLElement
  Object.defineProperties(rail, { clientWidth: { value: width }, scrollWidth: { value: 540 } })
  let scroll = 0
  Object.defineProperty(rail, 'scrollLeft', {
    get: () => scroll,
    set: (value) => {
      scroll = Math.max(0, Math.min(240, value))
    },
  })
  rail.getBoundingClientRect = () => rect(0, width)
  w.findAll('.composer-reference').forEach((tile, index) => {
    tile.element.getBoundingClientRect = () => rect(index * 180 - scroll, 180)
  })
  return w
}
function pointer(element: Element, type: string, x: number, y = 40, pointerType = 'mouse') {
  const event = new MouseEvent(type, {
    bubbles: true,
    cancelable: true,
    button: 0,
    clientX: x,
    clientY: y,
  })
  Object.defineProperties(event, {
    pointerId: { value: 1 },
    isPrimary: { value: true },
    pointerType: { value: pointerType },
  })
  element.dispatchEvent(event)
}
beforeEach(() => {
  frames.clear()
  nextFrame = 0
  vi.spyOn(globalThis, 'requestAnimationFrame').mockImplementation((callback) => {
    frames.set(++nextFrame, callback)
    return nextFrame
  })
  vi.spyOn(globalThis, 'cancelAnimationFrame').mockImplementation((id) => {
    frames.delete(id)
  })
})
afterEach(() => {
  wrappers.splice(0).forEach((w) => w.unmount())
  vi.restoreAllMocks()
})

describe('studio reference sorting', () => {
  it.each(['mouse', 'touch'])(
    'moves across multiple references with %s without also opening a preview',
    async (kind) => {
      const w = render()
      const image = w.get('.reference-preview').element
      pointer(image, 'pointerdown', 40, 40, kind)
      pointer(image, 'pointermove', 450, 40, kind)
      await w.vm.$nextTick()
      expect(w.findAll('.is-sort-target')).toHaveLength(1)
      pointer(image, 'pointerup', 450, 40, kind)
      await w.get('.reference-preview').trigger('click')
      expect(w.emitted('reorder')).toEqual([[0, 2]])
      expect(w.emitted('preview')).toBeUndefined()
      expect(w.get('[role="status"]').text()).toContain('第3位')
      expect(frames.size).toBe(0)
    },
  )
  it('keeps a tap as a preview and cancels a drag dropped outside the rail', async () => {
    const w = render()
    const image = w.get('.reference-preview').element
    pointer(image, 'pointerdown', 40)
    pointer(image, 'pointerup', 40)
    await w.get('.reference-preview').trigger('click')
    expect(w.emitted('preview')).toEqual([[7]])
    pointer(image, 'pointerdown', 40)
    pointer(image, 'pointermove', 450, 150)
    pointer(image, 'pointerup', 450, 150)
    expect(w.emitted('reorder')).toBeUndefined()
    expect(frames.size).toBe(0)
  })
  it('auto-scrolls to a hidden destination and releases the frame loop after dropping', () => {
    const w = render(300)
    const handle = w.get('.reference-sort-handle').element
    pointer(handle, 'pointerdown', 40)
    pointer(handle, 'pointermove', 292)
    for (let i = 0; i < 20; i++) {
      const [id, callback] = frames.entries().next().value!
      frames.delete(id)
      callback(i * 16)
    }
    expect((w.get('.composer-references').element as HTMLElement).scrollLeft).toBe(160)
    pointer(handle, 'pointerup', 292)
    expect(w.emitted('reorder')).toEqual([[0, 2]])
    expect(frames.size).toBe(0)
  })
  it('cancels capture and scrolling when a submission locks the draft', async () => {
    const w = render(300)
    const handle = w.get('.reference-sort-handle').element as HTMLElement
    const release = vi.fn()
    Object.assign(handle, {
      setPointerCapture: vi.fn(),
      hasPointerCapture: () => true,
      releasePointerCapture: release,
    })
    pointer(handle, 'pointerdown', 40)
    pointer(handle, 'pointermove', 292)
    expect(frames.size).toBe(1)
    await w.setProps({ disabled: true })
    pointer(handle, 'pointerup', 292)
    expect(w.emitted('reorder')).toBeUndefined()
    expect(release).toHaveBeenCalledWith(1)
    expect(frames.size).toBe(0)
  })
  it('stops edge scrolling at the original extent even while the drag remains active', () => {
    const w = render(300)
    const handle = w.get('.reference-sort-handle').element
    pointer(handle, 'pointerdown', 40)
    pointer(handle, 'pointermove', 292)
    for (let i = 0; i < 100 && frames.size; i++) {
      const [id, callback] = frames.entries().next().value!
      frames.delete(id)
      callback(i * 16)
    }
    expect((w.get('.composer-references').element as HTMLElement).scrollLeft).toBe(240)
    expect(frames.size).toBe(0)
    pointer(handle, 'pointercancel', 292)
    expect(w.emitted('reorder')).toBeUndefined()
  })
  it('clears a cancelled gesture and keeps keyboard sorting available without arrow buttons', async () => {
    const w = render()
    const handle = w.get('.reference-sort-handle')
    pointer(handle.element, 'pointerdown', 40)
    pointer(handle.element, 'pointermove', 450)
    pointer(handle.element, 'pointercancel', 450)
    expect(w.emitted('reorder')).toBeUndefined()
    await handle.trigger('keydown', { key: 'End' })
    expect(w.emitted('reorder')).toEqual([[0, 2]])
    expect(w.find('[aria-label="下移图片1"]').exists()).toBe(false)
    expect(frames.size).toBe(0)
  })
})

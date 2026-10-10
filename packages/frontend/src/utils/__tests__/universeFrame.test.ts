import { describe, expect, it, vi } from 'vitest'
import { createUniverseFrame } from '../universeFrame'

describe('universe refresh scheduling', () => {
  it('coalesces interaction bursts and renders the next screen frame without a 30fps gate', () => {
    const callbacks = new Map<number, FrameRequestCallback>()
    let id = 0
    const request = vi.fn((cb: FrameRequestCallback) => { callbacks.set(++id, cb); return id })
    const cancel = vi.fn((id: number) => callbacks.delete(id))
    const render = vi.fn()
    const frames = createUniverseFrame(render, request, cancel)
    frames.setActive(true)
    for (let n=0;n<100;n++) frames.invalidate()
    expect(request).toHaveBeenCalledTimes(1)
    callbacks.get(1)!(16.7)
    expect(render).toHaveBeenCalledOnce()
    frames.invalidate(); callbacks.get(2)!(25)
    expect(render).toHaveBeenCalledTimes(2)
    expect(request).toHaveBeenCalledTimes(2)
    frames.dispose()
  })
  it('cancels pending work when hidden or disposed and can redraw after resuming', () => {
    const request = vi.fn((_callback: FrameRequestCallback) => 1), cancel = vi.fn(), render = vi.fn()
    const frames = createUniverseFrame(render, request, cancel)
    frames.invalidate(); expect(request).not.toHaveBeenCalled()
    frames.setActive(true); frames.setActive(false)
    expect(cancel).toHaveBeenCalledWith(1)
    frames.invalidate(); expect(request).toHaveBeenCalledOnce()
    frames.setActive(true); frames.dispose(); frames.invalidate()
    expect(request).toHaveBeenCalledTimes(2)
    const late = request.mock.calls[1]![0] as FrameRequestCallback
    late(100)
    expect(render).not.toHaveBeenCalled()
  })
})

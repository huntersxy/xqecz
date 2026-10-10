import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ArtworkUniverse from '../ArtworkUniverse.vue'
import { createUniverseScene } from '@/utils/universeScene'

const scene = vi.hoisted(() => ({ imageSource: vi.fn<(id: number) => string | undefined>(), update: vi.fn(), enter: vi.fn(), fit: vi.fn(), zoom: vi.fn(), focus: vi.fn(), project: vi.fn(() => ({ x: 200, y: 200, diameter: 40 })), setActive: vi.fn(), dispose: vi.fn() }))
vi.mock('@/utils/universeScene', () => ({ createUniverseScene: vi.fn(() => scene) }))
vi.mock('@/utils/artworkPlanet', () => ({ loadMontageImage: vi.fn(async () => null),  planetImageSource: vi.fn((url: string) => url) }))
const origin = { left: 900, top: 140, width: 180, height: 180, image: '/planet.png' }
const artworks = [{ id: 1, title: 'First planet', thumb: '/first.webp' }, { id: 2, title: 'Text planet', thumb: '' }]
let wrapper: ReturnType<typeof mount> | undefined
let motion = false
const animateDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'animate')
const animationsDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, 'getAnimations')

beforeEach(() => {
  vi.useFakeTimers()
  vi.clearAllMocks()
  motion = false
  scene.imageSource.mockReturnValue(undefined)
  vi.stubGlobal('matchMedia', vi.fn(() => ({ get matches() { return motion }, addEventListener: vi.fn(), removeEventListener: vi.fn() })))
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => setTimeout(() => cb(performance.now()), 16))
  vi.stubGlobal('cancelAnimationFrame', clearTimeout)
  vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
  Object.defineProperty(HTMLElement.prototype, 'animate', { configurable: true, value: vi.fn(() => ({ cancel: vi.fn(), pause: vi.fn(), play: vi.fn(), playState: 'running' })) })
  Object.defineProperty(HTMLElement.prototype, 'getAnimations', { configurable: true, value: vi.fn(() => []) })
  document.body.innerHTML = '<div id="app"><button id="previous">Previous</button></div>'
  document.getElementById('previous')!.focus()
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.useRealTimers()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  if (animateDescriptor) Object.defineProperty(HTMLElement.prototype, 'animate', animateDescriptor)
  else Reflect.deleteProperty(HTMLElement.prototype, 'animate')
  if (animationsDescriptor) Object.defineProperty(HTMLElement.prototype, 'getAnimations', animationsDescriptor)
  else Reflect.deleteProperty(HTMLElement.prototype, 'getAnimations')
  document.body.innerHTML = ''
})
async function start() {
  wrapper = mount(ArtworkUniverse, { props: { artworks, origin }, attachTo: document.body })
  await flushPromises()
}
async function advance(ms: number) { await vi.advanceTimersByTimeAsync(ms); await flushPromises() }
function phase() { return document.querySelector('.au-root')?.getAttribute('data-phase') }

describe('artwork universe lifecycle', () => {
  it('progresses from flying to unfolding, marquee and interactive map', async () => {
    await start()
    expect(phase()).toBe('flight')
    await advance(1000)
    expect(phase()).toBe('unfold')
    await advance(1600)
    expect(phase()).toBe('marquee')
    await advance(3500)
    expect(phase()).toBe('collapse')
    await advance(1600)
    expect(phase()).toBe('map')
    expect(scene.setActive).toHaveBeenCalledWith(true)
    expect(document.querySelector('.au-montage')).toBeNull()
  })
  it('contracts visible artwork into a circle before handing it to its matching planet', async () => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      return this.classList.contains('au-tile')
        ? { x: 20, y: 40, left: 20, top: 40, right: 220, bottom: 160, width: 200, height: 120, toJSON() {} }
        : { x: 0, y: 0, left: 0, top: 0, right: 1024, bottom: 768, width: 1024, height: 768, toJSON() {} }
    })
    await start()
    document.querySelector<HTMLButtonElement>('[aria-label="跳过展开动画"]')!.click()
    await flushPromises()
    expect(phase()).toBe('collapse')
    expect(document.querySelector('.au-map')?.classList.contains('is-previewing')).toBe(true)
    const clones = document.querySelectorAll('.au-collapse-tile')
    expect(clones.length).toBeGreaterThan(0)
    const frames = vi.mocked(HTMLElement.prototype.animate).mock.calls.find(([keyframes]) =>
      Array.isArray(keyframes) && keyframes.some(frame => frame.offset === .35),
    )![0] as Keyframe[]
    expect(frames[1]).toMatchObject({ offset: .35, clipPath: 'inset(0 round 50%)', opacity: 1 })
    expect(frames[1]!.transform).toBe('translate(0,0) scale(0.6,1)')
    expect(frames[2]).toMatchObject({ offset: .82, opacity: 1, transform: 'translate(80px,100px) scale(0.2,0.3333333333333333)' })
    expect(scene.project).toHaveBeenCalledWith(expect.objectContaining({ artwork: expect.objectContaining({ id: 1 }) }))
    await advance(1600)
    expect(phase()).toBe('map')
    expect(document.querySelector('.au-map')?.classList.contains('is-visible')).toBe(true)
    expect(document.querySelector('.au-collapse-tile')).toBeNull()
  })
  it('can close during opening without a later phase resurrecting the universe', async () => {
    await start()
    document.querySelector<HTMLButtonElement>('.au-close')!.click()
    await advance(1500)
    expect(wrapper!.emitted('closed')).toHaveLength(1)
    await advance(9000)
    expect(phase()).toBe('closing')
    expect(scene.setActive).not.toHaveBeenCalledWith(true)
  })
  it('skips spatial animation for reduced motion and restores focus and background access on disposal', async () => {
    motion = true
    await start()
    await advance(100)
    expect(phase()).toBe('map')
    expect(document.getElementById('app')!.inert).toBe(true)
    wrapper!.unmount(); wrapper = undefined
    expect(scene.dispose).toHaveBeenCalledOnce()
    expect(document.getElementById('app')!.inert).toBe(false)
    expect(document.activeElement?.id).toBe('previous')
  })
  it('can skip the montage and safely incorporate more works', async () => {
    await start()
    document.querySelector<HTMLButtonElement>('[aria-label="跳过展开动画"]')!.click()
    await advance(1600)
    expect(phase()).toBe('map')
    await wrapper!.setProps({ artworks: [...artworks, { id: 3, title: 'Next page', thumb: '' }] })
    expect(scene.update.mock.lastCall![0]).toHaveLength(3)
    await advance(10000)
    expect(phase()).toBe('map')
  })
  it('searches text-only planets and emits the selected content id', async () => {
    motion = true
    await start(); await advance(100)
    const input = document.querySelector<HTMLInputElement>('input')!
    input.value = 'Text'; input.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()
    document.querySelector<HTMLButtonElement>('.au-search-results button')!.click()
    await flushPromises()
    expect(scene.focus.mock.lastCall![0].artwork.id).toBe(2)
    document.querySelector<HTMLButtonElement>('.au-open')!.click()
    expect(wrapper!.emitted('select')).toEqual([[2]])
  })
  it('reuses the loaded fallback URL and recovers a preview when loading completes', async () => {
    motion = true
    await start(); await advance(100)
    const input = document.querySelector<HTMLInputElement>('input')!
    input.value = 'First'; input.dispatchEvent(new Event('input', { bubbles: true }))
    await flushPromises()
    document.querySelector<HTMLButtonElement>('.au-search-results button')!.click()
    await flushPromises()
    document.querySelector<HTMLImageElement>('.au-selection-image')!.dispatchEvent(new Event('error'))
    await flushPromises()
    expect(document.querySelector('.au-selection-image')).toBeNull()
    scene.imageSource.mockReturnValue('/__planet-media/thumbs/first.webp')
    vi.mocked(createUniverseScene).mock.lastCall![2]!({ total: 1, ready: 1, failed: 0, pending: 0 })
    await flushPromises()
    expect(document.querySelector<HTMLImageElement>('.au-selection-image')!.getAttribute('src')).toBe('/__planet-media/thumbs/first.webp')
  })
})

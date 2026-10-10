import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ArtworkPlanet from '../ArtworkPlanet.vue'

const rendering = vi.hoisted(() => ({ render: vi.fn(), setAnimationLoop: vi.fn(), fillRect: vi.fn() }))
vi.mock('three', () => ({
  Scene: class { add() {} },
  OrthographicCamera: class { position = { z: 0 } },
  Vector3: class {},
  CanvasTexture: class { constructor(public image: HTMLCanvasElement) {} dispose() {} },
  MeshStandardMaterial: class { dispose() {} },
  SphereGeometry: class { dispose() {} },
  Mesh: class { geometry = { dispose() {} } },
  AmbientLight: class {},
  DirectionalLight: class { position = { set() {} } },
  SRGBColorSpace: 'srgb',
  RepeatWrapping: 1000,
  WebGLRenderer: class {
    capabilities = { getMaxAnisotropy: () => 1 }
    render = rendering.render
    setAnimationLoop = rendering.setAnimationLoop
    setPixelRatio() {}
    setSize() {}
    dispose() {}
  },
}))
vi.mock('three/addons/controls/OrbitControls.js', () => ({
  OrbitControls: class {
    addEventListener() {}
    update() {}
    dispose() {}
  },
}))
const artwork = vi.hoisted(() => ({ image: { naturalWidth: 100 }, paint: vi.fn() }))
vi.mock('@/utils/artworkPlanet', async importOriginal => ({
  ...await importOriginal<typeof import('@/utils/artworkPlanet')>(),
  loadPlanetImage: vi.fn(async (url: string) => url.startsWith('/thumbs/') ? artwork.image : null),
  paintPlanetTexture: artwork.paint,
}))
let wrapper: ReturnType<typeof mount> | undefined
let intersect: IntersectionObserverCallback

beforeEach(() => {
  vi.clearAllMocks()
  vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({ fillRect: rendering.fillRect } as unknown as CanvasRenderingContext2D)
  vi.stubGlobal('matchMedia', vi.fn((query: string) => ({ matches: query.includes('min-width'), addEventListener: vi.fn(), removeEventListener: vi.fn() })))
  vi.stubGlobal('IntersectionObserver', class {
    constructor(callback: IntersectionObserverCallback) { intersect = callback }
    observe() {}
    disconnect() {}
  })
})
afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})
async function start(artworks: { id: number; title: string; thumb: string }[]) {
  wrapper = mount(ArtworkPlanet, { props: { artworks } })
  intersect([{ isIntersecting: true }] as IntersectionObserverEntry[], {} as IntersectionObserver)
  await vi.dynamicImportSettled()
  await flushPromises()
}

describe('artwork planet initialization', () => {
  it('starts rendering and rotation even when the packaged decorative image fails', async () => {
    await start([])
    expect(wrapper!.find('.artwork-planet').classes()).toContain('is-ready')
    expect(wrapper!.find('[aria-label="暂停星球自转"]').exists()).toBe(true)
    expect(rendering.fillRect).toHaveBeenCalled()
    expect(rendering.setAnimationLoop).toHaveBeenCalledWith(expect.any(Function))
  })

  it('continues loading artwork after the decorative image fails', async () => {
    await start([{ id: 1, title: 'A', thumb: '/thumbs/a.webp' }, { id: 2, title: 'B', thumb: '/thumbs/b.webp' }])
    expect(artwork.paint).toHaveBeenCalled()
    expect(artwork.paint.mock.lastCall![1]).toHaveLength(12)
    expect(rendering.render).toHaveBeenCalled()
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'

afterEach(() => { vi.useRealTimers(); vi.unstubAllEnvs(); vi.unstubAllGlobals(); vi.resetModules() })

async function loader(mediaBase = '') {
  vi.stubEnv('VITE_MEDIA_BASE_URL', mediaBase)
  vi.stubEnv('DEV', true)
  vi.resetModules()
  return import('../artworkPlanet')
}

describe('planet image source', () => {
  it('does not start a request when the owner has already cancelled', async () => {
    const { loadPlanetImage } = await loader()
    const createImage = vi.fn()
    vi.stubGlobal('Image', createImage)
    const owner = new AbortController()
    owner.abort()
    expect(await loadPlanetImage('/thumbs/a.webp', owner.signal)).toBeNull()
    expect(createImage).not.toHaveBeenCalled()
  })

  it.each(['abort', 'timeout'])('settles pending artwork loads on %s and detaches late callbacks', async reason => {
    const { loadPlanetImage } = await loader()
    vi.useFakeTimers()
    const images: MockImage[] = []
    class MockImage {
      src = ''
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      constructor() { images.push(this) }
    }
    vi.stubGlobal('Image', MockImage)
    const owner = new AbortController()
    // 外部 URL 不走本站兜底，隔离单个请求的取消/超时契约。
    const pending = loadPlanetImage('https://external.example.invalid/a.webp', owner.signal)
    expect(images).toHaveLength(1)
    if (reason === 'abort') owner.abort()
    else await vi.advanceTimersByTimeAsync(4000)
    expect(await pending).toBeNull()
    expect(images[0]!.src).toBe('')
    expect(images[0]!.onload).toBeNull()
    expect(images[0]!.onerror).toBeNull()
    expect(vi.getTimerCount()).toBe(0)
    expect(images).toHaveLength(1)
  })

  it('resolves API media paths using the same configured base as normal cards', async () => {
    const { planetImageSource } = await loader('https://api39.xiey.work')
    expect(planetImageSource('/thumbs/abc_thumb.webp', 'http://localhost:5173', true)).toBe('/__planet-media/thumbs/abc_thumb.webp')
    expect(planetImageSource('/thumbs/abc_thumb.webp', 'https://xq.xiey.work', false)).toBe('https://api39.xiey.work/thumbs/abc_thumb.webp?texture_origin=https%3A%2F%2Fxq.xiey.work')
  })

  it('keeps same-origin files and packaged images unchanged', async () => {
    const { planetImageSource } = await loader()
    expect(planetImageSource('/thumbs/local.webp', 'http://localhost:5173', true)).toBe('/thumbs/local.webp')
    expect(planetImageSource('/src/assets/bg.webp', 'http://localhost:5173', true)).toBe('/src/assets/bg.webp')
    expect(planetImageSource('', 'http://localhost:5173', true)).toBe('')
  })

  it('keeps frontend build assets on the frontend when production media uses another host', async () => {
    const { planetImageSource } = await loader('https://api39.xiey.work')
    expect(planetImageSource('/assets/bg-C2y8ql2t.webp', 'https://xq.xiey.work', false)).toBe('/assets/bg-C2y8ql2t.webp')
    expect(planetImageSource('/src/assets/bg.webp', 'http://localhost:5173', true)).toBe('/src/assets/bg.webp')
  })

  it('never retries missing frontend assets against the media server', async () => {
    const { loadPlanetImage } = await loader('https://api39.xiey.work')
    const requested: string[] = []
    class MockImage {
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      set src(value: string) {
        if (!value) return
        requested.push(value)
        queueMicrotask(() => this.onerror?.())
      }
    }
    vi.stubGlobal('Image', MockImage)
    expect(await loadPlanetImage('/assets/missing.webp', new AbortController().signal)).toBeNull()
    expect(requested).toEqual(['/assets/missing.webp'])
  })

  it('separates CORS texture requests from ordinary image cache entries and preserves queries', async () => {
    const { planetImageSource } = await loader()
    const url = planetImageSource('https://api39.xiey.work/thumbs/a.webp?v=2', 'https://xq.xiey.work', false)
    expect(new URL(url).searchParams.get('v')).toBe('2')
    expect(new URL(url).searchParams.get('texture_origin')).toBe('https://xq.xiey.work')
    expect(planetImageSource(url, 'https://xq.xiey.work', false)).toBe(url)
    expect(planetImageSource('https://other.example/a.webp?signature=abc', 'http://localhost:5173', true)).toBe('https://other.example/a.webp?signature=abc')
  })

  it('falls back to the production media host through the canvas-safe development proxy', async () => {
    const { loadPlanetImage } = await loader()
    const requested: string[] = []
    class MockImage {
      crossOrigin = ''
      decoding = ''
      naturalWidth = 100
      onload: (() => void) | null = null
      onerror: (() => void) | null = null
      set src(value: string) {
        if (!value) return
        requested.push(value)
        queueMicrotask(() => value.startsWith('/__planet-media') ? this.onload?.() : this.onerror?.())
      }
    }
    vi.stubGlobal('Image', MockImage)
    const image = await loadPlanetImage('/thumbs/6c28799d8edb4a3327fcd0062ec695bf_thumb.webp', new AbortController().signal)
    expect(image).not.toBeNull()
    expect(requested).toEqual(['/thumbs/6c28799d8edb4a3327fcd0062ec695bf_thumb.webp', '/__planet-media/thumbs/6c28799d8edb4a3327fcd0062ec695bf_thumb.webp'])
  })
})

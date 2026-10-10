import { afterEach, describe, expect, it, vi } from 'vitest'

afterEach(() => { vi.unstubAllEnvs(); vi.unstubAllGlobals(); vi.resetModules() })

async function loader(mediaBase = '') {
  vi.stubEnv('VITE_MEDIA_BASE_URL', mediaBase)
  vi.stubEnv('DEV', true)
  vi.resetModules()
  return import('../artworkPlanet')
}

describe('planet image source', () => {
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

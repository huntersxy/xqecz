import { afterEach, describe, expect, it, vi } from 'vitest'
import { createUniverseImages } from '../universeImages'
import { loadUniverseImage } from '../artworkPlanet'
import type { UniverseNode } from '../artworkUniverse'

vi.mock('../artworkPlanet', () => ({ loadUniverseImage: vi.fn() }))
const image = {} as HTMLImageElement
const nodes = (count: number): UniverseNode[] => Array.from({ length: count }, (_, id) => ({ artwork: { id, title: 'Work ' + id, thumb: '/thumbs/' + id + '.webp' }, x: id, y: 0, radius: 100, hue: 20 }))
afterEach(() => vi.resetAllMocks())
async function settle() { await Promise.resolve(); await Promise.resolve(); await Promise.resolve() }

describe('full universe images', () => {
  it('starts all artwork requests before the map is active and retains more than 64 images', async () => {
    vi.mocked(loadUniverseImage).mockResolvedValue(image)
    const rendered = vi.fn()
    const progress = vi.fn()
    const collection = createUniverseImages(rendered, progress)
    collection.update(nodes(300))
    expect(loadUniverseImage).toHaveBeenCalledTimes(300)
    expect(progress).toHaveBeenLastCalledWith({ total: 300, ready: 0, failed: 0, pending: 300 })
    await settle()
    expect(rendered).toHaveBeenCalledTimes(300)
    expect(progress).toHaveBeenLastCalledWith({ total: 300, ready: 300, failed: 0, pending: 0 })
    collection.update(nodes(300).reverse())
    expect(loadUniverseImage).toHaveBeenCalledTimes(300)
    expect(rendered).toHaveBeenCalledTimes(300)
    collection.dispose()
  })
  it('deduplicates shared images and only fetches newly appended works', async () => {
    vi.mocked(loadUniverseImage).mockResolvedValue(image)
    const rendered = vi.fn()
    const collection = createUniverseImages(rendered, vi.fn())
    const works = nodes(2)
    works[1]!.artwork.thumb = works[0]!.artwork.thumb
    collection.update(works)
    await settle()
    expect(loadUniverseImage).toHaveBeenCalledTimes(1)
    expect(rendered).toHaveBeenCalledTimes(2)
    collection.update([...works, nodes(3)[2]!])
    await settle()
    expect(loadUniverseImage).toHaveBeenCalledTimes(2)
    expect(rendered).toHaveBeenCalledTimes(3)
    collection.dispose()
  })
  it('distinguishes unavailable pictures from text-only works', async () => {
    vi.mocked(loadUniverseImage).mockResolvedValue(null)
    const progress = vi.fn()
    const collection = createUniverseImages(vi.fn(), progress)
    const works = nodes(2)
    works[1]!.artwork.thumb = ''
    collection.update(works)
    await settle()
    expect(loadUniverseImage).toHaveBeenCalledTimes(1)
    expect(progress).toHaveBeenLastCalledWith({ total: 1, ready: 0, failed: 1, pending: 0 })
    collection.dispose()
  })
  it('cancels owner requests and ignores late images after closing', async () => {
    let resolve!: (image: HTMLImageElement) => void
    vi.mocked(loadUniverseImage).mockReturnValue(new Promise(done => { resolve = done }))
    const rendered = vi.fn()
    const progress = vi.fn()
    const collection = createUniverseImages(rendered, progress)
    collection.update(nodes(1))
    const signal = vi.mocked(loadUniverseImage).mock.calls[0]![1]
    collection.dispose()
    expect(signal.aborted).toBe(true)
    resolve(image)
    await settle()
    expect(rendered).not.toHaveBeenCalled()
    expect(progress).toHaveBeenCalledTimes(1)
  })
  it('ignores a stale image after its work was removed or changed', async () => {
    const completions = new Map<string, (image: HTMLImageElement) => void>()
    vi.mocked(loadUniverseImage).mockImplementation(artwork => new Promise(done => { completions.set(artwork.thumb, done) }))
    const rendered = vi.fn()
    const collection = createUniverseImages(rendered, vi.fn())
    const works = nodes(2)
    collection.update(works)
    collection.update([{ ...works[0]!, artwork: { ...works[0]!.artwork, thumb: '/thumbs/changed.webp' } }])
    completions.get('/thumbs/0.webp')!(image)
    completions.get('/thumbs/1.webp')!(image)
    await settle()
    expect(rendered).not.toHaveBeenCalled()
    completions.get('/thumbs/changed.webp')!(image)
    await settle()
    expect(rendered).toHaveBeenCalledTimes(1)
    collection.dispose()
  })
})

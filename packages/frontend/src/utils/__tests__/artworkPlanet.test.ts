import { afterEach, describe, expect, it, vi } from 'vitest'
import { PlanetArtworkDeck, planetSlotSamples, PLANET_SLOTS, paintPlanetTexture } from '../artworkPlanet'

afterEach(() => vi.restoreAllMocks())

const items = Array.from({ length: 30 }, (_, id) => ({ id, thumb: `/thumbs/${id}.webp` }))

describe('PlanetArtworkDeck', () => {
  it('deduplicates sources and visits all candidates before reshuffling', () => {
    const deck = new PlanetArtworkDeck()
    deck.update([...items, items[0]!, { id: 31, thumb: '' }])
    const visited = Array.from({ length: 30 }, () => deck.next(new Set()))
    expect(new Set(visited).size).toBe(30)
    expect(deck.next(new Set())).toBeDefined()
  })

  it('prioritizes new pages and avoids resident and loading images', () => {
    const deck = new PlanetArtworkDeck()
    deck.update(items.slice(0, 4))
    deck.next(new Set())
    deck.update(items.slice(0, 5))
    expect(deck.next(new Set())).toBe('/thumbs/4.webp')
    const blocked = new Set(items.slice(0, 5).map(item => item.thumb))
    expect(deck.next(blocked)).toBeUndefined()
  })

  it('drops removed works without growing an image cache', () => {
    const deck = new PlanetArtworkDeck()
    deck.update(items)
    deck.update([{ id: 100, thumb: '/thumbs/only.webp' }])
    expect(deck.next(new Set())).toBe('/thumbs/only.webp')
    expect(deck.next(new Set())).toBe('/thumbs/only.webp')
  })

  it('keeps cycling when the end of the deck contains only resident images', () => {
    vi.spyOn(Math, 'random').mockReturnValue(.999)
    const deck = new PlanetArtworkDeck()
    deck.update(items.slice(0, 3))
    expect(deck.next(new Set())).toBe('/thumbs/0.webp')
    const blocked = new Set(['/thumbs/1.webp', '/thumbs/2.webp'])
    expect(deck.next(blocked)).toBe('/thumbs/0.webp')
  })
})

describe('planet texture', () => {
  it('samples all twelve patches on the unit sphere including their overlap', () => {
    for (let index = 0; index < PLANET_SLOTS; index++) {
      const points = planetSlotSamples(index)
      expect(points).toHaveLength(25)
      points.forEach(([x, y, z]) => expect(Math.hypot(x, y, z)).toBeCloseTo(1))
    }
  })

  it('wraps faded edge tiles across the longitude seam without resizing the atlas', () => {
    const gradient = { addColorStop: vi.fn() }
    const ctx = { clearRect: vi.fn(), drawImage: vi.fn(), createLinearGradient: vi.fn(() => gradient), fillRect: vi.fn(), globalCompositeOperation: 'source-over', fillStyle: '' }
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(ctx as unknown as CanvasRenderingContext2D)
    const canvas = document.createElement('canvas')
    canvas.width = 1024
    canvas.height = 512
    const image = { naturalWidth: 400, naturalHeight: 600 } as HTMLImageElement
    paintPlanetTexture(canvas, [image])
    expect(canvas.width).toBe(1024)
    expect(canvas.height).toBe(512)
    expect(ctx.createLinearGradient).toHaveBeenCalledTimes(24)
    expect(ctx.drawImage.mock.calls.some(call => Number(call[1]) > 1000)).toBe(true)
    expect(ctx.drawImage.mock.calls.some(call => Number(call[1]) < 0)).toBe(true)
    expect(ctx.globalCompositeOperation).toBe('source-over')
  })
})

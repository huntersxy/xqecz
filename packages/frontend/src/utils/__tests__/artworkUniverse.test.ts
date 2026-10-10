import { describe, expect, it } from 'vitest'
import { UniverseLayout, universeBounds, visibleUniverseNodes, universeOpeningView } from '../artworkUniverse'

const artworks = Array.from({ length: 400 }, (_, index) => ({ id: index + 1, title: `Work ${index}`, thumb: index % 3 ? `/thumbs/${index}.webp` : '' }))

describe('universe layout', () => {
  it('includes text-only works and deduplicates overlapping recommendations', () => {
    const nodes = new UniverseLayout(42).update([...artworks, artworks[0]!])
    expect(nodes).toHaveLength(400)
    expect(nodes.filter(node => !node.artwork.thumb).length).toBeGreaterThan(0)
  })
  it('places all planets without collisions', () => {
    const nodes = new UniverseLayout(123).update(artworks)
    nodes.forEach((a, index) => nodes.slice(index + 1).forEach(b => {
      expect(Math.hypot(a.x - b.x, a.y - b.y)).toBeGreaterThan(a.radius + b.radius)
    }))
  })
  it('keeps existing coordinates when pages arrive or the list order changes', () => {
    const layout = new UniverseLayout(54)
    const first = layout.update(artworks.slice(0, 50))
    const appended = layout.update([...artworks].reverse())
    first.forEach(node => {
      const next = appended.find(item => item.artwork.id === node.artwork.id)!
      expect([next.x, next.y]).toEqual([node.x, node.y])
    })
  })
  it('uses different arrangements on separate visits', () => {
    expect(new UniverseLayout(11).update(artworks)).not.toEqual(new UniverseLayout(12).update(artworks))
  })
  it('has varied local spacing rather than uniform spiral or grid spacing', () => {
    for (const seed of [11, 123, 9876]) {
      const nodes = new UniverseLayout(seed).update(artworks)
      const distances = nodes.map((node, index) => Math.min(...nodes.filter((_, other) => other !== index).map(other => Math.hypot(node.x - other.x, node.y - other.y))))
      const mean = distances.reduce((sum, distance) => sum + distance, 0) / distances.length
      const deviation = Math.sqrt(distances.reduce((sum, distance) => sum + (distance - mean) ** 2, 0) / distances.length)
      expect(deviation / mean).toBeGreaterThan(.2)
    }
  })
  it('is repeatable for a seed and preserves removed coordinates when a work returns', () => {
    const layout = new UniverseLayout(65)
    const original = layout.update(artworks)
    expect(new UniverseLayout(65).update(artworks)).toEqual(original)
    layout.update(artworks.slice(30))
    expect(layout.update(artworks)).toEqual(original)
  })
  it('bounds encompass every planet without hiding visible works behind a fixed cap', () => {
    const nodes = new UniverseLayout(92).update(artworks)
    const size = universeBounds(nodes)
    expect(nodes.every(node => Math.abs(node.x) + node.radius < size / 2 && Math.abs(node.y) + node.radius < size / 2)).toBe(true)
    expect(visibleUniverseNodes(nodes, 0, 0, size, size)).toHaveLength(400)
    const visible = visibleUniverseNodes(nodes, 0, 0, 300, 300)
    expect(visible.length).toBeLessThan(10)
    expect(visible.every(node => Math.abs(node.x) < 150 + node.radius && Math.abs(node.y) < 150 + node.radius)).toBe(true)
  })
  it('opens close to image works with readable planet diameters', () => {
    const nodes = new UniverseLayout(42).update(artworks)
    const view = universeOpeningView(nodes, 16 / 9)
    const visible = visibleUniverseNodes(nodes, view.x, view.y, view.height * 16 / 9, view.height)
    expect(visible.filter(node => node.artwork.thumb).length).toBeGreaterThan(30)
    const images = visible.filter(node => node.artwork.thumb)
    const averageDiameter = images.reduce((sum, node) => sum + node.radius * 2 * 1080 / view.height, 0) / images.length
    expect(averageDiameter).toBeGreaterThan(70)
    expect(nodes.filter(node => node.artwork.thumb).every(node => node.radius >= 82)).toBe(true)
  })

})

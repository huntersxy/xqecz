export interface UniverseArtwork { id: number; title?: string; thumb: string; text?: string }
export interface UniverseNode { artwork: UniverseArtwork; x: number; y: number; radius: number; hue: number }
export interface PlanetOrigin { left: number; top: number; width: number; height: number; image: string; atlas?: string }
export type UniversePhase = 'flight' | 'unfold' | 'marquee' | 'collapse' | 'map' | 'closing'

function hash(value: number) {
  let n = value | 0
  n = Math.imul(n ^ (n >>> 16), 0x45d9f3b)
  n = Math.imul(n ^ (n >>> 16), 0x45d9f3b)
  return (n ^ (n >>> 16)) >>> 0
}

function randomGenerator(seed: number) {
  let state = seed >>> 0
  return () => {
    state += 0x6d2b79f5
    let value = Math.imul(state ^ (state >>> 15), state | 1)
    value ^= value + Math.imul(value ^ (value >>> 7), value | 61)
    return ((value ^ (value >>> 14)) >>> 0) / 4294967296
  }
}

interface Cloud { x: number; y: number; width: number; height: number; angle: number }
type Coordinate = Omit<UniverseNode, 'artwork'>

/** Random cloud sampling with local collision checks; appended works never move existing planets. */
export class UniverseLayout {
  private coordinates = new Map<number, Coordinate>()
  private cells = new Map<string, Coordinate[]>()
  private clouds: Cloud[] = []
  private extent = 1000
  constructor(private seed: number) {}

  private createClouds(count: number) {
    const random = randomGenerator(this.seed)
    this.extent = Math.max(1000, Math.sqrt(count) * 120)
    for (let index = 0; index < 4; index++) {
      let x = 0
      let y = 0
      for (let attempt = 0; attempt < 80; attempt++) {
        x = (random() - .5) * this.extent * 2.2
        y = (random() - .5) * this.extent * 1.6
        if (this.clouds.every(cloud => Math.hypot(cloud.x - x, cloud.y - y) > this.extent * .75)) break
      }
      this.clouds.push({ x, y, width: this.extent * (.25 + random() * .2), height: this.extent * (.16 + random() * .15), angle: random() * Math.PI * 2 })
    }
  }

  private place(id: number): Coordinate {
    const noise = hash(id ^ this.seed)
    const random = randomGenerator(noise)
    const radius = 39 + ((noise >>> 16) & 255) / 255 * 16
    const gap = 12 + random() * 28
    const clustered = random() < .86
    const cloud = this.clouds[Math.floor(random() * this.clouds.length)]!
    for (let attempt = 0; ; attempt++) {
      let x: number
      let y: number
      if (attempt < 192 && clustered) {
        // Bounded bell-shaped sampling produces uneven clusters without remote outliers.
        const dx = (random() + random() + random() - 1.5) * cloud.width
        const dy = (random() + random() + random() - 1.5) * cloud.height
        x = cloud.x + dx * Math.cos(cloud.angle) - dy * Math.sin(cloud.angle)
        y = cloud.y + dx * Math.sin(cloud.angle) + dy * Math.cos(cloud.angle)
      } else {
        const angle = random() * Math.PI * 2
        const distance = Math.sqrt(random()) * this.extent * (1.55 + Math.floor(attempt / 192) * .18)
        x = Math.cos(angle) * distance
        y = Math.sin(angle) * distance * .8
      }
      const col = Math.floor(x / 160)
      const row = Math.floor(y / 160)
      let occupied = false
      for (let dx = -1; dx <= 1 && !occupied; dx++) {
        for (let dy = -1; dy <= 1 && !occupied; dy++) {
          occupied = (this.cells.get(`${col + dx}:${row + dy}`) || [])
            .some(other => Math.hypot(other.x - x, other.y - y) < radius + other.radius + gap)
        }
      }
      if (occupied) continue
      const coordinate = { x, y, radius, hue: noise % 360 }
      const key = `${col}:${row}`
      const cell = this.cells.get(key) || []
      cell.push(coordinate)
      this.cells.set(key, cell)
      return coordinate
    }
  }

  update(artworks: readonly UniverseArtwork[]): UniverseNode[] {
    const unique = new Map(artworks.map(artwork => [artwork.id, artwork]))
    if (!this.clouds.length && unique.size) this.createClouds(unique.size)
    const fresh = [...unique.values()].filter(item => !this.coordinates.has(item.id))
    fresh.sort((a, b) => hash(a.id ^ this.seed) - hash(b.id ^ this.seed))
    fresh.forEach(item => this.coordinates.set(item.id, this.place(item.id)))
    return [...unique.values()].map(artwork => ({ artwork, ...this.coordinates.get(artwork.id)! }))
  }
}

export function universeBounds(nodes: readonly UniverseNode[]) {
  const span = nodes.reduce((value, node) => Math.max(value, Math.abs(node.x) + node.radius, Math.abs(node.y) + node.radius), 240)
  return span * 2 + 180
}

export function visibleUniverseNodes(nodes: readonly UniverseNode[], x: number, y: number, width: number, height: number, limit = 64) {
  return nodes.filter(node => Math.abs(node.x - x) < width / 2 + node.radius && Math.abs(node.y - y) < height / 2 + node.radius)
    .sort((a, b) => (a.x - x) ** 2 + (a.y - y) ** 2 - (b.x - x) ** 2 - (b.y - y) ** 2)
    .slice(0, limit)
}

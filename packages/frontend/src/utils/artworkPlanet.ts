import { getImageUrl, getRemoteFallbackUrl, REMOTE_MEDIA_BASE } from '@/utils'

// 普通 <img> 能显示不代表 Canvas/WebGL 可读取：贴图加载必须统一走本文件的 CORS 入口。
// 地址归属由 utils/index.ts 判定；这里仅负责贴图的跨域策略，详见 docs/frontend-media.md。
export interface PlanetArtwork { id: number; thumb: string }
export const PLANET_COLUMNS = 6
export const PLANET_ROWS = 2
export const PLANET_SLOTS = PLANET_COLUMNS * PLANET_ROWS
export const PLANET_TEXTURE_WIDTH = 1024
export const PLANET_TEXTURE_HEIGHT = 512
const OVERLAP = 22

function shuffle<T>(items: T[]) {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[items[i], items[j]] = [items[j]!, items[i]!]
  }
  return items
}

/** 新页作品优先进入队列；遍历完才重新洗牌，不把随机固定在首次加载。 */
export class PlanetArtworkDeck {
  private urls: string[] = []
  private pending: string[] = []

  update(items: readonly PlanetArtwork[]) {
    const urls = [...new Set(items.map(item => getImageUrl(item.thumb)).filter(Boolean))]
    const previous = new Set(this.urls)
    const current = new Set(urls)
    this.pending = [...shuffle(urls.filter(url => !previous.has(url))), ...this.pending.filter(url => current.has(url))]
    this.urls = urls
  }

  next(excluded: ReadonlySet<string>): string | undefined {
    if (!this.pending.length) this.pending = shuffle([...this.urls])
    let index = this.pending.findIndex(url => !excluded.has(url))
    if (index < 0 && this.urls.some(url => !excluded.has(url))) {
      this.pending = shuffle(this.urls.filter(url => !excluded.has(url)))
      index = 0
    }
    if (index < 0) return undefined
    return this.pending.splice(index, 1)[0]
  }
}

/** 在共享地址解析之后，为本站跨域贴图设置独立缓存键；不改第三方签名地址。 */
export function planetImageSource(url: string, origin = window.location.origin, development = import.meta.env.DEV): string {
  const resolved = getImageUrl(url)
  if (!resolved) return ''
  const parsed = new URL(resolved, origin)
  if (parsed.origin === origin || !/^https?:$/.test(parsed.protocol)) return resolved
  if (parsed.origin !== REMOTE_MEDIA_BASE && !getRemoteFallbackUrl(resolved, origin)) return resolved
  if (development && parsed.origin === REMOTE_MEDIA_BASE && /^\/(thumbs|uploads|images)\//.test(parsed.pathname)) {
    return `/__planet-media${parsed.pathname}${parsed.search}`
  }
  // Ordinary <img> CDN entries may lack ACAO. Separate origins to avoid reusing
  // a response cached without CORS or with another site's Allow-Origin header.
  parsed.searchParams.set('texture_origin', origin)
  return parsed.href
}

/** 只读取缩略图，超时 / 离开视口取消；CORS 失败时保留其余可用作品。 */
export async function loadPlanetImage(url: string, signal: AbortSignal): Promise<HTMLImageElement | null> {
  const source = planetImageSource(url)
  if (!source || signal.aborted) return null
  async function load(src: string): Promise<HTMLImageElement | null> {
    if (signal.aborted) return null
    return new Promise(resolve => {
      const image = new Image()
      image.crossOrigin = 'anonymous'
      image.decoding = 'async'
      let settled = false
      const timer = window.setTimeout(() => finish(null), 4000)
      function finish(result: HTMLImageElement | null) {
        if (settled) return
        settled = true
        window.clearTimeout(timer)
        signal.removeEventListener('abort', abort)
        image.onload = null
        image.onerror = null
        if (!result) image.src = ''
        resolve(result)
      }
      function abort() { finish(null) }
      signal.addEventListener('abort', abort, { once: true })
      image.onload = () => finish(image.naturalWidth > 0 ? image : null)
      image.onerror = () => finish(null)
      image.src = src
    })
  }
  const image = await load(source)
  if (image || signal.aborted) return image
  const fallback = getRemoteFallbackUrl(getImageUrl(url))
  return fallback ? load(planetImageSource(fallback)) : null
}

/** 局部重绘不重置尺寸；贴片边缘渐隐并重叠，横向 wrap 保证经度首尾无缝。 */
export function paintPlanetTexture(canvas: HTMLCanvasElement, images: readonly HTMLImageElement[]) {
  const ctx = canvas.getContext('2d')
  if (!ctx || !images.length) return
  const width = PLANET_TEXTURE_WIDTH / PLANET_COLUMNS
  const height = PLANET_TEXTURE_HEIGHT / PLANET_ROWS
  ctx.clearRect(0, 0, canvas.width, canvas.height)
  // 预乘 alpha 加法合成相邻渐隐片，重叠区域权重相加为一，不残留硬边。
  ctx.globalCompositeOperation = 'lighter'
  const tile = document.createElement('canvas')
  tile.width = Math.ceil(width + OVERLAP * 2)
  tile.height = height + OVERLAP * 2
  const tileCtx = tile.getContext('2d')
  if (!tileCtx) return
  for (let index = 0; index < PLANET_SLOTS; index++) {
    tileCtx.clearRect(0, 0, tile.width, tile.height)
    tileCtx.globalCompositeOperation = 'source-over'
    drawCover(tileCtx, images[index % images.length]!, 0, 0, tile.width, tile.height)
    tileCtx.globalCompositeOperation = 'destination-in'
    const horizontal = tileCtx.createLinearGradient(0, 0, tile.width, 0)
    horizontal.addColorStop(0, 'transparent')
    horizontal.addColorStop(OVERLAP * 2 / tile.width, '#fff')
    horizontal.addColorStop(1 - OVERLAP * 2 / tile.width, '#fff')
    horizontal.addColorStop(1, 'transparent')
    tileCtx.fillStyle = horizontal
    tileCtx.fillRect(0, 0, tile.width, tile.height)
    const vertical = tileCtx.createLinearGradient(0, 0, 0, tile.height)
    vertical.addColorStop(0, index < PLANET_COLUMNS ? '#fff' : 'transparent')
    vertical.addColorStop(OVERLAP * 2 / tile.height, '#fff')
    vertical.addColorStop(1 - OVERLAP * 2 / tile.height, '#fff')
    vertical.addColorStop(1, index >= PLANET_COLUMNS ? '#fff' : 'transparent')
    tileCtx.fillStyle = vertical
    tileCtx.fillRect(0, 0, tile.width, tile.height)
    const x = (index % PLANET_COLUMNS) * width - OVERLAP
    const y = Math.floor(index / PLANET_COLUMNS) * height - OVERLAP
    ctx.drawImage(tile, x, y)
    if (x < 0) ctx.drawImage(tile, x + canvas.width, y)
    if (x + tile.width > canvas.width) ctx.drawImage(tile, x - canvas.width, y)
  }
  ctx.globalCompositeOperation = 'source-over'
}

function drawCover(ctx: CanvasRenderingContext2D, image: HTMLImageElement, x: number, y: number, width: number, height: number) {
  const scale = Math.max(width / image.naturalWidth, height / image.naturalHeight)
  const sw = width / scale
  const sh = height / scale
  ctx.drawImage(image, (image.naturalWidth - sw) / 2, (image.naturalHeight - sh) / 2, sw, sh, x, y, width, height)
}

/** 球面坐标与 SphereGeometry 的 UV 一致，含渐隐外沿；用于确认整片已转到背面。 */
export function planetSlotSamples(index: number): [number, number, number][] {
  const col = index % PLANET_COLUMNS
  const row = Math.floor(index / PLANET_COLUMNS)
  const phiPad = OVERLAP / PLANET_TEXTURE_WIDTH * Math.PI * 2
  const thetaPad = OVERLAP / PLANET_TEXTURE_HEIGHT * Math.PI
  const result: [number, number, number][] = []
  for (let y = 0; y <= 4; y++) {
    const theta = Math.max(.04, Math.min(Math.PI - .04, (row + y / 4) / PLANET_ROWS * Math.PI + (y === 0 ? -thetaPad : y === 4 ? thetaPad : 0)))
    for (let x = 0; x <= 4; x++) {
      const phi = (col + x / 4) / PLANET_COLUMNS * Math.PI * 2 + (x === 0 ? -phiPad : x === 4 ? phiPad : 0)
      result.push([-Math.cos(phi) * Math.sin(theta), Math.cos(theta), Math.sin(phi) * Math.sin(theta)])
    }
  }
  return result
}

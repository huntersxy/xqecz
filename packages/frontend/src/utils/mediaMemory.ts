const STORE_KEY = 'xqecz:media-loaded-v1'
const MAX_ENTRIES = 400
const TTL_MS = 30 * 24 * 60 * 60 * 1000
const REFRESH_MS = 24 * 60 * 60 * 1000
const MEDIA_PATH = /\/(uploads|thumbs|images)\/([^/]+)$/

interface Entry {
  url: string
  at: number
}

export function mediaKeyOf(url: string): string | null {
  if (!url) return null
  const m = MEDIA_PATH.exec(url.replace(/[?#].*$/, ''))
  return m ? `${m[1]}/${m[2]}` : null
}

export function sameMedia(a: string, b: string): boolean {
  const key = mediaKeyOf(a)
  return key !== null && key === mediaKeyOf(b)
}

function load(): Record<string, Entry> {
  try {
    const raw = localStorage.getItem(STORE_KEY)
    const parsed = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function save(map: Record<string, Entry>): void {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify(map))
  } catch {}
}

export function rememberLoaded(localUrl: string, loadedUrl: string, now: number = Date.now()): void {
  const key = mediaKeyOf(localUrl)
  if (!key || !sameMedia(localUrl, loadedUrl)) return
  const map = load()
  const prev = map[key]
  if (prev?.url === loadedUrl && now - prev.at < REFRESH_MS) return
  map[key] = { url: loadedUrl, at: now }
  const keys = Object.keys(map)
  if (keys.length > MAX_ENTRIES) {
    keys.sort((a, b) => map[a].at - map[b].at)
    for (const k of keys.slice(0, keys.length - MAX_ENTRIES)) delete map[k]
  }
  save(map)
}

export function recallLoaded(localUrl: string, now: number = Date.now()): string | null {
  const key = mediaKeyOf(localUrl)
  if (!key) return null
  const e = load()[key]
  if (!e || typeof e.url !== 'string' || now - e.at > TTL_MS) return null
  return sameMedia(localUrl, e.url) ? e.url : null
}

export function forgetLoaded(localUrl: string): void {
  const key = mediaKeyOf(localUrl)
  if (!key) return
  const map = load()
  if (!(key in map)) return
  delete map[key]
  save(map)
}


export type MediaSource = 'origin' | 'mirror' | 'mirror2'

export type MirrorTier = 'mirror' | 'mirror2'

export const SOURCE_TTL_MS = 7 * 24 * 60 * 60 * 1000

export const PROBE_TIMEOUT_MS = 4000

export const PROBE_BYTES = 32768

export const PROBE_RETRY_DELAY_MS = 1000

export const BROKEN_COOLDOWN_MS = 60 * 1000

const IP_ECHO_URL = 'https://1.1.1.1/cdn-cgi/trace'

const IP_TIMEOUT_MS = 3000

const RECORD_KEY = 'xqecz:image-source-v3'

interface SourceRecord {
  openlist?: boolean
  r2?: boolean
  at: number
  net: string
  ip?: string
}

let knownIp: string | undefined
let networkCheck: Promise<boolean> | null = null
const brokenMarks = new Map<MirrorTier, number>()

function tierBroken(tier: MirrorTier, now: number): boolean {
  const at = brokenMarks.get(tier)
  return at !== undefined && now - at < BROKEN_COOLDOWN_MS
}


function readRecord(): SourceRecord | null {
  try {
    const raw = localStorage.getItem(RECORD_KEY)
    if (!raw) return null
    const r = JSON.parse(raw) as SourceRecord
    const hasTier = typeof r?.openlist === 'boolean' || typeof r?.r2 === 'boolean'
    if (!hasTier || typeof r?.net !== 'string') return null
    return r
  } catch {
    return null
  }
}

function writeRecord(rec: SourceRecord): void {
  try {
    localStorage.setItem(RECORD_KEY, JSON.stringify(rec))
  } catch {
  }
}

function clearRecord(): void {
  try {
    localStorage.removeItem(RECORD_KEY)
  } catch {
  }
}

function bestSource(rec: SourceRecord): MediaSource {
  if (rec.openlist) return 'mirror2'
  if (rec.r2) return 'mirror'
  return 'origin'
}

export function localNetFingerprint(): string {
  const nav = navigator as Navigator & {
    connection?: { effectiveType?: string; rtt?: number; downlink?: number }
  }
  const c = nav.connection
  const rtt = typeof c?.rtt === 'number' ? Math.round(c.rtt / 50) : -1
  const dl = typeof c?.downlink === 'number' ? Math.round(c.downlink) : -1
  return `${nav.onLine ? 1 : 0}|${c?.effectiveType ?? ''}|${rtt}|${dl}`
}


export function cachedImageSource(now: number = Date.now()): MediaSource | null {
  const rec = readRecord()
  if (!rec) return null
  if (rec.net !== localNetFingerprint()) return null
  if (now - rec.at > SOURCE_TTL_MS) return null
  const eff = { ...rec }
  if (tierBroken('mirror2', now)) eff.openlist = false
  if (tierBroken('mirror', now)) eff.r2 = false
  return bestSource(eff)
}

export async function probeMirror(openlistUrl: string, r2Url = ''): Promise<MediaSource> {
  const rec: SourceRecord = {
    at: Date.now(),
    net: localNetFingerprint(),
    ...(knownIp ? { ip: knownIp } : {}),
  }

  if (openlistUrl) {
    rec.openlist = await reachable(openlistUrl)
    if (rec.openlist) brokenMarks.delete('mirror2')
  }
  if (!rec.openlist && r2Url) {
    rec.r2 = await reachable(r2Url)
    if (rec.r2) brokenMarks.delete('mirror')
  }

  writeRecord(rec)
  return bestSource(rec)
}

export function markUnreachable(tier: MirrorTier): void {
  brokenMarks.set(tier, Date.now())
}

export function recheckNetwork(): Promise<boolean> {
  networkCheck ??= checkNetwork()
  return networkCheck
}

async function checkNetwork(): Promise<boolean> {
  const rec = readRecord()
  if (!rec) return false
  const ip = await fetchIp()
  if (ip) knownIp = ip
  if (!ip || !rec.ip) {
    if (ip) writeRecord({ ...rec, ip })
    return false
  }
  if (ip === rec.ip) return false
  clearRecord()
  brokenMarks.clear()
  return true
}

async function fetchIp(): Promise<string | undefined> {
  try {
    const ctrl = typeof AbortController !== 'undefined' ? new AbortController() : null
    const timer = ctrl ? setTimeout(() => ctrl.abort(), IP_TIMEOUT_MS) : null
    const resp = await fetch(IP_ECHO_URL, {
      cache: 'no-store',
      signal: ctrl ? ctrl.signal : undefined,
    })
    if (!resp.ok) return undefined
    const text = await resp.text()
    const m = /\bip=(\S+)/.exec(text)
    return m?.[1]
  } catch {
    return undefined
  } finally {
  }
}

async function reachableOnce(url: string): Promise<boolean> {
  try {
    const ctrl = typeof AbortController !== 'undefined' ? new AbortController() : null
    const timer = ctrl ? setTimeout(() => ctrl.abort(), PROBE_TIMEOUT_MS) : null
    const resp = await fetch(url, {
      cache: 'no-store',
      signal: ctrl ? ctrl.signal : undefined,
      headers: { Range: `bytes=0-${PROBE_BYTES - 1}` },
    })
    return resp.ok || resp.status === 206
  } catch {
    return false
  } finally {
  }
}

async function reachable(url: string): Promise<boolean> {
  if (await reachableOnce(url)) return true
  await delay(PROBE_RETRY_DELAY_MS)
  if (await reachableOnce(url)) return true
  return reachableOnce(url)
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}


export function pickImageUrl(
  local: string,
  mirror: string,
  mirror2: string,
  source: MediaSource | null,
): string {
  if (source === 'mirror' && mirror) return mirror
  if (source === 'mirror2' && mirror2) return mirror2
  return local || mirror2 || mirror
}

export function fallbackChain(
  local: string,
  mirror: string,
  mirror2: string,
  source: MediaSource | null,
): string[] {
  const chain: string[] = []
  if (source === 'mirror2') {
    if (mirror) chain.push(mirror)
    if (local) chain.push(local)
  } else if (source === 'mirror') {
    if (local) chain.push(local)
  }
  return chain
}

export function resetImageSourceState(): void {
  clearRecord()
  brokenMarks.clear()
  knownIp = undefined
  networkCheck = null
}

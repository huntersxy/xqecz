/**
 * 详情页大图的来源决策：**可达性优先，失败回退源站**。
 *
 * 与旧的「测速择快」不同，这里的取舍由业务约束决定：
 * - 服务器出网额度（20GB/月）是瓶颈 → 镜像（R2）只要可达就用，哪怕它更慢；
 * - 只探测镜像侧：源站本来就是回退目标，探它既没有决策价值，还会白发一次整图请求；
 * - 探测流量是「浏览器 → Cloudflare」，**不占服务器的出网额度**。
 *
 * 结论按「网络标识」记在本机 `localStorage`（不上传），以便同一网络回访时直接复用：
 * 首图零等待、也不用重复探测。网络标识 = 公网 IP（主，能识别代理开/关）
 * + navigator.connection（辅，IP 接口不可用时的退路；Chromium 才有）。
 * IP 是异步取的，所以同步读取时只能先用本地指纹判断，IP 回来后若发现网络已变
 * 再重新探测——宁可多探测一次，也不让旧结论一直挂在错误的网络上。
 *
 * 结论永远可能出错（R2 中途挂、指纹太粗），因此渲染层还有第二道兜底：
 * `markMirrorUnreachable()` 在镜像加载失败时把当前网络标记为不可达。
 */

/** 图片来源。`origin` 是权威副本（源站），`mirror` 是镜像（R2）。 */
export type MediaSource = 'origin' | 'mirror'

/** 本机记录的有效期：网络标识没变时最多复用这么久。 */
export const SOURCE_TTL_MS = 7 * 24 * 60 * 60 * 1000

/** 单次探测超时：超过即判为不可达（阻断型网络会一直等到这里）。 */
export const PROBE_TIMEOUT_MS = 4000

/** 只要一小段字节，够判定可达性即可，不必下载整图。 */
export const PROBE_BYTES = 32768

/** 公网 IP 回显：Cloudflare 自家接口，纯文本 `ip=...`，CORS 为 `*`。 */
const IP_ECHO_URL = 'https://1.1.1.1/cdn-cgi/trace'

/** IP 接口自身的超时：它只用于「网络是否变了」的校验，不该拖慢首图。 */
const IP_TIMEOUT_MS = 3000

const RECORD_KEY = 'xqecz:image-source'

interface SourceRecord {
  /** 镜像是否可达 */
  reachable: boolean
  /** 记录时间（用于 TTL） */
  at: number
  /** 本地网络指纹（同步可算），变了就不再信任本条记录 */
  net: string
  /** 公网 IP（异步补全），用于识别代理开/关 */
  ip?: string
}

/** 内存里最近一次拿到的公网 IP，避免同一次会话重复取。 */
let knownIp: string | undefined
/** IP 校验在一次页面会话内只做一次。 */
let networkCheck: Promise<boolean> | null = null

/* ---------------- 本机记录 ---------------- */

function readRecord(): SourceRecord | null {
  try {
    const raw = localStorage.getItem(RECORD_KEY)
    if (!raw) return null
    const r = JSON.parse(raw) as SourceRecord
    if (typeof r?.reachable !== 'boolean' || typeof r?.net !== 'string') return null
    return r
  } catch {
    return null
  }
}

function writeRecord(rec: SourceRecord): void {
  try {
    localStorage.setItem(RECORD_KEY, JSON.stringify(rec))
  } catch {
    // 隐身模式 / 存储配额满：读不到记录只意味着每次都探测，功能不受影响。
  }
}

function clearRecord(): void {
  try {
    localStorage.removeItem(RECORD_KEY)
  } catch {
    /* 同上 */
  }
}

/**
 * 本地网络指纹：同步可得，所以它决定「这次访问要不要等探测」。
 * 保持简略——只取能区分网络切换的少数几个字段；不区分网络的字段（如 UA）一律不取。
 * `navigator.connection` 仅 Chromium 提供，缺失时该部分退化为空串（仍与 IP 配合判断）。
 */
export function localNetFingerprint(): string {
  const nav = navigator as Navigator & {
    connection?: { effectiveType?: string; rtt?: number; downlink?: number }
  }
  const c = nav.connection
  const rtt = typeof c?.rtt === 'number' ? Math.round(c.rtt / 50) : -1
  const dl = typeof c?.downlink === 'number' ? Math.round(c.downlink) : -1
  return `${nav.onLine ? 1 : 0}|${c?.effectiveType ?? ''}|${rtt}|${dl}`
}

/* ---------------- 对外读接口 ---------------- */

/**
 * 同步取本机已记录的结论；网络指纹已变、记录过期或没有记录时返回 null（调用方需探测）。
 * 注意：这里看不到「IP 变了」这种情况，那要等 `recheckNetwork()` 异步确认。
 */
export function cachedImageSource(now: number = Date.now()): MediaSource | null {
  const rec = readRecord()
  if (!rec) return null
  if (rec.net !== localNetFingerprint()) return null
  if (now - rec.at > SOURCE_TTL_MS) return null
  return rec.reachable ? 'mirror' : 'origin'
}

/**
 * 探测镜像可达性：一次带 `Range` 的小请求，成功即认为可达。
 * 失败（超时 / 非 2xx206 / 网络错误）一律判为不可达，并把结论记到本机。
 * 源站不参与探测——它就是回退目标。
 */
export async function probeMirror(mirrorUrl: string): Promise<MediaSource> {
  // 注意不要用与模块函数 `reachable` 同名的局部常量——会遮蔽它并触发 TDZ。
  const ok = await reachable(mirrorUrl)
  writeRecord({
    reachable: ok,
    at: Date.now(),
    net: localNetFingerprint(),
    ...(knownIp ? { ip: knownIp } : {}),
  })
  return ok ? 'mirror' : 'origin'
}

/** 渲染层兜底：镜像加载失败时把当前网络标记为不可达，后续一律走源站。 */
export function markMirrorUnreachable(): void {
  writeRecord({
    reachable: false,
    at: Date.now(),
    net: localNetFingerprint(),
    ...(knownIp ? { ip: knownIp } : {}),
  })
}

/**
 * 后台校验公网 IP：与记录里的不同就说明网络变了（典型是代理开/关），
 * 清掉旧结论并返回 true，调用方据此重新探测。
 * 一次页面会话内只做一次；IP 接口失败不判「变了」——拿不准就不推翻旧结论。
 */
export function recheckNetwork(): Promise<boolean> {
  networkCheck ??= checkNetwork()
  return networkCheck
}

async function checkNetwork(): Promise<boolean> {
  const rec = readRecord()
  if (!rec) return false
  const ip = await fetchIp()
  if (ip) knownIp = ip
  // 记录里还没有 IP（历史遗留或本次刚写入）：补上，不视为网络变化。
  if (!ip || !rec.ip) {
    if (ip) writeRecord({ ...rec, ip })
    return false
  }
  if (ip === rec.ip) return false
  clearRecord()
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
    // `ip=1.2.3.4` 换行其它字段；直接取 ip= 之后那一段，不依赖 JSON 结构。
    const m = /\bip=(\S+)/.exec(text)
    return m?.[1]
  } catch {
    return undefined
  } finally {
    /* timer 在下面统一清理 */
  }
}

/** 镜像侧可达性探测。非 2xx/206、超时、网络错误都视为不可达。 */
async function reachable(url: string): Promise<boolean> {
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
    /* 见 fetchIp：超时用完即弃 */
  }
}

/* ---------------- 地址拼装 ---------------- */

/**
 * 按已定的来源选地址；`source` 尚未确定（null）时用源站，
 * 因为源站是权威副本——不确定就先出不会出错的那一侧，绝不猜。
 */
export function pickImageUrl(local: string, mirror: string, source: MediaSource | null): string {
  if (source === 'mirror' && mirror) return mirror
  return local || mirror
}

/** 测试与手动重测用：清空本机记录与内存状态。 */
export function resetImageSourceState(): void {
  clearRecord()
  knownIp = undefined
  networkCheck = null
}

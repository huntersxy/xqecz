/**
 * 详情页大图的来源决策：**可达性优先，逐级回退**。
 *
 * 两级镜像 + 源站，顺序固定：**首选镜像（自建 OpenList）→ 次选镜像（R2）→ 源站**。
 * OpenList 在前是因为它由我们自己的机器直出、没有第三方额度与计费；
 * R2 走 Cloudflare，国内访问常被拖慢甚至不通（实测 `img.xqecz.bond` 明显更差），
 * 故排在源站之前、OpenList 之后。
 * 与旧的「测速择快」不同，取舍由业务约束决定：
 * - 服务器出网额度是瓶颈 → 镜像只要可达就用，哪怕它更慢；
 * - 只探测镜像侧：源站本来就是回退目标，探它既没有决策价值，还会白发一次整图请求；
 * - 探测流量是「浏览器 → 镜像」，**不占源站的出网带宽**。
 *
 * 探测是**按优先级串行且短路**的：首选（OpenList）通了就不探次选（R2）。
 * 多数网络下首选可达，因此常规路径只多一次请求；只有首选不通时才会再等一个超时去试次选。
 * 这是刻意的——并行探两个会在每次冷启动都白付一倍探测流量。
 * 单级探测**试满 3 次（一次失败 → 退避 `PROBE_RETRY_DELAY_MS` → 重试 → 快速复核）**
 * 才判不可达：持久结论的代价高，判错一次就是整个 TTL 绕开该级。
 *
 * 结论按「网络标识」记在本机 `localStorage`（不上传），以便同一网络回访时直接复用：
 * 首图零等待、也不用重复探测。网络标识 = 公网 IP（主，能识别代理开/关）
 * + navigator.connection（辅，IP 接口不可用时的退路；Chromium 才有）。
 * IP 是异步取的，所以同步读取时只能先用本地指纹判断，IP 回来后若发现网络已变
 * 再重新探测——宁可多探测一次，也不让旧结论一直挂在错误的网络上。
 *
 * 结论永远可能出错（镜像中途挂、指纹太粗），因此渲染层还有第二道兜底：
 * `markUnreachable()` 在某一级加载失败时给它记一条**内存**负标记并立刻往下退一级。
 * 标记不落盘：只在冷却窗口（`BROKEN_COOLDOWN_MS`）内让 `cachedImageSource`
 * 绕开该级，窗口一过该级自动恢复资格（这就是给镜像的「宽裕重试」）；
 * 持久化的「不可达」只由探测写入，而探测对单级要试满 3 次才认输，瞬时抖动不至于拉黑。
 */

/** 图片来源。`origin` 是权威副本（源站），另两级是镜像。 */
export type MediaSource = 'origin' | 'mirror' | 'mirror2'

/** 可达性探测的目标级，用于回写失败结论。 */
export type MirrorTier = 'mirror' | 'mirror2'

/** 本机记录的有效期：网络标识没变时最多复用这么久。 */
export const SOURCE_TTL_MS = 7 * 24 * 60 * 60 * 1000

/** 单次探测超时：超过即判为不可达（阻断型网络会一直等到这里）。 */
export const PROBE_TIMEOUT_MS = 4000

/** 只要一小段字节，够判定可达性即可，不必下载整图。 */
export const PROBE_BYTES = 32768

/** 探测的重试间隔：两次失败之间先等一下，把瞬时抖动让过去。 */
export const PROBE_RETRY_DELAY_MS = 1000

/** 渲染层失败负标记的冷却窗口：窗口内绕开该级，窗口过后自动恢复重试资格。 */
export const BROKEN_COOLDOWN_MS = 60 * 1000

/** 公网 IP 回显：Cloudflare 自家接口，纯文本 `ip=...`，CORS 为 `*`。 */
const IP_ECHO_URL = 'https://1.1.1.1/cdn-cgi/trace'

/** IP 接口自身的超时：它只用于「网络是否变了」的校验，不该拖慢首图。 */
const IP_TIMEOUT_MS = 3000

// 带 v3：首选镜像从七牛换成自建 OpenList，旧记录里「七牛可达」的结论会把
// 用户继续引向一个已经下线的域名，换个键让它们整体作废、重新探测一次。
// （v2 那次是优先级翻转，同样是为了不让旧结论继续误导。）浏览器只会留着旧键，
// 不读了而已；条目本身很小，等 localStorage 自然淘汰即可。
const RECORD_KEY = 'xqecz:image-source-v3'

interface SourceRecord {
  /**
   * 首选镜像（自建 OpenList）是否可达；
   * `undefined` 表示本次没探（未配置或更高优先级的已通）
   */
  openlist?: boolean
  /** 次选镜像（R2）是否可达；`undefined` 同上 */
  r2?: boolean
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
/** 渲染层失败负标记（内存态）：tier → 失败时刻。不落盘，语义见 `markUnreachable`。 */
const brokenMarks = new Map<MirrorTier, number>()

/** 该级的负标记是否还在冷却窗口内（窗口一过即视为恢复资格）。 */
function tierBroken(tier: MirrorTier, now: number): boolean {
  const at = brokenMarks.get(tier)
  return at !== undefined && now - at < BROKEN_COOLDOWN_MS
}

/* ---------------- 本机记录 ---------------- */

function readRecord(): SourceRecord | null {
  try {
    const raw = localStorage.getItem(RECORD_KEY)
    if (!raw) return null
    const r = JSON.parse(raw) as SourceRecord
    // 至少得有一级的结论，否则这条记录没有任何信息量
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
 * 按记录挑最优可达来源。
 *
 * **OpenList（mirror2）优先于 R2（mirror）**：OpenList 是我们自己的机器直出，
 * 不受第三方计费与额度影响；R2 走 Cloudflare，国内访问常被拖慢甚至不通
 * （实测 `img.xqecz.bond` 体验明显更差）。R2 仍比源站优先——
 * 源站出网带宽是最稀缺的资源，能用镜像就不回源。
 */
function bestSource(rec: SourceRecord): MediaSource {
  if (rec.openlist) return 'mirror2'
  if (rec.r2) return 'mirror'
  return 'origin'
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
  // 渲染层负标记只在冷却窗口内临时绕开对应一级，**不动落盘结论**：
  // 窗口过后该级自动回到候选里（下次渲染就是一次重试机会），真正判死交给探测。
  const eff = { ...rec }
  if (tierBroken('mirror2', now)) eff.openlist = false
  if (tierBroken('mirror', now)) eff.r2 = false
  return bestSource(eff)
}

/**
 * 探测镜像可达性：带 `Range` 的小请求，单级**试满 3 次才判不可达**，
 * 并把结论记到本机。失败（超时 / 非 2xx206 / 网络错误）才落盘为「不可达」。
 *
 * **按优先级串行短路**：首选（OpenList）通了就不再探 R2——首选可达是常态，
 * 不该为每次冷启动多付一次请求。任一侧地址为空（未配置）即跳过该级。
 * 源站不参与探测：它就是回退目标。
 *
 * 入参顺序即优先级（首选在前），与 `bestSource` 保持一致。
 */
export async function probeMirror(openlistUrl: string, r2Url = ''): Promise<MediaSource> {
  // 注意不要用与模块函数 `reachable` 同名的局部常量——会遮蔽它并触发 TDZ。
  const rec: SourceRecord = {
    at: Date.now(),
    net: localNetFingerprint(),
    ...(knownIp ? { ip: knownIp } : {}),
  }

  if (openlistUrl) {
    rec.openlist = await reachable(openlistUrl)
    // 探测（自带 3 次尝试）是最权威的判据：探通了顺手解除渲染层负标记，
    // 不让一条单次失败的记忆压住刚确认过的可达结论。
    if (rec.openlist) brokenMarks.delete('mirror2')
  }
  // 首选已通就不再探次选：那一级的结论留空（undefined），别把「没探」记成「不可达」——
  // 记错了会让首选中途失效时直接掉到源站，白白绕开还能用的 R2。
  if (!rec.openlist && r2Url) {
    rec.r2 = await reachable(r2Url)
    if (rec.r2) brokenMarks.delete('mirror')
  }

  writeRecord(rec)
  return bestSource(rec)
}

/**
 * 渲染层兜底：某一级加载失败时给它记一条**内存**负标记，调用方沿链退一级。
 * **只标失败的那一级**——OpenList 挂了不连累 R2，它正是链上该顶上的下一级。
 *
 * 与探测结论的本质区别：渲染失败可能只是一次瞬时错误（连接被掐断、CDN 抖动），
 * 因此标记不写 localStorage——冷却窗口（`BROKEN_COOLDOWN_MS`）内
 * `cachedImageSource` 绕开该级，窗口一过该级自动恢复资格，等于免费重试；
 * 持久化的 `openlist=false` / `r2=false` 只由 `probeMirror` 写入（它自带 3 次尝试）。
 */
export function markUnreachable(tier: MirrorTier): void {
  brokenMarks.set(tier, Date.now())
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
  // 换网（典型是代理开/关）意味着旧网络下的渲染失败也一并作废：负标记跟着记录一起清。
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
    // `ip=1.2.3.4` 换行其它字段；直接取 ip= 之后那一段，不依赖 JSON 结构。
    const m = /\bip=(\S+)/.exec(text)
    return m?.[1]
  } catch {
    return undefined
  } finally {
    /* timer 在下面统一清理 */
  }
}

/** 单次可达性探测：非 2xx/206、超时、网络错误都算这一次失败。 */
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
    /* 见 fetchIp：超时用完即弃 */
  }
}

/**
 * 镜像侧可达性判定：**试满 3 次才认输**——普通一次 + 退避
 * `PROBE_RETRY_DELAY_MS` 后一次 + 快速复核一次。误判的代价是整条持久记录
 * 在 TTL 内都绕开该级，比多等一秒重试贵得多；串行探测只在冷启动付这个时间，
 * 首选可达（常态）第一次就返回，不为常态多花任何请求。
 */
async function reachable(url: string): Promise<boolean> {
  if (await reachableOnce(url)) return true
  await delay(PROBE_RETRY_DELAY_MS)
  if (await reachableOnce(url)) return true
  return reachableOnce(url)
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/* ---------------- 地址拼装 ---------------- */

/**
 * 按已定的来源选地址；`source` 尚未确定（null）时用源站，
 * 因为源站是权威副本——不确定就先出不会出错的那一侧，绝不猜。
 * 某一级地址为空时继续往后落，避免「选了替补却没有替补地址」直接渲染成空。
 */
export function pickImageUrl(
  local: string,
  mirror: string,
  mirror2: string,
  source: MediaSource | null,
): string {
  if (source === 'mirror' && mirror) return mirror
  if (source === 'mirror2' && mirror2) return mirror2
  // 落到这里说明来源未定或选中的那一级没地址：按 源站 → 首选 → 次选 兜，
  // 顺序与镜像链一致（OpenList 优先于 R2），不是随便取一个非空值。
  return local || mirror2 || mirror
}

/**
 * 当前来源之后还能退的地址，**按优先级**排列（渲染层兜底链）。
 * 首选 OpenList → [R2, 源站]；次选 R2 → [源站]；已是源站则空。
 * 空串一律剔除：没配镜像时不该在链里留一个注定失败的占位。
 */
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

/** 测试与手动重测用：清空本机记录、内存负标记与 IP 状态。 */
export function resetImageSourceState(): void {
  clearRecord()
  brokenMarks.clear()
  knownIp = undefined
  networkCheck = null
}

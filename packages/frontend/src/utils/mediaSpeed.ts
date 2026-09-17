/**
 * 详情页大图源选择：在本站（源站）与 R2 之间测速，谁快用谁。
 *
 * 设计要点：
 * - 只在详情页对大图生效；缩略图保持纯本地，不参与选择。
 * - 偏好按「本机 + 本次会话」缓存（默认 30 分钟）：测速是真实网络请求，
 *   不能每张图都测；缓存在窗口内让同一轮浏览里的所有图片保持一致。
 * - 任何一侧没有地址、测速失败、环境不支持时不猜：宁可退回源站，
 *   也不要把用户引到一个可能 404 的地址上。
 */

/** 缓存有效期：30 分钟内不再重复测速。 */
export const SPEED_PREF_TTL_MS = 30 * 60 * 1000

/** 单侧探测字节数：够量出吞吐，又不至于是可感知的额外流量。 */
const SAMPLE_BYTES = 32768

/** 单次探测超时：超过这个时间就没必要等，直接判这一侧更慢。 */
const PROBE_TIMEOUT_MS = 4000

/**
 * 打分权重：延迟（首字节）比吞吐更影响「点开即见」的体感，
 * 因此把两者折算成一个可比较的毫秒分数。
 */
const LATENCY_WEIGHT = 0.4
const THROUGHPUT_WEIGHT = 0.6

export type MediaSource = 'origin' | 'mirror'

export interface SourceCandidate {
  source: MediaSource
  url: string
}

export interface SpeedProbeResult {
  winner: MediaSource
  scores: Partial<Record<MediaSource, number>>
}

/** 探测结果缓存（按候选地址组合区分）。 */
const cache = new Map<string, { at: number; result: SpeedProbeResult }>()

function cacheKey(local: string, mirror: string): string {
  return 'media-speed:' + local + '::' + mirror
}

/** 清空缓存（测试与手动重测用）。 */
export function resetMediaSpeedCache(): void {
  cache.clear()
}

/**
 * 对候选地址发起一次真实请求并计时。
 * cache: 'no-store' 确保拿到的是网络真实耗时，不被浏览器缓存美化。
 * 兼容性：老浏览器没有 AbortController 时退化为「等到自然结束」，不报错。
 */
async function probe(
  candidate: SourceCandidate,
  clock: () => number = () => performance.now(),
): Promise<number> {
  const started = clock()
  const hasAbort = typeof AbortController !== 'undefined'
  const controller = hasAbort ? new AbortController() : null
  const timer = controller ? setTimeout(() => controller.abort(), PROBE_TIMEOUT_MS) : null
  try {
    const headers: Record<string, string> = {}
    if (candidate.source === 'mirror') {
      // 只多要一段字节，语义上仍是「取这张图」，不改变返回类型。
      headers.Range = 'bytes=0-' + (SAMPLE_BYTES - 1)
    }
    const resp = await fetch(candidate.url, {
      cache: 'no-store',
      signal: controller ? controller.signal : undefined,
      headers,
    })
    if (!resp.ok && resp.status !== 206) return Number.POSITIVE_INFINITY
    const body = await resp.arrayBuffer()
    const elapsed = clock() - started
    const bytes = body.byteLength
    // 源站不支持 Range 时拿到整图，用实际字节数同样能算出吞吐。
    const perByte = bytes > 0 ? elapsed / bytes : elapsed
    return elapsed * LATENCY_WEIGHT + perByte * bytes * THROUGHPUT_WEIGHT
  } catch {
    return Number.POSITIVE_INFINITY
  } finally {
    if (timer) clearTimeout(timer)
  }
}

/**
 * 在源站与 R2 之间测速，返回更快的一侧。
 * 只给一侧地址时直接返回该侧（不测速，也不编造结果）。
 */
export async function probeMediaSources(
  local: string,
  mirror: string,
  now: () => number = Date.now,
  // 每侧一个独立时钟：两侧是并发探测的，共享时钟会互相污染耗时读数。
  clockFor: (url: string) => () => number = () => () => performance.now(),
): Promise<SpeedProbeResult> {
  if (!mirror) return { winner: 'origin', scores: {} }
  if (!local) return { winner: 'mirror', scores: {} }

  const key = cacheKey(local, mirror)
  const hit = cache.get(key)
  if (hit && now() - hit.at < SPEED_PREF_TTL_MS) return hit.result

  const candidates: SourceCandidate[] = [
    { source: 'origin', url: local },
    { source: 'mirror', url: mirror },
  ]
  const scores: Partial<Record<MediaSource, number>> = {}
  await Promise.all(
    candidates.map(async (c) => {
      scores[c.source] = await probe(c, clockFor(c.url))
    }),
  )

  const originScore = scores.origin ?? Number.POSITIVE_INFINITY
  const mirrorScore = scores.mirror ?? Number.POSITIVE_INFINITY
  // 两侧都不可达时退回源站：源站是权威副本，R2 只是备份。
  const winner: MediaSource = mirrorScore < originScore ? 'mirror' : 'origin'

  const result: SpeedProbeResult = { winner, scores }
  cache.set(key, { at: now(), result })
  return result
}

/**
 * 同步取已缓存的偏好；没有缓存返回 null
 *（调用方先用源站渲染，测速完成后再切源，避免白屏等待）。
 */
export function cachedMediaSource(
  local: string,
  mirror: string,
  now: () => number = Date.now,
): MediaSource | null {
  if (!mirror) return 'origin'
  const key = cacheKey(local, mirror)
  const hit = cache.get(key)
  if (!hit) return null
  if (now() - hit.at >= SPEED_PREF_TTL_MS) {
    cache.delete(key)
    return null
  }
  return hit.result.winner
}

/**
 * 当前应使用的大图地址：偏好为 R2 且确有 R2 地址时用 R2，其余一律源站。
 */
export function pickImageUrl(local: string, mirror: string, source: MediaSource | null): string {
  if (source === 'mirror' && mirror) return mirror
  return local || mirror
}

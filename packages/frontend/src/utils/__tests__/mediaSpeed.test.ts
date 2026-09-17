import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  SPEED_PREF_TTL_MS,
  cachedMediaSource,
  pickImageUrl,
  probeMediaSources,
  resetMediaSpeedCache,
} from '@/utils/mediaSpeed'

const LOCAL = '/uploads/ab12.webp'
const MIRROR = 'https://file.example.com/ab12.webp'

/**
 * 按 URL 分别计时的可控时钟。
 * 探测是并发发起的，若共用一个全局时钟，先返回的一侧会把另一侧的耗时算大；
 * 真实世界里每次请求各自计时，这里也必须如此，否则测出来的是假象。
 */
/**
 * makeClock 提供两套独立计时：
 * - now() 是墙钟，只用于缓存的写入/过期时间戳；
 * - clockFor(url) 是每侧独立的网络耗时钟，保证并发探测互不污染。
 */
function makeClock() {
  const elapsed = new Map<string, { t: number }>()
  let wall = 1000
  return {
    now: () => wall,
    advanceWall: (ms: number) => {
      wall += ms
    },
    clockFor: (url: string) => {
      const slot = elapsed.get(url) ?? { t: 0 }
      elapsed.set(url, slot)
      return () => slot.t
    },
    advance: (url: string, ms: number) => {
      const slot = elapsed.get(url) ?? { t: 0 }
      slot.t += ms
      elapsed.set(url, slot)
    },
    reset: (url: string) => elapsed.set(url, { t: 0 }),
  }
}

/** 构造 fetch 替身：按 URL 决定耗时与是否失败，并推进假时钟。 */
function mockFetch(
  plan: Record<string, { ms: number; ok?: boolean; bytes?: number }>,
  clock: ReturnType<typeof makeClock>,
) {
  const calls: string[] = []
  const fn = vi.fn(async (url: string) => {
    calls.push(url)
    const rule = plan[url]
    clock.advance(url, rule?.ms ?? 1)
    // 模拟真实的异步 IO：让出微任务，使另一侧的分支得以交错。
    await Promise.resolve()
    if (rule && rule.ok === false) {
      return {
        ok: false,
        status: 500,
        arrayBuffer: async () => new ArrayBuffer(0),
      } as unknown as Response
    }
    return {
      ok: true,
      status: 200,
      arrayBuffer: async () => new ArrayBuffer(rule?.bytes ?? 1024),
    } as unknown as Response
  })
  vi.stubGlobal('fetch', fn)
  return { fn, calls }
}

beforeEach(() => {
  resetMediaSpeedCache()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('probeMediaSources', () => {
  it('没有任何地址时直接返回源站，且不发请求', async () => {
    const clock = makeClock()
    const { fn } = mockFetch({}, clock)
    const res = await probeMediaSources('', '', clock.now, clock.clockFor)
    expect(res.winner).toBe('origin')
    expect(fn).not.toHaveBeenCalled()
  })

  it('只有源站地址时不测速，直接选源站', async () => {
    const clock = makeClock()
    const { fn } = mockFetch({}, clock)
    const res = await probeMediaSources(LOCAL, '', clock.now, clock.clockFor)
    expect(res.winner).toBe('origin')
    expect(fn).not.toHaveBeenCalled()
  })

  it('只有 R2 地址时直接选 R2', async () => {
    const clock = makeClock()
    const res = await probeMediaSources('', MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('mirror')
  })

  it('两侧都探测，返回更快的一侧', async () => {
    const clock = makeClock()
    const { calls } = mockFetch({ [LOCAL]: { ms: 100 }, [MIRROR]: { ms: 20 } }, clock)
    const res = await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('mirror')
    expect(new Set(calls)).toEqual(new Set([LOCAL, MIRROR]))
  })

  it('源站更快时选出源站', async () => {
    const clock = makeClock()
    mockFetch({ [LOCAL]: { ms: 30 }, [MIRROR]: { ms: 300 } }, clock)
    const res = await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('origin')
  })

  it('某一侧请求失败时判该侧不可达', async () => {
    const clock = makeClock()
    mockFetch({ [LOCAL]: { ms: 10, ok: false }, [MIRROR]: { ms: 200 } }, clock)
    const res = await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('mirror')
    expect(res.scores.origin).toBe(Number.POSITIVE_INFINITY)
  })

  it('两侧都不可达时退回源站（R2 只是备份，不能把用户带偏）', async () => {
    const clock = makeClock()
    mockFetch({ [LOCAL]: { ms: 10, ok: false }, [MIRROR]: { ms: 10, ok: false } }, clock)
    const res = await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('origin')
  })

  it('网络异常（抛出）视为不可达，不影响另一侧判定', async () => {
    const clock = makeClock()
    vi.stubGlobal(
      'fetch',
      vi.fn(async (url: string) => {
        clock.reset(url)
        clock.advance(url, 5)
        await Promise.resolve()
        if (url === LOCAL) throw new Error('offline')
        return {
          ok: true,
          status: 200,
          arrayBuffer: async () => new ArrayBuffer(64),
        } as unknown as Response
      }),
    )
    const res = await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(res.winner).toBe('mirror')
  })

  it('TTL 内复用缓存，不重复发请求', async () => {
    const clock = makeClock()
    const { fn } = mockFetch({ [LOCAL]: { ms: 10 }, [MIRROR]: { ms: 5 } }, clock)
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    const afterFirst = fn.mock.calls.length
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(fn.mock.calls.length).toBe(afterFirst)
  })

  it('超过 TTL 后重新测速', async () => {
    const clock = makeClock()
    const { fn } = mockFetch({ [LOCAL]: { ms: 10 }, [MIRROR]: { ms: 5 } }, clock)
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    const afterFirst = fn.mock.calls.length
    clock.advanceWall(SPEED_PREF_TTL_MS + 1)
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(fn.mock.calls.length).toBeGreaterThan(afterFirst)
  })
})

describe('cachedMediaSource', () => {
  it('没有 R2 地址时直接返回源站', () => {
    expect(cachedMediaSource(LOCAL, '')).toBe('origin')
  })

  it('未测速时返回 null（调用方先用源站渲染）', () => {
    expect(cachedMediaSource(LOCAL, MIRROR)).toBeNull()
  })

  it('测速后返回缓存里的胜出方', async () => {
    const clock = makeClock()
    mockFetch({ [LOCAL]: { ms: 500 }, [MIRROR]: { ms: 5 } }, clock)
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    // 与探测使用同一时钟，才是在校验「TTL 内可读」这件事本身。
    expect(cachedMediaSource(LOCAL, MIRROR, clock.now)).toBe('mirror')
  })

  it('缓存过期后重新变为 null', async () => {
    const clock = makeClock()
    mockFetch({ [LOCAL]: { ms: 5 }, [MIRROR]: { ms: 5 } }, clock)
    await probeMediaSources(LOCAL, MIRROR, clock.now, clock.clockFor)
    expect(cachedMediaSource(LOCAL, MIRROR, clock.now)).not.toBeNull()
    clock.advanceWall(SPEED_PREF_TTL_MS + 1)
    expect(cachedMediaSource(LOCAL, MIRROR, clock.now)).toBeNull()
  })
})

describe('pickImageUrl', () => {
  it('偏好为 R2 且确有地址时用 R2', () => {
    expect(pickImageUrl(LOCAL, MIRROR, 'mirror')).toBe(MIRROR)
  })

  it('偏好为源站时用源站', () => {
    expect(pickImageUrl(LOCAL, MIRROR, 'origin')).toBe(LOCAL)
  })

  it('还没测出结果（null）时用源站', () => {
    expect(pickImageUrl(LOCAL, MIRROR, null)).toBe(LOCAL)
  })

  it('没有源站地址时退回 R2', () => {
    expect(pickImageUrl('', MIRROR, null)).toBe(MIRROR)
  })
})

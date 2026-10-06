import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  BROKEN_COOLDOWN_MS,
  cachedImageSource,
  fallbackChain,
  localNetFingerprint,
  markUnreachable,
  pickImageUrl,
  probeMirror,
  recheckNetwork,
  SOURCE_TTL_MS,
  resetImageSourceState,
} from '@/utils/imageSource'

const R2 = 'https://img.xqecz.bond/uploads/ab12.webp'
// 首选镜像（自建 OpenList）的直链：免签，因此不带任何查询参数。
// 注意 URL 里那个 `/d` 是实例的免签直链前缀，少了它 OpenList 只会回 SPA 首页 HTML。
const QN = 'https://drive.xiey.work/d/uploads/ab12.webp'
const LOCAL = 'https://api39.xiey.work/uploads/ab12.webp'
const STORE_KEY = 'xqecz:image-source-v3'

function seed(rec: { openlist?: boolean; r2?: boolean; at?: number; net?: string; ip?: string }) {
  localStorage.setItem(
    STORE_KEY,
    JSON.stringify({
      at: rec.at ?? Date.now(),
      net: rec.net ?? localNetFingerprint(),
      ...rec,
    }),
  )
}

function readStore(): Record<string, unknown> | null {
  const raw = localStorage.getItem(STORE_KEY)
  return raw ? JSON.parse(raw) : null
}

/** 让 fetch 按 url 决定响应；同时记录调用参数供断言。 */
function stubFetch(handler: (url: string, init?: RequestInit) => Response | never) {
  const calls: { url: string; init?: RequestInit }[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      return handler(url, init)
    }),
  )
  return calls
}

beforeEach(() => {
  resetImageSourceState()
  localStorage.clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('cachedImageSource（同步读本机结论）', () => {
  it('没有记录时返回 null，交由调用方去探测', () => {
    expect(cachedImageSource()).toBeNull()
  })

  it('首选镜像（OpenList）可达 → mirror2', () => {
    seed({ openlist: true })
    expect(cachedImageSource()).toBe('mirror2')
  })

  it('首发不可达但 R2 可达 → mirror（用 R2，不回源站）', () => {
    seed({ openlist: false, r2: true })
    expect(cachedImageSource()).toBe('mirror')
  })

  it('两级都不可达 → origin', () => {
    seed({ openlist: false, r2: false })
    expect(cachedImageSource()).toBe('origin')
  })

  it('本地网络指纹变了 → 视作换网，返回 null 重新探测', () => {
    seed({ openlist: true, net: '0||-1|-1' })
    expect(cachedImageSource()).toBeNull()
  })

  it('超过有效期 → 返回 null 重新探测', () => {
    seed({ openlist: true, at: Date.now() - SOURCE_TTL_MS - 1 })
    expect(cachedImageSource()).toBeNull()
  })
})

describe('probeMirror（按优先级串行短路地探镜像侧）', () => {
  it('首选（OpenList）可达 → mirror2，且不再探次选（省一次请求）', async () => {
    const calls = stubFetch(() => new Response(null, { status: 206 }))
    await expect(probeMirror(QN, R2)).resolves.toBe('mirror2')
    expect(calls).toHaveLength(1)
    expect(calls[0].url).toBe(QN)
    expect(readStore()?.openlist).toBe(true)
  })

  it('首选已通时不把次选记成不可达（否则首选失效会直接掉到源站）', async () => {
    stubFetch(() => new Response(null, { status: 206 }))
    await probeMirror(QN, R2)
    expect(readStore()?.r2).toBeUndefined()
  })

  it('首选不通、次选可达 → mirror', async () => {
    stubFetch((url) => new Response(null, { status: url === R2 ? 206 : 500 }))
    await expect(probeMirror(QN, R2)).resolves.toBe('mirror')
    expect(readStore()?.openlist).toBe(false)
    expect(readStore()?.r2).toBe(true)
  })

  it('两级都不通 → origin', async () => {
    stubFetch(() => new Response(null, { status: 500 }))
    await expect(probeMirror(QN, R2)).resolves.toBe('origin')
    expect(readStore()?.openlist).toBe(false)
    expect(readStore()?.r2).toBe(false)
  })

  it('请求抛错（网络错误 / 超时中断）同样判为不可达', async () => {
    stubFetch(() => {
      throw new Error('blocked by network')
    })
    await expect(probeMirror(QN, R2)).resolves.toBe('origin')
  })

  it('没配次选时只探首选（失败也试满 3 次）', async () => {
    const calls = stubFetch(() => new Response(null, { status: 500 }))
    await expect(probeMirror(QN)).resolves.toBe('origin')
    expect(calls).toHaveLength(3)
    expect(calls.every((c) => c.url === QN)).toBe(true)
    expect(readStore()?.r2).toBeUndefined()
  })

  it('首选未配置时直接探次选，不空转', async () => {
    const calls = stubFetch(() => new Response(null, { status: 206 }))
    await expect(probeMirror('', R2)).resolves.toBe('mirror')
    expect(calls.map((c) => c.url)).toEqual([R2])
  })

  it('从不请求源站（源站是回退目标，探它纯浪费）', async () => {
    const calls = stubFetch(() => new Response(null, { status: 200 }))
    await probeMirror(QN, R2)
    expect(calls.map((c) => c.url)).not.toContain(LOCAL)
  })

  it('探测带 Range 头且绕过缓存', async () => {
    const calls = stubFetch(() => new Response(null, { status: 206 }))
    await probeMirror(QN)
    expect(calls[0].init?.headers).toMatchObject({ Range: 'bytes=0-32767' })
    expect(calls[0].init?.cache).toBe('no-store')
  })

  it('单次失败不判死：退避重试成功仍算可达（瞬时抖动不拉黑）', async () => {
    const hits: Record<string, number> = {}
    stubFetch((url) => {
      hits[url] = (hits[url] ?? 0) + 1
      if (url === QN) return new Response(null, { status: hits[url] === 1 ? 500 : 206 })
      return new Response(null, { status: 206 })
    })
    await expect(probeMirror(QN, R2)).resolves.toBe('mirror2')
    expect(hits[QN]).toBe(2)
  })

  it('两连败后第三次复核通过也算可达', async () => {
    const hits: Record<string, number> = {}
    stubFetch((url) => {
      hits[url] = (hits[url] ?? 0) + 1
      if (url === QN) return new Response(null, { status: hits[url] < 3 ? 500 : 206 })
      return new Response(null, { status: 206 })
    })
    await expect(probeMirror(QN, R2)).resolves.toBe('mirror2')
    expect(hits[QN]).toBe(3)
  })

  it('三连败才判不可达：每级恰好 3 次尝试', async () => {
    const calls = stubFetch(() => new Response(null, { status: 500 }))
    await expect(probeMirror(QN, R2)).resolves.toBe('origin')
    expect(calls.filter((c) => c.url === QN)).toHaveLength(3)
    expect(calls.filter((c) => c.url === R2)).toHaveLength(3)
  })
})

describe('recheckNetwork（IP 变了才推翻旧结论）', () => {
  it('本机没有记录 → 无需校验', async () => {
    const calls = stubFetch(() => new Response('ip=1.2.3.4'))
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(calls).toHaveLength(0)
  })

  it('公网 IP 与记录一致 → 不动记录', async () => {
    seed({ r2: true, ip: '1.2.3.4' })
    stubFetch(() => new Response('ip=1.2.3.4\nloc=CN\n'))
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()).not.toBeNull()
  })

  it('公网 IP 变了（典型：代理开/关）→ 清掉记录，要求重新探测', async () => {
    seed({ r2: true, ip: '1.2.3.4' })
    stubFetch(() => new Response('ip=5.6.7.8\nloc=CN\n'))
    await expect(recheckNetwork()).resolves.toBe(true)
    expect(localStorage.getItem(STORE_KEY)).toBeNull()
  })

  it('换网同时清掉渲染层负标记（旧网络下的失败不该拖累新网络）', async () => {
    seed({ openlist: true, r2: true, ip: '1.2.3.4' })
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('mirror')
    stubFetch(() => new Response('ip=5.6.7.8\nloc=CN\n'))
    await expect(recheckNetwork()).resolves.toBe(true)
    // 记录已被换网清掉，负标记也一并作废；重新 seed 后 mirror2 恢复资格。
    seed({ openlist: true, r2: true, ip: '5.6.7.8' })
    expect(cachedImageSource()).toBe('mirror2')
  })

  it('IP 接口失败时保持旧结论——拿不准就不推翻', async () => {
    seed({ r2: true, ip: '1.2.3.4' })
    stubFetch(() => {
      throw new Error('ip echo down')
    })
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()).not.toBeNull()
  })

  it('记录还没有 IP（首次）→ 补上，但不算网络变化', async () => {
    seed({ r2: false })
    stubFetch(() => new Response('ip=9.9.9.9'))
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()?.ip).toBe('9.9.9.9')
  })

  it('同一次页面会话内只取一次 IP', async () => {
    seed({ r2: true, ip: '1.2.3.4' })
    const calls = stubFetch(() => new Response('ip=1.2.3.4'))
    await recheckNetwork()
    await recheckNetwork()
    expect(calls).toHaveLength(1)
  })
})

describe('markUnreachable（渲染层兜底：内存负标记，60s 冷却）', () => {
  it('首选（OpenList）取不到 → 冷却窗口内只绕开首选，R2 可达时改判为 mirror', () => {
    seed({ openlist: true, r2: true })
    expect(cachedImageSource()).toBe('mirror2')
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('mirror')
    // 不落盘：持久结论保持原样，判死只归探测管。
    expect(readStore()?.openlist).toBe(true)
    expect(readStore()?.r2).toBe(true)
  })

  it('两级都取不到 → 冷却窗口内 origin', () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror2')
    markUnreachable('mirror')
    expect(cachedImageSource()).toBe('origin')
  })

  it('R2 取不到不会连累首选（OpenList）的资格', () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror')
    expect(cachedImageSource()).toBe('mirror2')
    expect(readStore()?.r2).toBe(true)
  })

  it('冷却窗口一过，该级自动恢复资格（这就是宽裕的重试）', () => {
    seed({ openlist: true, r2: false })
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('origin')
    expect(cachedImageSource(Date.now() + BROKEN_COOLDOWN_MS + 1)).toBe('mirror2')
  })

  it('冷却窗口内重提同一级只是续期，不会把失败永久化', () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror2')
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('mirror')
    expect(readStore()?.openlist).toBe(true)
  })

  it('没有记录时负标记也安全（最多降到 origin，不抛错）', () => {
    markUnreachable('mirror')
    expect(cachedImageSource()).toBeNull()
  })

  it('探测探通后解除该级的负标记', async () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('mirror')
    stubFetch(() => new Response(null, { status: 206 }))
    await probeMirror(QN)
    expect(cachedImageSource()).toBe('mirror2')
  })
})

describe('pickImageUrl', () => {
  it('来源为 mirror（R2）且确有地址时用 R2', () => {
    expect(pickImageUrl(LOCAL, R2, QN, 'mirror')).toBe(R2)
  })

  it('来源为 mirror2（OpenList）时用首选', () => {
    expect(pickImageUrl(LOCAL, R2, QN, 'mirror2')).toBe(QN)
  })

  it('来源为 origin 时用源站', () => {
    expect(pickImageUrl(LOCAL, R2, QN, 'origin')).toBe(LOCAL)
  })

  it('尚未确定来源（null）时用源站——不确定就不猜', () => {
    expect(pickImageUrl(LOCAL, R2, QN, null)).toBe(LOCAL)
  })

  it('选中的那一级没地址时继续往后落，不渲染成空', () => {
    expect(pickImageUrl(LOCAL, '', '', 'mirror')).toBe(LOCAL)
    expect(pickImageUrl('', R2, '', 'mirror2')).toBe(R2)
  })

  it('没有源站地址时才用镜像，且优先首选（OpenList）', () => {
    expect(pickImageUrl('', R2, QN, null)).toBe(QN)
  })
})

describe('fallbackChain（渲染层逐级回退）', () => {
  it('首选 OpenList → [R2, 源站]', () => {
    expect(fallbackChain(LOCAL, R2, QN, 'mirror2')).toEqual([R2, LOCAL])
  })

  it('没配 R2 时链里不留注定失败的占位', () => {
    expect(fallbackChain(LOCAL, '', QN, 'mirror2')).toEqual([LOCAL])
  })

  it('R2 → [源站]', () => {
    expect(fallbackChain(LOCAL, R2, QN, 'mirror')).toEqual([LOCAL])
  })

  it('已在源站则无可退', () => {
    expect(fallbackChain(LOCAL, R2, QN, 'origin')).toEqual([])
    expect(fallbackChain(LOCAL, R2, QN, null)).toEqual([])
  })
})

describe('localNetFingerprint', () => {
  it('字段简略且稳定（本地生成，不上传）', () => {
    const a = localNetFingerprint()
    expect(typeof a).toBe('string')
    expect(a).toBe(localNetFingerprint())
    // 段位：onLine | effectiveType | rtt 桶 | downlink 桶
    expect(a.split('|')).toHaveLength(4)
  })
})

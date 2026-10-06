import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
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

  it('没配次选时只探首选', async () => {
    const calls = stubFetch(() => new Response(null, { status: 500 }))
    await expect(probeMirror(QN)).resolves.toBe('origin')
    expect(calls).toHaveLength(1)
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

describe('markUnreachable（渲染层兜底的回写）', () => {
  it('首选（OpenList）取不到 → 只标首选，R2 可达时改判为 mirror', async () => {
    seed({ openlist: true, r2: true })
    expect(cachedImageSource()).toBe('mirror2')
    markUnreachable('mirror2')
    expect(cachedImageSource()).toBe('mirror')
    expect(readStore()?.openlist).toBe(false)
    expect(readStore()?.r2).toBe(true)
  })

  it('两级都取不到 → origin', () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror2')
    markUnreachable('mirror')
    expect(cachedImageSource()).toBe('origin')
  })

  it('R2 取不到不会连累首选（OpenList）的结论', () => {
    seed({ openlist: false, r2: true })
    markUnreachable('mirror')
    expect(readStore()?.openlist).toBe(false)
    expect(cachedImageSource()).toBe('origin')
  })

  it('首选结论保留、只标死 R2 时仍回到首选', () => {
    seed({ openlist: true, r2: true })
    markUnreachable('mirror')
    expect(readStore()?.openlist).toBe(true)
    expect(cachedImageSource()).toBe('mirror2')
  })

  it('没有任何记录时兜底标记也安全，且写在「当前」网络指纹下', () => {
    markUnreachable('mirror')
    expect(cachedImageSource()).toBe('origin')
    expect(readStore()?.net).toBe(localNetFingerprint())
    expect(readStore()?.r2).toBe(false)
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

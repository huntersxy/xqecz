import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import {
  cachedImageSource,
  localNetFingerprint,
  markMirrorUnreachable,
  pickImageUrl,
  probeMirror,
  recheckNetwork,
  SOURCE_TTL_MS,
  resetImageSourceState,
} from '@/utils/imageSource'

const MIRROR = 'https://img.xqecz.bond/uploads/ab12.webp'
const LOCAL = 'https://api39.xiey.work/uploads/ab12.webp'
const STORE_KEY = 'xqecz:image-source'

function seed(rec: { reachable: boolean; at?: number; net?: string; ip?: string }) {
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

/** 让 fetch 返回指定状态；同时记录调用参数供断言。 */
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

  it('同一网络下记录为可达 → mirror', () => {
    seed({ reachable: true })
    expect(cachedImageSource()).toBe('mirror')
  })

  it('同一网络下记录为不可达 → origin（不用 R2）', () => {
    seed({ reachable: false })
    expect(cachedImageSource()).toBe('origin')
  })

  it('本地网络指纹变了 → 视作换网，返回 null 重新探测', () => {
    seed({ reachable: true, net: '0||-1|-1' }) // 指纹与当前不同
    expect(cachedImageSource()).toBeNull()
  })

  it('超过有效期 → 返回 null 重新探测', () => {
    seed({ reachable: true, at: Date.now() - SOURCE_TTL_MS - 1 })
    expect(cachedImageSource()).toBeNull()
  })
})

describe('probeMirror（只探镜像侧）', () => {
  it('镜像可达 → mirror，并把结论记到本机', async () => {
    stubFetch(() => new Response(null, { status: 206 }))
    await expect(probeMirror(MIRROR)).resolves.toBe('mirror')
    expect(readStore()?.reachable).toBe(true)
  })

  it('镜像不可达（HTTP 错误）→ origin，并记录为不可达', async () => {
    stubFetch(() => new Response(null, { status: 500 }))
    await expect(probeMirror(MIRROR)).resolves.toBe('origin')
    expect(readStore()?.reachable).toBe(false)
  })

  it('镜像请求抛错（网络错误 / 超时中断）→ origin', async () => {
    stubFetch(() => {
      throw new Error('blocked by network')
    })
    await expect(probeMirror(MIRROR)).resolves.toBe('origin')
  })

  it('只探测镜像地址，绝不请求源站（源站是回退目标，探它纯浪费）', async () => {
    const calls = stubFetch(() => new Response(null, { status: 200 }))
    await probeMirror(MIRROR)
    expect(calls).toHaveLength(1)
    expect(calls[0].url).toBe(MIRROR)
    expect(calls.map((c) => c.url)).not.toContain(LOCAL)
  })

  it('探测带 Range 头，只要一小段字节，不下载整图', async () => {
    const calls = stubFetch(() => new Response(null, { status: 206 }))
    await probeMirror(MIRROR)
    expect(calls[0].init?.headers).toMatchObject({ Range: 'bytes=0-32767' })
  })

  it('探测结果绕过缓存，拿到的是网络真实状态', async () => {
    const calls = stubFetch(() => new Response(null, { status: 200 }))
    await probeMirror(MIRROR)
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
    seed({ reachable: true, ip: '1.2.3.4' })
    stubFetch(() => new Response('ip=1.2.3.4\nloc=CN\n'))
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()).not.toBeNull()
  })

  it('公网 IP 变了（典型：代理开/关）→ 清掉记录，要求重新探测', async () => {
    seed({ reachable: true, ip: '1.2.3.4' })
    stubFetch(() => new Response('ip=5.6.7.8\nloc=CN\n'))
    await expect(recheckNetwork()).resolves.toBe(true)
    expect(localStorage.getItem(STORE_KEY)).toBeNull()
  })

  it('IP 接口失败时保持旧结论——拿不准就不推翻', async () => {
    seed({ reachable: true, ip: '1.2.3.4' })
    stubFetch(() => {
      throw new Error('ip echo down')
    })
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()).not.toBeNull()
  })

  it('记录还没有 IP（首次）→ 补上，但不算网络变化', async () => {
    seed({ reachable: false })
    stubFetch(() => new Response('ip=9.9.9.9'))
    await expect(recheckNetwork()).resolves.toBe(false)
    expect(readStore()?.ip).toBe('9.9.9.9')
  })

  it('同一次页面会话内只取一次 IP', async () => {
    seed({ reachable: true, ip: '1.2.3.4' })
    const calls = stubFetch(() => new Response('ip=1.2.3.4'))
    await recheckNetwork()
    await recheckNetwork()
    expect(calls).toHaveLength(1)
  })
})

describe('markMirrorUnreachable（渲染层兜底的回写）', () => {
  it('镜像取不到时标记当前网络不可达，之后同步读到 origin', async () => {
    seed({ reachable: true })
    expect(cachedImageSource()).toBe('mirror')
    markMirrorUnreachable()
    expect(cachedImageSource()).toBe('origin')
  })

  it('标记写在「当前」网络指纹下，同一网络的后续访问直接得到 origin', () => {
    markMirrorUnreachable()
    expect(cachedImageSource()).toBe('origin')
    expect(readStore()?.net).toBe(localNetFingerprint())
    expect(readStore()?.reachable).toBe(false)
  })
})

describe('pickImageUrl', () => {
  it('来源为 mirror 且确有镜像地址时用镜像', () => {
    expect(pickImageUrl(LOCAL, MIRROR, 'mirror')).toBe(MIRROR)
  })

  it('来源为 origin 时用源站', () => {
    expect(pickImageUrl(LOCAL, MIRROR, 'origin')).toBe(LOCAL)
  })

  it('尚未确定来源（null）时用源站——不确定就不猜', () => {
    expect(pickImageUrl(LOCAL, MIRROR, null)).toBe(LOCAL)
  })

  it('内容没有镜像副本时退回源站', () => {
    expect(pickImageUrl(LOCAL, '', 'mirror')).toBe(LOCAL)
  })

  it('没有源站地址时才用镜像', () => {
    expect(pickImageUrl('', MIRROR, null)).toBe(MIRROR)
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

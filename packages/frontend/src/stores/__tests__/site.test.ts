import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSiteStore } from '@/stores/site'

const api = vi.hoisted(() => ({ config: vi.fn() }))
vi.mock('@/api', () => ({ siteApi: api }))

function response(data: Record<string, unknown>) {
  return { code: 200, message: 'ok', data }
}

beforeEach(() => {
  setActivePinia(createPinia())
  document.head.innerHTML = ''
  api.config.mockReset()
})

describe('site store', () => {
  it('keeps usable defaults before the backend config arrives', () => {
    const store = useSiteStore()

    expect(store.config.name).toBe('小泉宇宙')
    expect(store.text('nav.gallery')).toBe('作品广场')
    expect(store.lines('login.storyTitle')).toEqual(['在小泉宇宙，', '写下你的', '第一颗星球'])
    expect(store.format('characters.detailMeta', { title: '小泉' })).toContain('小泉 · 角色设定')
  })

  it('loads the public config, resolves text elements and updates document metadata', async () => {
    api.config.mockResolvedValue(
      response({
        name: '模板宇宙',
        short_name: '模板',
        tagline: '自定义画廊',
        browser_title: '{name} · {tagline}',
        description: '新的站点描述',
        logo_url: '/images/site/logo-1.webp',
        favicon_url: '/images/site/favicon-1.png',
        friend_links: [{ name: '友站', url: 'https://example.com/', icon_url: '' }],
        texts: {
          'nav.gallery': '作品集',
          'login.titleRegister': '加入{name}',
        },
      }),
    )
    const store = useSiteStore()

    await store.load()

    expect(store.loaded).toBe(true)
    expect(store.config.name).toBe('模板宇宙')
    expect(store.text('nav.gallery')).toBe('作品集')
    expect(store.text('login.titleRegister')).toBe('加入模板宇宙')
    expect(store.text('home.featuredTitle')).toBe('正在被看见')
    expect(store.config.friend_links).toHaveLength(1)
    expect(store.baseTitle).toBe('模板宇宙 · 自定义画廊')
    expect(document.title).toBe('模板宇宙 · 自定义画廊')
    expect(document.querySelector('meta[name="description"]')?.getAttribute('content')).toBe('新的站点描述')
    expect(document.querySelector('link[rel="icon"]')?.getAttribute('href')).toBe('/images/site/favicon-1.png')
    expect(document.querySelector('link[rel="manifest"]')?.getAttribute('href')).toBe('/api/site/manifest')
  })

  it('accepts an admin save without refetching and keeps placeholders resolved', () => {
    const store = useSiteStore()
    store.applyConfig({
      name: '后台站名',
      short_name: '后台',
      tagline: '副标题',
      browser_title: '{name} · {tagline}',
      description: '描述',
      logo_url: '',
      favicon_url: '/favicon.webp',
      friend_links: [],
      texts: {
        'login.submitLogin': '回到{name}',
      },
    })

    expect(store.text('login.submitLogin')).toBe('回到后台站名')
    expect(store.baseTitle).toBe('后台站名 · 副标题')
    expect(api.config).not.toHaveBeenCalled()
  })

  it('falls back to defaults when the public config request fails', async () => {
    api.config.mockRejectedValue(new Error('network down'))
    const store = useSiteStore()

    await store.load()

    expect(store.loaded).toBe(true)
    expect(store.error).toBe('network down')
    expect(store.config.name).toBe('小泉宇宙')
    expect(store.text('nav.gallery')).toBe('作品广场')
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import AdminSiteSettings from '@/components/admin/AdminSiteSettings.vue'
import { DEFAULT_SITE_TEXTS } from '@/utils/siteText'
import type { SiteSettings } from '@/types'

const mocks = vi.hoisted(() => ({
  settings: vi.fn(),
  save: vi.fn(),
  upload: vi.fn(),
  success: vi.fn(),
}))

vi.mock('@/api', () => ({
  adminApi: {
    siteSettings: mocks.settings,
    saveSiteSettings: mocks.save,
    uploadSiteAsset: mocks.upload,
  },
  siteApi: { config: vi.fn() },
}))
vi.mock('@/composables/useToast', () => ({ toast: { success: mocks.success, error: vi.fn() } }))

const site: SiteSettings = {
  name: '测试站',
  short_name: '测试',
  tagline: '副标题',
  browser_title: '{name} · {tagline}',
  description: '测试描述',
  logo_url: '',
  favicon_url: '/favicon.webp',
  friend_links: [],
  texts: { ...DEFAULT_SITE_TEXTS },
  elements: [
    { key: 'nav.gallery', label: '作品广场标签', group: '导航', default: '作品广场', multiline: false, max_length: 20 },
    { key: 'login.storyText', label: '登录页故事说明', group: '登录', default: '上传你的作品。', multiline: true, max_length: 300 },
  ],
}

const mounted: ReturnType<typeof mount>[] = []
function create() {
  const wrapper = mount(AdminSiteSettings, { global: { plugins: [createPinia()] } })
  mounted.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal(
    'matchMedia',
    vi.fn((media: string) => ({
      matches: false,
      media,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  )
  mocks.settings.mockResolvedValue({ code: 200, data: { ...site } })
  mocks.save.mockResolvedValue({ code: 200, data: { ...site, name: '新站名' } })
  mocks.upload.mockResolvedValue({ code: 200, data: { kind: 'logo', url: '/images/site/logo-1.webp' } })
})

afterEach(() => {
  mounted.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.unstubAllGlobals()
})

describe('AdminSiteSettings', () => {
  it('renders brand fields, the server text catalog and a live preview', async () => {
    const wrapper = create()
    await flushPromises()

    expect(wrapper.text()).toContain('品牌与 SEO')
    expect(wrapper.text()).toContain('文本元素')
    expect(wrapper.text()).toContain('导航')
    expect(wrapper.text()).toContain('登录')
    expect(wrapper.text()).toContain('测试站')
    expect(wrapper.text()).toContain('测试站 · 副标题')
    expect(wrapper.find('.site-preview').exists()).toBe(true)
  })

  it('sends the edited config and refreshes the saved snapshot', async () => {
    const wrapper = create()
    await flushPromises()

    const nameInput = wrapper
      .findAll('input')
      .find((input) => (input.element as HTMLInputElement).value === '测试站')
    expect(nameInput).toBeTruthy()
    await nameInput!.setValue('新站名')

    const saveButton = wrapper.findAll('button').find((button) => button.text() === '保存更改')
    await saveButton!.trigger('click')
    await flushPromises()

    expect(mocks.save).toHaveBeenCalledWith(
      expect.objectContaining({ name: '新站名', texts: expect.any(Object) }),
      expect.any(AbortSignal),
    )
    expect(mocks.success).toHaveBeenCalledWith('站点外观已保存')
    expect(wrapper.text()).toContain('与线上一致')
  })

  it('uploads a logo and previews it before saving', async () => {
    const wrapper = create()
    await flushPromises()

    const logoInput = wrapper.find('input[type="file"]')
    const file = new File([new Uint8Array([1, 2, 3])], 'logo.png', { type: 'image/png' })
    Object.defineProperty(logoInput.element, 'files', { value: [file], configurable: true })
    await logoInput.trigger('change')
    await flushPromises()

    expect(mocks.upload).toHaveBeenCalledWith('logo', file, expect.any(AbortSignal))
    expect(mocks.success).toHaveBeenCalledWith('图片已上传，保存后生效')
    expect(wrapper.find('.site-preview-brand img').exists()).toBe(true)
  })

  it('resets one text element back to the server default', async () => {
    const wrapper = create()
    await flushPromises()

    const input = wrapper.findAll('input').find((item) => (item.element as HTMLInputElement).value === '作品广场')
    expect(input).toBeTruthy()
    await input!.setValue('自定义画廊')

    const resetButton = wrapper.findAll('button').find((button) => button.text() === '恢复默认')
    await resetButton!.trigger('click')

    const restored = wrapper.findAll('input').find((item) => (item.element as HTMLInputElement).value === '作品广场')
    expect(restored).toBeTruthy()
  })

  it('filters the text element catalog by label or key', async () => {
    const wrapper = create()
    await flushPromises()

    const search = wrapper.find('input[placeholder="搜索文案元素、分组或 key"]')
    await search.setValue('登录页')

    expect(wrapper.text()).toContain('登录页故事说明')
    expect(wrapper.text()).not.toContain('作品广场标签')
    expect(wrapper.text()).toContain('匹配 1 项')
  })
})

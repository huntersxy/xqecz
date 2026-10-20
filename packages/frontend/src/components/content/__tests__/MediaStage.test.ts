import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import MediaStage from '../MediaStage.vue'
import { ContentSchema } from '@/types'

const mocks = vi.hoisted(() => ({ probe: vi.fn(), chain: vi.fn(), broken: vi.fn() }))
vi.mock('@/utils/imageSource', () => ({
  cachedImageSource: () => null,
  fallbackChain: mocks.chain,
  markUnreachable: mocks.broken,
  pickImageUrl: (local: string, _r2: string, openlist: string, source: string | null) =>
    source === 'mirror2' ? openlist : local,
  probeMirror: mocks.probe,
  recheckNetwork: async () => false,
}))

function content(overrides: Record<string, unknown> = {}) {
  return ContentSchema.parse({
    id: 1, title: '透明图片', img: 'https://origin.example/art.png',
    mirror2_img: 'https://mirror.example/art.png',
    user: { id: 1, username: '作者' }, tags: [], created_at: '2026-10-10T00:00:00.000Z',
    ...overrides,
  })
}
const wrappers: ReturnType<typeof mount>[] = []
function createStage(overrides: Record<string, unknown> = {}) {
  const wrapper = mount(MediaStage, { props: { content: content(overrides) } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => { mocks.probe.mockReset(); mocks.chain.mockReset().mockReturnValue([]); mocks.broken.mockReset() })
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

describe('MediaStage loading feedback', () => {
  it('keeps feedback visible from source selection through the actual image load', async () => {
    let resolve!: (source: string) => void
    mocks.probe.mockImplementation(() => new Promise(done => { resolve = done }))
    const wrapper = createStage()
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    expect(wrapper.find('img.cd-image').exists()).toBe(false)

    resolve('mirror2')
    await flushPromises()
    expect(wrapper.get('img.cd-image').attributes('src')).toBe('https://mirror.example/art.png')
    expect(wrapper.get('[role="status"]').attributes('aria-busy')).toBe('true')

    await wrapper.get('img.cd-image').trigger('load')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })

  it('only shows the blurred backdrop after the real image has loaded, reusing its URL', async () => {
    mocks.probe.mockResolvedValue('mirror2')
    const wrapper = createStage()
    await flushPromises()
    expect(wrapper.find('.cd-ambient').exists()).toBe(false)

    await wrapper.get('img.cd-image').trigger('load')
    expect(wrapper.get('.cd-ambient').attributes('src')).toBe('https://mirror.example/art.png')
  })

  it('shows a clear failure with a retry instead of an endless loader', async () => {
    const wrapper = createStage({ mirror2_img: '' })
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    await wrapper.get('img.cd-image').trigger('error')
    expect(wrapper.get('[role="status"]').text()).toContain('图片加载失败，请稍后重试')
    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(false)

    await wrapper.get('.cd-retry').trigger('click')
    expect(wrapper.find('img.cd-image').exists()).toBe(true)
  })
})

describe('MediaStage fallback chain', () => {
  it('moves to the next candidate when a mirror fails and marks only that tier broken', async () => {
    mocks.probe.mockResolvedValue('mirror2')
    mocks.chain.mockReturnValue(['https://origin.example/art.png'])
    const wrapper = createStage()
    await flushPromises()
    await wrapper.get('img.cd-image').trigger('error')
    expect(mocks.broken).toHaveBeenCalledWith('mirror2')
    expect(wrapper.get('img.cd-image').attributes('src')).toBe('https://origin.example/art.png')
    expect(wrapper.find('.cd-retry').exists()).toBe(false)
  })
})

describe('MediaStage text works', () => {
  it('renders text-only content without requesting any media', async () => {
    const wrapper = createStage({ img: '', mirror2_img: '', text: '正文 **加粗**' })
    await flushPromises()
    expect(mocks.probe).not.toHaveBeenCalled()
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.get('.cd-text-card strong').text()).toBe('加粗')
  })
})

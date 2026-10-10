import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ContentMedia from '../ContentMedia.vue'
import { ContentSchema } from '@/types'

const mocks = vi.hoisted(() => ({ probe: vi.fn() }))
vi.mock('@/utils/imageSource', () => ({
  cachedImageSource: () => null,
  fallbackChain: () => [],
  markUnreachable: vi.fn(),
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
function createMedia(overrides: Record<string, unknown> = {}) {
  const wrapper = mount(ContentMedia, { props: { content: content(overrides) } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => { mocks.probe.mockReset() })
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

describe('ContentMedia loading feedback', () => {
  it('keeps feedback visible from source selection through the actual image load', async () => {
    let resolve!: (source: string) => void
    mocks.probe.mockImplementation(() => new Promise(done => { resolve = done }))
    const wrapper = createMedia()
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    expect(wrapper.find('.arco-image-img').exists()).toBe(false)

    resolve('mirror2')
    await flushPromises()
    expect(wrapper.get('.arco-image-img').attributes('src')).toBe('https://mirror.example/art.png')
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    expect(wrapper.get('[role="status"]').attributes('aria-busy')).toBe('true')

    await wrapper.get('.arco-image-img').trigger('load')
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.find('.arco-image-loading').exists()).toBe(false)
  })

  it('shows a clear failure instead of an endless loader when the image cannot load', async () => {
    const wrapper = createMedia({ mirror2_img: '' })
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toBe('正在加载作品图片…')
    await wrapper.get('.arco-image-img').trigger('error')
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toBe('图片加载失败，请稍后重试')
    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(false)
  })
})

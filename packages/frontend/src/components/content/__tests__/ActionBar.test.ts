import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ActionBar from '../ActionBar.vue'
import { ContentSchema } from '@/types'

const mocks = vi.hoisted(() => ({
  user: { isLoggedIn: false },
  likeStatus: vi.fn(), toggleLike: vi.fn(), toggleFavorite: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}))
vi.mock('@/stores/user', () => ({ useUserStore: () => mocks.user }))
vi.mock('@/api', () => ({ contentApi: mocks }))
vi.mock('@/composables/useToast', () => ({ toast: mocks.toast }))

function content(overrides: Record<string, unknown> = {}) {
  return ContentSchema.parse({
    id: 1,
    title: '作品',
    tags: [],
    created_at: '2026-10-10T00:00:00.000Z',
    origin: '/uploads/art.png?source=original',
    img: '/uploads/art.png',
    thumb: '/thumbs/art.webp',
    file_size: 720077,
    user: { id: 1, username: '作者' },
    ...overrides,
  })
}

const wrappers: ReturnType<typeof mount>[] = []
function createActions(overrides: Record<string, unknown> = {}) {
  const wrapper = mount(ActionBar, { props: { content: content(overrides) } })
  wrappers.push(wrapper)
  return wrapper
}
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  mocks.user.isLoggedIn = false
  vi.clearAllMocks()
})

describe('ActionBar downloads', () => {
  it('links directly to the original file and displays its size once', () => {
    const wrapper = createActions()
    const link = wrapper.get('a[aria-label="下载作品"]')
    expect(link.attributes('href')).toBe('/uploads/art.png?source=original&download=1')
    expect(link.get('small').text()).toBe('PNG · 703.2 KB')
    expect(wrapper.text().match(/703\.2 KB/g)).toHaveLength(1)
    expect(wrapper.text()).not.toContain('缩略图')
    expect(wrapper.find('[aria-haspopup]').exists()).toBe(false)
  })

  it('updates the download file and metadata when navigating to a video', async () => {
    const wrapper = createActions()
    await wrapper.setProps({ content: content({
      id: 2, origin: '', img: '', video: '/uploads/movie.mp4', file_size: 2 * 1024 * 1024,
    }) })
    const link = wrapper.get('a[aria-label="下载作品"]')
    expect(link.attributes('href')).toBe('/uploads/movie.mp4?download=1')
    expect(link.get('small').text()).toBe('MP4 · 2.0 MB')
    expect(wrapper.text()).not.toContain('703.2 KB')
  })

  it.each([
    { origin: '', img: '', video: '', thumb: '' },
    { origin: '', img: '/thumbs/art.webp', video: '', thumb: '/thumbs/art.webp' },
  ])('does not offer downloads for text or thumbnail-only content', (media) => {
    const wrapper = createActions(media)
    expect(wrapper.find('a[aria-label="下载作品"]').exists()).toBe(false)
  })

  it('supports an original image fallback without displaying an unknown size', () => {
    const wrapper = createActions({ origin: '', file_size: 0 })
    const link = wrapper.get('a[aria-label="下载作品"]')
    expect(link.attributes('href')).toBe('/uploads/art.png?download=1')
    expect(link.get('small').text()).toBe('PNG')
  })
})

describe('ActionBar interactions', () => {
  it('explains why like and favorite are disabled when logged out', () => {
    const wrapper = createActions()
    expect(wrapper.get('.cd-like-action').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('登录后即可点亮、收藏作品')
  })

  it('does not apply a status response from the previous work after switching', async () => {
    mocks.user.isLoggedIn = true
    let finish!: (v: unknown) => void
    mocks.likeStatus.mockImplementationOnce(() => new Promise(r => { finish = r }))
    mocks.likeStatus.mockResolvedValueOnce({ code: 200, data: { liked: false, favorited: false, like_count: 7 } })
    const wrapper = createActions({ like_count: 3 })
    await wrapper.setProps({ content: content({ id: 2, like_count: 7 }) })
    await flushPromises()
    finish({ code: 200, data: { liked: true, favorited: true, like_count: 99 } })
    await flushPromises()
    expect(wrapper.get('.cd-like-action').attributes('aria-pressed')).toBe('false')
    expect(wrapper.get('.cd-like-count').text()).toBe('7')
  })

  it('ignores a second click while the first toggle is still pending', async () => {
    mocks.user.isLoggedIn = true
    mocks.likeStatus.mockResolvedValue({ code: 200, data: { liked: false, favorited: false, like_count: 0 } })
    let finish!: (v: unknown) => void
    mocks.toggleLike.mockImplementation(() => new Promise(r => { finish = r }))
    const wrapper = createActions()
    await flushPromises()
    await wrapper.get('.cd-like-action').trigger('click')
    await wrapper.get('.cd-like-action').trigger('click')
    expect(mocks.toggleLike).toHaveBeenCalledTimes(1)
    finish({ code: 200, data: { liked: true, like_count: 1 } })
    await flushPromises()
    expect(wrapper.get('.cd-like-action').attributes('aria-pressed')).toBe('true')
  })
})

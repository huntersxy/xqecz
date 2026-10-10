import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ContentActions from '../ContentActions.vue'
import { ContentSchema } from '@/types'

vi.mock('@/stores/user', () => ({ useUserStore: () => ({ isLoggedIn: false }) }))
vi.mock('@/api', () => ({ contentApi: {} }))

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
  const wrapper = mount(ContentActions, { props: { content: content(overrides) } })
  wrappers.push(wrapper)
  return wrapper
}
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()) })

describe('ContentActions downloads', () => {
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

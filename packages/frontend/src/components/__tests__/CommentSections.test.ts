import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CommentSections from '../CommentSections.vue'
import { CommentSchema } from '@/types/schemas'

const mocks = vi.hoisted(() => ({ list: vi.fn(), add: vi.fn(), delete: vi.fn(), success: vi.fn(), error: vi.fn(), warning: vi.fn(), confirm: vi.fn() }))
vi.mock('@/api', () => ({ commentApi: mocks }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ isLoggedIn: true, user: { id: 8 } }) }))
vi.mock('@/composables/useToast', () => ({ useConfirm: () => ({ confirm: mocks.confirm }) }))
vi.mock('@arco-design/web-vue', async importOriginal => ({ ...await importOriginal<typeof import('@arco-design/web-vue')>(), Message: mocks }))

function comment(id: number, parentId: number | null, text: string) {
  return CommentSchema.parse({ id, content_id: 9, user_id: id, text, parent_id: parentId, is_banned: false, created_at: '2026-10-10T00:00:00.000Z', user: { id, username: '作者' + text } })
}
const a = comment(1, null, '评论A'), b = comment(2, 1, '评论B'), c = comment(3, 2, '评论C')
const thread = { ...a, replies: [{ ...b, parent: { id: 1, user_id: 1, text: a.text, user: a.user } }, { ...c, parent: { id: 2, user_id: 2, text: b.text, user: b.user } }] }
const wrappers: ReturnType<typeof mount>[] = []
function create() {
  const wrapper = mount(CommentSections, { props: { contentId: 9, isLoggedIn: true }, global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => { vi.clearAllMocks(); mocks.list.mockResolvedValue({ code: 200, data: { list: [thread], total: 41, total_page: 3 } }); mocks.add.mockResolvedValue({ code: 200, data: c }) })
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

describe('comment reply chains', () => {
  it('loads existing comments immediately when opening the detail', async () => {
    const wrapper = create()
    await flushPromises()
    expect(mocks.list).toHaveBeenCalledWith(9, 1, 20)
    expect(wrapper.findAll('.cd-comment')).toHaveLength(3)
  })
  it('discards an old response when switching to another content', async () => {
    let finish!: (result: unknown) => void
    mocks.list.mockImplementationOnce(() => new Promise(resolve => { finish = resolve }))
    const wrapper = create()
    mocks.list.mockResolvedValueOnce({ code: 200, data: { list: [], total: 0, total_page: 0 } })
    await wrapper.setProps({ contentId: 10 })
    await flushPromises()
    finish({ code: 200, data: { list: [thread], total: 1, total_page: 1 } })
    await flushPromises()
    expect(wrapper.findAll('.cd-comment')).toHaveLength(0)
    expect(wrapper.find('.cd-comment-empty').exists()).toBe(true)
  })
  it('offers a retry after loading fails instead of claiming there are no comments', async () => {
    mocks.list.mockRejectedValueOnce(new Error('offline'))
    const wrapper = create()
    await flushPromises()
    expect(wrapper.find('.cd-comment-status').text()).toContain('评论加载失败')
    expect(wrapper.find('.cd-comment-empty').exists()).toBe(false)
    await wrapper.find('.cd-comment-status button').trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.cd-comment')).toHaveLength(3)
  })
  it('renders C under the same thread and quotes its actual parent B', async () => {
    const wrapper = create()
    await (wrapper.vm as unknown as { loadComments: (page?: number) => Promise<void> }).loadComments()
    await flushPromises()
    expect(wrapper.findAll('.cd-comment')).toHaveLength(3)
    const rows = wrapper.findAll('.cd-comment')
    expect(rows[2].find('.cd-comment-body').text()).toContain('评论C')
    expect(rows[2].find('.cd-comment-quote').text()).toContain('作者评论B: 评论B')
  })
  it('submits a reply to B with B id and reloads the current page', async () => {
    const wrapper = create()
    await (wrapper.vm as unknown as { loadComments: (page?: number) => Promise<void> }).loadComments(2)
    await flushPromises()
    await wrapper.findAll('.cd-comment-reply')[1].trigger('click')
    await wrapper.find('textarea').setValue('新的评论C')
    await wrapper.find('.cd-comment-submit').trigger('click')
    await flushPromises()
    expect(mocks.add).toHaveBeenCalledWith(9, '新的评论C', 2)
    expect(mocks.list).toHaveBeenLastCalledWith(9, 2, 20)
    expect(wrapper.find('.cd-reply-hint').exists()).toBe(false)
  })
  it('prevents duplicate submissions while waiting and keeps text when the request fails', async () => {
    let finish!: (result: unknown) => void
    mocks.add.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    const wrapper = create()
    await wrapper.find('textarea').setValue('等待提交')
    await wrapper.find('.cd-comment-submit').trigger('click')
    await wrapper.find('textarea').trigger('keyup', { key: 'Enter', ctrlKey: true })
    expect(mocks.add).toHaveBeenCalledTimes(1)
    finish({ code: 400, message: '回复的评论不存在或不可见' })
    await flushPromises()
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('等待提交')
    expect(mocks.error).toHaveBeenCalledWith('回复的评论不存在或不可见')
  })
})

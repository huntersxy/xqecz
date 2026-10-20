import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import PollsView from '@/views/PollsView.vue'
import type { Poll, PollDetail } from '@/types/schemas'

const api = vi.hoisted(() => ({ list: vi.fn(), detail: vi.fn(), vote: vi.fn() }))
vi.mock('@/api', () => ({ pollApi: api }))
const wrappers: ReturnType<typeof mount>[] = []
const poll = (id = 1): Poll => ({
  id,
  title: `下一次想看什么场景 ${id}`,
  description: '选择你喜欢的场景',
  options: ['校园午后', '雨中街道', '海边日落', '冬日庭院'],
  vote_count: 9,
  user_id: 2,
  user: { id: 2, username: '活动发起人' },
  created_at: '2026-10-15T02:00:00.000Z',
})
function detail(myVote: number | null = null, id = 1): PollDetail {
  return { poll: poll(id), vote_counts: { '0': 3, '1': 6 }, total_votes: 9, my_vote: myVote }
}
const ok = <T>(data: T) => ({ code: 200, message: 'ok', data })
function list(rows = [poll()], page = 1, total = rows.length) {
  return ok({ list: rows, page, total, page_size: 12, total_page: Math.ceil(total / 12) })
}
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => {
    resolve = done
  })
  return { promise, resolve }
}
async function render(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/polls', component: PollsView },
      { path: '/polls/:id', component: PollsView },
    ],
  })
  await router.push(path)
  const wrapper = mount(PollsView, { global: { plugins: [router] } })
  wrappers.push(wrapper)
  await flushPromises()
  return { wrapper, router }
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  api.list.mockResolvedValue(list())
  api.detail.mockResolvedValue(ok(detail()))
  api.vote.mockResolvedValue(ok(detail(0)))
})
afterEach(() => {
  wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('public poll list', () => {
  it('paginates the public list without exposing poll creation', async () => {
    api.list
      .mockResolvedValueOnce(list([poll()], 1, 13))
      .mockResolvedValueOnce(list([poll(2)], 2, 13))
    const { wrapper, router } = await render('/polls')
    expect(wrapper.get('.poll-index-count').text()).toBe('13')
    expect(wrapper.get('h1').text()).toBe('投票广场')
    expect(wrapper.text()).not.toContain('发起投票')
    expect(wrapper.find('form').exists()).toBe(false)
    const buttons = wrapper.findAll('.poll-pagination button')
    expect(buttons[0].attributes('disabled')).toBeDefined()
    await buttons[1].trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.page).toBe('2')
    expect(api.list).toHaveBeenLastCalledWith({ page: 2, page_size: 12 }, expect.any(AbortSignal))
    expect(wrapper.get('.poll-preview').attributes('href')).toBe('/polls/2')
    expect(wrapper.findAll('.poll-pagination button')[1].attributes('disabled')).toBeDefined()
  })
})

describe('public ballot and statistics', () => {
  it('accepts option zero for a guest, prevents duplicate submission and uses server totals', async () => {
    const pending = deferred<ReturnType<typeof ok<PollDetail>>>()
    api.vote.mockReturnValueOnce(pending.promise)
    const { wrapper } = await render('/polls/1')
    expect(wrapper.get('.poll-submit').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('游客也可参与')
    await wrapper.findAll('input[type="radio"]')[0].setValue()
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(api.vote).toHaveBeenCalledOnce()
    expect(api.vote).toHaveBeenCalledWith(1, 0, expect.any(AbortSignal))
    expect(wrapper.get('fieldset').attributes('disabled')).toBeDefined()
    pending.resolve(ok({ ...detail(0), total_votes: 10, vote_counts: { '0': 4, '1': 6 } }))
    await flushPromises()
    expect(wrapper.get('.poll-byline').text()).toContain('10 票')
    expect(wrapper.findAll('.poll-option-result')[0].text()).toContain('40%')
    expect(wrapper.get('.poll-vote-status').text()).toContain('投票已记录')
    expect(wrapper.get('.poll-submit').attributes('disabled')).toBeDefined()
  })

  it('restores a previous selection and changes its vote without adding a voter', async () => {
    api.detail.mockResolvedValue(ok(detail(0)))
    api.vote.mockResolvedValue(ok({ ...detail(1), vote_counts: { '0': 2, '1': 7 } }))
    const { wrapper } = await render('/polls/1')
    expect((wrapper.findAll('input')[0].element as HTMLInputElement).checked).toBe(true)
    expect(wrapper.findAll('.poll-option-result')[1].text()).toContain('67%')
    await wrapper.findAll('input')[1].setValue()
    expect(wrapper.get('.poll-submit').text()).toBe('确认改票')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(api.vote).toHaveBeenCalledWith(1, 1, expect.any(AbortSignal))
    expect(wrapper.get('.poll-byline').text()).toContain('9 票')
    expect(wrapper.findAll('.poll-option-result')[1].text()).toContain('78%')
    expect(wrapper.findAll('.poll-option')[1].text()).toContain('已投')
  })

  it('can show zero-vote results before voting without producing invalid percentages', async () => {
    api.detail.mockResolvedValue(ok({ ...detail(), total_votes: 0, vote_counts: {} }))
    const { wrapper } = await render('/polls/1')
    expect(wrapper.find('.poll-option-result').exists()).toBe(false)
    await wrapper.get('.poll-ballot-actions .poll-secondary').trigger('click')
    expect(wrapper.findAll('.poll-option-result').every((row) => row.text() === '0 票 0%')).toBe(
      true,
    )
    expect(wrapper.text()).not.toContain('NaN')
    expect(
      wrapper.findAll('.poll-option-fill').every((row) => row.attributes('style') === 'width: 0%;'),
    ).toBe(true)
  })

  it.each(['business', 'transport'])(
    'keeps the selection after a %s failure so the voter can retry',
    async (failure) => {
      if (failure === 'business')
        api.vote.mockResolvedValueOnce({ code: 400, message: '暂时无法投票', data: null })
      else api.vote.mockRejectedValueOnce(new Error('连接失败'))
      const { wrapper } = await render('/polls/1')
      await wrapper.findAll('input')[0].setValue()
      await wrapper.get('form').trigger('submit')
      await flushPromises()
      expect(wrapper.get('[role="alert"]').text()).toBe(
        failure === 'business' ? '暂时无法投票' : '连接失败',
      )
      expect((wrapper.findAll('input')[0].element as HTMLInputElement).checked).toBe(true)
      expect(wrapper.get('.poll-submit').attributes('disabled')).toBeUndefined()
      await wrapper.get('form').trigger('submit')
      await flushPromises()
      expect(wrapper.find('[role="alert"]').exists()).toBe(false)
      expect(wrapper.text()).toContain('投票已记录')
    },
  )

  it('does not let a late detail or vote overwrite the next poll', async () => {
    const pending = deferred<ReturnType<typeof ok<PollDetail>>>()
    api.detail.mockReturnValueOnce(pending.promise).mockResolvedValueOnce(ok(detail(null, 2)))
    const { wrapper, router } = await render('/polls/1')
    const firstSignal = api.detail.mock.calls[0][1] as AbortSignal
    await router.push('/polls/2')
    await flushPromises()
    expect(firstSignal.aborted).toBe(true)
    pending.resolve(ok(detail(null, 1)))
    await flushPromises()
    expect(wrapper.get('.poll-ballot h2').text()).toContain('场景 2')
    const pendingVote = deferred<ReturnType<typeof ok<PollDetail>>>()
    api.vote.mockReturnValueOnce(pendingVote.promise)
    await wrapper.findAll('input')[0].setValue()
    await wrapper.get('form').trigger('submit')
    const voteSignal = api.vote.mock.calls[0][2] as AbortSignal
    api.detail.mockResolvedValueOnce(ok(detail(null, 3)))
    await router.push('/polls/3')
    await flushPromises()
    expect(voteSignal.aborted).toBe(true)
    pendingVote.resolve(ok(detail(0, 2)))
    await flushPromises()
    expect(wrapper.get('.poll-ballot h2').text()).toContain('场景 3')
    expect(wrapper.find('.poll-vote-status').exists()).toBe(false)
  })

  it('reports a missing poll and cancels owned work on unmount', async () => {
    const { wrapper } = await render('/polls/not-a-number')
    expect(wrapper.get('[role="alert"]').text()).toContain('投票不存在')
    expect(api.detail).not.toHaveBeenCalled()
    api.detail.mockRejectedValueOnce(new Error('投票不存在'))
    const missing = await render('/polls/404')
    expect(missing.wrapper.get('[role="alert"]').text()).toContain('投票不存在')
    const pending = deferred<ReturnType<typeof ok<PollDetail>>>()
    api.detail.mockReturnValueOnce(pending.promise)
    const active = await render('/polls/1')
    const signal = api.detail.mock.calls.at(-1)![1] as AbortSignal
    active.wrapper.unmount()
    expect(signal.aborted).toBe(true)
    pending.resolve(ok(detail()))
    await flushPromises()
  })
})

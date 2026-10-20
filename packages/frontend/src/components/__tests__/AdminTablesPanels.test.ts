import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h, type Component } from 'vue'
import AdminPollPanel from '@/components/admin/AdminPollPanel.vue'
import AdminClaimList from '@/components/admin/AdminClaimList.vue'
import AdminReportList from '@/components/admin/AdminReportList.vue'
import { useAdminStore } from '@/stores/admin'
import { formatTime } from '@/utils'
import type { Claim, CommentReport, Poll } from '@/types'

const confirmMock = vi.hoisted(() => vi.fn(async () => true))
const toastMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  info: vi.fn(),
}))

vi.mock('@/composables/useToast', () => ({
  toast: toastMock,
  useConfirm: () => ({ pendingConfirm: { value: null }, confirm: confirmMock, respond: vi.fn() }),
}))

vi.mock('@/api', () => ({
  adminApi: { handleClaim: vi.fn(async () => ({ data: null })) },
  contentApi: {},
  commentApi: { handleReport: vi.fn(async () => ({ data: null })), delete: vi.fn(async () => ({ data: null })) },
  pollApi: { list: vi.fn(async () => ({ data: { list: [] } })), delete: vi.fn(async () => ({ data: null })) },
}))

const EmptyRoute = defineComponent({ name: 'EmptyRoute', render: () => h('div') })

const pollFixture: Poll = {
  id: 7,
  title: '最想看的番',
  description: '只能选一个',
  options: ['选项甲', '选项乙'],
  vote_count: 12,
  user_id: 1,
  created_at: '2026-10-01T00:00:00.000Z',
}

const claimFixture = {
  id: 5,
  content_id: 3,
  user_id: 2,
  user: { id: 2, username: 'alice' },
  content: { id: 3, title: '测试内容', thumb: '' },
  reason: '这是我的作品',
  status: 'pending',
  remark: '',
  created_at: '2026-10-02T00:00:00.000Z',
} as unknown as Claim

const reportFixture: CommentReport = {
  id: 9,
  comment_id: 33,
  user_id: 4,
  reason: '广告引流',
  handled: false,
  created_at: '2026-10-03T00:00:00.000Z',
  Comment: { id: 33, text: '违规评论内容' },
  User: { id: 4, username: 'bob' },
}

let pinia: Pinia
let admin: ReturnType<typeof useAdminStore>
const wrappers: VueWrapper[] = []

function mountPanel(component: Component) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: EmptyRoute },
      { path: '/polls', component: EmptyRoute },
      { path: '/polls/:id', component: EmptyRoute },
    ],
  })
  const wrapper = mount(component, { global: { plugins: [pinia, router] } })
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.clearAllMocks()
  confirmMock.mockReset()
  confirmMock.mockResolvedValue(true)

  vi.stubGlobal('matchMedia', vi.fn((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })))

  pinia = createPinia()
  setActivePinia(pinia)
  admin = useAdminStore()

  vi.spyOn(admin, 'loadPolls').mockResolvedValue(undefined)
  vi.spyOn(admin, 'loadClaims').mockResolvedValue(undefined)
  vi.spyOn(admin, 'loadReports').mockResolvedValue(undefined)

  admin.polls = []
  admin.pollsLoading = false
  admin.reports = []
  admin.reportsLoading = false
  admin.claims.list = []
  admin.claims.total = 0
  admin.claims.page = 1
  admin.claims.totalPages = 1
  admin.claims.loading = false
  admin.showCreatePollModal = false
})

afterEach(() => {
  wrappers.splice(0).forEach(w => w.unmount())
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('投票管理面板', () => {
  it('无数据时渲染空态与创建引导', async () => {
    const wrapper = mountPanel(AdminPollPanel)
    await flushPromises()

    expect(wrapper.find('.admin-empty').exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无投票')
    expect(wrapper.text()).toContain('创建投票')
  })

  it('首屏加载时渲染卡片骨架，不闪空态', async () => {
    admin.pollsLoading = true
    const wrapper = mountPanel(AdminPollPanel)
    await flushPromises()

    expect(wrapper.find('.admin-skeleton').exists()).toBe(true)
    expect(wrapper.findAll('.admin-skeleton-card')).toHaveLength(5)
    expect(wrapper.find('.admin-empty').exists()).toBe(false)
  })

  it('有数据时渲染标题、票数、描述与选项', async () => {
    admin.polls = [pollFixture]
    const wrapper = mountPanel(AdminPollPanel)
    await flushPromises()

    expect(wrapper.find('.poll-list').exists()).toBe(true)
    expect(wrapper.text()).toContain('最想看的番')
    expect(wrapper.text()).toContain('12 票')
    expect(wrapper.text()).toContain('只能选一个')
    expect(wrapper.text()).toContain('选项甲')
    expect(wrapper.find('.admin-empty').exists()).toBe(false)
  })

  it('删除需二次确认：取消不请求，确认后卡片淡出再删除并刷新', async () => {
    admin.polls = [pollFixture]
    const deletePoll = vi.spyOn(admin, 'deletePoll').mockResolvedValue(true)
    const wrapper = mountPanel(AdminPollPanel)
    await flushPromises()

    confirmMock.mockResolvedValueOnce(false)
    await wrapper.find('.poll-delete').trigger('click')
    await flushPromises()
    expect(confirmMock).toHaveBeenCalledTimes(1)
    expect(deletePoll).not.toHaveBeenCalled()
    expect(wrapper.find('.adm-card').classes()).not.toContain('is-leaving')

    await wrapper.find('.poll-delete').trigger('click')
    await flushPromises()
    expect(wrapper.find('.adm-card').classes()).toContain('is-leaving')
    await vi.waitFor(() => {
      expect(deletePoll).toHaveBeenCalledWith(7)
    })
    await vi.waitFor(() => {
      expect(admin.loadPolls).toHaveBeenCalledTimes(2)
    })
  })
})

describe('认领管理列表', () => {
  it('无数据时渲染空态', async () => {
    const wrapper = mountPanel(AdminClaimList)
    await flushPromises()

    expect(wrapper.find('.admin-empty').exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无认领')
    expect(wrapper.find('.arco-table').exists()).toBe(false)
    expect(wrapper.find('.adm-card').exists()).toBe(false)
  })

  it('有数据时渲染卡片：标题、申请人、理由、时间与状态', async () => {
    admin.claims.list = [claimFixture]
    admin.claims.total = 1
    const wrapper = mountPanel(AdminClaimList)
    await flushPromises()

    const card = wrapper.find('.adm-card')
    expect(card.exists()).toBe(true)
    expect(card.find('.adm-card-title').text()).toContain('测试内容')
    expect(wrapper.text()).toContain('alice')
    expect(card.find('.claim-reason').text()).toContain('这是我的作品')
    expect(card.find('.claim-reason').attributes('title')).toBe('这是我的作品')
    expect(wrapper.text()).toContain(formatTime(claimFixture.created_at))
    expect(card.find('.adm-card-badge').text()).toContain('待处理')
    expect(wrapper.find('.arco-table').exists()).toBe(false)
  })

  it('已处理的认领只显示状态，不再出现操作按钮', async () => {
    admin.claims.list = [{ ...claimFixture, status: 'approved' } as Claim]
    admin.claims.total = 1
    const wrapper = mountPanel(AdminClaimList)
    await flushPromises()

    expect(wrapper.find('.adm-card-badge').text()).toContain('已通过')
    expect(wrapper.text()).toContain('已处理')
    expect(wrapper.find('[aria-label="通过认领"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="拒绝认领"]').exists()).toBe(false)
  })

  it('通过按钮调用 handleClaim(id, approve)，卡片淡出后刷新列表', async () => {
    admin.claims.list = [claimFixture]
    admin.claims.total = 1
    const handleClaim = vi.spyOn(admin, 'handleClaim').mockResolvedValue(true)
    const wrapper = mountPanel(AdminClaimList)
    await flushPromises()

    await wrapper.find('[aria-label="通过认领"]').trigger('click')
    await flushPromises()

    expect(handleClaim).toHaveBeenCalledWith(5, 'approve')
    expect(wrapper.find('.adm-card').classes()).toContain('is-leaving')
    await vi.waitFor(() => {
      expect(admin.loadClaims).toHaveBeenCalledTimes(2)
    })
    await vi.waitFor(() => {
      expect(wrapper.find('.adm-card').classes()).not.toContain('is-leaving')
    })
  })

  it('拒绝按钮调用 handleClaim(id, reject)，卡片淡出后刷新列表', async () => {
    admin.claims.list = [claimFixture]
    admin.claims.total = 1
    const handleClaim = vi.spyOn(admin, 'handleClaim').mockResolvedValue(true)
    const wrapper = mountPanel(AdminClaimList)
    await flushPromises()

    await wrapper.find('[aria-label="拒绝认领"]').trigger('click')
    await flushPromises()

    expect(handleClaim).toHaveBeenCalledWith(5, 'reject')
    expect(wrapper.find('.adm-card').classes()).toContain('is-leaving')
    await vi.waitFor(() => {
      expect(admin.loadClaims).toHaveBeenCalledTimes(2)
    })
  })
})

describe('举报管理列表', () => {
  it('无数据时渲染空态', async () => {
    const wrapper = mountPanel(AdminReportList)
    await flushPromises()

    expect(wrapper.find('.admin-empty').exists()).toBe(true)
    expect(wrapper.text()).toContain('暂无举报')
    expect(wrapper.find('.arco-table').exists()).toBe(false)
    expect(wrapper.find('.adm-card').exists()).toBe(false)
  })

  it('有数据时渲染举报原因、被举报内容、举报人与状态', async () => {
    admin.reports = [reportFixture]
    const wrapper = mountPanel(AdminReportList)
    await flushPromises()

    const card = wrapper.find('.adm-card')
    expect(card.exists()).toBe(true)
    expect(card.find('.adm-card-title').text()).toContain('广告引流')
    expect(card.find('.report-quote').text()).toContain('违规评论内容')
    expect(card.find('.report-quote').attributes('title')).toBe('违规评论内容')
    expect(wrapper.text()).toContain('bob')
    expect(wrapper.text()).toContain(formatTime(reportFixture.created_at))
    expect(card.find('.adm-card-badge').text()).toContain('待处理')
    expect(wrapper.find('.arco-table').exists()).toBe(false)
  })

  it('已处理的举报不再显示处理按钮，删除仍然可用', async () => {
    admin.reports = [{ ...reportFixture, handled: true }]
    const wrapper = mountPanel(AdminReportList)
    await flushPromises()

    expect(wrapper.find('.adm-card-badge').text()).toContain('已处理')
    expect(wrapper.find('[aria-label="标记已处理"]').exists()).toBe(false)
    expect(wrapper.find('[aria-label="删除评论"]').exists()).toBe(true)
  })

  it('标记已处理调用 handleReport(id)，卡片淡出后刷新列表', async () => {
    admin.reports = [reportFixture]
    const handleReport = vi.spyOn(admin, 'handleReport').mockResolvedValue(true)
    const wrapper = mountPanel(AdminReportList)
    await flushPromises()

    await wrapper.find('[aria-label="标记已处理"]').trigger('click')
    await flushPromises()

    expect(handleReport).toHaveBeenCalledWith(9)
    expect(wrapper.find('.adm-card').classes()).toContain('is-leaving')
    await vi.waitFor(() => {
      expect(admin.loadReports).toHaveBeenCalledTimes(2)
    })
    await vi.waitFor(() => {
      expect(wrapper.find('.adm-card').classes()).not.toContain('is-leaving')
    })
  })

  it('删除评论需二次确认：取消不请求，确认后删除、标记处理并刷新', async () => {
    admin.reports = [reportFixture]
    const deleteComment = vi.spyOn(admin, 'deleteComment').mockResolvedValue(true)
    const handleReport = vi.spyOn(admin, 'handleReport').mockResolvedValue(true)
    const wrapper = mountPanel(AdminReportList)
    await flushPromises()

    confirmMock.mockResolvedValueOnce(false)
    await wrapper.find('[aria-label="删除评论"]').trigger('click')
    await flushPromises()
    expect(confirmMock).toHaveBeenCalledTimes(1)
    expect(deleteComment).not.toHaveBeenCalled()

    await wrapper.find('[aria-label="删除评论"]').trigger('click')
    await flushPromises()
    expect(deleteComment).toHaveBeenCalledWith(33)
    expect(handleReport).toHaveBeenCalledWith(9)
    await vi.waitFor(() => {
      expect(admin.loadReports).toHaveBeenCalledTimes(2)
    })
  })
})

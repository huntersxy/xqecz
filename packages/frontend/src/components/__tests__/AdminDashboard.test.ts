import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AdminDashboard from '@/components/admin/AdminDashboard.vue'
import { ContentSchema } from '@/types'
import type { DashboardStats } from '@/types'

const { getDashboard, openDrawer } = vi.hoisted(() => ({
  getDashboard: vi.fn(),
  openDrawer: vi.fn(),
}))

vi.mock('@/api', () => ({ adminApi: { getDashboard } }))
vi.mock('@/stores/admin', () => ({ useAdminStore: () => ({ openDrawer }) }))

const baseStats: DashboardStats = {
  content: { total: 100, approved: 70, pending: 12, rejected: 18, today: 3 },
  users: { total: 60, admins: 2, banned: 1, today: 1 },
  comments: { total: 240, today: 5 },
  claims: { total: 9, pending: 4, approved: 4, rejected: 1 },
  reports: { total: 11, unhandled: 3, handled: 8 },
  polls: { total: 7, votes: 88 },
  views: 12345,
  topTags: [
    { tag: '原神', count: 30 },
    { tag: '手书', count: 10 },
  ],
  recentContents: [
    ContentSchema.parse({
      id: 1,
      title: '初音手书',
      thumb: '/uploads/a.webp',
      view_count: 12,
      created_at: 1_700_000_000_000,
      audit_status: 'approved',
      tags: ['初音'],
      user: { id: 7, username: '画手甲' },
    }),
  ],
  recentUsers: [
    { id: 9, username: '管理员甲', email: 'admin@example.com', is_admin: true, created_at: 1_700_000_000_000 },
  ],
}

const mountedWrappers: ReturnType<typeof mount>[] = []

function renderDashboard() {
  const wrapper = mount(AdminDashboard)
  mountedWrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: query === '(prefers-reduced-motion: reduce)',
    media: query,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
  getDashboard.mockResolvedValue({ code: 0, data: baseStats })
})

afterEach(() => {
  mountedWrappers.splice(0).forEach((wrapper) => wrapper.unmount())
  getDashboard.mockReset()
  openDrawer.mockReset()
  vi.unstubAllGlobals()
})

describe('AdminDashboard', () => {
  it('首次加载先渲染骨架，数据就绪后渲染内容', async () => {
    const wrapper = renderDashboard()
    expect(wrapper.find('.admin-skeleton').exists()).toBe(true)

    await flushPromises()

    expect(wrapper.find('.admin-skeleton').exists()).toBe(false)
    expect(wrapper.findAll('.adm-kpi')).toHaveLength(6)
    expect(getDashboard).toHaveBeenCalledTimes(1)
    expect(getDashboard).toHaveBeenLastCalledWith(false)
  })

  it('KPI 卡片点击 emit 对应页签 key', async () => {
    const wrapper = renderDashboard()
    await flushPromises()

    const cards = wrapper.findAll('.adm-kpi')
    expect(cards).toHaveLength(6)

    const expected = ['all', 'pending', 'users', 'reports', 'polls']
    for (let index = 0; index < expected.length; index += 1) {
      await cards[index].trigger('click')
    }

    expect(wrapper.emitted('select')?.map((args) => args[0])).toEqual(expected)
  })

  it('没有目标页签的「总浏览量」禁用且点击不 emit', async () => {
    const wrapper = renderDashboard()
    await flushPromises()

    const viewsCard = wrapper.findAll('.adm-kpi')[5]
    expect(viewsCard.text()).toContain('总浏览量')
    expect(viewsCard.attributes('disabled')).toBeDefined()

    await viewsCard.trigger('click')

    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('接口返回空数据时渲染 AdminEmpty 空态', async () => {
    getDashboard.mockResolvedValue({ code: 0, data: null })

    const wrapper = renderDashboard()
    await flushPromises()

    expect(wrapper.find('.admin-skeleton').exists()).toBe(false)
    expect(wrapper.find('.admin-empty-title').text()).toBe('暂无数据')
    expect(wrapper.findAll('.adm-kpi')).toHaveLength(0)
  })

  it('刷新按钮触发既有加载方法并携带 force', async () => {
    const wrapper = renderDashboard()
    await flushPromises()
    expect(getDashboard).toHaveBeenCalledTimes(1)

    await wrapper.find('.adm-dash-refresh').trigger('click')
    await flushPromises()

    expect(getDashboard).toHaveBeenCalledTimes(2)
    expect(getDashboard).toHaveBeenLastCalledWith(true)
  })

  it('reduced-motion 下 KPI 数字直接给终值且不建动画帧', async () => {
    const requestAnimationFrame = vi.fn(() => 0)
    vi.stubGlobal('requestAnimationFrame', requestAnimationFrame)
    vi.stubGlobal('cancelAnimationFrame', vi.fn())

    const wrapper = renderDashboard()
    await flushPromises()

    const values = wrapper.findAll('.adm-kpi-value').map((node) => node.text())
    expect(values).toEqual(['100', '12', '60', '3', '7', '12,345'])
    expect(requestAnimationFrame).not.toHaveBeenCalled()
  })

  it('待处理带深链页签，最新内容点击打开抽屉', async () => {
    const wrapper = renderDashboard()
    await flushPromises()

    const pendingItems = wrapper.findAll('.adm-pending-item')
    expect(pendingItems).toHaveLength(3)
    for (const item of pendingItems) {
      await item.trigger('click')
    }
    expect(wrapper.emitted('select')?.map((args) => args[0])).toEqual(['pending', 'claims', 'reports'])

    await wrapper.find('.adm-content-item').trigger('click')
    expect(openDrawer).toHaveBeenCalledTimes(1)
    expect(openDrawer.mock.calls[0][1]).toBe('view')
  })
})

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { nextTick } from 'vue'
import AdminContentList from '@/components/admin/AdminContentList.vue'
import AdminUserTable from '@/components/admin/AdminUserTable.vue'
import { useAdminStore } from '@/stores/admin'
import { useUserStore } from '@/stores/user'
import type { Content, User } from '@/types'

const media = vi.hoisted(() => ({ mobile: false }))

vi.mock('@vueuse/core', async () => {
  const actual = await vi.importActual<typeof import('@vueuse/core')>('@vueuse/core')
  const { ref } = await import('vue')
  return { ...actual, useMediaQuery: () => ref(media.mobile) }
})

vi.mock('@/composables/useToast', async () => {
  const { ref } = await import('vue')
  return {
    toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
    useConfirm: () => ({ pendingConfirm: ref(null), confirm: vi.fn(async () => true), respond: vi.fn() }),
  }
})

const currentUser = { id: 1, username: 'root', email: 'root@example.com', is_admin: true } as User
const otherUser = { id: 2, username: 'reader', email: 'reader@example.com', is_admin: false, is_banned: true } as User

function content(id: number, overrides: Partial<Content> = {}): Content {
  return {
    id,
    title: '内容 ' + id,
    text: '',
    thumb: '',
    video: '',
    img: '',
    origin: '',
    mirror_img: '',
    mirror_video: '',
    mirror2_img: '',
    mirror2_video: '',
    file_size: 0,
    user: { id: 9, username: 'creator' },
    avatar_url: '',
    tags: ['标签'],
    view_count: 0,
    like_count: 0,
    audit_status: 'pending',
    ai_enhanced: false,
    created_at: 0,
    ...overrides,
  }
}

const mountedWrappers: VueWrapper[] = []

function track<T extends VueWrapper>(wrapper: T): T {
  mountedWrappers.push(wrapper)
  return wrapper
}

function seedContentList(state: { list: Content[]; total: number; totalPages: number; loading: boolean }, list: Content[]) {
  state.list = list
  state.total = list.length
  state.totalPages = 1
  state.loading = false
}

async function settle() {
  await flushPromises()
  await nextTick()
  await flushPromises()
}

let pinia: Pinia
let admin: ReturnType<typeof useAdminStore>
let userStore: ReturnType<typeof useUserStore>

beforeEach(() => {
  media.mobile = false
  pinia = createPinia()
  setActivePinia(pinia)
  admin = useAdminStore()
  userStore = useUserStore()
  userStore.user = currentUser

  vi.spyOn(admin, 'loadMyContent').mockResolvedValue(undefined)
  vi.spyOn(admin, 'loadAllContent').mockResolvedValue(undefined)
  vi.spyOn(admin, 'loadPendingContent').mockResolvedValue(undefined)
  vi.spyOn(admin, 'loadUsers').mockResolvedValue(undefined)
  vi.spyOn(admin, 'auditContent').mockResolvedValue(true)
  vi.spyOn(admin, 'updateUserRole').mockResolvedValue(true)
  vi.spyOn(admin, 'updateUserBan').mockResolvedValue(true)
  vi.spyOn(admin, 'deleteUser').mockResolvedValue(true)
  vi.spyOn(admin, 'confirmDelete').mockImplementation(() => {})
})

afterEach(() => {
  mountedWrappers.splice(0).forEach(wrapper => wrapper.unmount())
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

function mountContent(mode: 'my' | 'pending' | 'all') {
  return track(mount(AdminContentList, { props: { mode }, global: { plugins: [pinia] } }))
}

function mountUsers() {
  return track(mount(AdminUserTable, { global: { plugins: [pinia] } }))
}

describe('AdminContentList 三种 mode', () => {
  it('页头 / 工具区计数 / 卡片操作按 mode 变化，并拉取对应列表', async () => {
    seedContentList(admin.myContent, [content(1, { audit_status: 'approved' })])
    seedContentList(admin.pendingContent, [content(2), content(3)])
    seedContentList(admin.allContent, [content(4), content(5), content(6)])

    const my = mountContent('my')
    await settle()
    expect(my.find('.admin-panel-title').text()).toBe('我的内容')
    expect(my.find('.admin-data-toolbar-summary').text()).toBe('共 1 条内容')
    expect(my.findAll('[aria-label="编辑"]')).toHaveLength(1)
    expect(my.findAll('[aria-label="删除"]')).toHaveLength(1)
    expect(my.findAll('[aria-label="通过"]')).toHaveLength(0)
    expect(my.text()).not.toContain('批量生成缩略图')
    expect(admin.loadMyContent).toHaveBeenCalledTimes(1)

    const pending = mountContent('pending')
    await settle()
    expect(pending.find('.admin-panel-title').text()).toBe('待审核内容')
    expect(pending.find('.admin-data-toolbar-summary').text()).toBe('共 2 条待审内容')
    expect(pending.findAll('[aria-label="通过"]')).toHaveLength(2)
    expect(pending.findAll('[aria-label="拒绝"]')).toHaveLength(2)
    expect(pending.findAll('[aria-label="更多操作"]')).toHaveLength(2)
    expect(pending.findAll('[aria-label="编辑"]')).toHaveLength(0)
    expect(admin.loadPendingContent).toHaveBeenCalledTimes(1)

    const all = mountContent('all')
    await settle()
    expect(all.find('.admin-panel-title').text()).toBe('所有内容')
    expect(all.find('.admin-data-toolbar-summary').text()).toBe('共 3 条内容')
    expect(all.text()).toContain('批量生成缩略图')
    expect(all.findAll('[aria-label="编辑"]')).toHaveLength(3)
    expect(admin.loadAllContent).toHaveBeenCalledTimes(1)
  })

  it('桌面与手机渲染同一套卡片网格，不再有表格和旧的手机卡片', async () => {
    seedContentList(admin.pendingContent, [content(7)])

    media.mobile = false
    const desktop = mountContent('pending')
    await settle()
    expect(desktop.find('.arco-table').exists()).toBe(false)
    expect(desktop.find('.admin-mobile-card').exists()).toBe(false)
    expect(desktop.find('.adm-card-grid').exists()).toBe(true)
    expect(desktop.find('.adm-card').text()).toContain('内容 7')
    expect(desktop.find('.adm-card').text()).toContain('审核中')

    media.mobile = true
    const mobile = mountContent('pending')
    await settle()
    expect(mobile.find('.arco-table').exists()).toBe(false)
    expect(mobile.find('.admin-mobile-card').exists()).toBe(false)
    expect(mobile.find('.adm-card-grid').exists()).toBe(true)
    expect(mobile.find('.adm-card').text()).toContain('内容 7')
    expect(mobile.find('.adm-card').text()).toContain('审核中')
    expect(mobile.findAll('.adm-card [aria-label="通过"]')).toHaveLength(1)
    expect(mobile.findAll('.adm-card [aria-label="拒绝"]')).toHaveLength(1)
    expect(mobile.findAll('.adm-card [aria-label="更多操作"]')).toHaveLength(1)
  })

  it('卡片带状态徽标与作者 / 标签，我的内容不重复显示作者', async () => {
    seedContentList(admin.allContent, [content(8, { audit_status: 'rejected', tags: ['甲', '乙', '丙'] })])
    const all = mountContent('all')
    await settle()
    expect(all.find('.adm-card-badge').text()).toBe('已拒绝')
    expect(all.find('.adm-card-meta').text()).toContain('creator')
    expect(all.find('.adm-card-meta').text()).toContain('甲')
    expect(all.find('.adm-card-meta').text()).toContain('+1')

    seedContentList(admin.myContent, [content(9, { audit_status: 'approved' })])
    const my = mountContent('my')
    await settle()
    expect(my.find('.adm-card-meta').text()).not.toContain('creator')
  })
})

describe('AdminContentList 卡片交互', () => {
  it('整卡点击打开内容抽屉，键盘 Enter 等价，底部按钮不连带触发抽屉', async () => {
    seedContentList(admin.myContent, [content(21, { audit_status: 'approved' })])
    const openDrawer = vi.spyOn(admin, 'openDrawer').mockImplementation(() => {})

    const wrapper = mountContent('my')
    await settle()

    await wrapper.get('.adm-card').trigger('click')
    expect(openDrawer).toHaveBeenNthCalledWith(1, expect.objectContaining({ id: 21 }), 'view')

    await wrapper.get('.adm-card').trigger('keydown', { key: 'Enter' })
    expect(openDrawer).toHaveBeenNthCalledWith(2, expect.objectContaining({ id: 21 }), 'view')

    await wrapper.get('[aria-label="编辑"]').trigger('click')
    expect(openDrawer).toHaveBeenNthCalledWith(3, expect.objectContaining({ id: 21 }), 'edit')

    await wrapper.get('[aria-label="删除"]').trigger('click')
    expect(openDrawer).toHaveBeenCalledTimes(3)
  })
})

describe('AdminContentList 审核与删除', () => {
  it('pending 模式下通过 / 拒绝调用 auditContent，并给该卡片打 M8 退场标记', async () => {
    seedContentList(admin.pendingContent, [content(11)])

    const wrapper = mountContent('pending')
    await settle()

    await wrapper.get('[aria-label="通过"]').trigger('click')
    await settle()
    expect(admin.auditContent).toHaveBeenNthCalledWith(1, 11, 'approved', currentUser.id)
    expect(wrapper.find('.adm-card.is-leaving').exists()).toBe(true)

    await wrapper.get('[aria-label="拒绝"]').trigger('click')
    await settle()
    expect(admin.auditContent).toHaveBeenNthCalledWith(2, 11, 'rejected', currentUser.id)
    expect(admin.loadPendingContent).toHaveBeenCalledTimes(1)
  })

  it('退场 280ms 后真正刷新列表，并清掉卡片上的退场标记', async () => {
    seedContentList(admin.pendingContent, [content(31)])

    const wrapper = mountContent('pending')
    await settle()

    await wrapper.get('[aria-label="通过"]').trigger('click')
    await settle()
    expect(wrapper.find('.adm-card.is-leaving').exists()).toBe(true)

    await new Promise(resolve => setTimeout(resolve, 340))
    await settle()

    expect(admin.loadPendingContent).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.adm-card.is-leaving').exists()).toBe(false)
  })

  it('删除按钮走 admin.confirmDelete，成功回调后才进入退场', async () => {
    seedContentList(admin.myContent, [content(12, { audit_status: 'approved' })])
    const confirmDelete = vi.spyOn(admin, 'confirmDelete').mockImplementation((_id, onOk) => { onOk?.() })

    const wrapper = mountContent('my')
    await settle()

    await wrapper.get('[aria-label="删除"]').trigger('click')
    await settle()

    expect(confirmDelete).toHaveBeenCalledWith(12, expect.any(Function))
    expect(wrapper.find('.adm-card.is-leaving').exists()).toBe(true)
  })
})

describe('AdminUserTable 用户管理', () => {
  it('卡片流渲染徽标与操作，角色切换 / 封禁 / 删除三个动作都走 store', async () => {
    admin.users.list = [currentUser, otherUser]
    admin.users.total = 2
    admin.users.totalPages = 1
    admin.users.loading = false

    const wrapper = mountUsers()
    await settle()

    expect(wrapper.find('.admin-panel-title').text()).toBe('用户管理')
    expect(wrapper.find('.admin-data-toolbar-summary').text()).toBe('共 2 位用户')
    expect(wrapper.find('.arco-table').exists()).toBe(false)
    expect(wrapper.find('.adm-card-grid.is-dense').exists()).toBe(true)
    expect(wrapper.findAll('.user-card')).toHaveLength(2)
    expect(wrapper.findAll('.adm-card-badge')).toHaveLength(4)
    expect(wrapper.get('.user-card').text()).toContain('管理员')
    expect(wrapper.get('.user-card').text()).toContain('自己')
    expect(wrapper.findAll('[aria-label="删除"]')).toHaveLength(1)

    await wrapper.get('[aria-label="设为管理员"]').trigger('click')
    await settle()
    expect(admin.updateUserRole).toHaveBeenCalledWith(2, true)

    await wrapper.get('[aria-label="解封"]').trigger('click')
    await settle()
    expect(admin.updateUserBan).toHaveBeenCalledWith(2, false)

    await wrapper.get('[aria-label="删除"]').trigger('click')
    await settle()
    expect(wrapper.find('.user-card.is-leaving').exists()).toBe(true)
    expect(admin.deleteUser).not.toHaveBeenCalled()

    await new Promise(resolve => setTimeout(resolve, 340))
    await settle()
    expect(admin.deleteUser).toHaveBeenCalledWith(2)
    expect(admin.loadUsers).toHaveBeenCalled()
    expect(wrapper.find('.user-card.is-leaving').exists()).toBe(false)
  })

  it('概览指标按当前页统计，页内筛选只过滤当前页', async () => {
    admin.users.list = [currentUser, otherUser, { id: 3, username: 'writer', is_admin: true } as User]
    admin.users.total = 6
    admin.users.totalPages = 2
    admin.users.page = 2
    admin.users.loading = false

    const wrapper = mountUsers()
    await settle()

    const stats = wrapper.findAll('.adm-stat')
    expect(stats).toHaveLength(3)
    expect(stats[0].text()).toContain('6')
    expect(stats[1].text()).toContain('2')
    expect(stats[1].text()).toContain('当前页')
    expect(stats[2].text()).toContain('1')

    const filter = wrapper.get('[aria-label="筛选当前页用户"]')
    expect(filter.text()).toContain('当前页')
    const buttons = filter.findAll('button')
    expect(buttons.map(button => button.text())).toEqual(['全部', '管理员', '已封禁'])
    expect(wrapper.findAll('.user-card')).toHaveLength(3)

    await buttons[1].trigger('click')
    await settle()
    expect(wrapper.findAll('.user-card')).toHaveLength(2)

    await buttons[2].trigger('click')
    await settle()
    expect(wrapper.findAll('.user-card')).toHaveLength(1)
    expect(wrapper.get('.user-card').text()).toContain('已封禁')
  })
})

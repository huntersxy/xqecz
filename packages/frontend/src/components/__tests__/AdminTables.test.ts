// WP5a：内容卡片流 / 用户表格的定向行为测试。
// 覆盖：三种 mode 的渲染差异、卡片网格（桌面与手机同一套）、pending 审核动作、
// 卡片 M8 退场与 280ms 后的延迟刷新、删除确认流，以及用户管理的角色 / 封禁 / 删除三个动作。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { nextTick } from 'vue'
import AdminContentList from '@/components/admin/AdminContentList.vue'
import AdminUserTable from '@/components/admin/AdminUserTable.vue'
import { useAdminStore } from '@/stores/admin'
import { useUserStore } from '@/stores/user'
import type { Content, User } from '@/types'

// useMediaQuery 在 jsdom 里没有 matchMedia，这里改成由测试直接拨动的开关。
const media = vi.hoisted(() => ({ mobile: false }))

vi.mock('@vueuse/core', async () => {
  const actual = await vi.importActual<typeof import('@vueuse/core')>('@vueuse/core')
  const { ref } = await import('vue')
  return { ...actual, useMediaQuery: () => ref(media.mobile) }
})

// stores/admin.ts 在模块顶层取 confirm；这里让确认框一律返回 true。
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

  // 列表加载与写操作全部换成桩：只断言调用契约，不碰真实网络。
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
    // pending 的编辑 / 删除收进「更多」菜单：卡片底部只放通过 / 拒绝 / 更多三个按钮。
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
    // 手机与桌面共用同一份卡片 DOM：通过 / 拒绝 / 更多都在卡片底部。
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
    // 超过 2 个标签折成 +N，原文进 title 提示，不在卡片里铺开
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

    // 操作区已 @click.stop：点「删除」不会再冒泡出一次开抽屉。
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
    // 退场期间只打了标记，刷新要等 280ms 后才发生。
    expect(admin.loadPendingContent).toHaveBeenCalledTimes(1)
  })

  it('退场 280ms 后真正刷新列表，并清掉卡片上的退场标记', async () => {
    seedContentList(admin.pendingContent, [content(31)])

    const wrapper = mountContent('pending')
    await settle()

    await wrapper.get('[aria-label="通过"]').trigger('click')
    await settle()
    expect(wrapper.find('.adm-card.is-leaving').exists()).toBe(true)

    // 真实等待 280ms（不用 fake timers：@vue/test-utils 的 flushPromises 自身依赖 setTimeout）
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
  it('角色 / 封禁用状态点，角色切换 / 封禁 / 删除三个动作都走 store', async () => {
    admin.users.list = [currentUser, otherUser]
    admin.users.total = 2
    admin.users.totalPages = 1
    admin.users.loading = false

    const wrapper = mountUsers()
    await settle()

    expect(wrapper.find('.admin-panel-title').text()).toBe('用户管理')
    expect(wrapper.find('.admin-data-toolbar-summary').text()).toBe('共 2 位用户')
    // 两位用户 × 角色 + 状态 = 4 个状态点；本人不渲染操作按钮。
    expect(wrapper.findAll('.admin-status')).toHaveLength(4)
    expect(wrapper.findAll('[aria-label="删除"]')).toHaveLength(1)

    await wrapper.get('[aria-label="设为管理员"]').trigger('click')
    await settle()
    expect(admin.updateUserRole).toHaveBeenCalledWith(2, true)

    await wrapper.get('[aria-label="解封"]').trigger('click')
    await settle()
    expect(admin.updateUserBan).toHaveBeenCalledWith(2, false)

    await wrapper.get('[aria-label="删除"]').trigger('click')
    await settle()
    expect(admin.deleteUser).toHaveBeenCalledWith(2)
    expect(wrapper.find('tr.is-leaving').exists()).toBe(true)
  })
})

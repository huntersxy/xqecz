
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { createMemoryHistory, createRouter, type RouteRecordRaw, type Router } from 'vue-router'
import { defineComponent, h, nextTick, onMounted, ref, type Component } from 'vue'
import AdminNav from '@/components/admin/AdminNav.vue'
import AdminView from '@/views/AdminView.vue'
import PublicShell from '@/components/PublicShell.vue'
import App from '@/App.vue'
import { contentApi } from '@/api'
import { useAdminStore } from '@/stores/admin'
import { useUserStore } from '@/stores/user'
import { SHELL_SIDEBAR_CLOSE_EVENT } from '@/utils/shellSidebar'
import type { User } from '@/types'

type NavItem = { key: string; title: string; icon: Component; badge?: number }
type NavGroup = { label: string; items: NavItem[] }

vi.mock('@/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api')>()
  const page = async () => ({ data: { list: [], total: 0, page: 1, page_size: 1, total_page: 1 } })
  return {
    ...actual,
    adminApi: { ...actual.adminApi, pending: vi.fn(page), getClaims: vi.fn(page) },
    contentApi: { ...actual.contentApi, getTags: vi.fn(async () => ({ data: [] })) },
    commentApi: { ...actual.commentApi, getReports: vi.fn(async () => ({ data: [] })) },
  }
})

vi.mock('@/composables/useToast', () => ({
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
  useConfirm: () => ({ pendingConfirm: ref(null), confirm: vi.fn(async () => true), respond: vi.fn() }),
}))

const stubPage = (name: string, marker: string) =>
  defineComponent({
    name,
    setup() {
      return () => h('div', { 'data-stub': marker })
    },
  })

const AdminViewStubs = {
  AdminDashboard: stubPage('AdminDashboard', 'dashboard'),
  AdminContentList: defineComponent({
    name: 'AdminContentList',
    props: ['mode'],
    setup(props) {
      return () => h('div', { 'data-stub': 'content', 'data-mode': props.mode })
    },
  }),
  AdminReviewPanel: stubPage('AdminReviewPanel', 'review'),
  AdminUploadPanel: stubPage('AdminUploadPanel', 'upload'),
  AdminUserTable: stubPage('AdminUserTable', 'users'),
  AdminPollPanel: stubPage('AdminPollPanel', 'polls'),
  AdminClaimList: stubPage('AdminClaimList', 'claims'),
  AdminReportList: stubPage('AdminReportList', 'reports'),
  AdminVisionSettings: stubPage('AdminVisionSettings', 'vision'),
  AdminAI: stubPage('AdminAI', 'ai'),
  AdminLogsTerminal: stubPage('AdminLogsTerminal', 'logs'),
  AdminApiKeys: stubPage('AdminApiKeys', 'api-keys'),
  AdminApiDocs: stubPage('AdminApiDocs', 'api-docs'),
  AdminContentDrawer: stubPage('AdminContentDrawer', 'drawer'),
  transition: true,
}

const icon: Component = () => h('i', { 'data-test-icon': 'true' })

function fixtureGroups(): NavGroup[] {
  return [
    { label: '总览', items: [{ key: 'dashboard', title: '仪表盘', icon }] },
    {
      label: '内容',
      items: [
        { key: 'my', title: '我的内容', icon },
        { key: 'upload', title: '上传内容', icon },
      ],
    },
    {
      label: '管理',
      items: [
        { key: 'pending', title: '审核内容', icon, badge: 3 },
        { key: 'users', title: '用户管理', icon },
      ],
    },
    { label: '系统', items: [{ key: 'api-keys', title: 'API 密钥', icon }] },
  ]
}

const adminUser = { id: 1, username: 'root', email: 'root@example.com', is_admin: true } as unknown as User
const normalUser = { id: 2, username: 'reader', email: 'reader@example.com', is_admin: false } as unknown as User

const mountedWrappers: VueWrapper[] = []

function track<T extends VueWrapper>(wrapper: T): T {
  mountedWrappers.push(wrapper)
  return wrapper
}

async function settle() {
  await flushPromises()
  await nextTick()
  await flushPromises()
}

function ensureWorkspaceSlot() {
  if (!document.querySelector('#public-shell-workspace')) {
    const slot = document.createElement('div')
    slot.id = 'public-shell-workspace'
    document.body.appendChild(slot)
  }
}

function seedUser(pinia: Pinia, isAdmin: boolean) {
  setActivePinia(pinia)
  const userStore = useUserStore()
  userStore.user = isAdmin ? adminUser : normalUser
  userStore.isLoggedIn = true
}

function navRoot(): HTMLElement {
  const el = document.querySelector<HTMLElement>('#public-shell-workspace .admin-nav')
  if (!el) throw new Error('AdminNav 没有 teleport 进 #public-shell-workspace')
  return el
}

function navItemButtons(): HTMLButtonElement[] {
  return Array.from(navRoot().querySelectorAll<HTMLButtonElement>('.admin-nav-item'))
}

function navItemByTitle(title: string): HTMLButtonElement {
  const button = navItemButtons().find(
    (item) => item.querySelector('.admin-nav-item-label')?.textContent?.trim() === title,
  )
  if (!button) throw new Error('找不到导航项：' + title)
  return button
}

function navGroupLabels(): string[] {
  return Array.from(navRoot().querySelectorAll<HTMLElement>('.admin-nav-group')).map(
    (group) => group.getAttribute('aria-label') ?? '',
  )
}

function groupToggle(label: string): HTMLButtonElement {
  const toggle = Array.from(navRoot().querySelectorAll<HTMLButtonElement>('.admin-nav-group-toggle')).find(
    (button) => button.textContent?.trim() === label,
  )
  if (!toggle) throw new Error('找不到分组：' + label)
  return toggle
}

function groupBody(label: string): HTMLElement {
  const id = groupToggle(label).getAttribute('aria-controls')
  const body = id ? document.getElementById(id) : null
  if (!body) throw new Error('找不到分组容器：' + label)
  return body
}

async function clickNavItem(title: string) {
  navItemByTitle(title).click()
  await settle()
}

async function clickGroupToggle(label: string) {
  groupToggle(label).click()
  await settle()
}

function mountNav(overrides: { groups?: NavGroup[]; active?: string; isAdmin?: boolean } = {}) {
  ensureWorkspaceSlot()
  return track(
    mount(AdminNav, {
      props: { groups: fixtureGroups(), active: 'dashboard', isAdmin: true, ...overrides },
      attachTo: document.body,
    }),
  )
}

const EmptyRoute = defineComponent({ name: 'EmptyRoute', render: () => h('div') })

function adminRoutes(): RouteRecordRaw[] {
  return [
    { path: '/admin', name: 'admin', component: EmptyRoute },
    { path: '/', name: 'home', component: EmptyRoute },
  ]
}

async function mountAdminView(path: string, isAdmin = true) {
  ensureWorkspaceSlot()
  const pinia = createPinia()
  seedUser(pinia, isAdmin)
  const router = createRouter({ history: createMemoryHistory(), routes: adminRoutes() })
  await router.push(path)
  await router.isReady()
  const wrapper = track(
    mount(AdminView, {
      global: { plugins: [pinia, router], stubs: AdminViewStubs },
      attachTo: document.body,
    }),
  )
  await settle()
  setActivePinia(pinia)
  return { wrapper, router, store: useAdminStore() }
}

async function routerAt(path: string): Promise<Router> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: EmptyRoute }],
  })
  await router.push(path)
  await router.isReady()
  return router
}

beforeEach(() => {
  localStorage.clear()
  document.body.innerHTML = ''
  vi.clearAllMocks()
  vi.stubGlobal('matchMedia', (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
  Object.defineProperty(window, 'scrollTo', { value: vi.fn(), writable: true, configurable: true })
})

afterEach(() => {
  for (const wrapper of mountedWrappers.splice(0)) {
    try {
      wrapper.unmount()
    } catch {
    }
  }
  document.body.innerHTML = ''
  localStorage.clear()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

describe('AdminNav 后台导航', () => {
  describe('分组与页签', () => {
    it('管理员看到 总览 / 内容 / 管理 / 系统 四组，且各组页签 key 与预期顺序一致', async () => {
      const { wrapper } = await mountAdminView('/admin?tab=dashboard')

      expect(navGroupLabels()).toEqual(['总览', '内容', '管理', '系统'])

      const nav = wrapper.findComponent(AdminNav)
      const clickedKeys: string[] = []
      for (const button of navItemButtons()) {
        button.click()
        await settle()
        const emitted = nav.emitted('select') ?? []
        clickedKeys.push(String(emitted[emitted.length - 1]?.[0]))
      }

      expect(clickedKeys).toEqual([
        'dashboard',
        'my',
        'upload',
        'review',
        'all',
        'users',
        'polls',
        'reports',
        'vision',
        'ai',
        'api-keys',
        'api-docs',
        'logs',
      ])
    })

    it('普通用户只看到 内容 / 系统 两组，且不含任何管理员专属页签', async () => {
      await mountAdminView('/admin', false)

      expect(navGroupLabels()).toEqual(['内容', '系统'])
      expect(navItemButtons().map((button) => button.textContent?.trim())).toEqual([
        '我的内容',
        '上传内容',
        'API 密钥',
        'API 文档',
      ])
      expect(document.body.textContent).not.toContain('用户管理')
    })
  })

  describe('交互与无障碍', () => {
    it('点击导航项派发 select 事件，事件载荷是该页签的 key', async () => {
      const wrapper = mountNav({ active: 'dashboard' })
      await settle()

      await clickNavItem('上传内容')

      expect(wrapper.emitted('select')).toEqual([['upload']])
    })

    it('徽标的 aria-label 为「{标题}，{n} 项待处理」，数量超过 99 时显示 99+', async () => {
      mountNav({
        groups: [
          {
            label: '管理',
            items: [
              { key: 'pending', title: '审核内容', icon, badge: 7 },
              { key: 'claims', title: '认领管理', icon, badge: 150 },
            ],
          },
        ],
        active: 'pending',
      })
      await settle()

      const pending = navItemByTitle('审核内容')
      expect(pending.getAttribute('aria-label')).toBe('审核内容，7 项待处理')
      expect(pending.querySelector('.admin-nav-badge')?.textContent?.trim()).toBe('7')

      const claims = navItemByTitle('认领管理')
      expect(claims.getAttribute('aria-label')).toBe('认领管理，150 项待处理')
      expect(claims.querySelector('.admin-nav-badge')?.textContent?.trim()).toBe('99+')
    })

    it('当前页签项带 aria-current="page"，其余项不带该属性', async () => {
      mountNav({ active: 'my' })
      await settle()

      expect(navItemByTitle('我的内容').getAttribute('aria-current')).toBe('page')
      for (const title of ['仪表盘', '上传内容', '审核内容', '用户管理', 'API 密钥']) {
        expect(navItemByTitle(title).hasAttribute('aria-current')).toBe(false)
      }
    })
  })

  describe('折叠状态持久化', () => {
    it('分组折叠状态以 JSON 数组写入 localStorage 的 admin_nav_collapsed_groups', async () => {
      const first = mountNav({ active: 'dashboard' })
      await settle()

      await clickGroupToggle('管理')

      expect(JSON.parse(localStorage.getItem('admin_nav_collapsed_groups') ?? '[]')).toEqual(['管理'])
      expect(groupToggle('管理').getAttribute('aria-expanded')).toBe('false')
      expect(groupToggle('管理').classList.contains('is-collapsed')).toBe(true)
      expect(groupBody('管理').classList.contains('is-collapsed')).toBe(true)

      first.unmount()
      mountNav({ active: 'dashboard' })
      await settle()
      expect(groupToggle('管理').getAttribute('aria-expanded')).toBe('false')
      expect(groupBody('管理').classList.contains('is-collapsed')).toBe(true)
    })

    it('localStorage 读写抛出异常时静默降级为全部展开，不向外抛错', async () => {
      const getItem = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
        throw new Error('storage disabled')
      })
      const setItem = vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
        throw new Error('quota exceeded')
      })

      expect(() => mountNav({ active: 'dashboard' })).not.toThrow()
      await settle()

      for (const label of ['总览', '内容', '管理', '系统']) {
        expect(groupToggle(label).getAttribute('aria-expanded')).toBe('true')
      }

      await clickGroupToggle('管理')
      expect(groupToggle('管理').getAttribute('aria-expanded')).toBe('false')
      expect(getItem).toHaveBeenCalled()
      expect(setItem).toHaveBeenCalled()
    })
  })

  describe('布局与挂载', () => {
    it('手机宽度下选中导航项会触发关闭站点侧栏抽屉', async () => {
      const pinia = createPinia()
      seedUser(pinia, true)
      const router = await routerAt('/admin')
      const wrapper = track(
        mount(PublicShell, {
          global: { plugins: [pinia, router] },
          slots: {
            default: () => h(AdminNav, { groups: fixtureGroups(), active: 'my', isAdmin: true }),
          },
          attachTo: document.body,
        }),
      )
      await settle()

      await wrapper.get('.public-shell-menu').trigger('click')
      expect(wrapper.classes()).toContain('is-mobile-open')

      const closed = vi.fn()
      window.addEventListener(SHELL_SIDEBAR_CLOSE_EVENT, closed)
      await clickNavItem('用户管理')
      window.removeEventListener(SHELL_SIDEBAR_CLOSE_EVENT, closed)

      expect(closed).toHaveBeenCalledTimes(1)
      expect(wrapper.classes()).not.toContain('is-mobile-open')
    })

    it('导航渲染在站点侧栏工作区槽 #public-shell-workspace 内', async () => {
      const wrapper = mountNav({ active: 'dashboard' })
      await settle()

      expect(document.querySelector('#public-shell-workspace .admin-nav')).not.toBeNull()
      expect(wrapper.find('.admin-nav').exists()).toBe(false)
    })
  })
})

describe('AdminView 页签与 URL 契约', () => {
  it('深链 ?tab=users 进入即显示用户管理，刷新后仍保持', async () => {
    const first = await mountAdminView('/admin?tab=users')
    expect(first.store.activeTab).toBe('users')
    expect(first.router.currentRoute.value.query.tab).toBe('users')
    expect(first.wrapper.find('[data-stub="users"]').exists()).toBe(true)
    expect(navItemByTitle('用户管理').getAttribute('aria-current')).toBe('page')

    first.wrapper.unmount()
    const second = await mountAdminView('/admin?tab=users')
    expect(second.store.activeTab).toBe('users')
    expect(second.wrapper.find('[data-stub="users"]').exists()).toBe(true)
    expect(navItemByTitle('用户管理').getAttribute('aria-current')).toBe('page')
  })

  it('首次进入 /admin（无 ?tab=）会补写默认页签，刷新后仍可保持', async () => {
    const { router, store } = await mountAdminView('/admin')

    expect(store.activeTab).toBe('dashboard')
    expect(router.currentRoute.value.query.tab).toBe('dashboard')
  })

  it('切换页签走 push，浏览器返回键回到上一页签', async () => {
    const { wrapper, router, store } = await mountAdminView('/admin?tab=my')
    expect(store.activeTab).toBe('my')

    await clickNavItem('用户管理')
    expect(store.activeTab).toBe('users')
    expect(router.currentRoute.value.query.tab).toBe('users')
    expect(wrapper.find('[data-stub="users"]').exists()).toBe(true)

    await router.back()
    await settle()

    expect(router.currentRoute.value.query.tab).toBe('my')
    expect(store.activeTab).toBe('my')
    expect(wrapper.find('[data-stub="content"]').attributes('data-mode')).toBe('my')
    expect(navItemByTitle('我的内容').getAttribute('aria-current')).toBe('page')
  })

  it('普通用户访问越权 ?tab=users 会被 replace 回 my，且不渲染 403', async () => {
    const { wrapper, router, store } = await mountAdminView('/admin?tab=users', false)

    expect(store.activeTab).toBe('my')
    expect(router.currentRoute.value.path).toBe('/admin')
    expect(router.currentRoute.value.query.tab).toBe('my')
    expect(wrapper.text()).not.toContain('没有访问此管理功能的权限')
    expect(wrapper.find('[data-stub="users"]').exists()).toBe(false)
    expect(wrapper.find('[data-stub="content"]').attributes('data-mode')).toBe('my')
    expect(navGroupLabels()).toEqual(['内容', '系统'])
  })

  it('组件层：切换页签不会重建 AdminView 内的 AdminNav 实例', async () => {
    const { wrapper, store } = await mountAdminView('/admin?tab=dashboard')

    const navUidBefore = wrapper.findComponent(AdminNav).vm.$.uid
    const navNodeBefore = navRoot()
    expect(contentApi.getTags).toHaveBeenCalledTimes(1)

    await clickNavItem('用户管理')

    expect(store.activeTab).toBe('users')
    expect(wrapper.findComponent(AdminNav).vm.$.uid).toBe(navUidBefore)
    expect(navRoot()).toBe(navNodeBefore)
    expect(contentApi.getTags).toHaveBeenCalledTimes(1)

  })
})

describe('App.vue 的 /admin 稳定 key', () => {
  const probeMounts = { admin: 0, plain: 0 }

  const adminProbe = defineComponent({
    name: 'AdminProbe',
    setup() {
      onMounted(() => {
        probeMounts.admin += 1
      })
      return () => h('div', { 'data-probe': 'admin' })
    },
  })

  const plainProbe = defineComponent({
    name: 'PlainProbe',
    setup() {
      onMounted(() => {
        probeMounts.plain += 1
      })
      return () => h('div', { 'data-probe': 'plain' })
    },
  })

  function appRoutes(): RouteRecordRaw[] {
    return [
      { path: '/', name: 'home', component: EmptyRoute },
      { path: '/admin', name: 'admin', component: adminProbe },
      { path: '/plain', name: 'plain', component: plainProbe },
      { path: '/:pathMatch(.*)*', name: 'not-found', component: EmptyRoute },
    ]
  }

  async function mountApp(path: string) {
    const pinia = createPinia()
    seedUser(pinia, true)
    const router = createRouter({ history: createMemoryHistory(), routes: appRoutes() })
    await router.push(path)
    await router.isReady()
    const wrapper = track(
      mount(App, { global: { plugins: [pinia, router], stubs: { transition: true } }, attachTo: document.body }),
    )
    await settle()
    return { wrapper, router }
  }

  it('/admin 内只改 query 不重建页面，其它路由仍按 fullPath 重建（行为不变）', async () => {
    probeMounts.admin = 0
    probeMounts.plain = 0

    const { router } = await mountApp('/admin')
    expect(probeMounts.admin).toBe(1)
    expect(document.querySelector('[data-probe="admin"]')).not.toBeNull()

    await router.push('/admin?tab=users')
    await settle()
    expect(probeMounts.admin).toBe(1)

    await router.push('/admin?tab=dashboard')
    await settle()
    expect(probeMounts.admin).toBe(1)

    await router.push('/admin')
    await settle()
    expect(probeMounts.admin).toBe(1)

    await router.push('/plain?page=1')
    await settle()
    expect(probeMounts.plain).toBe(1)

    await router.push('/plain?page=2')
    await settle()
    expect(probeMounts.plain).toBe(2)
    expect(probeMounts.admin).toBe(1)
  })
})

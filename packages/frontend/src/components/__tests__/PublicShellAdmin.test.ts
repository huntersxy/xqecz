// WP2 · D1：/admin 复用站点壳侧栏的受控改动回归。
// 只验证「/admin 的槽可见 + 站点壳拿到的状态类」与「/studio、其它路由表现不变」，
// 槽内滚动样式属于纯 CSS，留给 Lead 浏览器实测（WP2 验收第 9 条）。

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h, type Component } from 'vue'
import PublicShell from '@/components/PublicShell.vue'
import AdminNav from '@/components/admin/AdminNav.vue'
import { useUserStore } from '@/stores/user'
import type { User } from '@/types'

const EmptyRoute = defineComponent({ name: 'EmptyRoute', render: () => h('div') })
const icon: Component = () => h('i')
const groups = [{ label: '内容', items: [{ key: 'my', title: '我的内容', icon }] }]

const mountedWrappers: VueWrapper[] = []

async function render(path: string, withNav = false) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const userStore = useUserStore()
  userStore.user = { id: 1, username: 'root', is_admin: true } as unknown as User
  userStore.isLoggedIn = true

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: EmptyRoute }],
  })
  await router.push(path)
  await router.isReady()

  const wrapper = mount(PublicShell, {
    global: { plugins: [pinia, router] },
    slots: withNav
      ? { default: () => h(AdminNav, { groups, active: 'my', isAdmin: true }) }
      : undefined,
    attachTo: document.body,
  })
  mountedWrappers.push(wrapper)
  await flushPromises()
  return wrapper
}

function assertSidebarOrder(wrapper: VueWrapper) {
  const order = [
    '.public-shell-brand-row',
    '.public-shell-nav',
    '#public-shell-workspace',
    '.public-shell-bottom',
  ]
  const nodes = order.map((selector) => wrapper.get(selector).element)
  for (let i = 1; i < nodes.length; i += 1) {
    expect(nodes[i - 1].compareDocumentPosition(nodes[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  }
}

beforeEach(() => {
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
})

afterEach(() => {
  for (const wrapper of mountedWrappers.splice(0)) {
    try {
      wrapper.unmount()
    } catch {
      // 已手动卸载
    }
  }
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('PublicShell 在 /admin 下的受控改动（D1）', () => {
  it('/admin：打开工作区模式并标记 is-admin，侧栏槽可见', async () => {
    const wrapper = await render('/admin')

    expect(wrapper.classes()).toContain('is-workspace')
    expect(wrapper.classes()).toContain('is-admin')
    expect(wrapper.find('#public-shell-workspace').exists()).toBe(true)
    assertSidebarOrder(wrapper)
  })

  it('/studio：仍只打开工作区模式，不误加 is-admin', async () => {
    const wrapper = await render('/studio')

    expect(wrapper.classes()).toContain('is-workspace')
    expect(wrapper.classes()).not.toContain('is-admin')
    expect(wrapper.find('#public-shell-workspace').exists()).toBe(true)
  })

  it('其它路由：既不打开工作区模式也不带 is-admin（槽仍按既有规则隐藏）', async () => {
    const wrapper = await render('/')

    expect(wrapper.classes()).not.toContain('is-workspace')
    expect(wrapper.classes()).not.toContain('is-admin')
  })

  it('/admin 下 AdminNav teleport 进站点侧栏槽，而不是留在页面内容区', async () => {
    await render('/admin', true)

    expect(document.querySelector('#public-shell-workspace .admin-nav')).not.toBeNull()
    // 槽本身在 PublicShell 的 DOM 里，改用「不在页面内容区」判定 teleport 生效。
    expect(document.querySelector('.public-shell-content .admin-nav')).toBeNull()
  })
})

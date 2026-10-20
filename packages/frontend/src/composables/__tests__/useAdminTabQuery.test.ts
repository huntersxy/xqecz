import { describe, expect, it, vi } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import {
  ADMIN_TAB_KEYS,
  COMMON_TAB_KEYS,
  adminTabsFor,
  defaultAdminTab,
  useAdminTabQuery,
} from '../useAdminTabQuery'

const Empty = defineComponent({ render: () => h('div') })

async function settle() {
  await flushPromises()
  await nextTick()
  await flushPromises()
}

async function mountHarness(url: string, isAdminFlag: boolean) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Empty },
      { path: '/admin', component: Empty },
    ],
  })
  await router.push(url)
  await router.isReady()
  const replaceSpy = vi.spyOn(router, 'replace')
  const pushSpy = vi.spyOn(router, 'push')

  const activeTab = ref('my')
  const isAdmin = ref(isAdminFlag)
  let api!: ReturnType<typeof useAdminTabQuery>
  mount(
    defineComponent({
      setup() {
        api = useAdminTabQuery({ isAdmin, activeTab })
        return () => h('div')
      },
    }),
    { global: { plugins: [router] } },
  )
  await settle()

  return { router, activeTab, isAdmin, api, replaceSpy, pushSpy }
}

describe('useAdminTabQuery（§7.4 URL 页签同步）', () => {
  it('管理员访问合法 query 时原样采用，不做 replace', async () => {
    const { router, activeTab, api, replaceSpy } = await mountHarness('/admin?tab=logs', true)

    expect(activeTab.value).toBe('logs')
    expect(router.currentRoute.value.query.tab).toBe('logs')
    expect(replaceSpy).not.toHaveBeenCalled()
    expect(api.allowedTabs.value).toEqual(ADMIN_TAB_KEYS)
  })

  it('管理员缺少 query 时用默认页总览，并把地址补全', async () => {
    const { router, activeTab, replaceSpy } = await mountHarness('/admin', true)

    expect(activeTab.value).toBe('dashboard')
    expect(router.currentRoute.value.query.tab).toBe('dashboard')
    expect(replaceSpy).toHaveBeenCalledTimes(1)
    expect(replaceSpy).toHaveBeenCalledWith({ query: { tab: 'dashboard' } })
  })

  it('非法 query 规范化到默认页，并保留其余参数', async () => {
    const { router, activeTab, replaceSpy } = await mountHarness('/admin?tab=nope&keep=1', true)

    expect(activeTab.value).toBe('dashboard')
    expect(router.currentRoute.value.query).toMatchObject({ tab: 'dashboard', keep: '1' })
    expect(replaceSpy).toHaveBeenCalledTimes(1)
  })

  it('重复参数（数组）同样视为非法', async () => {
    const { router, activeTab } = await mountHarness('/admin?tab=my&tab=users', true)

    expect(activeTab.value).toBe('dashboard')
    expect(router.currentRoute.value.query.tab).toBe('dashboard')
  })

  it('旧别名 pending / claims 规范化到合并后的 review，其余参数保留', async () => {
    const one = await mountHarness('/admin?tab=pending', true)

    expect(one.activeTab.value).toBe('review')
    expect(one.router.currentRoute.value.query.tab).toBe('review')
    expect(one.replaceSpy).toHaveBeenCalledTimes(1)

    const two = await mountHarness('/admin?tab=claims&keep=2', true)

    expect(two.activeTab.value).toBe('review')
    expect(two.router.currentRoute.value.query).toMatchObject({ tab: 'review', keep: '2' })
  })

  it('合并前的遗留 sub 参数被清掉，不影响页签', async () => {
    const { router, activeTab } = await mountHarness('/admin?tab=review&sub=claims', true)

    expect(activeTab.value).toBe('review')
    expect(router.currentRoute.value.query).toEqual({ tab: 'review' })
  })

  it('selectTab 收到旧 key 时落到 review', async () => {
    const { router, activeTab, api } = await mountHarness('/admin?tab=my', true)

    api.selectTab('claims')
    await settle()

    expect(activeTab.value).toBe('review')
    expect(router.currentRoute.value.query).toEqual({ tab: 'review' })
  })

  it('普通用户访问越权页签 users 时替换为「我的内容」', async () => {
    const { router, activeTab, api, replaceSpy } = await mountHarness('/admin?tab=users', false)

    expect(activeTab.value).toBe('my')
    expect(router.currentRoute.value.query.tab).toBe('my')
    expect(replaceSpy).toHaveBeenCalledTimes(1)
    expect(api.allowedTabs.value).toEqual(COMMON_TAB_KEYS)
    expect(api.isAllowed('users')).toBe(false)
  })

  it('普通用户访问公共页签 upload 时保持不动', async () => {
    const { router, activeTab, replaceSpy } = await mountHarness('/admin?tab=upload', false)

    expect(activeTab.value).toBe('upload')
    expect(router.currentRoute.value.query.tab).toBe('upload')
    expect(replaceSpy).not.toHaveBeenCalled()
  })

  it('选择页签：合法则 push，非法直接忽略', async () => {
    const { router, activeTab, api } = await mountHarness('/admin?tab=my', true)

    api.selectTab('polls')
    await settle()
    expect(activeTab.value).toBe('polls')
    expect(router.currentRoute.value.query.tab).toBe('polls')

    api.selectTab('nope')
    await settle()
    expect(activeTab.value).toBe('polls')
    expect(router.currentRoute.value.query.tab).toBe('polls')
  })

  it('普通用户调用 selectTab 也无法进入管理员页签', async () => {
    const { router, activeTab, api } = await mountHarness('/admin?tab=my', false)

    api.selectTab('users')
    await settle()
    expect(activeTab.value).toBe('my')
    expect(router.currentRoute.value.query.tab).toBe('my')
  })

  it('角色掉权后，当前越权页签被送回默认页', async () => {
    const { router, activeTab, api, isAdmin } = await mountHarness('/admin?tab=my', true)

    api.selectTab('users')
    await settle()
    expect(activeTab.value).toBe('users')

    isAdmin.value = false
    await settle()
    expect(activeTab.value).toBe('my')
    expect(router.currentRoute.value.query.tab).toBe('my')
    expect(api.allowedTabs.value).toEqual(COMMON_TAB_KEYS)
  })

  it('浏览器前进后退时，页签跟随 query 更新', async () => {
    const { router, activeTab, api } = await mountHarness('/admin?tab=my', true)

    api.selectTab('all')
    await settle()
    api.selectTab('reports')
    await settle()
    expect(router.currentRoute.value.query.tab).toBe('reports')

    router.back()
    await settle()
    expect(activeTab.value).toBe('all')
    expect(router.currentRoute.value.query.tab).toBe('all')

    router.back()
    await settle()
    expect(activeTab.value).toBe('my')

    router.forward()
    await settle()
    expect(activeTab.value).toBe('all')
  })

  it('纯函数：角色 → 合法页签集合与默认页', () => {
    expect(adminTabsFor(true)).toEqual(ADMIN_TAB_KEYS)
    expect(adminTabsFor(false)).toEqual(COMMON_TAB_KEYS)
    expect(defaultAdminTab(true)).toBe('dashboard')
    expect(defaultAdminTab(false)).toBe('my')
  })
})

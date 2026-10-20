import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, onActivated, onDeactivated, onMounted, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import App from '@/App.vue'

vi.mock('@/stores/theme', () => ({ useThemeStore: () => ({}) }))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ isLoggedIn: true }) }))

afterEach(() => vi.restoreAllMocks())

describe('page navigation with real Vue transitions', () => {
  it('mounts the next page immediately, survives rapid switches and restores the cached homepage', async () => {
    const homeMounted = vi.fn()
    const activated = vi.fn()
    const deactivated = vi.fn()
    const scroll = vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
    const Home = defineComponent({
      name: 'HomeView',
      setup() {
        onMounted(homeMounted)
        onActivated(activated)
        onDeactivated(deactivated)
        return () => h('div', { class: 'home' }, '首页')
      },
    })
    const Page = defineComponent({ setup: () => () => h('div', { class: 'page' }, '页面') })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'home', component: Home },
        { path: '/ideas', component: Page },
        { path: '/polls', component: Page },
        { path: '/content/:id', name: 'content-detail', component: Page },
      ],
    })
    await router.push('/')
    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: {
          transition: false,
          PublicShell: { template: '<div><slot /></div>' },
          ErrorBoundary: { template: '<div><slot /></div>' },
          Toaster: true,
          ConfirmDialog: true,
        },
      },
    })
    await router.push('/ideas')
    await nextTick()
    expect(wrapper.find('.page').exists()).toBe(true)
    expect(wrapper.find('.home').exists()).toBe(false)
    await router.push('/polls')
    await router.push('/')
    await flushPromises()
    expect(wrapper.find('.home').exists()).toBe(true)
    expect(homeMounted).toHaveBeenCalledTimes(1)
    expect(activated).toHaveBeenCalledTimes(2)
    expect(deactivated).toHaveBeenCalledTimes(1)
    scroll.mockClear()
    await router.push('/content/1')
    await router.push('/content/2')
    await router.push('/')
    await flushPromises()
    expect(wrapper.find('.home').exists()).toBe(true)
    expect(scroll).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

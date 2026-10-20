import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import RouteProgress from '../RouteProgress.vue'

const Page = defineComponent({ setup: () => () => h('div') })

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: Page },
      { path: '/slow', component: Page },
      { path: '/boom', component: Page, beforeEnter: () => { throw new Error('boom') } },
    ],
  })
}

beforeEach(() => { vi.useFakeTimers() })
afterEach(() => { vi.useRealTimers() })

async function mountBar(router: ReturnType<typeof makeRouter>) {
  await router.push('/')
  return mount(RouteProgress, { global: { plugins: [router] } })
}

describe('RouteProgress', () => {
  it('stays hidden for navigations that finish before the delay', async () => {
    const router = makeRouter()
    const wrapper = await mountBar(router)
    await router.push('/slow')
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.classes()).not.toContain('is-visible')
    wrapper.unmount()
  })

  it('shows while a slow navigation is pending, then completes and hides', async () => {
    const router = makeRouter()
    let release!: () => void
    const wrapper = await mountBar(router)
    router.beforeEach((to) => to.path === '/slow'
      ? new Promise<void>((resolve) => { release = resolve })
      : undefined)
    const pending = router.push('/slow')
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.classes()).toContain('is-visible')

    release()
    await pending
    await flushPromises()
    expect(wrapper.get('.route-progress-bar').attributes('style')).toContain('scaleX(1)')
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.classes()).not.toContain('is-visible')
    wrapper.unmount()
  })

  it('ignores same-path query changes', async () => {
    const router = makeRouter()
    const wrapper = await mountBar(router)
    await router.push({ path: '/', query: { q: '1' } })
    await vi.advanceTimersByTimeAsync(400)
    expect(wrapper.classes()).not.toContain('is-visible')
    wrapper.unmount()
  })

  it('wraps up when a navigation fails', async () => {
    const router = makeRouter()
    const wrapper = await mountBar(router)
    router.beforeEach(async (to) => {
      if (to.path === '/boom') await new Promise((r) => setTimeout(r, 300))
    })
    const failed = router.push('/boom').catch(() => {})
    await vi.advanceTimersByTimeAsync(200)
    expect(wrapper.classes()).toContain('is-visible')
    await vi.advanceTimersByTimeAsync(400)
    await failed
    await vi.advanceTimersByTimeAsync(600)
    expect(wrapper.classes()).not.toContain('is-visible')
    wrapper.unmount()
  })
})

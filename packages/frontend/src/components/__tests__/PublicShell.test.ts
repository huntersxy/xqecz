import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PublicShell from '@/components/PublicShell.vue'

async function render(path: string, slot = '') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  await router.push(path)
  await router.isReady()
  return mount(PublicShell, {
    slots: { default: slot },
    global: {
      plugins: [createPinia(), router],
    },
  })
}

describe('public shell sidebar', () => {
  it.each(['pointerover', 'focusin', 'pointerdown'])('preloads nested links on %s without navigating', async (event) => {
    const wrapper = await render('/', '<a href="#/intent-page/1"><span class="intent-label">页面</span></a>')
    const loader = vi.fn().mockResolvedValue({ template: '<div />' })
    wrapper.vm.$router.addRoute({ path: '/intent-page/:id', component: loader })
    await wrapper.get('.intent-label').trigger(event)
    await wrapper.get('.intent-label').trigger(event)
    await flushPromises()
    expect(loader).toHaveBeenCalledTimes(1)
    expect(wrapper.vm.$router.currentRoute.value.path).toBe('/')
    wrapper.unmount()
  })

  it('preloads publish buttons and dynamic content cards, but ignores external links', async () => {
    const wrapper = await render('/', '<div data-preload-route="/content/1"><span class="card-label">作品</span></div><a href="https://example.com/#/external" class="external">站外</a>')
    const publish = vi.fn().mockResolvedValue({ template: '<div />' })
    const detail = vi.fn().mockResolvedValue({ template: '<div />' })
    const external = vi.fn().mockResolvedValue({ template: '<div />' })
    wrapper.vm.$router.addRoute({ path: '/quick-upload', component: publish })
    wrapper.vm.$router.addRoute({ path: '/content/:id', component: detail })
    wrapper.vm.$router.addRoute({ path: '/external', component: external })
    await wrapper.get('.public-shell-publish').trigger('pointerdown')
    await wrapper.get('.card-label').trigger('focusin')
    await wrapper.get('.external').trigger('pointerover')
    await flushPromises()
    expect(publish).toHaveBeenCalledTimes(1)
    expect(detail).toHaveBeenCalledTimes(1)
    expect(external).not.toHaveBeenCalled()
    expect(wrapper.vm.$router.currentRoute.value.path).toBe('/')
    wrapper.unmount()
  })

  it('keeps the shared nav above the studio workspace slot', async () => {
    const wrapper = await render('/studio')
    const order = [
      '.public-shell-brand-row',
      '.public-shell-nav',
      '#public-shell-workspace',
      '.public-shell-bottom',
    ]
    const nodes = order.map((selector) => wrapper.get(selector).element)
    for (let index = 1; index < nodes.length; index += 1) {
      expect(
        nodes[index - 1]!.compareDocumentPosition(nodes[index]!) &
          Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy()
    }
  })
})

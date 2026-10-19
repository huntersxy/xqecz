import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PublicShell from '@/components/PublicShell.vue'

async function render(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }],
  })
  await router.push(path)
  await router.isReady()
  return mount(PublicShell, {
    global: {
      plugins: [createPinia(), router],
    },
  })
}

describe('public shell sidebar', () => {
  it('keeps the shared nav and library above the studio workspace slot', async () => {
    const wrapper = await render('/studio')
    const order = [
      '.public-shell-brand-row',
      '.public-shell-nav',
      '.public-shell-library',
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

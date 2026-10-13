import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import IdeasView from '@/views/IdeasView.vue'
import CreativeCatalogView from '@/views/CreativeCatalogView.vue'

const api = vi.hoisted(() => ({ ideas: vi.fn(), resources: vi.fn() }))
vi.mock('@/api/creative', () => ({ aiApi: { ideas: api.ideas }, creativeApi: { resources: api.resources } }))
const wrappers: ReturnType<typeof mount>[] = []
async function render(view: typeof IdeasView | typeof CreativeCatalogView, path: string) {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path, component: view }] })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(view, { global: { plugins: [router, createPinia()], stubs: { CreativeImage: true } } })
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}
beforeEach(() => {
  vi.clearAllMocks()
  api.resources.mockResolvedValue([])
  api.ideas.mockResolvedValue([])
})
afterEach(() => wrappers.splice(0).forEach((wrapper) => wrapper.unmount()))

describe('creative discovery pages', () => {
  it('gives a text-only idea a visible preview while preserving its detail destination and cost', async () => {
    api.ideas.mockResolvedValue([{ idea: { id: 7, title: '午后的小泉', description: '窗边的一束光', status: 'waiting', model: 'image-model' }, author: { username: '创作者' }, references: [], results: [], sponsorable: true, current_cost: 3 }])
    const wrapper = await render(IdeasView, '/ideas')
    const card = wrapper.get('a.idea-card')
    expect(card.attributes('href')).toBe('/ideas/7')
    expect(card.text()).toContain('一段灵感，等待被点亮')
    expect(card.text()).toContain('3 额度')
    expect(card.text()).toContain('午后的小泉')
  })
  it('makes the character card destination explicit and keeps search filtering working', async () => {
    api.resources.mockResolvedValue([{ id: 2, slug: 'koizumi', title: '小泉', summary: '角色简介', payload: { asset_ids: [] } }])
    const wrapper = await render(CreativeCatalogView, '/characters')
    expect(wrapper.get('a.character-card').attributes('href')).toBe('/characters/koizumi')
    expect(wrapper.get('a.character-card').text()).toContain('查看角色设定')
    await wrapper.get('input').setValue('不存在的角色')
    expect(wrapper.find('a.character-card').exists()).toBe(false)
    await wrapper.get('input').setValue('小泉')
    expect(wrapper.find('a.character-card').exists()).toBe(true)
  })
})

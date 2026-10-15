import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import IdeasView from '@/views/IdeasView.vue'
import CreativeCatalogView from '@/views/CreativeCatalogView.vue'
import IdeasSection from '@/components/IdeasSection.vue'

const api = vi.hoisted(() => ({ ideas: vi.fn(), resources: vi.fn() }))
vi.mock('@/api/creative', () => ({
  aiApi: { ideas: api.ideas },
  creativeApi: { resources: api.resources },
}))
const wrappers: ReturnType<typeof mount>[] = []
async function render(
  view: typeof IdeasView | typeof CreativeCatalogView | typeof IdeasSection,
  path: string,
) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path, component: view }],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(view, {
    global: { plugins: [router, createPinia()], stubs: { CreativeImage: true } },
  })
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
  it('shows a bounded prompt and small original references on the homepage, including realized ideas', async () => {
    const prompt = '窗边的角色，午后暖光。'.repeat(30)
    api.ideas.mockResolvedValue([
      {
        idea: {
          id: 7,
          title: '窗边的午后',
          snapshot: JSON.stringify({ prompt }),
          status: 'realized',
        },
        author: { username: '创作者' },
        references: [1, 2, 3, 4].map((id) => ({ id })),
        results: [{ id: 99 }],
        current_cost: 3,
        sponsorable: false,
      },
    ])
    const wrapper = await render(IdeasSection, '/')
    const card = wrapper.get('.home-idea-preview')
    expect(card.attributes('href')).toBe('/ideas/7')
    expect(card.get('.home-idea-prompt').text()).toBe(
      Array.from(prompt).slice(0, 120).join('') + '…',
    )
    expect(card.text()).toContain('已实现')
    expect(card.text()).not.toContain('3 额度')
    const images = card.findAllComponents({ name: 'CreativeImage' })
    expect(images.map((image) => image.props('id'))).toEqual([1, 2, 3])
    expect(images.every((image) => image.props('ideaId') === 7)).toBe(true)
    expect(card.get('.home-idea-more').text()).toBe('+1')
  })
  it('uses the description for a text-only homepage proposal and hides the section when empty', async () => {
    api.ideas.mockResolvedValue([
      {
        idea: {
          id: 8,
          title: '文字创意',
          snapshot: 'null',
          description: '窗边的一束光',
          status: 'waiting',
        },
        author: { username: '作者' },
        references: [],
        results: [],
        current_cost: 2,
        sponsorable: true,
      },
    ])
    const wrapper = await render(IdeasSection, '/')
    expect(wrapper.get('.home-idea-prompt').text()).toBe('窗边的一束光')
    expect(wrapper.get('.home-idea-preview').text()).toContain('2 额度')
    expect(wrapper.find('.home-idea-references').exists()).toBe(false)
    api.ideas.mockResolvedValue([])
    const empty = await render(IdeasSection, '/')
    expect(empty.find('.home-ideas').exists()).toBe(false)
  })
  it('gives a text-only idea a visible preview while preserving its detail destination and cost', async () => {
    api.ideas.mockResolvedValue([
      {
        idea: {
          id: 7,
          title: '午后的小泉',
          description: '窗边的一束光',
          status: 'waiting',
          model: 'image-model',
        },
        author: { username: '创作者' },
        references: [],
        results: [],
        sponsorable: true,
        current_cost: 3,
      },
    ])
    const wrapper = await render(IdeasView, '/ideas')
    const card = wrapper.get('a.idea-card')
    expect(card.attributes('href')).toBe('/ideas/7')
    expect(card.get('.idea-card-prompt').text()).toBe('窗边的一束光')
    expect(card.text()).toContain('纯文字')
    expect(card.text()).toContain('3 额度')
    expect(card.text()).toContain('午后的小泉')
  })
  it('leads with a bounded snapshot prompt and previews original references instead of results', async () => {
    const prompt = '窗边的小泉，午后暖光。'.repeat(50)
    api.ideas.mockResolvedValue([
      {
        idea: {
          id: 8,
          title: '午后',
          description: '简短投稿说明',
          snapshot: JSON.stringify({ prompt }),
          status: 'realized',
          model: 'image-model',
        },
        author: { username: '创作者' },
        references: [1, 2, 3, 4].map((id) => ({ id })),
        results: [{ id: 99 }],
        sponsorable: false,
      },
    ])
    const wrapper = await render(IdeasView, '/ideas')
    const card = wrapper.get('.idea-card')
    expect(card.get('.idea-card-prompt').text()).toBe(
      Array.from(prompt).slice(0, 240).join('') + '…',
    )
    expect(
      card.findAllComponents({ name: 'CreativeImage' }).map((image) => image.props('id')),
    ).toEqual([1, 2, 3])
    expect(card.get('.idea-reference-more').text()).toBe('+1')
    expect(card.get('.idea-card-read').text()).toContain('完整提示词')
  })
  it.each(['broken json', 'null', '{"prompt":42}'])(
    'keeps older ideas readable for snapshot %s',
    async (snapshot) => {
      api.ideas.mockResolvedValue([
        {
          idea: {
            id: 9,
            title: '旧创意',
            description: '保留的说明',
            snapshot,
            status: 'waiting',
            model: 'model',
          },
          author: { username: '作者' },
          references: [],
          results: [],
          sponsorable: false,
        },
      ])
      const wrapper = await render(IdeasView, '/ideas')
      expect(wrapper.get('.idea-card-prompt').text()).toBe('保留的说明')
    },
  )
  it('makes the character card destination explicit and keeps search filtering working', async () => {
    api.resources.mockResolvedValue([
      { id: 2, slug: 'koizumi', title: '小泉', summary: '角色简介', payload: { asset_ids: [] } },
    ])
    const wrapper = await render(CreativeCatalogView, '/characters')
    expect(wrapper.get('a.character-card').attributes('href')).toBe('/characters/koizumi')
    expect(wrapper.get('a.character-card').text()).toContain('查看角色设定')
    await wrapper.get('input').setValue('不存在的角色')
    expect(wrapper.find('a.character-card').exists()).toBe(false)
    await wrapper.get('input').setValue('小泉')
    expect(wrapper.find('a.character-card').exists()).toBe(true)
  })
})

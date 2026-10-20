import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CreativeLibrary from '@/components/CreativeLibrary.vue'

const api = vi.hoisted(() => ({ resources: vi.fn(), expand: vi.fn() }))
vi.mock('@/api/creative', () => ({ creativeApi: api }))

type Variable = {
  name: string
  label: string
  type?: 'text' | 'select'
  required?: boolean
  help?: string
  options?: string[]
  default?: string
}
const row = (
  id: number,
  kind: 'character' | 'template',
  title: string,
  assets: number[] = [],
  variables: Variable[] = [],
) => ({
  id,
  kind,
  slug: `r-${id}`,
  title,
  summary: `${title}的简介`,
  status: 'published' as const,
  sort_order: 0,
  version: 1,
  payload: {
    description: '',
    appearance: '栗色长发',
    body: '正文 {名字}',
    asset_ids: assets,
    character_ids: [],
    tags: [],
    variables,
    parameters: {},
    preset_id: 0,
  },
})
const wrappers: ReturnType<typeof mount>[] = []
async function render(appliedCharacters: number[] = []) {
  const w = mount(CreativeLibrary, {
    attachTo: document.body,
    props: { appliedCharacters },
    global: {
      stubs: {
        CreativeImage: { props: ['id'], template: '<span class="test-image" :data-id="id" />' },
      },
    },
  })
  wrappers.push(w)
  await flushPromises()
  return w
}
beforeEach(() => {
  api.resources.mockResolvedValue([
    row(1, 'character', '小泉', [11]),
    row(2, 'character', '雨宫'),
    row(3, 'template', '电影海报'),
  ])
  api.expand.mockResolvedValue({ prompt: '展开后的提示词' })
})
afterEach(() => {
  wrappers.splice(0).forEach((w) => w.unmount())
  vi.clearAllMocks()
})

describe('creative library', () => {
  it('shows character cards and filters them by search and kind', async () => {
    const w = await render()
    expect(w.findAll('.lib-card')).toHaveLength(2)
    expect(w.get('.lib-card .test-image').attributes('data-id')).toBe('11')
    await w.get('input[type="search"]').setValue('雨宫')
    expect(w.findAll('.lib-card')).toHaveLength(1)
    await w.get('input[type="search"]').setValue('不存在')
    expect(w.text()).toContain('没有匹配的素材')
    await w.get('input[type="search"]').setValue('')
    await w.findAll('[role="radio"]')[1]!.trigger('click')
    expect(w.findAll('.lib-card')).toHaveLength(1)
    expect(w.text()).toContain('电影海报')
  })

  it('imports a character on the first click without opening a detail pane', async () => {
    const w = await render()
    await w.findAll('.lib-card')[0]!.trigger('click')
    const applied = w.emitted('apply')!
    expect(applied).toHaveLength(1)
    expect(applied[0]![0]).toMatchObject({
      action: 'append',
      assets: [11],
      characters: [1],
      label: '小泉',
    })
    expect(w.find('.lib-detail').exists()).toBe(false)
  })

  it('marks characters already in the draft as imported and locks them', async () => {
    const w = await render([1])
    const card = w.findAll('.lib-card')[0]!
    expect(card.attributes('disabled')).toBeDefined()
    expect(card.text()).toContain('已导入')
    await card.trigger('click')
    expect(w.emitted('apply')).toBeUndefined()
  })

  it('expands a template on click and imports its prompt', async () => {
    const w = await render()
    await w.findAll('[role="radio"]')[1]!.trigger('click')
    await w.get('.lib-card').trigger('click')
    await flushPromises()
    expect(api.expand).toHaveBeenCalledWith(3, {}, 1, expect.any(AbortSignal))
    expect(w.emitted('apply')![0]![0]).toMatchObject({
      text: '展开后的提示词',
      assets: [],
      characters: [],
      label: '电影海报',
    })
  })

  it('asks for missing required variables inline before importing a template', async () => {
    const variables = [{ name: '名字', label: '主角', required: true }]
    api.resources.mockResolvedValue([row(3, 'template', '电影海报', [], variables)])
    const w = await render()
    await w.findAll('[role="radio"]')[1]!.trigger('click')
    await w.get('.lib-card').trigger('click')
    await flushPromises()
    expect(w.find('.lib-fill').exists()).toBe(true)
    expect(w.get('.lib-fill-apply').attributes('disabled')).toBeDefined()
    await w.get('.lib-field input').setValue('小泉')
    await w.get('.lib-fill').trigger('submit')
    await flushPromises()
    expect(api.expand).toHaveBeenCalledWith(3, { 名字: '小泉' }, 1, expect.any(AbortSignal))
    expect(w.emitted('apply')).toHaveLength(1)
  })

  it('surfaces a load failure instead of showing an empty library', async () => {
    api.resources.mockRejectedValue(new Error('网络异常'))
    const w = await render()
    expect(w.get('.creative-error').text()).toContain('网络异常')
  })
})

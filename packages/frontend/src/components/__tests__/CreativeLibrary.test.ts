import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CreativeLibrary from '@/components/CreativeLibrary.vue'

const api = vi.hoisted(() => ({ resources: vi.fn(), expand: vi.fn() }))
vi.mock('@/api/creative', () => ({ creativeApi: api }))

const row = (id: number, kind: 'character' | 'template', title: string, assets: number[] = []) => ({
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
    variables: [],
    parameters: {},
    preset_id: 0,
  },
})
const wrappers: ReturnType<typeof mount>[] = []
async function render() {
  const w = mount(CreativeLibrary, {
    attachTo: document.body,
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
})
afterEach(() => {
  wrappers.splice(0).forEach((w) => w.unmount())
  vi.clearAllMocks()
})

describe('creative library', () => {
  it('lists characters as cards, filters by search and kind, and shows an empty hint', async () => {
    const w = await render()
    expect(w.findAll('.library-row')).toHaveLength(2)
    // 有参考图的资源用首张图作缩略图，没有的用占位图标。
    expect(w.get('.library-row .test-image').attributes('data-id')).toBe('11')
    await w.get('input[type="search"]').setValue('雨宫')
    expect(w.findAll('.library-row')).toHaveLength(1)
    await w.get('input[type="search"]').setValue('不存在')
    expect(w.text()).toContain('没有匹配的资源')
    await w.get('input[type="search"]').setValue('')
    await w.findAll('[role="radio"]')[1]!.trigger('click')
    expect(w.findAll('.library-row')).toHaveLength(1)
    expect(w.text()).toContain('电影海报')
  })

  it('opens a resource as a detail pane, applies it and goes back to the list', async () => {
    const w = await render()
    await w.findAll('.library-row')[0]!.trigger('click')
    await flushPromises()
    expect(w.find('.lib-list').exists()).toBe(false)
    expect(w.get('.lib-detail h3').text()).toBe('小泉')
    expect(w.get('.library-images .test-image').attributes('data-id')).toBe('11')
    await w.findAll('.lib-btn').find((b) => b.text() === '追加 / 导入')!.trigger('click')
    const applied = w.emitted('apply')!
    expect(applied).toHaveLength(1)
    expect(applied[0]![0]).toMatchObject({ action: 'append', assets: [11], characters: [1] })
    await w.get('.lib-back').trigger('click')
    await flushPromises()
    expect(w.find('.lib-detail').exists()).toBe(false)
    expect(w.findAll('.library-row')).toHaveLength(2)
  })

  it('surfaces a load failure instead of showing an empty library', async () => {
    api.resources.mockRejectedValue(new Error('网络异常'))
    const w = await render()
    expect(w.get('.creative-error').text()).toContain('网络异常')
  })
})

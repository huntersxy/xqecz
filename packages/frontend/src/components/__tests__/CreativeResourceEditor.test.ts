import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CreativeResourceEditor from '../admin/CreativeResourceEditor.vue'
import { CreativeResourceSchema } from '@/types/schemas'

const save = vi.hoisted(() => vi.fn().mockResolvedValue(null))
vi.mock('@/api/creative', () => ({ creativeApi: { save }, aiApi: {} }))

describe('character reference management', () => {
  it('preserves existing references without a public-image picker and can remove an association', async () => {
    const resource = CreativeResourceSchema.parse({
      id: 1,
      kind: 'character',
      slug: 'role',
      title: '角色',
      summary: '',
      status: 'draft',
      sort_order: 0,
      version: 1,
      payload: { asset_ids: [3, 4] },
      created_at: '2026-10-15T00:00:00.000Z',
      updated_at: '2026-10-15T00:00:00.000Z',
    })
    const wrapper = mount(CreativeResourceEditor, {
      props: { resource, kind: 'character', assets: [], resources: [] },
      global: {
        stubs: {
          CreativeImage: true,
          Form: { template: '<div><slot /></div>' },
          FormItem: { template: '<div><slot /></div>' },
        },
      },
    })
    try {
      expect(wrapper.text()).not.toContain('可选择已有公开图片')
      expect(wrapper.find('[aria-label="关联设定图"]').exists()).toBe(false)
      expect(
        wrapper.findAllComponents({ name: 'CreativeImage' }).map((image) => image.props('id')),
      ).toEqual([3, 4])
      await wrapper.get('button[aria-label="移除参考图 1"]').trigger('click')
      await wrapper
        .findAll('button')
        .find((button) => button.text() === '保存资源')!
        .trigger('click')
      await flushPromises()
      expect(save).toHaveBeenCalledWith(
        expect.objectContaining({ payload: expect.objectContaining({ asset_ids: [4] }) }),
      )
      expect(resource.payload.asset_ids).toEqual([3, 4])
    } finally {
      wrapper.unmount()
    }
  })
})

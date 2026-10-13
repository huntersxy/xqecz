import { describe, expect, it } from 'vitest'
import type { CreativeResource } from '@/types/schemas'
import { resolveCreativeResource } from '../creativeResources'
import { referenceMentionAt, remapReferenceMentions } from '../creativeMentions'

function row(
  id: number,
  kind: CreativeResource['kind'],
  asset_ids: number[],
  character_ids: number[] = [],
  preset_id = 0,
): CreativeResource {
  return {
    id,
    kind,
    title: `角色${id}`,
    summary: '',
    slug: `role-${id}`,
    status: 'published',
    sort_order: 0,
    version: 1,
    created_at: '',
    updated_at: '',
    payload: {
      appearance: kind === 'character' ? '粉色头发' : '',
      description: '',
      body: '',
      asset_ids,
      character_ids,
      preset_id,
      variables: [],
      parameters: {},
      tags: [],
    },
  }
}
describe('creative relationships and image mentions', () => {
  it('imports a template’s roles, their images and legacy preset images without duplication or cycles', () => {
    const role = row(1, 'character', [10]),
      preset = row(2, 'preset', [11], [1]),
      template = row(3, 'template', [10], [1], 2)
    preset.payload.character_ids.push(2)
    expect(resolveCreativeResource(template, [role, preset, template])).toEqual({
      assets: [10, 11],
      characters: [1],
      text: '角色1：粉色头发',
    })
  })
  it('opens mention completion only at an unfinished image reference', () => {
    expect(referenceMentionAt('画出 @', 4)).toEqual({ start: 3, end: 4, query: '' })
    expect(referenceMentionAt('画出 @图片2', 7)).toEqual({ start: 3, end: 7, query: '2' })
    expect(referenceMentionAt('普通描述', 4)).toBe(null)
  })
  it('preserves image targets when reordered and removes deleted references', () => {
    expect(remapReferenceMentions('@图片1穿着@图片2的衣服', [10, 20], [20, 10])).toBe(
      '@图片2穿着@图片1的衣服',
    )
    expect(remapReferenceMentions('参考@图片2、@图片1', [10, 20], [20])).toBe('参考@图片1、')
  })
})

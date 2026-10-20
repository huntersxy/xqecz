import { describe, expect, it } from 'vitest'
import { MetadataOptionsSchema, VisionSettingsSchema, VisionJobSchema, VisionReferenceSchema } from '@/types/schemas'
import { ContentSchema } from '@/types'

const base = process.env.XQECZ_LIVE_API || ''
const session = process.env.XQECZ_LIVE_SESSION || ''
const live = base ? describe : describe.skip
async function api(path: string, admin = false) {
  const res = await fetch(`${base}${path}`, { headers: admin && session ? { Cookie: `session_id=${session}` } : {} })
  expect(res.status).toBe(200)
  const json = await res.json()
  expect(json.code).toBe(200)
  return json.data
}
live('视觉生成响应契约', () => {
  it('公开能力及占位值符合上传契约', async () => { MetadataOptionsSchema.parse(await api('/content/metadata-options')) })
  it('内容列表与详情仍符合已有契约', async () => {
    const list = await api('/content/list?page_size=10')
    for (const item of list.list) {
      const parsed = ContentSchema.parse(item)
      ContentSchema.parse(await api(`/content/${parsed.id}?silent=1`))
    }
  })
  it.skipIf(!session)('管理员配置、任务与角色库可解析且没有秘密字段', async () => {
    const config = await api('/admin/vision/settings', true)
    VisionSettingsSchema.parse(config)
    expect(config).not.toHaveProperty('api_key')
    expect(config).not.toHaveProperty('key')
    for (const job of await api('/admin/vision/jobs', true)) VisionJobSchema.parse(job)
    for (const ref of await api('/admin/vision/references', true)) {
      VisionReferenceSchema.parse(ref)
      expect(ref).not.toHaveProperty('data_url')
    }
  })
})

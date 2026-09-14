import { describe, expect, it } from 'vitest'
import { ContentSchema, RecommendContentSchema } from '@/types'
import { formatTime } from '@/utils'

const ISO = '2026-09-13T11:28:04.865Z'
const baseContent = {
  id: 1,
  title: 't',
  user: { id: 2, username: 'u' },
  tags: [],
  created_at: ISO,
}

describe('时间字段解析', () => {
  it('ISO 字符串解析为 Unix 秒而不是 0', () => {
    const parsed = ContentSchema.parse(baseContent)
    expect(parsed.created_at).toBe(Math.floor(Date.parse(ISO) / 1000))
    expect(parsed.created_at).toBeGreaterThan(0)
  })

  it('解析结果可直接交给 formatTime 渲染', () => {
    const parsed = ContentSchema.parse(baseContent)
    expect(formatTime(parsed.created_at)).toMatch(/^\d{4}\/\d{2}\/\d{2} /)
  })

  it('数字与缺失值保持原有兜底语义', () => {
    expect(ContentSchema.parse({ ...baseContent, created_at: 1700000000 }).created_at).toBe(1700000000)
    expect(ContentSchema.parse({ ...baseContent, created_at: null }).created_at).toBe(0)
    expect(ContentSchema.parse({ ...baseContent, created_at: 0 }).created_at).toBe(0)
  })

  it('推荐位 schema 同样保留时间', () => {
    const rec = RecommendContentSchema.parse({
      id: 3,
      title: 'r',
      user: { id: 1, username: 'u' },
      tags: [],
      created_at: ISO,
    })
    expect(rec.created_at).toBe(Math.floor(Date.parse(ISO) / 1000))
  })
})

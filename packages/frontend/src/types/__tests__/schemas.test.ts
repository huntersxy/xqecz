import { describe, expect, it } from 'vitest'
import { ContentSchema, RecommendContentSchema } from '@/types'
import { CommentSchema } from '@/types/schemas'
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


describe('评论作者头像', () => {
  const comment = { id: 1, content_id: 9, user_id: 2, text: '评论', parent_id: null, is_banned: false, created_at: ISO }
  it('保留顶层、回复和父评论引用中的公开头像地址', () => {
    const user = { id: 2, username: '作者', avatar_url: 'https://example.com/avatar.png' }
    const parsed = CommentSchema.parse({ ...comment, user, replies: [{ ...comment, id: 3, parent_id: 1, user, parent: { id: 1, user_id: 2, text: '评论', user } }] })
    expect(parsed.user?.avatar_url).toBe(user.avatar_url)
    expect(parsed.replies?.[0].user?.avatar_url).toBe(user.avatar_url)
    expect(parsed.replies?.[0].parent?.user?.avatar_url).toBe(user.avatar_url)
  })
  it('兼容缺少头像字段的历史响应，并不保留公开作者中的邮箱', () => {
    const parsed = CommentSchema.parse({ ...comment, user: { id: 2, username: '作者', email: 'test@example.com' } })
    expect(parsed.user?.avatar_url).toBeUndefined()
    expect(parsed.user).not.toHaveProperty('email')
  })
})

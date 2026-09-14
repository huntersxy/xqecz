// 契约冒烟：把「运行中的后端」的真实响应交给前端自己的 zod schema 校验。
//
// 只在设置了 XQECZ_LIVE_API 时执行（例如 XQECZ_LIVE_API=http://localhost:5173/api），
// 因此 CI（无后端）会自动跳过，不会拖慢或污染流水线。
import { describe, expect, it } from 'vitest'
import { ContentSchema, RecommendContentSchema } from '@/types'

const BASE = process.env.XQECZ_LIVE_API || ''
const live = BASE ? describe : describe.skip

interface Envelope<T> {
  code: number
  message: string
  data: T
}

async function api<T>(path: string): Promise<Envelope<T>> {
  const res = await fetch(`${BASE}${path}`)
  expect(res.status, `${path} 状态码`).toBe(200)
  return (await res.json()) as Envelope<T>
}

interface PageData {
  list: unknown[]
  total: number
  page: number
  page_size: number
  total_page: number
}

live('运行中的后端与前端契约一致', () => {
  it('内容列表全部通过 ContentSchema 校验', async () => {
    const res = await api<PageData>('/content/list?page=1&page_size=20')
    expect(res.code).toBe(200)
    expect(res.data.list.length).toBeGreaterThan(0)
    expect(res.data.total).toBeGreaterThan(0)
    for (const item of res.data.list) {
      const parsed = ContentSchema.parse(item)
      expect(parsed.id).toBeGreaterThan(0)
      expect(typeof parsed.title).toBe('string')
      expect(Array.isArray(parsed.tags)).toBe(true)
    }
  })

  it('标签筛选结果同样通过 schema 校验', async () => {
    const tags = await api<string[]>('/content/tags')
    expect(Array.isArray(tags.data)).toBe(true)
    expect(tags.data.length).toBeGreaterThan(0)
    const tag = encodeURIComponent(tags.data[0])
    const res = await api<PageData>(`/content/list?page=1&page_size=10&tag=${tag}`)
    for (const item of res.data.list) {
      const parsed = ContentSchema.parse(item)
      expect(parsed.tags).toContain(tags.data[0])
    }
  })

  it('搜索与推荐响应可被前端解析', async () => {
    const search = await api<PageData>('/content/search?keyword=a&page_size=5')
    for (const item of search.data.list) ContentSchema.parse(item)

    const rec = await api<{ list: unknown[]; count: number }>('/content/recommend?count=10')
    expect(rec.data.count).toBeGreaterThan(0)
    expect(rec.data.list.length).toBeGreaterThan(0)
    for (const item of rec.data.list) {
      const parsed = RecommendContentSchema.parse(item)
      expect(parsed.id).toBeGreaterThan(0)
    }
  })

  it('详情响应可被前端解析且含展示所需字段', async () => {
    const list = await api<PageData>('/content/list?page_size=1')
    const first = ContentSchema.parse(list.data.list[0])
    const detail = await api<unknown>(`/content/${first.id}?silent=1`)
    const parsed = ContentSchema.parse(detail.data)
    expect(parsed.id).toBe(first.id)
    expect(typeof parsed.thumb).toBe('string')
    expect(parsed.user.id).toBeGreaterThanOrEqual(0)
  })

  it('评论与投票响应结构符合前端预期', async () => {
    const list = await api<PageData>('/content/list?page_size=1')
    const contentId = ContentSchema.parse(list.data.list[0]).id

    const comments = await api<PageData>(`/comment/list/${contentId}?page=1&page_size=20`)
    for (const c of comments.data.list as Record<string, unknown>[]) {
      expect(typeof c['id']).toBe('number')
      expect(typeof c['text']).toBe('string')
      expect(c['parent_id']).toBeNull()
      expect(Array.isArray(c['replies'])).toBe(true)
      expect(typeof (c['user'] as Record<string, unknown>)['username']).toBe('string')
    }

    const count = await api<{ content_id: number; count: number }>(`/comment/count/${contentId}`)
    expect(count.data.content_id).toBe(contentId)
    expect(count.data.count).toBeGreaterThanOrEqual(0)

    const polls = await api<PageData>('/poll/list')
    for (const p of polls.data.list as Record<string, unknown>[]) {
      expect(typeof p['id']).toBe('number')
      expect(typeof p['title']).toBe('string')
      expect(Array.isArray(p['options'])).toBe(true)
    }
  })
})

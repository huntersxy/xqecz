import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useRecommendLoader } from '@/composables/useRecommendLoader'

const mocks = vi.hoisted(() => ({ recommend: vi.fn() }))
vi.mock('@/api', () => ({ contentApi: { recommend: mocks.recommend } }))

function items(start: number, count: number) {
  return Array.from({ length: count }, (_, index) => ({
    id: start + index,
    title: '作品 ' + (start + index),
    thumb: '/thumbs/' + (start + index) + '.webp',
    tags: [],
    view_count: 3,
    like_count: 1,
    user: { id: 1, username: 'creator' },
    created_at: '2026-10-18T00:00:00.000Z',
  }))
}

describe('recommend strip paging', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the next server page before advancing past the pool', async () => {
    mocks.recommend
      .mockResolvedValueOnce({ code: 200, data: { list: items(0, 16), count: 32 } })
      .mockResolvedValueOnce({ code: 200, data: { list: items(16, 16), count: 32 } })
      .mockResolvedValueOnce({ code: 200, data: { list: [], count: 32 } })

    const loader = useRecommendLoader()
    await loader.loadRecommendContents(1)
    expect(loader.recommendContents.value).toHaveLength(16)
    expect(loader.displayStart.value).toBe(0)

    await loader.refreshRecommend(8)
    expect(loader.displayStart.value).toBe(8)
    expect(mocks.recommend).toHaveBeenCalledTimes(1)

    // 窗口走到 16/16：必须先拉第 2 页，不能把窗口推过池尾显示空条。
    await loader.refreshRecommend(8)
    expect(mocks.recommend).toHaveBeenLastCalledWith(16, 2)
    expect(loader.recommendContents.value).toHaveLength(32)
    expect(loader.displayStart.value).toBe(16)

    await loader.refreshRecommend(8)
    expect(loader.displayStart.value).toBe(24)

    // 池已耗尽：回卷到 0，仍保留现有内容。
    await loader.refreshRecommend(8)
    expect(loader.displayStart.value).toBe(0)
    expect(loader.recommendContents.value).toHaveLength(32)
  })
})

import { describe, it, expect } from 'vitest'
import { diffLists } from '../useListCache'
import type { Content } from '@/types'

function makeContent(id: number, likeCount = 0): Content {
  return {
    id,
    title: `内容${id}`,
    text: '',
    thumb: '',
    video: '',
    img: '',
    origin: '',
    file_size: 0,
    user: { id: 1, username: 'u1' },
    avatar_url: '',
    tags: [],
    view_count: 0,
    like_count: likeCount,
    created_at: id,
  }
}

describe('diffLists', () => {
  it('全量快照：正常判定新增/删除/更新', () => {
    const cached = [makeContent(1), makeContent(2), makeContent(3)]
    const fresh = [makeContent(4), makeContent(1, 5), makeContent(2)]

    const r = diffLists(cached, fresh)

    expect([...r.added]).toEqual([4])
    expect([...r.removed]).toEqual([3])
    expect([...r.updated].sort()).toEqual([1, 2])
    // merged 以 fresh 顺序为准，字段取 fresh 值
    expect(r.merged.map((i) => i.id)).toEqual([4, 1, 2])
    expect(r.merged.find((i) => i.id === 1)?.like_count).toBe(5)
  })

  it('部分拉取（fresh 数量不足）：removed 恒空，不误删未覆盖的旧卡片', () => {
    const cached = [makeContent(1), makeContent(2), makeContent(3), makeContent(4), makeContent(5)]
    // 只拉最新 2 条（如增量探测/部分分页），2 < 5 → 非全量快照
    const fresh = [makeContent(6), makeContent(5)]

    const r = diffLists(cached, fresh)

    // 新增照常
    expect([...r.added]).toEqual([6])
    // 未被 fresh 覆盖的 1/2/3/4 不能判为已删除（可能只是没拉到）
    expect(r.removed.size).toBe(0)
  })

  it('fresh 数量足够但内容不同：按全量快照处理（removed 生效）', () => {
    const cached = [makeContent(1), makeContent(2), makeContent(3)]
    // 数量相等但 id 集合不同 → 视为全量替换快照，删除判定有效
    const fresh = [makeContent(3), makeContent(4), makeContent(5)]

    const r = diffLists(cached, fresh)

    expect([...r.added].sort()).toEqual([4, 5])
    expect([...r.removed].sort()).toEqual([1, 2])
    expect(r.merged.map((i) => i.id)).toEqual([3, 4, 5])
  })
})

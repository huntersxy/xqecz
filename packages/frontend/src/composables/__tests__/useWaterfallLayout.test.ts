import { describe, it, expect } from 'vitest'
import { computeLayout, type Position } from '../useWaterfallLayout'

const GAP = 10

function posMap(positions: Map<string | number, Position>) {
  return Object.fromEntries(
    [...positions.entries()].map(([id, p]) => [String(id), { x: p.x, y: p.y, w: p.w, h: p.h, col: p.col }]),
  )
}

describe('computeLayout 最短列分配', () => {
  it('两列时按最短列依次落位，列高推进含 gap', () => {
    const items = [
      { id: 1, height: 100 },
      { id: 2, height: 120 },
      { id: 3, height: 90 },
      { id: 4, height: 110 },
    ]
    const result = computeLayout(items, 2, 200, GAP)
    // 1→col0 y0，2→col1 y0，3→col0 y110（col0 更矮），4→col1 y130
    expect(posMap(result.positions)).toEqual({
      '1': { x: 0, y: 0, w: 200, h: 100, col: 0 },
      '2': { x: 210, y: 0, w: 200, h: 120, col: 1 },
      '3': { x: 0, y: 110, w: 200, h: 90, col: 0 },
      '4': { x: 210, y: 130, w: 200, h: 110, col: 1 },
    })
    expect(result.height).toBe(250) // max(110+90+10, 130+110+10)
  })

  it('preserveColumns=false 时忽略旧列，按当前高度重新分配且互不重叠', () => {
    const previous = new Map<string | number, Position>([
      [1, { x: 0, y: 0, w: 200, h: 100, col: 0 }],
      [2, { x: 210, y: 0, w: 200, h: 120, col: 1 }],
      [3, { x: 0, y: 110, w: 200, h: 90, col: 0 }],
      [4, { x: 210, y: 130, w: 200, h: 110, col: 1 }],
    ])
    const items = [
      { id: 1, height: 100 },
      { id: 2, height: 120 },
      { id: 3, height: 90 },
      { id: 4, height: 110 },
    ]
    // 3 列：全部重新分配，忽略旧列（1→col0，2→col1，3→col2，4→col2）
    const result = computeLayout(items, 3, 150, GAP, previous, false)
    expect(result.positions.get(1)?.col).toBe(0)
    expect(result.positions.get(2)?.col).toBe(1)
    expect(result.positions.get(3)?.col).toBe(2)
    expect(result.positions.get(4)?.col).toBe(2) // col2 最矮（100 < 110/130）
    // 校验不重叠：同一列 y 严格递增
    const colBottom = new Map<number, number>()
    for (const p of result.positions.values()) {
      const bottom = colBottom.get(p.col) ?? 0
      expect(p.y).toBeGreaterThanOrEqual(bottom)
      colBottom.set(p.col, p.y + p.h + GAP)
    }
  })
})

describe('computeLayout 稳定列（增量重排）', () => {
  const previous = new Map<string | number, Position>([
    [1, { x: 0, y: 0, w: 200, h: 100, col: 0 }],
    [2, { x: 210, y: 0, w: 200, h: 120, col: 1 }],
    [3, { x: 0, y: 110, w: 200, h: 90, col: 0 }],
    [4, { x: 210, y: 130, w: 200, h: 110, col: 1 }],
  ])

  it('某卡高度变化时只顺移本列下方卡片，其他列不动', () => {
    // 卡片 1（col0）高度 100 → 200：col0 下方卡片 3 从 y110 → y210
    const items = [
      { id: 1, height: 200 },
      { id: 2, height: 120 },
      { id: 3, height: 90 },
      { id: 4, height: 110 },
    ]
    const result = computeLayout(items, 2, 200, GAP, previous, true)
    expect(result.positions.get(1)).toMatchObject({ col: 0, y: 0, h: 200 })
    expect(result.positions.get(3)).toMatchObject({ col: 0, y: 210, h: 90 })
    // col1 完全不变
    expect(result.positions.get(2)).toMatchObject({ col: 1, y: 0 })
    expect(result.positions.get(4)).toMatchObject({ col: 1, y: 130 })
    expect(result.height).toBe(310) // max(210+90+10, 130+110+10)
  })

  it('新增卡片落到最短列，已有卡片列不变', () => {
    const items = [
      { id: 1, height: 100 },
      { id: 2, height: 120 },
      { id: 3, height: 90 },
      { id: 4, height: 110 },
      { id: 5, height: 80 },
    ]
    const result = computeLayout(items, 2, 200, GAP, previous, true)
    expect(result.positions.get(1)?.col).toBe(0)
    expect(result.positions.get(2)?.col).toBe(1)
    expect(result.positions.get(3)?.col).toBe(0)
    expect(result.positions.get(4)?.col).toBe(1)
    // 新增卡片 5 去最短列：col0 已到 210，col1 已到 250 → col0 更矮，y=210
    expect(result.positions.get(5)).toMatchObject({ col: 0, y: 210, h: 80 })
  })

  it('卡片高度为 0（尚未量到 DOM）时回退到上次已知高度', () => {
    const items = [
      { id: 1, height: 0 },
      { id: 2, height: 120 },
      { id: 3, height: 90 },
      { id: 4, height: 110 },
    ]
    const result = computeLayout(items, 2, 200, GAP, previous, true)
    // 卡片 1 高度用上次的 100，不塌成 0
    expect(result.positions.get(1)?.h).toBe(100)
    expect(result.positions.get(3)?.y).toBe(110)
  })
})

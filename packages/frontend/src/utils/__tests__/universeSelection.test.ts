import { describe, expect, it } from 'vitest'
import { universeSelectionPosition } from '../universeSelection'

describe('planet anchored selection', () => {
  it('stays beside the clicked planet and moves when the projected planet moves', () => {
    const first=universeSelectionPosition({x:500,y:500,diameter:120},1920,1080,340,300)
    const moved=universeSelectionPosition({x:650,y:580,diameter:120},1920,1080,340,300)
    expect(first).toEqual({x:578,y:350,visible:true})
    expect(moved.x-first.x).toBe(150); expect(moved.y-first.y).toBe(80)
  })
  it('flips beside the right edge and stays inside narrow screens', () => {
    const edge=universeSelectionPosition({x:1800,y:500,diameter:100},1920,1080,340,300)
    expect(edge.x).toBe(1392)
    const mobile=universeSelectionPosition({x:350,y:800,diameter:90},390,844,240,300)
    expect(mobile.x).toBeGreaterThanOrEqual(16); expect(mobile.x+240).toBeLessThanOrEqual(374)
    expect(mobile.y+300).toBeLessThanOrEqual(828)
  })
  it('hides the card when its planet is completely outside the viewport', () => {
    expect(universeSelectionPosition({x:-300,y:500,diameter:100},1920,1080,340,300).visible).toBe(false)
  })
})

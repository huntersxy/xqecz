import { describe, expect, it } from 'vitest'
import { hasAIGCTag, mergeAIGCTag } from '../contentTags'

describe('AIGC declaration merging', () => {
  it('merges without mutating manual tags and unchecking restores the draft', () => {
    const draft = Object.freeze(['校园', '小泉'])
    expect(mergeAIGCTag(draft, true)).toEqual(['校园', '小泉', 'AIGC'])
    expect(mergeAIGCTag(draft, false)).toEqual(['校园', '小泉'])
    expect(draft).toEqual(['校园', '小泉'])
  })
  it('keeps a manual marker independently of the checkbox and canonicalizes duplicates', () => {
    expect(mergeAIGCTag(['校园', 'aigc', '#AIGC', 'ＡＩＧＣ'], false)).toEqual(['校园', 'AIGC'])
    expect(mergeAIGCTag(['AIGC', '校园'], true)).toEqual(['AIGC', '校园'])
    expect(hasAIGCTag(['ＡＩＧＣ'])).toBe(true)
    expect(hasAIGCTag(['AIGC风格'])).toBe(false)
  })
})

export const AIGC_TAG = 'AIGC'

function isAIGCTag(tag: string): boolean {
  return tag.normalize('NFKC').trim().replace(/^#+/, '').trim().toUpperCase() === AIGC_TAG
}

export function hasAIGCTag(tags: readonly string[]): boolean {
  return tags.some(isAIGCTag)
}

// Merge at submission/display time without mutating the manual draft; unchecking only removes the checkbox-owned addition.
export function mergeAIGCTag(tags: readonly string[], declared: boolean): string[] {
  const result: string[] = []
  let seen = false
  for (const tag of tags) {
    if (isAIGCTag(tag)) {
      if (!seen) result.push(AIGC_TAG)
      seen = true
    } else result.push(tag)
  }
  if (declared && !seen) result.push(AIGC_TAG)
  return result
}

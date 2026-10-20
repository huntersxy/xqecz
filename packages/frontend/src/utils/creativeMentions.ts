export function referenceMentionAt(text: string, cursor: number) {
  const match = /@(?:图片)?(\d*)$/.exec(text.slice(0, cursor))
  return match ? { start: cursor - match[0].length, end: cursor, query: match[1] ?? '' } : null
}

export function remapReferenceMentions(text: string, before: number[], after: number[]) {
  return text.replace(/@图片([1-9]\d*)(?!\d)/g, (token, index: string) => {
    const id = before[Number(index) - 1]
    if (id === undefined) return token
    const next = after.indexOf(id)
    return next < 0 ? '' : `@图片${next + 1}`
  })
}

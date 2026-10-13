import type { CreativeResource } from '@/types/schemas'

// All consumers use the same relationship traversal. Legacy presets remain
// readable, while new role resources keep their own images directly together.
export function resolveCreativeResource(row: CreativeResource, resources: CreativeResource[]) {
  const assets = new Set<number>(),
    characters = new Set<number>(),
    text = new Set<string>(),
    seen = new Set<number>()
  const visit = (resource: CreativeResource) => {
    if (seen.has(resource.id)) return
    seen.add(resource.id)
    resource.payload.asset_ids.forEach((id) => assets.add(id))
    if (resource.kind === 'character') {
      characters.add(resource.id)
      const description =
        resource.payload.appearance || resource.payload.description || resource.summary
      if (description) text.add(`${resource.title}：${description}`)
    }
    for (const id of [...resource.payload.character_ids, resource.payload.preset_id]) {
      const related = resources.find((candidate) => candidate.id === id)
      if (related) visit(related)
    }
  }
  visit(row)
  return { assets: [...assets], characters: [...characters], text: [...text].join('\n') }
}

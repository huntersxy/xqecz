import type { CreativeResource } from '@/types/schemas'

export function resolveCreativeResource(row: CreativeResource, resources: CreativeResource[]) {
  const assets = new Set<number>(),
    characters = new Set<number>(),
    seen = new Set<number>()
  const visit = (resource: CreativeResource) => {
    if (seen.has(resource.id)) return
    seen.add(resource.id)
    resource.payload.asset_ids.forEach((id) => assets.add(id))
    if (resource.kind === 'character') characters.add(resource.id)
    for (const id of [...resource.payload.character_ids, resource.payload.preset_id]) {
      const related = resources.find((candidate) => candidate.id === id)
      if (related) visit(related)
    }
  }
  visit(row)
  return { assets: [...assets], characters: [...characters] }
}

import type { AITaskView } from '@/types/schemas'

export function defaultPublishDescription(
  tasks: AITaskView[],
  assetId: number | undefined,
  fallback: string,
): string {
  if (assetId === undefined) return fallback
  const view = tasks.find((row) => row.assets.some((asset) => asset.id === assetId))
  const base = view?.base_prompt?.trim()
  if (base) return base
  return view?.task.prompt || fallback
}

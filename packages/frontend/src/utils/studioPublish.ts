import type { AITaskView } from '@/types/schemas'

/**
 * 发布对话框的默认作品描述：优先用生成该图片的任务所保存的原始提示词
 * （BasePrompt，不含站点提示词增强后缀）；历史任务缺失时回退当前输入区。
 * 纯函数，便于单测覆盖「选中资源不在当前任务列表」等分支。
 */
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

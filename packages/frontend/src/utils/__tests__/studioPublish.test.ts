import { describe, expect, it } from 'vitest'
import { defaultPublishDescription } from '../studioPublish'
import type { AITaskView } from '@/types/schemas'

const time = '2026-10-14T00:00:00.000Z'
function view(id: number, prompt: string, base?: string, assetIds = [id * 2]): AITaskView {
  return {
    task: { id, prompt } as AITaskView['task'],
    assets: assetIds.map((assetId) => ({ id: assetId }) as AITaskView['assets'][number]),
    base_prompt: base,
  } as AITaskView
}

describe('defaultPublishDescription', () => {
  it('prefers the task base prompt (without the enhancement suffix)', () => {
    const tasks = [view(1, '原始提示词\n\n站点增强后缀', '原始提示词', [4])]
    expect(defaultPublishDescription(tasks, 4, '输入区')).toBe('原始提示词')
  })

  it('falls back to the stored task prompt when a historical task has no base prompt', () => {
    const tasks = [view(1, '旧任务提示词', undefined, [4])]
    expect(defaultPublishDescription(tasks, 4, '输入区')).toBe('旧任务提示词')
  })

  it('falls back to the composer when the asset is not in the loaded tasks', () => {
    expect(defaultPublishDescription([view(1, '任务', '任务', [4])], 9, '输入区')).toBe('输入区')
    expect(defaultPublishDescription([], undefined, '输入区')).toBe('输入区')
  })
})

import type { AIIdeaView } from '@/types/schemas'

// Discovery surfaces read the same sanitized proposal snapshot. Older or invalid
// snapshots fall back to the public description; result images are not the plan.
export function ideaPromptPreview(idea: AIIdeaView['idea'], limit = 240) {
  let prompt = ''
  try {
    const snapshot: unknown = JSON.parse(idea.snapshot)
    if (
      snapshot &&
      typeof snapshot === 'object' &&
      'prompt' in snapshot &&
      typeof snapshot.prompt === 'string'
    )
      prompt = snapshot.prompt.trim()
  } catch {
    // Historical snapshots can be absent or malformed.
  }
  const text = Array.from(prompt || idea.description || '投稿者暂未提供提示词说明')
  return text.length > limit ? `${text.slice(0, limit).join('')}…` : text.join('')
}

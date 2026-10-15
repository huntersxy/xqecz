import { describe, expect, it } from 'vitest'
import { z } from 'zod'
import {
  AIConfigSchema,
  AIAccountSchema,
  AISessionSchema,
  AIAssetSchema,
  AITaskViewSchema,
  AIProviderSchema,
  AILedgerSchema,
  AIIdeaViewSchema,
  CreativeResourceSchema,
} from '@/types/schemas'
const base = process.env.XQECZ_AI_FIXTURE_API
const isolated = base && ['127.0.0.1', 'localhost'].includes(new URL(base).hostname)
describe.skipIf(!isolated)('isolated AI API contract', () => {
  it('parses real public, private and admin handler responses', async () => {
    const endpoints: [string, z.ZodType][] = [
      ['/ai/config', AIConfigSchema],
      ['/ai/account', AIAccountSchema],
      ['/ai/sessions', z.array(AISessionSchema)],
      ['/ai/assets', z.array(AIAssetSchema)],
      ['/ai/tasks', z.array(AITaskViewSchema)],
      ['/admin/ai/tasks', z.array(AITaskViewSchema)],
      ['/admin/ai/providers', z.array(AIProviderSchema)],
      ['/admin/ai/ledger', z.array(AILedgerSchema)],
      ['/ideas', z.array(AIIdeaViewSchema)],
      ['/creative/resources', z.array(CreativeResourceSchema)],
    ]
    for (const [path, schema] of endpoints) {
      const res = await fetch(base + path, { headers: { Cookie: 'session_id=fixture-admin' } })
      const wire = await res.json()
      expect(wire.code, path).toBe(200)
      expect(schema.safeParse(wire.data).success, path).toBe(true)
      expect(JSON.stringify(wire)).not.toContain('fixture-only-secret')
      expect(JSON.stringify(wire)).not.toContain('cipher_key')
    }
  })
})

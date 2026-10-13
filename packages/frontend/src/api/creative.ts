import { z } from 'zod'
import { request, fetchApiBlob } from './index'
import {
  AIConfigSchema,
  AIAccountSchema,
  AISessionSchema,
  AITaskSchema,
  AITaskViewSchema,
  AIAssetSchema,
  AIProviderSchema,
  AILedgerSchema,
  AIIdeaSchema,
  AIIdeaViewSchema,
  CreativeResourceSchema,
} from '@/types/schemas'
import type { AIProvider, CreativeResource } from '@/types/schemas'

export interface GenerationInput {
  session_id: number
  idempotency_key: string
  prompt: string
  provider_id: number
  mode: 'site' | 'byok'
  size: string
  quality: string
  count: number
  references: { asset_id: number; note: string }[]
  parent_asset_id: number
  expected_cost: number
  provider_revision: number
  character_ids: number[]
}
type Options = Parameters<typeof request>[1]
// Mutations never retry automatically. Their stable client key is retained by
// the workbench until the durable task has been acknowledged.
async function checked<T>(url: string, schema: z.ZodType<T>, options: Options = {}): Promise<T> {
  const result = await request<unknown>(url, { ...options, retry: 0 })
  if (result.code !== 200) {
    const error = new Error(result.message)
    error.name = 'BusinessError'
    throw error
  }
  return schema.parse(result.data)
}
const nil = z.unknown()
export const aiApi = {
  config: (signal?: AbortSignal) => checked('/ai/config', AIConfigSchema, { signal }),
  account: (signal?: AbortSignal) => checked('/ai/account', AIAccountSchema, { signal }),
  ledger: () => checked('/ai/ledger', z.array(AILedgerSchema)),
  sessions: (signal?: AbortSignal, beforeId?: number) =>
    checked('/ai/sessions', z.array(AISessionSchema), { signal, query: { before_id: beforeId } }),
  session: (id: number, signal?: AbortSignal) =>
    checked(`/ai/sessions/${id}`, AISessionSchema, { signal }),
  createSession: (name: string) =>
    checked('/ai/sessions', AISessionSchema, { method: 'POST', body: { name } }),
  saveSession: (id: number, name: string, draft: unknown, version: number, signal?: AbortSignal) =>
    checked(`/ai/sessions/${id}`, AISessionSchema, {
      method: 'PATCH',
      body: { name, draft, version },
      signal,
    }),
  tasks: (query: Record<string, unknown> = {}, signal?: AbortSignal) =>
    checked('/ai/tasks', z.array(AITaskViewSchema), { query, signal }),
  submit: (input: GenerationInput, signal?: AbortSignal) =>
    checked('/ai/tasks', AITaskSchema, { method: 'POST', body: input, signal }),
  quote: (input: GenerationInput) =>
    checked('/ai/quote', z.object({ cost: z.number() }), { method: 'POST', body: input }),
  cancel: (id: number) => checked(`/ai/tasks/${id}/cancel`, nil, { method: 'POST' }),
  assets: (signal?: AbortSignal, beforeId?: number) =>
    checked('/ai/assets', z.array(AIAssetSchema), { signal, query: { before_id: beforeId } }),
  publicAssets: (signal?: AbortSignal, ids?: number[]) =>
    checked('/creative/assets', z.array(AIAssetSchema), { signal, query: { ids: ids?.join(',') } }),
  upload: (file: File) => {
    const body = new FormData()
    body.append('file', file)
    return checked('/ai/assets', AIAssetSchema, { method: 'POST', body, timeout: 60000 })
  },
  archive: (id: number) => checked(`/ai/assets/${id}`, nil, { method: 'DELETE' }),
  blob: (id: number, signal?: AbortSignal, preview = false) =>
    fetchApiBlob(`/ai/assets/${id}${preview ? '?preview=1' : ''}`, signal),
  publicBlob: (id: number, signal?: AbortSignal, preview = false) =>
    fetchApiBlob(`/creative/assets/${id}${preview ? '?preview=1' : ''}`, signal),
  ideaBlob: (ideaId: number, id: number, signal?: AbortSignal, preview = false) =>
    fetchApiBlob(`/ideas/${ideaId}/assets/${id}${preview ? '?preview=1' : ''}`, signal),
  publish: (id: number, title: string, description: string) =>
    checked(`/ai/assets/${id}/publish`, z.object({ content_id: z.number() }), {
      method: 'POST',
      body: { title, description, public_consent: true },
    }),
  saveBYOK: (data: {
    id?: number
    name: string
    allowed_provider_id: number
    api_key?: string
    default: boolean
  }) => checked('/ai/providers', AIProviderSchema, { method: 'POST', body: data }),
  deleteBYOK: (id: number) => checked(`/ai/providers/${id}`, nil, { method: 'DELETE' }),
  testBYOK: (id: number, key: string) =>
    checked(`/ai/providers/${id}/test`, AITaskSchema, {
      method: 'POST',
      body: { idempotency_key: key },
    }),
  ideas: (query: Record<string, unknown> = {}, signal?: AbortSignal) =>
    checked('/ideas', z.array(AIIdeaViewSchema), { query, signal }),
  idea: (id: number, signal?: AbortSignal) => checked(`/ideas/${id}`, AIIdeaViewSchema, { signal }),
  postIdea: (data: Record<string, unknown>) =>
    checked('/ai/ideas', AIIdeaSchema, { method: 'POST', body: data }),
  sponsor: (id: number, expectedCost: number, key: string) =>
    checked(`/ai/ideas/${id}/sponsor`, AITaskSchema, {
      method: 'POST',
      body: { expected_cost: expectedCost, idempotency_key: key },
    }),
  withdraw: (id: number) => checked(`/ai/ideas/${id}`, nil, { method: 'DELETE' }),
  providers: () => checked('/admin/ai/providers', z.array(AIProviderSchema)),
  saveProvider: (data: Omit<AIProvider, 'owner' | 'has_key'> & { api_key?: string }) =>
    checked('/admin/ai/providers', AIProviderSchema, { method: 'POST', body: data }),
  testProvider: (id: number, key: string) =>
    checked(`/admin/ai/providers/${id}/test`, AITaskSchema, {
      method: 'POST',
      body: { idempotency_key: key },
    }),
  adminTasks: (query: Record<string, unknown> = {}, signal?: AbortSignal) =>
    checked('/admin/ai/tasks', z.array(AITaskViewSchema), { query, signal }),
  resolve: (id: number, consumed: boolean, note: string) =>
    checked(`/admin/ai/tasks/${id}/resolve`, nil, { method: 'POST', body: { consumed, note } }),
  grant: (userId: number, amount: number, note: string, key: string) =>
    checked('/admin/ai/grants', nil, {
      method: 'POST',
      body: { user_id: userId, amount, note, idempotency_key: key },
    }),
  adminLedger: (userId?: number) =>
    checked('/admin/ai/ledger', z.array(AILedgerSchema), { query: { user_id: userId } }),
  adminIdeas: (query: Record<string, unknown> = {}, signal?: AbortSignal) =>
    checked('/admin/ai/ideas', z.array(AIIdeaViewSchema), { query, signal }),
  ideaEvents: (id: number, signal?: AbortSignal) =>
    checked(
      `/admin/ai/ideas/${id}/events`,
      z.array(
        z.object({
          id: z.number(),
          task_id: z.number(),
          actor_id: z.number(),
          action: z.string(),
          note: z.string(),
          created_at: z.string(),
        }),
      ),
      { signal },
    ),
  hideIdea: (id: number, note: string) =>
    checked(`/admin/ai/ideas/${id}/hide`, nil, { method: 'POST', body: { note } }),
  adminAssets: () => checked('/admin/ai/assets', z.array(AIAssetSchema)),
  metadata: (id: number, data: Record<string, unknown>) =>
    checked(`/admin/ai/assets/${id}`, nil, { method: 'PATCH', body: data }),
}
export const creativeApi = {
  resources: (kind?: string, search?: string, signal?: AbortSignal) =>
    checked('/creative/resources', z.array(CreativeResourceSchema), {
      query: { kind, search },
      signal,
    }),
  resource: (id: number, signal?: AbortSignal) =>
    checked(`/creative/resources/${id}`, CreativeResourceSchema, { signal }),
  character: async (slug: string, signal?: AbortSignal) => {
    const rows = await checked('/creative/resources', z.array(CreativeResourceSchema), {
      query: { kind: 'character', slug },
      signal,
    })
    if (!rows[0]) throw new Error('角色不存在')
    return rows[0]
  },
  expand: (id: number, values: Record<string, string>, version: number, signal?: AbortSignal) =>
    checked(`/creative/templates/${id}/expand`, z.object({ prompt: z.string() }), {
      method: 'POST',
      body: { values, version },
      signal,
    }),
  adminResources: () => checked('/admin/creative/resources', z.array(CreativeResourceSchema)),
  save: (data: CreativeResource) =>
    checked(
      data.id ? `/admin/creative/resources/${data.id}` : '/admin/creative/resources',
      CreativeResourceSchema,
      { method: data.id ? 'PUT' : 'POST', body: data },
    ),
}

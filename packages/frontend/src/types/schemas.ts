import { z } from 'zod'

const aiTime = z.string().regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/)
export const AIProviderSchema = z.object({
  id: z.number(),
  revision: z.number(),
  owner: z.number(),
  name: z.string(),
  endpoint: z.string(),
  model: z.string(),
  allowed_provider_id: z.number(),
  enabled: z.boolean(),
  default: z.boolean(),
  has_key: z.boolean(),
  sizes: z.array(z.string()),
  qualities: z
    .array(z.string())
    .nullable()
    .transform((v) => v ?? []),
  max_count: z.number(),
  max_references: z.number(),
  edit_transport: z.enum(['none', 'multipart', 'json']),
  timeout_seconds: z.number(),
  concurrency: z.number(),
  cost: z.number(),
  quality_costs: z
    .record(z.string(), z.number())
    .nullable()
    .transform((v) => v ?? {}),
})
export const AIConfigSchema = z.object({ providers: z.array(AIProviderSchema) })
export const AIAccountSchema = z.object({
  user_id: z.number(),
  available: z.number().int().nonnegative(),
  reserved: z.number().int().nonnegative(),
})
export const AISessionSchema = z.object({
  id: z.number().int().positive(),
  name: z.string(),
  draft: z.string(),
  version: z.number(),
  created_at: aiTime,
  updated_at: aiTime,
})
export const AIAssetSchema = z.object({
  id: z.number().int().positive(),
  task_id: z.number(),
  user_id: z.number(),
  mime: z.string(),
  bytes: z.number(),
  width: z.number(),
  height: z.number(),
  title: z.string(),
  description: z.string(),
  license: z.string(),
  purpose: z.string(),
  public: z.boolean(),
  download_allowed: z.boolean(),
  archived: z.boolean(),
  content_id: z.number().nullable(),
  created_at: aiTime,
})
export const AITaskSchema = z.object({
  id: z.number().int().positive(),
  user_id: z.number(),
  session_id: z.number(),
  prompt: z.string(),
  size: z.string(),
  quality: z.string(),
  count: z.number().int().positive(),
  model: z.string(),
  mode: z.string(),
  provider_id: z.number(),
  parent_task_id: z.number(),
  parent_asset_id: z.number(),
  idea_id: z.number(),
  status: z.enum(['queued', 'running', 'succeeded', 'failed', 'cancelled', 'unknown']),
  cost: z.number().int(),
  error: z.string(),
  created_at: aiTime,
  updated_at: aiTime,
})
export const AITaskViewSchema = z.object({
  task: AITaskSchema,
  user: z.object({ id: z.number(), username: z.string() }).optional(),
  assets: z.array(AIAssetSchema),
  references: z.array(AIAssetSchema).optional(),
})
export const AILedgerSchema = z.object({
  id: z.number(),
  user_id: z.number(),
  actor_id: z.number(),
  task_id: z.number(),
  kind: z.string(),
  amount: z.number(),
  note: z.string(),
  created_at: aiTime,
})
export const CreativeVariableSchema = z.object({
  name: z.string(),
  label: z.string(),
  type: z.enum(['text', 'select']),
  default: z.string(),
  required: z.boolean(),
  options: z
    .array(z.string())
    .nullable()
    .transform((v) => v ?? []),
  help: z.string(),
})
export const CreativeResourceSchema = z.object({
  id: z.number(),
  kind: z.enum(['character', 'preset', 'template']),
  slug: z.string(),
  title: z.string(),
  summary: z.string(),
  status: z.enum(['draft', 'published', 'archived']),
  sort_order: z.number(),
  version: z.number(),
  payload: z.object({
    description: z.string().default(''),
    appearance: z.string().default(''),
    body: z.string().default(''),
    asset_ids: z
      .array(z.number())
      .nullable()
      .optional()
      .transform((v) => v ?? []),
    character_ids: z
      .array(z.number())
      .nullable()
      .optional()
      .transform((v) => v ?? []),
    tags: z
      .array(z.string())
      .nullable()
      .optional()
      .transform((v) => v ?? []),
    variables: z
      .array(CreativeVariableSchema)
      .nullable()
      .optional()
      .transform((v) => v ?? []),
    preset_id: z.number().default(0),
    parameters: z
      .record(z.string(), z.unknown())
      .nullable()
      .optional()
      .transform((v) => v ?? {}),
  }),
  created_at: aiTime,
  updated_at: aiTime,
})
export const AIIdeaSchema = z.object({
  id: z.number(),
  user_id: z.number(),
  title: z.string(),
  description: z.string(),
  snapshot: z.string(),
  character_ids: z.string(),
  model: z.string(),
  status: z.string(),
  sponsor_id: z.number(),
  task_id: z.number(),
  estimated_cost: z.number(),
  created_at: aiTime,
  updated_at: aiTime,
})
export const AIIdeaViewSchema = z.object({
  idea: AIIdeaSchema,
  author: z.object({ id: z.number(), username: z.string() }),
  sponsor: z.object({ id: z.number(), username: z.string() }).nullable(),
  references: z.array(AIAssetSchema),
  results: z.array(AIAssetSchema),
  current_cost: z.number(),
  sponsorable: z.boolean(),
})
export type AIConfig = z.infer<typeof AIConfigSchema>
export type AIAccount = z.infer<typeof AIAccountSchema>
export type AISession = z.infer<typeof AISessionSchema>
export type AIAsset = z.infer<typeof AIAssetSchema>
export type AITask = z.infer<typeof AITaskSchema>
export type AIProvider = z.infer<typeof AIProviderSchema>
export type AITaskView = z.infer<typeof AITaskViewSchema>
export type CreativeResource = z.infer<typeof CreativeResourceSchema>
export type CreativeVariable = z.infer<typeof CreativeVariableSchema>
export type AIIdeaView = z.infer<typeof AIIdeaViewSchema>

export const MetadataOptionsSchema = z.object({ available: z.boolean(), ready: z.boolean(), placeholder_title: z.string().min(1), placeholder_tag: z.string().min(1) })
export const VisionSettingsSchema = z.object({
  reference_mode: z.enum(['images', 'text']).optional(), reference_prompt: z.string().optional(),
  enabled: z.boolean(), base_url: z.string().url(), model: z.string().min(1), prompt: z.string().min(1),
  max_tags: z.number().int().min(1).max(12), timeout_seconds: z.number().int().min(10).max(180),
  disable_thinking: z.boolean(), token_parameter: z.enum(['max_tokens', 'max_completion_tokens']), use_references: z.boolean(), has_api_key: z.boolean(),
})
export const VisionJobSchema = z.object({
  content_id: z.number().int().positive(), status: z.enum(['pending', 'running', 'succeeded', 'failed', 'cancelled']),
  attempts: z.number().int().min(0), error: z.string(), created_at: z.string().datetime({ offset: true }), updated_at: z.string().datetime({ offset: true }),
})
export const VisionResultSchema = z.object({ title: z.string().min(2), tags: z.array(z.string().min(2)).min(1).max(12) })
export const VisionReferenceSchema = z.object({ name: z.string().min(1), bytes: z.number().int().positive(), version: z.string().optional(), description: z.string().optional(), needs_description: z.boolean().optional() })

// ── 通用宽松 transform(沿用旧实现 normalize 的 `Number()||0` / `typeof==='string'? : ''` 兜底语义)──
// 用 z.unknown + transform 而不是 z.coerce,避免 NaN 不触发 default 的边界问题;
// 严格校验留给"值域有界的字段"(type/audit_status)用 enum/catch。
const num = z.unknown().transform((v) => Number(v) || 0)
const str = z.unknown().transform((v) => (typeof v === 'string' ? v : ''))
// 时间字段专用：后端返回 ISO 字符串（如 '2026-09-13T11:28:04.865Z'），而 formatTime 对数字按 Unix 秒解释。
// 若沿用 num，Number(ISO) === NaN → 0，时间会被整体丢成 0（详情页/后台一旦消费解析结果就会显示空）。
const dateNum = z.unknown().transform((v) => {
  if (typeof v === 'number') return Number.isFinite(v) ? v : 0
  if (typeof v === 'string' && v) {
    const ms = Date.parse(v)
    if (!Number.isNaN(ms)) return Math.floor(ms / 1000)
    const n = Number(v)
    return Number.isFinite(n) ? n : 0
  }
  return 0
})
const bool = z.unknown().transform((v) => Boolean(v))
const tagsArr = z
  .unknown()
  .transform((v) =>
    Array.isArray(v) ? v.filter((t): t is string => typeof t === 'string') : [],
  )

// ── User(完整) ──
// 与旧实现 normalize 等价:id/username 必填(契约保证),其他 optional 兜空。
const UserSchema = z.object({
  id: num,
  username: str,
  email: str.optional(),
  is_admin: bool.optional(),
  is_banned: bool.optional(),
  created_at: dateNum.optional(),
  updated_at: dateNum.optional(),
})
export type User = z.infer<typeof UserSchema>

// ── User(简介,推荐页用) ──
const UserBriefSchema = z.object({
  id: num,
  username: str,
  avatar_url: str.optional(),
})

// ── RecommendContent(推荐页) ──
export const RecommendContentSchema = z.object({
  id: num,
  title: str,
  thumb: str.optional().default(''),
  // 星图专用低体积图片；可选以兼容旧服务响应和首页缓存。
  atlas_thumb: str.optional(),
  montage_thumb: str.optional(),
  tags: tagsArr,
  view_count: num.optional().default(0),
  like_count: num.optional().default(0),
  user: UserBriefSchema,
  created_at: dateNum,
})
export type RecommendContent = z.infer<typeof RecommendContentSchema>

// ── Content(列表/详情) ──
export const ContentSchema = z.object({
  id: num,
  title: str,
  text: str.optional().default(''),
  thumb: str.optional().default(''),
  // 星图专用低体积图片；可选以兼容旧服务响应和首页缓存。
  atlas_thumb: str.optional(),
  montage_thumb: str.optional(),
  video: str.optional().default(''),
  img: str.optional().default(''),
  origin: str.optional().default(''),
  // R2 地址（未接入 R2 时为空）。与 mirror2_* 平级、各自独立；
  // 详情页按 OpenList → R2 → 源站 的可达性选择取用，缩略图不镜像故 thumb 无对应字段。
  mirror_img: str.optional().default(''),
  mirror_video: str.optional().default(''),
  // 首选镜像（自建 OpenList）地址。与 mirror_* 平级、各自独立：
  // 未配公开域名时为空，前端跳过该级继续往下回退。
  mirror2_img: str.optional().default(''),
  mirror2_video: str.optional().default(''),
  file_size: num.optional().default(0),
  user: UserSchema,
  avatar_url: str.optional().default(''),
  tags: tagsArr,
  view_count: num.optional().default(0),
  like_count: num.optional().default(0),
  audit_status: z.enum(['pending', 'approved', 'rejected']).optional(),
  created_at: dateNum,
  updated_at: dateNum.optional(),
})
export type Content = z.infer<typeof ContentSchema>

// ── Comment ──
export const CommentSchema: z.ZodType<{
  id: number
  content_id: number
  user_id: number
  text: string
  parent_id: number | null
  is_banned: boolean
  created_at: number
  updated_at?: number
  user?: z.infer<typeof UserBriefSchema>
  parent?: { id: number; user_id: number; text: string; user?: z.infer<typeof UserBriefSchema> }
  replies?: Comment[]
}> = z.object({
  id: num,
  content_id: num,
  user_id: num,
  text: str,
  parent_id: num.nullable(),
  is_banned: bool,
  created_at: dateNum,
  updated_at: dateNum.optional(),
  user: UserBriefSchema.optional(),
  parent: z.object({
    id: num,
    user_id: num,
    text: str,
    user: UserBriefSchema.optional(),
  }).optional(),
  replies: z.lazy(() => z.array(CommentSchema)).optional(),
})
export type Comment = z.infer<typeof CommentSchema>

// ── Poll ──
const PollSchema = z.object({
  id: num,
  title: str,
  description: str,
  options: z.unknown().transform((v) => Array.isArray(v) ? v.filter((o): o is string => typeof o === 'string') : []),
  vote_count: num,
  user_id: num,
  user: UserBriefSchema.optional(),
  created_at: str,
  updated_at: str.optional(),
})
export type Poll = z.infer<typeof PollSchema>

// ── Claim ──
const ClaimSchema = z.object({
  id: num,
  content_id: num,
  user_id: num,
  user: UserBriefSchema,
  content: ContentSchema,
  reason: str,
  status: z.enum(['pending', 'approved', 'rejected']),
  remark: str,
  created_at: str,
  updated_at: str.optional(),
})
export type Claim = z.infer<typeof ClaimSchema>

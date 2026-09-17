import { z } from 'zod'

// ── 通用宽松 transform(匹配原 normalize 的 `Number()||0` / `typeof==='string'? : ''` 兜底语义)──
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
// 与原 normalize 等价:id/username 必填(契约保证),其他 optional 兜空。
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
})

// ── RecommendContent(推荐页) ──
export const RecommendContentSchema = z.object({
  id: num,
  title: str,
  thumb: str.optional().default(''),
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
  video: str.optional().default(''),
  img: str.optional().default(''),
  origin: str.optional().default(''),
  // R2 备份地址（未接入 R2 时为空）。详情页据此在源站与 R2 之间测速择快；
  // 缩略图不镜像，因此 thumb 没有对应字段。
  mirror_img: str.optional().default(''),
  mirror_video: str.optional().default(''),
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
const CommentSchema: z.ZodType<{
  id: number
  content_id: number
  user_id: number
  text: string
  parent_id: number | null
  is_banned: boolean
  created_at: number
  updated_at?: number
  user?: { id: number; username: string }
  parent?: { id: number; user_id: number; text: string; user?: { id: number; username: string } }
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

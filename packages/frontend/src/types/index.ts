import { CONTENT_TYPES, ContentSchema, RecommendContentSchema } from './schemas'
import type { ContentType, User, Content, RecommendContent, Comment, Poll, Claim } from './schemas'

export { CONTENT_TYPES, ContentSchema, RecommendContentSchema }
export type { ContentType, User, Content, RecommendContent, Comment, Poll, Claim }

export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PaginatedResponse<T = unknown> {
  list: T[]
  total: number
  page: number
  page_size: number
  total_page: number
}


export interface LoginResponse {
  user: User
  needs_email?: boolean
}

export interface RegisterResponse {
  user_id: number
}


export interface AuditRequest {
  status: 'approved' | 'rejected'
  remark?: string
}

// 2026-07-29 改造：上传不再让用户选 type，后端根据"是否有 file"自动设 image / text。
// 描述字段统一用 `content`（对应后端 contents.content / DB column）。
export interface UploadContentData {
  title: string
  content?: string
  tags?: string[]
  user_id: number
  file?: File
}

// 游客快速上传：title 必填；text（content）与 file 至少一个（前端校验，后端兜底）。
export interface QuickUploadData {
  title: string
  nickname: string
  email: string
  content?: string
  tags?: string[]
  file?: File
}

export interface CommentReport {
  id: number
  comment_id: number
  user_id: number
  reason: string
  handled: boolean
  created_at: string
  Comment?: Pick<Comment, 'id' | 'text'>
  User?: User
}

export interface CommentCount {
  content_id: number
  count: number
}

export interface ListParams {
  page?: number
  page_size?: number
  tag?: string
  type?: string
  audit_status?: string
  sort_by?: string
  order?: string
  keyword?: string
}


export interface RecommendResponse {
  list: RecommendContent[]
  count: number
}

export interface PollDetail {
  poll: Poll
  vote_counts: Record<string, number>
  total_votes: number
  my_vote: number | null
}

export interface CreatePollData {
  title: string
  description?: string
  options: string[]
}

// ── API Key 类型 ──

export interface ApiKey {
  id: number
  name: string
  key_prefix: string
  permissions: string[]
  is_active: boolean
  last_used_at: number | null
  created_at: number
}

export interface ApiKeyCreated extends ApiKey {
  key: string // 完整 key，仅创建时返回一次
}

export interface CreateApiKeyData {
  name: string
  permissions: string[]
}

export interface UpdateApiKeyData {
  name?: string
  permissions?: string[]
  is_active?: boolean
}

// ── API 响应类型（契约单一来源，前端不再内联匿名结构）──

export interface CommentListResponse {
  list: Comment[]
  total: number
  page: number
  page_size: number
  total_page: number
}

export interface PollListResponse {
  list: Poll[]
  total: number
  page: number
  page_size: number
  total_page: number
}

export interface RegenerateThumbnailResponse {
  id: number
  thumb_path: string
}

export interface RegenerateAllResponse {
  count: number
  running?: boolean
}

export interface UpdateContentAuthorResponse {
  content_id: number
  old_user_id: number
  new_user_id: number
  new_username: string
}

export interface ClaimListResponse {
  list: Claim[]
  total: number
  page: number
  page_size: number
  total_page: number
}

// ── 仪表盘统计 ──

interface DashboardContentStats {
  total: number
  pending: number
  approved: number
  rejected: number
  image: number
  text: number
  today: number
}

interface DashboardUsersStats {
  total: number
  admins: number
  banned: number
  today: number
}

interface DashboardTagCount {
  tag: string
  count: number
}

export interface DashboardStats {
  content: DashboardContentStats
  users: DashboardUsersStats
  comments: { total: number; today: number }
  claims: { total: number; pending: number; approved: number; rejected: number }
  reports: { total: number; unhandled: number; handled: number }
  polls: { total: number; votes: number }
  views: number
  topTags: DashboardTagCount[]
  recentContents: Content[]
  recentUsers: User[]
}

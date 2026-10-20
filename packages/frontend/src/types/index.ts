import { ContentSchema, RecommendContentSchema } from './schemas'
import type { User, Content, RecommendContent, Comment, Poll, Claim, AIConfig, AIAccount, AISession, AITask, AIAsset } from './schemas'
import type { z } from 'zod'
import type { MetadataOptionsSchema, VisionSettingsSchema, VisionJobSchema, VisionResultSchema, VisionReferenceSchema } from './schemas'

export { ContentSchema, RecommendContentSchema }
export type { User, Content, RecommendContent, Comment, Poll, Claim }
export type { PollDetail, PollListResponse } from './schemas'
export type { AIConfig, AIAccount, AISession, AITask, AIAsset }

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

export interface UploadContentData {
  title: string
  content?: string
  tags?: string[]
  user_id: number
  file?: File
  auto_metadata?: boolean
  is_ai_generated?: boolean
}

export interface QuickUploadData {
  title: string
  nickname: string
  email: string
  content?: string
  tags?: string[]
  file?: File
  auto_metadata?: boolean
  is_ai_generated?: boolean
}

export type MetadataOptions = z.infer<typeof MetadataOptionsSchema>
export type VisionSettings = z.infer<typeof VisionSettingsSchema>
export type VisionJob = z.infer<typeof VisionJobSchema>
export type VisionResult = z.infer<typeof VisionResultSchema>
export type VisionReference = z.infer<typeof VisionReferenceSchema>

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
  audit_status?: string
  sort_by?: string
  order?: string
  keyword?: string
}


export interface RecommendResponse {
  list: RecommendContent[]
  count: number
}

export interface CreatePollData {
  title: string
  description?: string
  options: string[]
}

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
  key: string
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

export interface CommentListResponse {
  list: Comment[]
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
  ok?: number
  fail?: number
  total?: number
  running?: boolean
}

export interface RegenerateAllStatus {
  status: 'idle' | 'running' | 'done' | 'error'
  total: number
  ok: number
  fail: number
  started_at?: number
  updated_at?: number
  finished_at?: number
  error?: string
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

export type AdminLogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface AdminLogEntry {
  id?: string | number
  time: string
  level: AdminLogLevel | Uppercase<AdminLogLevel>
  message: string
  line?: string
  attrs?: Record<string, string | number | boolean | null>
}

export interface AdminLogsResponse {
  entries: AdminLogEntry[]
  next_cursor?: number
  truncated?: boolean
  server_time?: string
}

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

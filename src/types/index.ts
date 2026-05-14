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

export interface User {
  id: number
  username: string
  is_admin: boolean
  is_banned: boolean
  created_at: number
  updated_at: number
}

export interface LoginResponse {
  user: User
}

export interface RegisterResponse {
  user_id: number
}

export interface Content {
  id: number
  title: string
  type: 'video' | 'image' | 'text' | 'link'
  text?: string
  url?: string
  thumb?: string
  video?: string
  img?: string
  file_size?: number
  user: User
  tags: string[]
  view_count: number
  audit_status?: 'pending' | 'approved' | 'rejected'
  created_at: number
  updated_at?: number
}

export interface AuditRequest {
  admin_id: number
  status: 'approved' | 'rejected'
  remark?: string
}

export interface UploadContentData {
  title: string
  type: 'video' | 'image' | 'text' | 'link'
  content?: string
  url?: string
  user_id: number
  tags?: string[]
  file?: File
}

export interface Comment {
  id: number
  content_id: number
  user_id: number
  text: string
  parent_id: number | null
  is_banned: boolean
  created_at: number
  updated_at?: number
  user?: { id: number; username: string; is_admin?: boolean }
  parent?: Pick<Comment, 'id' | 'user_id' | 'text'> & { user?: { id: number; username: string } }
  replies?: Comment[]
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

export interface RecommendContent {
  id: number
  title: string
  type: 'video' | 'image' | 'text' | 'link'
  url?: string
  thumb?: string
  tags: string[]
  view_count?: number
  user: { id: number; username: string }
  created_at: number
}

export interface RecommendResponse {
  list: RecommendContent[]
  count: number
}

// 投票相关类型
export interface Poll {
  id: number
  title: string
  description: string
  options: string[]
  vote_count: number
  user_id: number
  user: { id: number; username: string }
  created_at: string
  updated_at: string
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

export interface VoteData {
  option_index: number
}

export interface Claim {
  id: number
  content_id: number
  user_id: number
  user: { id: number; username: string }
  content: Content
  reason: string
  status: 'pending' | 'approved' | 'rejected'
  remark: string
  created_at: string
  updated_at: string
}

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
  ID?: number
  Username?: string
  IsAdmin?: boolean
  IsBanned?: boolean
  created_at: string
  updated_at: string
  CreatedAt?: string
  UpdatedAt?: string
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
  type: 'video' | 'image' | 'text'
  content: string
  file_path: string
  file_size: number
  thumb_path?: string
  user_id: number
  user: User
  tags: string[]
  audit_status: 'pending' | 'approved' | 'rejected'
  created_at: string
  updated_at: string
  image?: string
  video?: string
  ID?: number
  Title?: string
  Type?: string
  Content?: string
  FilePath?: string
  FileSize?: number
  UserID?: number
  User?: User
  Tags?: string[]
  AuditStatus?: string
  CreatedAt?: string
  UpdatedAt?: string
}

export interface AuditRequest {
  admin_id: number
  status: 'approved' | 'rejected'
  remark?: string
}

export interface UploadContentData {
  title: string
  type: 'video' | 'image' | 'text'
  content?: string
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
  created_at: string
  updated_at?: string
  User?: User
  user?: User
  Parent?: Pick<Comment, 'id' | 'user_id' | 'text'> & { User?: User }
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
  type: 'video' | 'image' | 'text'
  file_path: string
  image: string
  tags: string[]
  view_count?: number
  user: { id: number; username: string }
  created_at: string
  ID?: number
  Title?: string
  Type?: string
  FilePath?: string
  Tags?: string[]
  User?: { ID?: number; Username?: string }
  CreatedAt?: string
}

export interface RecommendResponse {
  list: RecommendContent[]
  count: number
}

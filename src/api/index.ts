import { ofetch } from 'ofetch'
import type {
  ApiResponse,
  User,
  LoginResponse,
  RegisterResponse,
  Content,
  UploadContentData,
  AuditRequest,
  ListParams,
  PaginatedResponse,
  Comment,
  CommentReport,
  CommentCount,
  RecommendResponse,
  Poll,
  PollDetail,
  CreatePollData,
} from '@/types'

export interface CommentListResponse {
  list: Comment[]
  total: number
  page: number
  page_size: number
  total_page: number
}

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const api = ofetch.create({
  baseURL: BASE_URL,
  credentials: 'include',
  onRequestError({ error }) {
    console.error('[API] 请求失败:', error)
  },
  onResponseError({ response }) {
    if (response.status === 401) {
      console.warn('[API] 未授权，请重新登录')
    } else if (response.status >= 500) {
      console.error('[API] 服务器错误:', response.status)
    }
  },
})

async function request<T>(
  url: string,
  options: Parameters<typeof api>[1] = {},
): Promise<ApiResponse<T>> {
  return api(url, options) as Promise<ApiResponse<T>>
}

export const authApi = {
  initAdmin: () => request('/auth/init-admin', { method: 'POST' }),

  register: (username: string, password: string) =>
    request<RegisterResponse>('/auth/register', {
      method: 'POST',
      body: { username, password },
    }),

  login: (username: string, password: string) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: { username, password },
    }),

  logout: () => request('/auth/logout', { method: 'POST' }),

  getMe: () => request<User>('/auth/me', { method: 'GET' }),
}

export const contentApi = {
  getTags: () => request<string[]>('/content/tags'),

  recommend: (count: number, page?: number) => {
    return request<RecommendResponse>('/content/recommend', {
      query: { count, page: page || 1 },
    })
  },

  upload: (
    data: UploadContentData,
    onProgress?: (percent: number) => void,
  ): Promise<ApiResponse<Content>> => {
    const formData = new FormData()

    if (data.title) {
      formData.append('title', data.title)
    }

    if (data.type) {
      formData.append('type', data.type)
    }

    if (data.user_id !== undefined && data.user_id !== null) {
      formData.append('user_id', data.user_id.toString())
    }

    if (data.content) {
      formData.append('content', data.content)
    }

    if (data.url) {
      formData.append('url', data.url)
    }

    if (data.tags && data.tags.length > 0) {
      data.tags.forEach((tag) => {
        formData.append('tags', tag)
      })
    }

    if (data.file) {
      formData.append('file', data.file)
    }

    return new Promise<ApiResponse<Content>>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', `${BASE_URL}/content/upload`)
      xhr.withCredentials = true

      if (onProgress) {
        xhr.upload.addEventListener('progress', (event) => {
          if (event.lengthComputable) {
            const percent = Math.round((event.loaded * 100) / event.total)
            onProgress(percent)
          }
        })
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve(JSON.parse(xhr.responseText))
        } else {
          reject(new Error(`Upload failed: ${xhr.status}`))
        }
      }

      xhr.onerror = () => reject(new Error('Network error'))
      xhr.send(formData)
    })
  },

  list: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/content/list', {
      query: params,
    })
  },

  myList: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/content/my', {
      query: params,
    })
  },

  detail: (id: number) => request<Content>(`/content/${id}`),

  search: (keyword: string, params?: Omit<ListParams, 'keyword'>) => {
    return request<PaginatedResponse<Content>>('/content/search', {
      query: { keyword, ...params },
    })
  },

  update: (
    id: number,
    data: { title?: string; content?: string; url?: string; tags?: string[]; file?: File },
  ) => {
    const formData = new FormData()
    if (data.title) formData.append('title', data.title)
    if (data.content) formData.append('content', data.content)
    if (data.url) formData.append('url', data.url)
    if (data.tags && data.tags.length > 0) {
      data.tags.forEach((tag) => {
        formData.append('tags', tag)
      })
    }
    if (data.file) formData.append('file', data.file)

    return request<Content>(`/content/${id}`, {
      method: 'PUT',
      body: formData,
    })
  },

  delete: (id: number) => request(`/content/${id}`, { method: 'DELETE' }),

  uploadImage: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{
      id: number
      filename: string
      file_size: number
      image_url: string
      upload_time: string
    }>('/content/upload-image', {
      method: 'POST',
      body: formData,
    })
  },

  submitClaim: (contentId: number, reason: string) =>
    request(`/content/${contentId}/claim`, {
      method: 'POST',
      body: { reason },
    }),
}

export const commentApi = {
  add: (contentId: number, text: string, parentId?: number) => {
    const formData = new FormData()
    formData.append('content_id', contentId.toString())
    formData.append('text', text)
    if (parentId !== undefined && parentId !== null) {
      formData.append('parent_id', parentId.toString())
    }
    return request<Comment>('/comment/add', {
      method: 'POST',
      body: formData,
    })
  },

  list: (contentId: number, page: number = 1, pageSize: number = 20) => {
    return request<CommentListResponse>(`/comment/list/${contentId}`, {
      query: { page, page_size: pageSize }
    })
  },

  delete: (id: number) => request(`/comment/${id}`, { method: 'DELETE' }),

  count: (contentId: number) => request<CommentCount>(`/comment/count/${contentId}`),

  report: (commentId: number, reason?: string) => {
    const formData = new FormData()
    formData.append('comment_id', commentId.toString())
    if (reason) {
      formData.append('reason', reason)
    }
    return request<CommentReport>('/comment/report', {
      method: 'POST',
      body: formData,
    })
  },

  handleReport: (reportId: number) => {
    return request(`/admin/comments/reports/${reportId}/handle`, {
      method: 'POST',
    })
  },

  getReports: () => request<CommentReport[]>('/admin/comments/reports'),
}

export const pollApi = {
  create: (data: CreatePollData) =>
    request<Poll>('/poll/create', {
      method: 'POST',
      body: data,
    }),

  list: () => request<{ list: Poll[] }>('/poll/list'),

  detail: (id: number) => request<PollDetail>(`/poll/${id}`),

  vote: (id: number, optionIndex: number) =>
    request(`/poll/${id}/vote`, {
      method: 'POST',
      body: {
        option_index: optionIndex,
      },
    }),

  delete: (id: number) => request(`/poll/${id}`, { method: 'DELETE' }),
}

export const adminApi = {
  audit: (id: number, data: AuditRequest) =>
    request<Content>(`/admin/audit/${id}`, {
      method: 'POST',
      body: data,
    }),

  pending: (params?: Pick<ListParams, 'page' | 'page_size'>) => {
    return request<PaginatedResponse<Content>>('/admin/pending', {
      query: params,
    })
  },

  getAllContent: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/admin/content/all', {
      query: params,
    })
  },

  getUsers: (params?: Pick<ListParams, 'page' | 'page_size' | 'keyword'>) => {
    return request<PaginatedResponse<User>>('/admin/users', {
      query: params,
    })
  },

  updateUserRole: (id: number, isAdmin: boolean) =>
    request<User>(`/admin/users/${id}/role`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: { is_admin: isAdmin },
    }),

  updateUserBan: (id: number, isBanned: boolean) =>
    request<User>(`/admin/users/${id}/ban`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: { is_banned: isBanned },
    }),

  regenerateThumbnail: (id: number) =>
    request<{ id: number; thumb: string }>(`/admin/content/${id}/regenerate-thumbnail`, {
      method: 'POST',
    }),

  regenerateAllThumbnails: () =>
    request<{ count: number }>('/admin/content/regenerate-all-thumbnails', {
      method: 'POST',
    }),

  deleteUser: (id: number) => request(`/admin/users/${id}`, { method: 'DELETE' }),

  updateContentAuthor: (contentId: number, userId: number) =>
    request<{
      content_id: number
      old_user_id: number
      new_user_id: number
      new_username: string
    }>(`/admin/content/${contentId}/author`, {
      method: 'PUT',
      body: { user_id: userId },
    }),

  getClaims: (params: { page?: number; page_size?: number; status?: string }) =>
    request<{
      list: any[]
      total: number
      page: number
      page_size: number
    }>('/admin/claims', {
      query: params,
    }),

  handleClaim: (claimId: number, action: 'approve' | 'reject', remark?: string) =>
    request(`/admin/claims/${claimId}/handle`, {
      method: 'POST',
      body: { action, remark },
    }),
}

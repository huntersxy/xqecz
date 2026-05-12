import axios, { type AxiosRequestConfig } from 'axios'
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
  RecommendResponse
} from '@/types'

const BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const instance = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
})

instance.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      console.warn('[API] 未授权，请重新登录')
    } else if (error.response?.status >= 500) {
      console.error('[API] 服务器错误:', error.response?.status)
    }
    return Promise.reject(error)
  }
)

async function request<T>(url: string, options: AxiosRequestConfig = {}): Promise<ApiResponse<T>> {
  try {
    const response = await instance(url, options)
    return response.data
  } catch (error) {
    console.error('[API] 请求失败:', error)
    throw error
  }
}

export const authApi = {
  initAdmin: () => request('/auth/init-admin', { method: 'POST' }),

  register: (username: string, password: string) =>
    request<RegisterResponse>('/auth/register', {
      method:
 'POST',
      data: { username, password },
    }),

  login: (username: string, password: string) =>
    request<LoginResponse>('/auth/login', {
      method: 'POST',
      data: { username, password },
    }),

  logout: () => request('/auth/logout', { method: 'POST' }),

  getMe: () => request<User>('/auth/me', { method: 'GET' }),
}

export const contentApi = {
  getTags: () => request<string[]>('/content/tags'),

recommend: (count: number, page?: number) => {
  return request<RecommendResponse>('/content/recommend', {
    params: { count, page: page || 1 },
  })
},

  upload: (data: UploadContentData, onProgress?: (percent: number) => void): Promise<ApiResponse<Content>> => {
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

    if (data.tags && data.tags.length > 0) {
      data.tags.forEach(tag => {
        formData.append('tags', tag)
      })
    }

    if (data.file) {
      formData.append('file', data.file)
    }

    if (onProgress) {
      return new Promise<ApiResponse<Content>>((resolve, reject) => {
        instance.post('/content/upload', formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
          onUploadProgress: (progressEvent) => {
            if (progressEvent.total) {
              const percent = Math.round((progressEvent.loaded * 100) / progressEvent.total)
              onProgress(percent)
            }
          },
        }).then(response => resolve(response.data)).catch(reject)
      })
    }

    return request<Content>('/content/upload', {
      method: 'POST',
      data: formData,
    })
  },

  list: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/content/list', {
      params,
    })
  },

  myList: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/content/my', {
      params,
    })
  },

  detail: (id: number) => request<Content>(`/content/${id}`),

  search: (keyword: string, params?: Omit<ListParams, 'keyword'>) => {
    return request<PaginatedResponse<Content>>('/content/search', {
      params: { keyword, ...params },
    })
  },

  update: (id: number, data: { title?: string; content?: string; tags?: string[]; file?: File }) => {
    const formData = new FormData()
    if (data.title) formData.append('title', data.title)
    if (data.content) formData.append('content', data.content)
    if (data.tags && data.tags.length > 0) {
      data.tags.forEach(tag => {
        formData.append('tags', tag)
      })
    }
    if (data.file) formData.append('file', data.file)

    return request<Content>(`/content/${id}`, {
      method: 'PUT',
      data: formData,
    })
  },

  delete: (id: number) => request(`/content/${id}`, { method: 'DELETE' }),

  uploadImage: (file: File) => {
    const formData = new FormData()
    formData.append('file', file)
    return request<{ id: number; filename: string; file_size: number; image_url: string; upload_time: string }>('/content/upload-image', {
      method: 'POST',
      data: formData,
    })
  },
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
      data: formData,
    })
  },

  list: (contentId: number) => {
    return request<Comment[]>(`/comment/list/${contentId}`)
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
      data: formData,
    })
  },

  handleReport: (reportId: number) => {
    return request(`/admin/comments/reports/${reportId}/handle`, {
      method: 'POST',
    })
  },

  getReports: () => request<CommentReport[]>('/admin/comments/reports'),
}

export const adminApi = {
  audit: (id: number, data: AuditRequest) =>
    request<Content>(`/admin/audit/${id}`, {
      method: 'POST',
      data,
    }),

  pending: (params?: Pick<ListParams, 'page' | 'page_size'>) => {
    return request<PaginatedResponse<Content>>('/admin/pending', {
      params,
    })
  },

  getAllContent: (params?: ListParams) => {
    return request<PaginatedResponse<Content>>('/admin/content/all', {
      params,
    })
  },

  getUsers: (params?: Pick<ListParams, 'page' | 'page_size' | 'keyword'>) => {
    return request<PaginatedResponse<User>>('/admin/users', {
      params,
    })
  },

  updateUserRole: (id: number, isAdmin: boolean) =>
    request<User>(`/admin/users/${id}/role`, {
      method: 'PUT',
      data: { is_admin: isAdmin },
    }),

  updateUserBan: (id: number, isBanned: boolean) =>
    request<User>(`/admin/users/${id}/ban`, {
      method: 'PUT',
      data: { is_banned: isBanned },
    }),

  regenerateThumbnail: (id: number) =>
    request<{ id: number; thumb_path: string }>(`/admin/content/${id}/regenerate-thumbnail`, {
      method: 'POST',
    }),

  regenerateAllThumbnails: () =>
    request<{ count: number }>('/admin/content/regenerate-all-thumbnails', {
      method: 'POST',
    }),

  deleteUser: (id: number) => request(`/admin/users/${id}`, { method: 'DELETE' }),
}

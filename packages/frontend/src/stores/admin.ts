import { defineStore } from 'pinia'
import { adminApi, contentApi, commentApi, pollApi } from '@/api'
import { toast, useConfirm } from '@/composables/useToast'
import type { Content, User, Claim, CommentReport, Poll, CreatePollData } from '@/types'

interface PaginatedState<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
  loading: boolean
}

function createPaginatedState<T>(pageSize = 20): PaginatedState<T> {
  return { list: [], total: 0, page: 1, pageSize, totalPages: 1, loading: false }
}

interface PageResult<T> { list: T[]; total: number; total_page: number }

async function loadPage<T>(state: PaginatedState<T>, page: number, fetch: (p: number) => Promise<PageResult<T>>): Promise<void> {
  state.loading = true
  state.page = page
  try {
    const data = await fetch(page)
    state.list = data.list
    state.total = data.total
    state.totalPages = data.total_page
  } catch (e: unknown) { toast.error((e as Error).message || '加载失败') } finally { state.loading = false }
}

const { confirm } = useConfirm()

async function runAdmin(action: () => Promise<unknown>, successMsg: string, failMsg: string): Promise<boolean> {
  try {
    await action()
    toast.success(successMsg)
    return true
  } catch (e: unknown) { toast.error((e as Error).message || failMsg); return false }
}

const apiChangeAuthor = (contentId: number, userId: number) =>
  runAdmin(() => adminApi.updateContentAuthor(contentId, userId), '作者已更新', '更新失败')

function apiConfirmDelete(id: number, onOk?: () => void) {
  confirm('确定要删除这条内容吗？此操作不可撤销。').then(async (ok) => {
    if (!ok) return
    try {
      await contentApi.delete(id)
      toast.success('删除成功')
      onOk?.()
    } catch (e: unknown) { toast.error((e as Error).message || '删除失败') }
  })
}

const apiAuditContent = (id: number, status: 'approved' | 'rejected', _adminId: number) =>
  runAdmin(() => adminApi.audit(id, { status, remark: '' }), '审核成功', '审核失败')

const apiUpdateUserRole = (id: number, isAdmin: boolean) =>
  runAdmin(() => adminApi.updateUserRole(id, isAdmin), '更新成功', '更新失败')

const apiUpdateUserBan = (id: number, isBanned: boolean) =>
  runAdmin(() => adminApi.updateUserBan(id, isBanned), isBanned ? '封禁成功' : '解封成功', isBanned ? '封禁失败' : '解封失败')

const apiDeleteUser = (id: number) =>
  runAdmin(() => adminApi.deleteUser(id), '删除成功', '删除失败')

const apiHandleClaim = async (claimId: number, action: 'approve' | 'reject'): Promise<boolean> => {
  if (action === 'reject') {
    const ok = await confirm('请输入拒绝原因（可选），点取消放弃操作。')
    if (!ok) return false
  }
  return runAdmin(() => adminApi.handleClaim(claimId, action, undefined), action === 'approve' ? '认领已通过' : '认领已拒绝', '操作失败')
}

const apiDeletePoll = (id: number) =>
  runAdmin(() => pollApi.delete(id), '删除成功', '删除失败')

const apiHandleReport = (reportId: number) =>
  runAdmin(() => commentApi.handleReport(reportId), '处理成功', '处理失败')

const apiDeleteComment = (commentId: number) =>
  runAdmin(() => commentApi.delete(commentId), '删除成功', '删除失败')

const apiRegenerateThumbnail = (id: number) =>
  runAdmin(() => adminApi.regenerateThumbnail(id), '封面更新成功', '封面更新失败')

async function apiRegenerateAllThumbnails() {
  try {
    const r = await adminApi.regenerateAllThumbnails()
    toast.success(`已开始处理 ${r.data.count} 条`)
  } catch (e: unknown) { toast.error((e as Error).message || '操作失败') }
}

export const useAdminStore = defineStore('admin', () => {
  const activeTab = ref('my')
  const tags = ref<string[]>([])
  const tagsLoading = ref(false)

  const myContent = reactive(createPaginatedState<Content>())
  const allContent = reactive(createPaginatedState<Content>())
  const pendingContent = reactive(createPaginatedState<Content>())
  const users = reactive(createPaginatedState<User>(24))
  const claims = reactive(createPaginatedState<Claim>())
  const polls = ref<Poll[]>([])
  const pollsLoading = ref(false)
  const showCreatePollModal = ref(false)
  const createPollForm = ref<CreatePollData>({ title: '', description: '', options: ['', ''] })
  const reports = ref<CommentReport[]>([])
  const reportsLoading = ref(false)

  const drawerOpen = ref(false)
  const drawerMode = ref<'view' | 'edit'>('view')
  const drawerContent = ref<Content | null>(null)
  const drawerSaving = ref(false)

  const uploadProgress = ref(0)
  const uploading = ref(false)

  // 侧边导航待办角标：独立于各列表分页状态，不随筛选/翻页漂移
  const pendingCounts = reactive({ content: 0, claims: 0, reports: 0 })

  async function loadPendingCounts() {
    try {
      const [c, cl, r] = await Promise.all([
        adminApi.pending({ page: 1, page_size: 1 }),
        adminApi.getClaims({ page: 1, page_size: 1, status: 'pending' }),
        commentApi.getReports(),
      ])
      pendingCounts.content = c.data.total
      pendingCounts.claims = cl.data.total
      pendingCounts.reports = r.data.filter((x) => !x.handled).length
    } catch { /* 角标加载失败静默降级 */ }
  }

  async function loadTags() {
    tagsLoading.value = true
    try {
      const r = await contentApi.getTags()
      tags.value = r.data
    } catch (e: unknown) { toast.error((e as Error).message || '加载失败') } finally { tagsLoading.value = false }
  }

  const loadMyContent = (page = 1) => loadPage(myContent, page, (p) => contentApi.myList({ page: p, page_size: myContent.pageSize }).then((r) => r.data))
  const loadAllContent = (page = 1) => loadPage(allContent, page, (p) => adminApi.getAllContent({ page: p, page_size: allContent.pageSize }).then((r) => r.data))
  const loadPendingContent = (page = 1) => loadPage(pendingContent, page, (p) => adminApi.pending({ page: p, page_size: pendingContent.pageSize }).then((r) => r.data))
  const loadUsers = (page = 1) => loadPage(users, page, (p) => adminApi.getUsers({ page: p, page_size: users.pageSize }).then((r) => r.data))
  const loadClaims = (page = 1, status?: string) => loadPage(claims, page, async (p) => {
    const params = { page: p, page_size: claims.pageSize, ...(status ? { status } : {}) }
    const r = await adminApi.getClaims(params)
    return { ...r.data, total_page: Math.ceil(r.data.total / r.data.page_size) }
  })

  async function loadPolls() {
    pollsLoading.value = true
    try {
      const r = await pollApi.list()
      polls.value = r.data.list
    } catch (e: unknown) { toast.error((e as Error).message || '加载失败') } finally { pollsLoading.value = false }
  }

  async function loadReports() {
    reportsLoading.value = true
    try {
      const r = await commentApi.getReports()
      reports.value = r.data
    } catch (e: unknown) { toast.error((e as Error).message || '加载举报列表失败') } finally { reportsLoading.value = false }
  }

  function openDrawer(content: Content, mode: 'view' | 'edit' = 'view') {
    drawerContent.value = content
    drawerMode.value = mode
    drawerOpen.value = true
  }

  function closeDrawer() {
    drawerOpen.value = false
    drawerContent.value = null
  }

  async function fetchContentDetail(id: number): Promise<Content | null> {
    try {
      const r = await contentApi.detail(id)
      return r.data
    } catch (e: unknown) { toast.error((e as Error).message || '加载失败'); return null }
  }

  async function saveContent(id: number, data: {
    title: string; content: string; url: string; tags: string[]; file?: File
  }): Promise<boolean> {
    drawerSaving.value = true
    try {
      await contentApi.update(id, data)
      toast.success('保存成功')
      return true
    } catch (e: unknown) { toast.error((e as Error).message || '保存失败'); return false }
    finally { drawerSaving.value = false }
  }

  async function createPoll(): Promise<boolean> {
    const valid = createPollForm.value.options.filter((o) => o.trim())
    if (valid.length < 2) { toast.error('至少需要2个有效选项'); return false }
    try {
      await pollApi.create({
        title: createPollForm.value.title.trim(),
        description: (createPollForm.value.description || '').trim(),
        options: valid.map((o) => o.trim()),
      })
      toast.success('投票创建成功')
      showCreatePollModal.value = false
      createPollForm.value = { title: '', description: '', options: ['', ''] }
      return true
    } catch (e: unknown) { toast.error((e as Error).message || '创建失败'); return false }
  }

  function addPollOption() { createPollForm.value.options.push('') }
  function removePollOption(i: number) {
    if (createPollForm.value.options.length > 2) createPollForm.value.options.splice(i, 1)
  }

  // 2026-07-29 改造：上传不再让用户选 type，后端按 file 是否存在自动设 image / text。
  async function uploadContent(data: {
    title: string; content?: string; tags: string[]; file?: File; userId: number
  }): Promise<boolean> {
    uploading.value = true
    uploadProgress.value = 0
    try {
      const res = await contentApi.upload({
        title: data.title,
        content: data.content,
        tags: data.tags,
        user_id: data.userId,
        file: data.file,
      }, (p) => { uploadProgress.value = p })
      toast.success(res.data.audit_status === 'approved' ? '上传成功' : '上传成功，审核通过后将进入推荐')
      return true
    } catch (e: unknown) { toast.error(`上传失败: ${(e as Error).message}`); return false }
    finally { uploading.value = false; uploadProgress.value = 0 }
  }

  return {
    activeTab,
    tags, tagsLoading,
    myContent, allContent, pendingContent, users, claims, polls, pollsLoading, reports, reportsLoading,
    showCreatePollModal, createPollForm,
    drawerOpen, drawerMode, drawerContent, drawerSaving,
    uploadProgress, uploading,
    pendingCounts, loadPendingCounts,
    loadTags,
    loadMyContent, loadAllContent, loadPendingContent, loadUsers, loadClaims, loadPolls, loadReports,
    openDrawer, closeDrawer, fetchContentDetail, saveContent,
    changeAuthor: apiChangeAuthor,
    confirmDelete: apiConfirmDelete,
    auditContent: apiAuditContent,
    updateUserRole: apiUpdateUserRole,
    updateUserBan: apiUpdateUserBan,
    deleteUser: apiDeleteUser,
    handleClaim: apiHandleClaim,
    createPoll, deletePoll: apiDeletePoll, addPollOption, removePollOption,
    handleReport: apiHandleReport,
    deleteComment: apiDeleteComment,
    regenerateThumbnail: apiRegenerateThumbnail,
    regenerateAllThumbnails: apiRegenerateAllThumbnails,
    uploadContent,
  }
})

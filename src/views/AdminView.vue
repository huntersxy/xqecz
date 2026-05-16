<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { contentApi, adminApi, pollApi, commentApi } from '@/api'
import type { Content, User, Poll, CreatePollData, Claim, CommentReport } from '@/types'
import { getPreviewText, renderMarkdown } from '@/utils'

function getImageUrl(image?: string): string {
  if (image) {
    let url = image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
    if (url.startsWith('/')) {
      url = `https://xqapi.xiey.work${url}`
    }
    return url
  }
  return ''
}

import AdminContentCard from '@/components/admin/AdminContentCard.vue'
import AdminUserCard from '@/components/admin/AdminUserCard.vue'
import AdminPagination from '@/components/admin/AdminPagination.vue'
import AdminEmptyState from '@/components/admin/AdminEmptyState.vue'
import AdminContentDetailModal from '@/components/admin/AdminContentDetailModal.vue'
import AdminEditModal from '@/components/admin/AdminEditModal.vue'
import AdminDeleteConfirm from '@/components/admin/AdminDeleteConfirm.vue'
import AdminQuickAddTag from '@/components/admin/AdminQuickAddTag.vue'
import AdminChangeAuthorModal from '@/components/admin/AdminChangeAuthorModal.vue'

const userStore = useUserStore()
const router = useRouter()

const activeTab = ref('my')
const message = ref('')
const agreeUpload = ref(false)
const uploadFormExpanded = ref(true)

const uploadForm = ref({
  title: '',
    type: 'image' as 'video' | 'image' | 'text' | 'link',
  content: '',
  url: '',
  tags: [] as string[],
  file: undefined as File | undefined,
})
const filePreview = ref<string>('')
const uploadProgress = ref(0)
const isUploading = ref(false)

const editForm = ref({
  id: 0,
  title: '',
  content: '',
  url: '',
    type: 'text' as 'video' | 'image' | 'text' | 'link',
  tags: [] as string[],
  file: undefined as File | undefined
})
const isEditModal = ref(false)

const myContents = ref<Content[]>([])
const myContentsPage = ref(1)
const myContentsTotal = ref(0)
const myContentsTotalPages = ref(1)

const pendingContents = ref<Content[]>([])
const pendingContentsPage = ref(1)
const pendingContentsTotal = ref(0)
const pendingContentsTotalPages = ref(1)

const allContents = ref<Content[]>([])
const allContentsPage = ref(1)
const allContentsTotal = ref(0)
const allContentsTotalPages = ref(1)

const users = ref<User[]>([])
const usersPage = ref(1)
const usersTotal = ref(0)
const usersTotalPages = ref(1)

const showQuickAddTag = ref(false)
const quickAddTagName = ref('')
const tagTargetForm = ref<'upload' | 'edit'>('upload')

const showDeleteConfirm = ref(false)
const deleteTargetId = ref(0)

const showContentDetail = ref(false)
const contentDetail = ref<Content | null>(null)

const showChangeAuthorModal = ref(false)
const changeAuthorContentId = ref(0)
const changeAuthorCurrentName = ref('')

const polls = ref<Poll[]>([])
const pollsPage = ref(1)
const pollsTotal = ref(0)
const pollsTotalPages = ref(1)

const claims = ref<Claim[]>([])
const claimsPage = ref(1)
const claimsTotal = ref(0)
const claimsTotalPages = ref(1)
const claimsStatusFilter = ref('')

const reports = ref<CommentReport[]>([])
const showReportDeleteConfirm = ref(false)
const deleteReportTargetId = ref(0)
const deleteReportId = ref(0)

async function loadReports() {
  try {
    const res = await commentApi.getReports()
    if (res.code === 200) {
      reports.value = res.data
    }
  } catch {
    message.value = '加载举报列表失败'
  }
}

async function handleReportAction(reportId: number) {
  try {
    const res = await commentApi.handleReport(reportId)
    if (res.code === 200) {
      message.value = '处理成功'
      await loadReports()
    } else {
      message.value = res.message || '处理失败'
    }
  } catch {
    message.value = '处理失败'
  }
}

function confirmReportDelete(commentId: number, reportId: number) {
  deleteReportTargetId.value = commentId
  deleteReportId.value = reportId
  showReportDeleteConfirm.value = true
}

async function deleteReportComment() {
  const commentId = deleteReportTargetId.value
  const reportId = deleteReportId.value
  
  try {
    const deleteRes = await commentApi.delete(commentId)
    if (deleteRes.code === 200) {
      await commentApi.handleReport(reportId)
      message.value = '删除成功'
      await loadReports()
    } else {
      message.value = deleteRes.message || '删除失败'
    }
  } catch {
    message.value = '删除失败'
  }
  showReportDeleteConfirm.value = false
}

const showCreatePollModal = ref(false)
const createPollForm = ref<CreatePollData>({
  title: '',
  description: '',
  options: ['', '']
})

const allTags = computed(() => {
  const tagsSet = new Set<string>()
  const all = [...myContents.value, ...pendingContents.value, ...allContents.value]
  all.forEach(content => {
    const tags = content.tags || []
    tags.forEach(tag => tagsSet.add(String(tag)))
  })
  return Array.from(tagsSet)
})

const allTagsList = ref<string[]>([])

async function loadAllTags() {
  try {
    const res = await contentApi.getTags()
    if (res.code === 200) {
      allTagsList.value = res.data as string[]
    }
  } catch {
    console.error('加载标签失败')
  }
}

const renderedDetailContent = computed(() => {
  if (!contentDetail.value) return ''
  const text = contentDetail.value.text || ''
  let processedText = text.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '[禁止的脚本]')
  processedText = processedText.replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '[禁止的iframe]')
  return renderMarkdown(processedText)
})

function insertMarkdown(prefix: string, suffix: string = '') {
  const editTextarea = document.querySelector('.edit-textarea') as HTMLTextAreaElement
  const uploadTextarea = document.querySelector('.upload-textarea') as HTMLTextAreaElement
  const textarea = editTextarea && editTextarea.offsetParent !== null ? editTextarea : uploadTextarea
  if (!textarea) return
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const currentForm = isEditModal.value ? editForm.value : uploadForm.value
  const text = currentForm.content || ''
  currentForm.content = text.substring(0, start) + prefix + text.substring(start, end) + suffix + text.substring(end)
  textarea.focus()
  const newPos = start + prefix.length + (end - start) + suffix.length
  setTimeout(() => {
    textarea.setSelectionRange(newPos, newPos)
  }, 0)
}

const imageUploadInput = ref<HTMLInputElement | null>(null)
const uploadFileInput = ref<HTMLInputElement | null>(null)
const editImageUploadInput = ref<HTMLInputElement | null>(null)

function triggerImageUpload() {
  imageUploadInput.value?.click()
}

function triggerEditImageUpload() {
  editImageUploadInput.value?.click()
}

async function handleEditImageUpload(event: Event) {
  const target = event.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    message.value = '只支持 jpg、jpeg、png、gif、webp 格式的图片'
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    message.value = '图片大小不能超过10MB'
    return
  }
  try {
    message.value = '图片上传中...'
    const res = await contentApi.uploadImage(file)
    if (res.code === 200) {
      const textarea = document.querySelector('.edit-textarea') as HTMLTextAreaElement
      if (textarea) {
        const start = textarea.selectionStart
        const end = textarea.selectionEnd
        editForm.value.content = editForm.value.content.substring(0, start) + `![${file.name}](${res.data.image_url})` + editForm.value.content.substring(end)
      } else {
        editForm.value.content += `![${file.name}](${res.data.image_url})`
      }
      message.value = '图片上传成功'
    } else {
      message.value = res.message || '上传失败'
    }
  } catch (error) {
    message.value = '上传失败，请稍后重试'
  } finally {
    target.value = ''
  }
}

async function handleImageUpload(event: Event) {
  const target = event.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    message.value = '只支持 jpg、jpeg、png、gif、webp 格式的图片'
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    message.value = '图片大小不能超过10MB'
    return
  }
  try {
    message.value = '图片上传中...'
    const res = await contentApi.uploadImage(file)
    if (res.code === 200) {
      insertMarkdown(`![${file.name}](${res.data.image_url})`, '')
      message.value = '图片上传成功'
    } else {
      message.value = res.message || '上传失败'
    }
  } catch (error) {
    message.value = '上传失败，请稍后重试'
  } finally {
    target.value = ''
  }
}

function openContentDetail(content: Content) {
  const linkUrl = content.type === 'link' && content.url
  if (linkUrl) {
    window.open(linkUrl, '_blank')
    return
  }
  contentDetail.value = content
  showContentDetail.value = true
}

function openChangeAuthorModal(content: Content) {
  changeAuthorContentId.value = content.id || 0
  changeAuthorCurrentName.value = content.user?.username || ''
  showChangeAuthorModal.value = true
}

function handleChangeAuthorSuccess() {
  if (activeTab.value === 'my') loadMyContents()
  if (activeTab.value === 'all') loadAllContents()
  if (activeTab.value === 'pending') loadPendingContents()
}

function addTag(target: 'upload' | 'edit' = 'upload') {
  tagTargetForm.value = target
  showQuickAddTag.value = true
}

function handleQuickAddTag() {
  if (!quickAddTagName.value.trim()) {
    message.value = '请输入标签名称'
    return
  }
  const targetForm = tagTargetForm.value === 'upload' ? uploadForm.value : editForm.value
  if (targetForm.tags.includes(quickAddTagName.value.trim())) {
    message.value = '标签已存在'
    return
  }
  targetForm.tags.push(quickAddTagName.value.trim())
  quickAddTagName.value = ''
  showQuickAddTag.value = false
}

function removeTag(tag: string) {
  const index = uploadForm.value.tags.indexOf(tag)
  if (index > -1) {
    uploadForm.value.tags.splice(index, 1)
  }
}

function removeEditTag(tag: string) {
  const index = editForm.value.tags.indexOf(tag)
  if (index > -1) {
    editForm.value.tags.splice(index, 1)
  }
}

function toggleTag(tag: string) {
  const index = uploadForm.value.tags.indexOf(tag)
  if (index > -1) {
    uploadForm.value.tags.splice(index, 1)
  } else {
    uploadForm.value.tags.push(tag)
  }
}

function toggleEditTag(tag: string) {
  const index = editForm.value.tags.indexOf(tag)
  if (index > -1) {
    editForm.value.tags.splice(index, 1)
  } else {
    editForm.value.tags.push(tag)
  }
}

async function loadMyContents() {
  try {
    const res = await contentApi.myList({ page: myContentsPage.value, page_size: 12 })
    if (res.code === 200) {
      myContents.value = res.data.list
      myContentsTotal.value = res.data.total
      myContentsTotalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载我的内容失败', error)
  }
}

async function loadPendingContents() {
  try {
    const res = await adminApi.pending({ page: pendingContentsPage.value, page_size: 12 })
    if (res.code === 200) {
      pendingContents.value = res.data.list
      pendingContentsTotal.value = res.data.total
      pendingContentsTotalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载待审核内容失败', error)
  }
}

async function loadAllContents() {
  try {
    const res = await adminApi.getAllContent({ page: allContentsPage.value, page_size: 12 })
    if (res.code === 200) {
      allContents.value = res.data.list
      allContentsTotal.value = res.data.total
      allContentsTotalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载所有内容失败', error)
  }
}

async function loadUsers() {
  try {
    const res = await adminApi.getUsers({ page: usersPage.value, page_size: 24 })
    if (res.code === 200) {
      users.value = res.data.list
      usersTotal.value = res.data.total
      usersTotalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载用户列表失败', error)
  }
}

async function loadPolls() {
  try {
    const res = await pollApi.list()
    if (res.code === 200) {
      polls.value = res.data.list
      pollsTotal.value = res.data.list.length
      pollsTotalPages.value = 1
    }
  } catch (error) {
    console.error('加载投票列表失败', error)
  }
}

async function loadClaims() {
  try {
    const params: { page: number; page_size: number; status?: string } = { page: claimsPage.value, page_size: 12 }
    if (claimsStatusFilter.value) params.status = claimsStatusFilter.value
    const res = await adminApi.getClaims(params)
    if (res.code === 200) {
      claims.value = res.data.list
      claimsTotal.value = res.data.total
      claimsTotalPages.value = Math.ceil(res.data.total / res.data.page_size)
    }
  } catch (error) {
    console.error('加载认领列表失败', error)
  }
}

async function handleClaim(claimId: number, action: 'approve' | 'reject') {
  const reason = action === 'reject' ? prompt('请输入拒绝原因（可选）：') : ''
  if (action === 'reject' && reason === null) return
  try {
    const res = await adminApi.handleClaim(claimId, action, reason || undefined)
    if (res.code === 200) {
      message.value = action === 'approve' ? '认领已通过' : '认领已拒绝'
      loadClaims()
    } else {
      message.value = res.message || '操作失败'
    }
  } catch (error) {
    message.value = '操作失败'
  }
}

async function handleCreatePoll() {
  if (!createPollForm.value.title.trim()) {
    message.value = '请输入投票标题'
    return
  }
  const validOptions = createPollForm.value.options.filter(opt => opt.trim())
  if (validOptions.length < 2) {
    message.value = '至少需要2个有效选项'
    return
  }
  try {
    const data: CreatePollData = {
      title: createPollForm.value.title.trim(),
      description: createPollForm.value.description.trim(),
      options: validOptions.map(opt => opt.trim())
    }
    const res = await pollApi.create(data)
    if (res.code === 200) {
      message.value = '投票创建成功'
      showCreatePollModal.value = false
      createPollForm.value = {
        title: '',
        description: '',
        options: ['', '']
      }
      loadPolls()
    } else {
      message.value = res.message || '创建失败'
    }
  } catch (error) {
    message.value = '创建失败'
    console.error('创建投票失败', error)
  }
}

async function handleDeletePoll(id: number) {
  if (!confirm('确定删除该投票？')) return
  try {
    const res = await pollApi.delete(id)
    if (res.code === 200) {
      message.value = '删除成功'
      loadPolls()
    } else {
      message.value = res.message || '删除失败'
    }
  } catch (error) {
    message.value = '删除失败'
    console.error('删除投票失败', error)
  }
}

function addPollOption() {
  createPollForm.value.options.push('')
}

function removePollOption(index: number) {
  if (createPollForm.value.options.length > 2) {
    createPollForm.value.options.splice(index, 1)
  }
}

async function handleUpload() {
  if (!userStore.user) {
    message.value = '请先登录'
    return
  }
  try {
    const userId = userStore.user.id
    if (!uploadForm.value.title && uploadForm.value.type !== 'link') {
      message.value = '请填写标题'
      return
    }
    isUploading.value = true
    uploadProgress.value = 0
    const res = await contentApi.upload({ ...uploadForm.value, user_id: userId }, (percent) => {
      uploadProgress.value = percent
    })
    if (res.code === 200) {
      message.value = '上传成功'
      uploadForm.value = { title: '', type: 'image', content: '', url: '', tags: [], file: undefined }
      filePreview.value = ''
      agreeUpload.value = false
      loadMyContents()
    } else {
      message.value = res.message || '上传失败'
    }
  } catch (error: any) {
    message.value = `上传失败: ${error.message}`
  } finally {
    isUploading.value = false
    uploadProgress.value = 0
  }
}

function openEditModal(content: Content) {
  editForm.value = {
    id: content.id,
    title: content.title,
    content: content.text,
    url: content.url || '',
    type: content.type as 'video' | 'image' | 'text' | 'link',
    tags: content.tags || [],
    file: undefined
  }
  isEditModal.value = true
}

async function handleUpdate() {
  try {
    const res = await contentApi.update(editForm.value.id, {
      title: editForm.value.title,
      content: editForm.value.content,
      url: editForm.value.url,
      tags: editForm.value.tags,
      file: editForm.value.file
    })
    if (res.code === 200) {
      message.value = '更新成功'
      isEditModal.value = false
      if (activeTab.value === 'my') loadMyContents()
      if (activeTab.value === 'all') loadAllContents()
      if (activeTab.value === 'pending') loadPendingContents()
    } else {
      message.value = res.message
    }
  } catch (error) {
    message.value = '更新失败'
  }
}

function confirmDelete(id: number) {
  deleteTargetId.value = id
  showDeleteConfirm.value = true
}

async function handleDelete() {
  const id = deleteTargetId.value
  try {
    const res = await contentApi.delete(id)
    if (res.code === 200) {
      message.value = '删除成功'
      if (activeTab.value === 'my') loadMyContents()
      if (activeTab.value === 'all') loadAllContents()
    }
  } catch (error) {
    message.value = '删除失败'
  }
  showDeleteConfirm.value = false
}

async function regenerateThumbnail(id: number) {
  try {
    const res = await adminApi.regenerateThumbnail(id)
    if (res.code === 200) {
      message.value = '封面更新成功'
      if (activeTab.value === 'my') loadMyContents()
      if (activeTab.value === 'all') loadAllContents()
    } else {
      message.value = res.message || '封面更新失败'
    }
  } catch (error) {
    message.value = '封面更新失败'
  }
}

async function regenerateAllThumbnails() {
  try {
    const res = await adminApi.regenerateAllThumbnails()
    if (res.code === 200) {
      message.value = `已开始处理 ${res.data.count} 条内容的缩略图生成`
    } else {
      message.value = res.message || '操作失败'
    }
  } catch (error) {
    message.value = '操作失败'
  }
}

async function handleAudit(id: number, status: 'approved' | 'rejected') {
  if (!userStore.user) return
  try {
    const adminId = userStore.user.id
    const res = await adminApi.audit(id, { admin_id: adminId, status, remark: '' })
    if (res.code === 200) {
      message.value = '审核成功'
      loadPendingContents()
    }
  } catch (error) {
    message.value = '审核失败'
  }
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    uploadForm.value.file = target.files[0]
    const file = target.files[0]
    const reader = new FileReader()
    reader.onload = (e) => {
      filePreview.value = e.target?.result as string
    }
    reader.readAsDataURL(file)
  }
}

function handleEditFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    editForm.value.file = target.files[0]
  }
}

function clearFilePreview() {
  filePreview.value = ''
  uploadForm.value.file = undefined
  const fileInput = document.querySelector('.upload-file-input') as HTMLInputElement
  if (fileInput) fileInput.value = ''
}

async function handleUpdateUserRole(id: number, isAdmin: boolean) {
  if (!confirm(isAdmin ? '确定设为管理员？' : '确定取消管理员？')) return
  try {
    const res = await adminApi.updateUserRole(id, isAdmin)
    if (res.code === 200) {
      message.value = '更新成功'
      loadUsers()
    }
  } catch (error) {
    message.value = '更新失败'
  }
}

async function handleUpdateUserBan(id: number, isBanned: boolean) {
  if (!confirm(isBanned ? '确定封禁该用户？' : '确定解封该用户？')) return
  try {
    const res = await adminApi.updateUserBan(id, isBanned)
    if (res.code === 200) {
      message.value = isBanned ? '封禁成功' : '解封成功'
      loadUsers()
    }
  } catch (error) {
    message.value = isBanned ? '封禁失败' : '解封失败'
  }
}

async function handleDeleteUser(id: number) {
  if (!confirm('确定删除？')) return
  try {
    const res = await adminApi.deleteUser(id)
    if (res.code === 200) {
      message.value = '删除成功'
      loadUsers()
    }
  } catch (error) {
    message.value = '删除失败'
  }
}

function onTabChange(tab: string) {
  activeTab.value = tab
  if (tab === 'my') loadMyContents()
  if (tab === 'pending') loadPendingContents()
  if (tab === 'all') loadAllContents()
  if (tab === 'users') loadUsers()
  if (tab === 'polls') loadPolls()
  if (tab === 'claims') loadClaims()
  if (tab === 'reports') loadReports()
}

onMounted(() => {
  if (userStore.isLoggedIn) {
    loadMyContents()
  }
  loadAllTags()
})
</script>

<template>
  <div class="min-h-screen p-2 sm:p-4 md:p-5 lg:p-6 flex justify-center bg-transparent">
    <div class="w-full max-w-6xl bg-[var(--theme-surface)] rounded-xl shadow-lg shadow-black/5 border border-[var(--theme-card-border)] overflow-hidden">
      <div class="flex items-center px-3 sm:px-4 py-2 sm:py-2.5 theme-header-bg">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm theme-text-secondary font-medium">小泉动漫 - 后台管理</div>
      </div>

      <div class="p-3 sm:p-4 md:p-6">
        <div v-if="message" class="px-3 sm:px-4 py-2 sm:py-3 rounded-lg mb-3 sm:mb-4 flex justify-between items-center" :class="message.includes('成功') ? 'bg-emerald-50/80 text-[var(--theme-success)]' : 'bg-[var(--theme-danger)]/5/80 text-[var(--theme-danger)]'">
          <span class="text-xs sm:text-sm">{{ message }}</span>
          <span class="text-sm font-bold cursor-pointer hover:opacity-70" @click="message = ''">×</span>
        </div>

        <div class="flex flex-wrap gap-1.5 sm:gap-2 mb-3 sm:mb-4 pb-2 sm:pb-3 border-b border-[var(--theme-card-border)]">
          <button
            v-for="tab in [{key:'my',label:'我的内容'},{key:'pending',label:'审核内容'},{key:'all',label:'所有内容'},{key:'users',label:'用户管理'},{key:'polls',label:'投票管理'},{key:'claims',label:'认领管理'},{key:'reports',label:'举报管理'}]"
            :key="tab.key"
            v-show="tab.key === 'my' || userStore.user?.is_admin"
            @click="onTabChange(tab.key)"
            :class="[
              'px-2.5 sm:px-4 py-1.5 sm:py-2 text-xs sm:text-sm rounded-md transition-all',
              activeTab === tab.key 
                ? 'bg-[var(--theme-primary)]/10 text-[var(--theme-primary)] font-medium border border-[var(--theme-primary)]/30' 
                : 'bg-[var(--theme-surface)] theme-text-secondary border border-[var(--theme-card-border)] hover:bg-[var(--theme-primary)]/8 hover:text-[var(--theme-primary)] hover:border-[var(--theme-primary)]/30'
            ]"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-if="activeTab === 'my'" class="space-y-3 sm:space-y-4">
          <input
            ref="imageUploadInput"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            style="display: none"
            @change="handleImageUpload"
          />
          <div class="mb-4 bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-xl overflow-hidden">
            <div class="flex justify-between items-center px-5 py-4 bg-gradient-to-b from-black/5 to-transparent border-b border-black/5 cursor-pointer" @click="uploadFormExpanded = !uploadFormExpanded">
              <h2 class="text-base font-semibold theme-text m-0">上传内容</h2>
              <span class="text-2xl theme-text-secondary leading-none transition-transform" :class="{ 'rotate-180': uploadFormExpanded }">▼</span>
            </div>
            <div v-if="uploadFormExpanded" class="p-5">
              <div class="mb-4">
                <label class="block text-[13px] font-medium theme-text mb-2">标题</label>
                <input v-model="uploadForm.title" type="text" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10" placeholder="输入标题" />
              </div>
              <div class="mb-4">
                <label class="block text-[13px] font-medium theme-text mb-2">类型</label>
                <div class="flex gap-4">
                  <label class="flex items-center gap-1.5 cursor-pointer text-sm theme-text">
                    <input type="radio" value="text" v-model="uploadForm.type" class="cursor-pointer" />
                    <span>文字</span>
                  </label>
                  <label class="flex items-center gap-1.5 cursor-pointer text-sm theme-text">
                    <input type="radio" value="image" v-model="uploadForm.type" class="cursor-pointer" />
                    <span>图片</span>
                  </label>
                  <label class="flex items-center gap-1.5 cursor-pointer text-sm theme-text">
                    <input type="radio" value="video" v-model="uploadForm.type" class="cursor-pointer" />
                    <span>视频</span>
                  </label>
                  <label class="flex items-center gap-1.5 cursor-pointer text-sm theme-text">
                    <input type="radio" value="link" v-model="uploadForm.type" class="cursor-pointer" />
                    <span>链接</span>
                  </label>
                </div>
              </div>
              <div v-if="uploadForm.type === 'text'" class="mb-4">
                <label class="block text-[13px] font-medium theme-text mb-2">内容</label>
                <div class="flex gap-1 mb-2 p-1.5 bg-[var(--theme-hover-bg)] rounded-md">
                  <button type="button" @click="insertMarkdown('## ')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="标题">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M4 12h16M4 6h16M4 18h10" />
                    </svg>
                  </button>
                  <button type="button" @click="insertMarkdown('**', '**')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="粗体">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                      <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                    </svg>
                  </button>
                  <button type="button" @click="insertMarkdown('*', '*')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="斜体">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="19" y1="4" x2="10" y2="4" />
                      <line x1="14" y1="20" x2="5" y2="20" />
                      <line x1="15" y1="4" x2="9" y2="20" />
                    </svg>
                  </button>
                  <button type="button" @click="triggerImageUpload" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="上传图片">
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                      <circle cx="8.5" cy="8.5" r="1.5" />
                      <polyline points="21 15 16 10 5 21" />
                    </svg>
                  </button>
                </div>
                <textarea v-model="uploadForm.content" class="w-full min-h-[200px] resize-y font-mono text-sm leading-relaxed box-border bg-[var(--theme-surface)] border border-gray-200 rounded-lg p-3 focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10 upload-textarea" placeholder="支持Markdown"></textarea>
              </div>
              <div v-else class="mb-4">
                <label class="block text-[13px] font-medium theme-text mb-2">文件</label>
                <input type="file" ref="uploadFileInput" @change="handleFileChange" class="hidden" :accept="uploadForm.type === 'image' ? 'image/*' : 'video/*'" />
                <div 
                  @click="uploadFileInput?.click()" 
                  class="w-full p-6 border-2 border-dashed border-[var(--theme-card-border)] rounded-lg bg-[var(--theme-surface)] cursor-pointer hover:border-[var(--theme-primary)]/50 hover:bg-[var(--theme-hover-bg)] transition-all flex flex-col items-center justify-center gap-2"
                >
                  <svg class="w-8 h-8 theme-text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                    <polyline points="17 8 12 3 7 8" />
                    <line x1="12" y1="3" x2="12" y2="15" />
                  </svg>
                  <span class="text-sm theme-text-secondary">{{ uploadForm.file ? uploadForm.file.name : '点击或拖拽文件至此处' }}</span>
                </div>
                <div v-if="filePreview" class="relative mt-3 rounded-lg overflow-hidden">
                  <img v-if="uploadForm.type === 'image'" :src="filePreview" class="w-full max-h-[300px] object-contain" />
                  <video v-else-if="uploadForm.type === 'video'" :src="filePreview" class="w-full max-h-[300px]" controls />
                  <button type="button" @click="filePreview = ''; uploadForm.file = undefined; if (uploadFileInput) uploadFileInput.value = ''" class="absolute top-2 right-2 w-7 h-7 bg-black/50 text-white rounded-full cursor-pointer text-lg flex items-center justify-center hover:bg-black/70 transition-colors">×</button>
                </div>
              </div>
              <div v-if="uploadForm.type === 'video'" class="border-t border-gray-200 pt-2.5 px-4 pb-0">
                <label class="flex gap-2 text-xs theme-text-secondary leading-relaxed cursor-pointer items-start hover:theme-text">
                  <input type="checkbox" v-model="agreeUpload" class="mt-0.5 shrink-0 cursor-pointer" />
                  <span>自2026年5月14日起，本平台建议使用「链接」类型来代替视频类型，您可以在链接类型中填入您的B站视频链接。若您同意，但您继续上传大于15MB的视频，则视为委托站长通过二创站官方B站账号代为上传，原视频将替换为该B站链接。该官方账号不会产生任何收益，亦不会用于上传其他内容。勾选即表示您已阅读并同意上述条款。</span>
                </label>
              </div>
              <div class="border-t border-gray-200 pt-1 pb-0 mt-4">
                <div class="text-xs theme-text-secondary leading-relaxed">上传至本站的内容如未另行标注版权信息，默认采用<a href="https://creativecommons.org/licenses/by/4.0/deed.zh-hans" target="_blank" rel="noopener" class="text-[var(--theme-primary)] no-underline hover:underline">知识共享署名 4.0 国际许可协议（CC BY 4.0）</a>进行授权。建议您在作品中添加水印或署名以保障自身权益。</div>
              </div>
              <div v-if="uploadForm.type === 'link'" class="mb-4 mt-4">
                <label class="block text-[13px] font-medium theme-text mb-2">链接地址</label>
                <input v-model="uploadForm.url" type="url" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10" placeholder="https://..." />
              </div>
              <div class="mb-4">
                <label class="block text-[13px] font-medium theme-text mb-2">标签</label>
                <div class="mt-2 space-y-3">
                  <div v-if="uploadForm.tags.length > 0">
                    <div class="text-xs theme-text-secondary mb-1.5">已选择</div>
                    <div class="flex flex-wrap gap-2">
                      <span v-for="tag in uploadForm.tags" :key="tag"
                        class="px-3 py-1 bg-[var(--theme-primary)]/10 border border-blue-300 text-[var(--theme-primary)] cursor-pointer transition-all text-sm rounded-full hover:border-blue-400"
                        @click="toggleTag(tag)">{{ tag }} <span class="ml-1 text-xs">×</span></span>
                    </div>
                  </div>
                  <div>
                    <div class="text-xs theme-text-secondary mb-1.5">全部标签</div>
                    <div class="flex flex-wrap gap-2">
                      <span v-for="tag in allTagsList" :key="tag"
                        :class="['px-3 py-1 bg-[var(--theme-surface)] border cursor-pointer transition-all text-sm rounded-full', uploadForm.tags.includes(tag) ? 'bg-[var(--theme-primary)]/10/50 border-blue-300 text-[var(--theme-primary)] hover:border-blue-400' : 'border-black/20 theme-text hover:border-blue-300 hover:text-[var(--theme-primary)]']"
                        @click="toggleTag(tag)">{{ tag }}</span>
                      <button type="button" @click="tagTargetForm = 'upload'; showQuickAddTag = true" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border-dashed border-black/20 rounded-md theme-text-secondary hover:border-blue-300 hover:text-[var(--theme-primary)] transition-all" title="添加标签">
                        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <line x1="12" y1="5" x2="12" y2="19" />
                          <line x1="5" y1="12" x2="19" y2="12" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
              <div v-if="isUploading" class="mb-4">
                <div class="relative w-full h-5 bg-gray-200 rounded-full overflow-hidden">
                  <div class="h-full bg-gradient-to-r from-blue-500 to-purple-500 rounded-full transition-all duration-300" :style="{ width: `${uploadProgress}%` }"></div>
                  <span class="absolute inset-0 flex items-center justify-center text-xs font-medium text-white drop-shadow-md">{{ uploadProgress }}%</span>
                </div>
              </div>
              <div class="flex justify-end gap-3 px-0 py-2">
                <button @click="uploadForm = { title: '', type: 'image', content: '', url: '', tags: [], file: undefined }; filePreview = ''; agreeUpload = false" class="px-4 py-2 bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-lg text-sm theme-text hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isUploading">清空</button>
                <button @click="handleUpload" class="px-4 py-2 bg-[var(--theme-primary)] border border-[var(--theme-primary)] rounded-lg text-sm text-white hover:bg-[var(--theme-primary)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isUploading || (uploadForm.type === 'video' && uploadForm.file && uploadForm.file.size > 15 * 1024 * 1024 && !agreeUpload)">
                  {{ isUploading ? '上传中...' : '提交' }}
                </button>
              </div>
            </div>
          </div>
          
          <div class="flex items-center justify-between">
            <h2 class="text-base sm:text-lg font-semibold theme-text">我的内容</h2>
            <span class="text-xs sm:text-sm theme-text-secondary">共 {{ myContentsTotal }} 条</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 sm:gap-3">
            <AdminContentCard
              v-for="content in myContents"
              :key="content.id"
              :content="content"
              :show-actions="true"
              :show-regenerate-thumbnail="true"
              @view="openContentDetail"
              @edit="openEditModal"
              @delete="confirmDelete"
              @regenerate-thumbnail="regenerateThumbnail"
            />
          </div>

          <AdminEmptyState v-if="myContents.length === 0" text="暂无内容" />

          <AdminPagination
            v-if="myContentsTotalPages > 1"
            :current-page="myContentsPage"
            :total-pages="myContentsTotalPages"
            @change="(page: number) => { myContentsPage = page; loadMyContents() }"
          />
        </div>

        <div v-if="activeTab === 'pending'" class="space-y-3 sm:space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-base sm:text-lg font-semibold theme-text">待审核内容</h2>
            <span class="text-xs sm:text-sm theme-text-secondary">共 {{ pendingContentsTotal }} 条</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 sm:gap-3">
            <AdminContentCard
              v-for="content in pendingContents"
              :key="content.id"
              :content="content"
              :show-audit-actions="true"
              :show-author="true"
              @view="openContentDetail"
              @audit="handleAudit"
            />
          </div>

          <AdminEmptyState v-if="pendingContents.length === 0" text="暂无待审核内容" />

          <AdminPagination
            v-if="pendingContentsTotalPages > 1"
            :current-page="pendingContentsPage"
            :total-pages="pendingContentsTotalPages"
            @change="(page: number) => { pendingContentsPage = page; loadPendingContents() }"
          />
        </div>

        <div v-if="activeTab === 'all'" class="space-y-3 sm:space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-base sm:text-lg font-semibold theme-text">所有内容</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm theme-text-secondary">共 {{ allContentsTotal }} 条</span>
              <button @click="regenerateAllThumbnails" class="flex items-center gap-1.5 px-2.5 sm:px-4 py-1.5 sm:py-2 bg-[var(--theme-surface)] theme-text-secondary border border-[var(--theme-card-border)] rounded-md text-xs sm:text-sm hover:bg-[var(--theme-primary)]/8 hover:text-[var(--theme-primary)] hover:border-[var(--theme-primary)]/30 transition-all">
                <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
                  <path d="M3 3v5h5"/>
                  <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
                  <path d="M16 21h5v-5"/>
                </svg>
                批量生成缩略图
              </button>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-2 sm:gap-3">
            <AdminContentCard
              v-for="content in allContents"
              :key="content.id"
              :content="content"
              :show-actions="true"
              :show-author="true"
              :show-regenerate-thumbnail="true"
              :show-change-author="true"
              @view="openContentDetail"
              @edit="openEditModal"
              @delete="confirmDelete"
              @regenerate-thumbnail="regenerateThumbnail"
              @change-author="openChangeAuthorModal"
            />
          </div>

          <AdminEmptyState v-if="allContents.length === 0" text="暂无内容" />

          <AdminPagination
            v-if="allContentsTotalPages > 1"
            :current-page="allContentsPage"
            :total-pages="allContentsTotalPages"
            @change="(page: number) => { allContentsPage = page; loadAllContents() }"
          />
        </div>

        <div v-if="activeTab === 'users'" class="space-y-3 sm:space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-base sm:text-lg font-semibold theme-text">用户管理</h2>
            <span class="text-xs sm:text-sm theme-text-secondary">共 {{ usersTotal }} 条</span>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 sm:gap-3">
            <AdminUserCard
              v-for="user in users"
              :key="user.id"
              :user="user"
              @update-role="handleUpdateUserRole"
              @update-ban="handleUpdateUserBan"
              @delete="handleDeleteUser"
            />
          </div>

          <AdminEmptyState v-if="users.length === 0" text="暂无用户" type="user" />

          <AdminPagination
            v-if="usersTotalPages > 1"
            :current-page="usersPage"
            :total-pages="usersTotalPages"
            @change="(page: number) => { usersPage = page; loadUsers() }"
          />
        </div>

        <div v-if="activeTab === 'polls'" class="space-y-3 sm:space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-base sm:text-lg font-semibold theme-text">投票管理</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm theme-text-secondary">共 {{ pollsTotal }} 条</span>
              <button @click="showCreatePollModal = true" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-[var(--theme-primary)]/95 text-white border border-[var(--theme-primary)]/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">
                创建投票
              </button>
            </div>
          </div>

          <div class="space-y-2 sm:space-y-3">
            <div
              v-for="poll in polls"
              :key="poll.id"
              class="bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-lg p-3 sm:p-4"
            >
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div class="flex-1">
                  <div class="text-base sm:text-lg font-semibold theme-text mb-1">{{ poll.title }}</div>
                  <div class="flex flex-wrap gap-2 text-xs sm:text-sm theme-text-secondary">
                    <span>{{ poll.vote_count }} 票</span>
                    <span>{{ new Date(poll.created_at).toLocaleString() }}</span>
                  </div>
                </div>
                <button
                  @click="handleDeletePoll(poll.id)"
                  class="px-2 sm:px-3 py-1 sm:py-1.5 bg-red-500/20 text-red-500 border border-red-500/50 rounded-md text-xs sm:text-sm hover:bg-red-500/30 hover:border-red-500/70 transition-all shrink-0"
                >
                  删除
                </button>
              </div>
              <div v-if="poll.description" class="text-xs sm:text-sm theme-text-secondary mt-2 sm:mt-3 leading-relaxed">
                {{ poll.description }}
              </div>
              <div class="flex flex-wrap gap-1.5 sm:gap-2 mt-2 sm:mt-3">
                <span v-for="(option, index) in poll.options" :key="index" class="px-2 sm:px-2.5 py-0.5 sm:py-1 bg-[var(--theme-primary)]/10 border border-[var(--theme-primary)]/30 rounded-full text-xs sm:text-sm text-[var(--theme-primary)]">
                  {{ option }}
                </span>
              </div>
            </div>
          </div>

          <AdminEmptyState v-if="polls.length === 0" text="暂无投票" />

          <AdminPagination
            v-if="pollsTotalPages > 1"
            :current-page="pollsPage"
            :total-pages="pollsTotalPages"
            @change="(page: number) => { pollsPage = page; loadPolls() }"
          />
        </div>

        <div v-if="activeTab === 'claims'" class="space-y-3 sm:space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-base sm:text-lg font-semibold theme-text">认领管理</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm theme-text-secondary">共 {{ claimsTotal }} 条</span>
              <select v-model="claimsStatusFilter" @change="claimsPage = 1; loadClaims()" class="px-2.5 sm:px-3 py-1.5 sm:py-2 border border-[var(--theme-card-border)] rounded-md text-xs sm:text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10">
                <option value="">全部状态</option>
                <option value="pending">待处理</option>
                <option value="approved">已通过</option>
                <option value="rejected">已拒绝</option>
              </select>
            </div>
          </div>

          <div class="space-y-2 sm:space-y-3">
            <div
              v-for="claim in claims"
              :key="claim.id"
              class="bg-[var(--theme-card-bg)] border border-[var(--theme-card-border)] rounded-xl p-3 sm:p-4 shadow-md shadow-black/5"
            >
              <div class="flex flex-col sm:flex-row gap-3 sm:gap-4">
                <div class="w-full sm:w-48 h-28 sm:h-36 bg-[var(--theme-hover-bg)] rounded-lg overflow-hidden shrink-0">
                  <template v-if="claim.content.type !== 'text' && claim.content.type !== 'link'">
                    <img
                      :src="getImageUrl(claim.content.thumb)"
                      :alt="claim.content.type === 'video' ? '视频封面' : '内容图片'"
                      class="w-full h-full object-cover"
                      loading="lazy"
                    />
                  </template>
                  <template v-else>
                    <div class="w-full h-full p-2 text-xs theme-text-secondary overflow-hidden flex items-center justify-center">
                      {{ getPreviewText(claim.content.text) }}
                    </div>
                  </template>
                </div>
                <div class="flex-1 space-y-2">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <span class="text-base sm:text-lg font-semibold theme-text">{{ claim.content?.title || '未知内容' }}</span>
                    <span :class="[
                      'px-2 sm:px-3 py-1 rounded-full text-xs font-medium',
                      claim.status === 'pending' ? 'bg-amber-500/10 text-[var(--theme-warning)] border border-amber-500/30' :
                      claim.status === 'approved' ? 'bg-green-500/10 text-green-600 border border-green-500/30' :
                      'bg-[var(--theme-danger)]/50/10 text-[var(--theme-danger)] border border-red-500/30'
                    ]">
                      {{ claim.status === 'pending' ? '待处理' : claim.status === 'approved' ? '已通过' : '已拒绝' }}
                    </span>
                  </div>
                  <div class="text-xs sm:text-sm theme-text-secondary space-y-1">
                    <div><span class="theme-text-secondary">认领者：</span>{{ claim.user?.username || '未知用户' }}</div>
                    <div><span class="theme-text-secondary">认领理由：</span>{{ claim.reason }}</div>
                    <div v-if="claim.remark"><span class="theme-text-secondary">备注：</span>{{ claim.remark }}</div>
                    <div><span class="theme-text-secondary">提交时间：</span>{{ new Date(claim.created_at).toLocaleString() }}</div>
                  </div>
                  <div v-if="claim.status === 'pending'" class="flex gap-2 mt-2">
                    <button @click="handleClaim(claim.id, 'approve')" class="flex-1 px-3 py-1.5 bg-[var(--theme-primary)]/95 text-white border border-[var(--theme-primary)]/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">通过</button>
                    <button @click="handleClaim(claim.id, 'reject')" class="flex-1 px-3 py-1.5 bg-[var(--theme-danger)]/50/95 text-white border border-red-500/95 rounded-md text-xs sm:text-sm hover:bg-red-700/95 hover:border-red-700/95 transition-all">拒绝</button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <AdminEmptyState v-if="claims.length === 0" text="暂无认领申请" />

          <AdminPagination
            v-if="claimsTotalPages > 1"
            :current-page="claimsPage"
            :total-pages="claimsTotalPages"
            @change="(page: number) => { claimsPage = page; loadClaims() }"
          />
        </div>

        <div v-if="activeTab === 'reports'" class="space-y-3 sm:space-y-4">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="text-base sm:text-lg font-semibold theme-text">举报列表</h2>
            <span class="text-xs sm:text-sm theme-text-secondary">共 {{ reports.length }} 条</span>
          </div>

          <div v-if="reports.length > 0" class="space-y-2 sm:space-y-3">
            <div v-for="report in reports" :key="report.id" class="bg-[var(--theme-card-bg)] border border-[var(--theme-card-border)] rounded-xl p-3 sm:p-4 shadow-md shadow-black/5">
              <div class="flex flex-wrap items-center gap-2 sm:gap-3 mb-3 sm:mb-4">
                <span class="font-semibold text-[var(--theme-primary)] text-sm">#{{ report.id }}</span>
                <span class="text-xs sm:text-sm theme-text-secondary">{{ report.created_at }}</span>
                <span :class="[report.handled ? 'bg-[var(--theme-success)]/10 text-[var(--theme-success)]' : 'bg-[var(--theme-warning)]/10 text-[var(--theme-warning)]']" class="px-2 py-0.5 sm:px-3 sm:py-1 rounded-full text-xs font-medium">
                  {{ report.handled ? '已处理' : '待处理' }}
                </span>
              </div>
              
              <div class="pb-3 sm:pb-4 border-b border-[var(--theme-card-border)] mb-3 sm:mb-4">
                <p class="text-xs sm:text-sm theme-text-secondary mb-1">
                  <span class="font-medium theme-text-secondary">举报原因：</span>
                  {{ report.reason || '其他' }}
                </p>
                <p class="text-xs sm:text-sm theme-text-secondary italic mb-1">
                  <span class="font-medium theme-text-secondary">被举报内容：</span>
                  {{ report.Comment?.text }}
                </p>
                <p class="text-xs sm:text-sm theme-text-secondary">
                  <span class="font-medium theme-text-secondary">举报人：</span>
                  {{ report.User?.username }}
                </p>
              </div>

              <div class="flex flex-col sm:flex-row justify-end gap-2 sm:gap-3">
                <button 
                  v-if="!report.handled" 
                  @click="handleReportAction(report.id)" 
                  class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-[var(--theme-primary)] text-white rounded-lg text-sm font-medium hover:brightness-90 transition-all"
                >
                  标记已处理
                </button>
                <button 
                  @click="confirmReportDelete(report.comment_id, report.id)" 
                  class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-red-500/20 text-red-500 border border-red-500/50 rounded-lg text-sm font-medium hover:bg-red-500/30 transition-all"
                >
                  删除评论
                </button>
              </div>
            </div>
          </div>

          <AdminEmptyState v-else text="暂无举报" />
        </div>
      </div>
    </div>

    <input
      ref="editImageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleEditImageUpload"
    />
    

    
    <AdminEditModal
      :visible="isEditModal"
      :edit-form="editForm"
      :all-tags="allTagsList"
      @close="isEditModal = false"
      @save="handleUpdate"
      @insert-markdown="insertMarkdown"
      @trigger-image-upload="triggerEditImageUpload"
      @handle-image-upload="handleEditImageUpload"
      @handle-file-change="handleEditFileChange"
      @add-tag="addTag"
      @remove-tag="removeEditTag"
      @toggle-tag="toggleEditTag"
    />

    <AdminContentDetailModal
      v-if="contentDetail"
      :content="contentDetail"
      :visible="showContentDetail"
      @close="showContentDetail = false"
    />

    <AdminChangeAuthorModal
      :visible="showChangeAuthorModal"
      :content-id="changeAuthorContentId"
      :current-author-name="changeAuthorCurrentName"
      @close="showChangeAuthorModal = false"
      @success="handleChangeAuthorSuccess"
    />

    <Teleport to="body">
      <AdminQuickAddTag
        :visible="showQuickAddTag"
        :target="tagTargetForm"
        v-model:new-tag="quickAddTagName"
        @close="showQuickAddTag = false"
        @add="handleQuickAddTag"
      />

      <AdminDeleteConfirm
        :visible="showDeleteConfirm"
        @close="showDeleteConfirm = false"
        @confirm="handleDelete"
      />

      <div
        v-if="showReportDeleteConfirm"
        class="fixed inset-0 flex items-center justify-center bg-[var(--theme-hover-bg)]0 z-[9999]"
        @click.self="showReportDeleteConfirm = false"
      >
        <div class="w-[90%] sm:w-[400px] bg-[var(--theme-surface)] rounded-xl shadow-2xl overflow-hidden">
          <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--theme-card-border)] bg-gradient-to-b from-black/5 to-transparent">
            <h3 class="font-semibold theme-text">确认删除</h3>
            <button @click="showReportDeleteConfirm = false" class="theme-text-secondary hover:text-[var(--theme-primary)] text-xl leading-none">×</button>
          </div>
          <div class="p-4">
            <p class="text-sm theme-text">确定要删除这条评论吗？此操作不可撤销。</p>
          </div>
          <div class="flex justify-end gap-3 px-4 py-3 border-t border-[var(--theme-card-border)] bg-[var(--theme-hover-bg)]">
            <button @click="showReportDeleteConfirm = false" class="px-4 py-2 text-sm theme-text hover:text-[var(--theme-primary)] transition-colors">取消</button>
            <button @click="deleteReportComment" class="px-4 py-2 bg-red-500/20 text-red-500 text-sm font-medium rounded-lg hover:bg-red-500/30 transition-colors">确定删除</button>
          </div>
        </div>
      </div>

      <div
        v-if="showCreatePollModal"
        class="fixed inset-0 bg-[var(--theme-hover-bg)]0 flex items-center justify-center z-50 p-3"
        @click.self="showCreatePollModal = false"
      >
        <div class="w-full max-w-md sm:max-w-lg bg-[var(--theme-surface)] rounded-xl shadow-xl overflow-hidden">
          <div class="flex items-center justify-between px-4 py-3 border-b border-[var(--theme-card-border)]">
            <h3 class="text-base sm:text-lg font-semibold theme-text">创建投票</h3>
            <button @click="showCreatePollModal = false" class="w-8 h-8 flex items-center justify-center rounded-full hover:border-[var(--theme-card-border)] theme-text-secondary hover:theme-text transition-all">
              <span class="text-xl font-bold">×</span>
            </button>
          </div>
          <div class="p-4 space-y-3">
            <div>
              <label class="block text-xs sm:text-sm font-medium theme-text mb-1.5">投票标题</label>
              <input
                v-model="createPollForm.title"
                type="text"
                placeholder="请输入投票标题"
                class="w-full px-3 py-2 sm:py-2.5 border border-[var(--theme-card-border)] rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10"
              />
            </div>
            <div>
              <label class="block text-xs sm:text-sm font-medium theme-text mb-1.5">投票描述</label>
              <textarea
                v-model="createPollForm.description"
                placeholder="请输入投票描述（可选）"
                rows="3"
                class="w-full px-3 py-2 sm:py-2.5 border border-[var(--theme-card-border)] rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10 resize-none"
              />
            </div>
            <div>
              <label class="block text-xs sm:text-sm font-medium theme-text mb-1.5">投票选项</label>
              <div class="space-y-2">
                <div
                  v-for="(option, index) in createPollForm.options"
                  :key="index"
                  class="flex gap-2 items-center"
                >
                  <input
                    v-model="createPollForm.options[index]"
                    type="text"
                    placeholder="请输入选项内容"
                    class="flex-1 px-3 py-2 sm:py-2.5 border border-[var(--theme-card-border)] rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10"
                  />
                  <button
                    v-if="createPollForm.options.length > 2"
                    @click="removePollOption(index)"
                    class="w-8 h-8 flex items-center justify-center bg-[var(--theme-danger)]/50/10 border border-red-500/30 rounded-md text-[var(--theme-danger)] hover:bg-[var(--theme-danger)]/50/20 hover:border-red-500/50 transition-all"
                  >
                    ×
                  </button>
                </div>
              </div>
              <button @click="addPollOption" class="mt-2 px-3 py-1.5 bg-[var(--theme-surface)] theme-text-secondary border border-[var(--theme-card-border)] rounded-md text-xs sm:text-sm hover:bg-[var(--theme-primary)]/8 hover:text-[var(--theme-primary)] hover:border-[var(--theme-primary)]/30 transition-all">
                + 添加选项
              </button>
            </div>
          </div>
          <div class="flex justify-end gap-2 px-4 py-3 border-t border-[var(--theme-card-border)]">
            <button @click="showCreatePollModal = false" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-[var(--theme-surface)] theme-text-secondary border border-[var(--theme-card-border)] rounded-md text-xs sm:text-sm hover:bg-[var(--theme-primary)]/8 hover:text-[var(--theme-primary)] hover:border-[var(--theme-primary)]/30 transition-all">取消</button>
            <button @click="handleCreatePoll" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-[var(--theme-primary)]/95 text-white border border-[var(--theme-primary)]/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">创建</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
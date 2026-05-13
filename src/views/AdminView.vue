<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { contentApi, adminApi, pollApi } from '@/api'
import type { Content, User, Poll, CreatePollData, Claim } from '@/types'
import { getPreviewText, renderMarkdown } from '@/utils'

function getImageUrl(
  image: string | undefined,
  filePath: string | undefined,
  contentType?: string,
): string {
  if (image) {
    return image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
  }
  if (!filePath) return ''
  if (contentType === 'video') {
    return `https://xqapi.xiey.work/uploads/${filePath}`
  }
  const thumbPath = filePath.includes('_thumb.') ? filePath : filePath.replace(/\.[^.]+$/, '_thumb.webp')
  return `https://xqapi.xiey.work/thumbnails/${thumbPath}`
}

import AdminContentCard from '@/components/admin/AdminContentCard.vue'
import AdminUserCard from '@/components/admin/AdminUserCard.vue'
import AdminPagination from '@/components/admin/AdminPagination.vue'
import AdminEmptyState from '@/components/admin/AdminEmptyState.vue'
import AdminContentDetailModal from '@/components/admin/AdminContentDetailModal.vue'
import AdminEditModal from '@/components/admin/AdminEditModal.vue'
import AdminDeleteConfirm from '@/components/admin/AdminDeleteConfirm.vue'
import AdminQuickAddTag from '@/components/admin/AdminQuickAddTag.vue'
import AdminUploadForm from '@/components/admin/AdminUploadForm.vue'
import AdminChangeAuthorModal from '@/components/admin/AdminChangeAuthorModal.vue'

const userStore = useUserStore()
const router = useRouter()

const activeTab = ref('my')
const message = ref('')
const showUploadModal = ref(false)

const uploadForm = ref({
  title: '',
  type: 'text' as 'video' | 'image' | 'text',
  content: '',
  tags: [] as string[],
  file: undefined as File | undefined,
  filePath: ''
})
const filePreview = ref<string>('')
const uploadProgress = ref(0)
const isUploading = ref(false)

const editForm = ref({
  id: 0,
  title: '',
  content: '',
  type: 'text' as 'video' | 'image' | 'text',
  filePath: '',
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

// 投票管理相关
const polls = ref<Poll[]>([])
const pollsPage = ref(1)
const pollsTotal = ref(0)
const pollsTotalPages = ref(1)

// 认领管理相关
const claims = ref<Claim[]>([])
const claimsPage = ref(1)
const claimsTotal = ref(0)
const claimsTotalPages = ref(1)
const claimsStatusFilter = ref('')

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
    const tags = content.tags || content.Tags || []
    tags.forEach(tag => tagsSet.add(String(tag)))
  })
  return Array.from(tagsSet)
})

const renderedDetailContent = computed(() => {
  if (!contentDetail.value) return ''
  const text = contentDetail.value.content || contentDetail.value.Content || ''
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
  contentDetail.value = content
  showContentDetail.value = true
}

function openChangeAuthorModal(content: Content) {
  changeAuthorContentId.value = content.id || content.ID || 0
  changeAuthorCurrentName.value = content.user?.username || content.User?.Username || ''
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
      pollsTotalPages.value = 1 // 假设目前只有一页
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
    const userId = userStore.user.id || userStore.user.ID
    if (!uploadForm.value.title) {
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
      uploadForm.value = { title: '', type: 'text', content: '', tags: [], file: undefined, filePath: '' }
      filePreview.value = ''
      showUploadModal.value = false
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
    id: content.id || content.ID,
    title: content.title || content.Title,
    content: content.content || content.Content,
    type: (content.type || content.Type) as 'video' | 'image' | 'text',
    filePath: content.file_path || content.FilePath || '',
    tags: (content.tags || content.Tags || []).map(t => String(t)),
    file: undefined
  }
  isEditModal.value = true
}

async function handleUpdate() {
  try {
    const res = await contentApi.update(editForm.value.id, {
      title: editForm.value.title,
      content: editForm.value.content,
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
    const adminId = userStore.user.id || userStore.user.ID
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
}

onMounted(() => {
  if (userStore.isLoggedIn) {
    loadMyContents()
  }
})
</script>

<template>
  <div class="admin-container">
    <div class="mac-window admin-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">小泉动漫 - 后台管理</div>
      </div>

      <div class="admin-content">
        <div v-if="message" class="message-bar" :class="{ success: message.includes('成功'), error: message.includes('失败') }">
          {{ message }}
          <span class="message-close" @click="message = ''">×</span>
        </div>

        <div class="nav-tabs">
          <button
            v-for="tab in [{key:'my',label:'我的内容'},{key:'pending',label:'审核内容'},{key:'all',label:'所有内容'},{key:'users',label:'用户管理'},{key:'polls',label:'投票管理'},{key:'claims',label:'认领管理'}]"
            :key="tab.key"
            v-show="tab.key === 'my' || userStore.user?.is_admin"
            @click="onTabChange(tab.key)"
            :class="['nav-tab', { active: activeTab === tab.key }]"
          >
            {{ tab.label }}
          </button>
        </div>

        <div v-if="activeTab === 'my'" class="tab-content">
          <input
            ref="imageUploadInput"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            style="display: none"
            @change="handleImageUpload"
          />
          <div class="section-header">
            <h2 class="section-title">上传内容</h2>
            <button @click="router.push('/upload')" class="mac-btn primary-btn">
              新建上传
            </button>
          </div>
          
          <div class="section-header">
            <h2 class="section-title">我的内容</h2>
            <span class="section-count">共 {{ myContentsTotal }} 条</span>
          </div>

          <div class="content-list">
            <AdminContentCard
              v-for="content in myContents"
              :key="content.id || content.ID"
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

        <div v-if="activeTab === 'pending'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">待审核内容</h2>
            <span class="section-count">共 {{ pendingContentsTotal }} 条</span>
          </div>

          <div class="content-list">
            <AdminContentCard
              v-for="content in pendingContents"
              :key="content.id || content.ID"
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

        <div v-if="activeTab === 'all'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">所有内容</h2>
            <div class="section-actions">
              <span class="section-count">共 {{ allContentsTotal }} 条</span>
              <button @click="regenerateAllThumbnails" class="mac-btn secondary-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
                  <path d="M3 3v5h5"/>
                  <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
                  <path d="M16 21h5v-5"/>
                </svg>
                批量生成缩略图
              </button>
            </div>
          </div>

          <div class="content-list">
            <AdminContentCard
              v-for="content in allContents"
              :key="content.id || content.ID"
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

        <div v-if="activeTab === 'users'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">用户管理</h2>
            <span class="section-count">共 {{ usersTotal }} 条</span>
          </div>

          <div class="user-list">
            <AdminUserCard
              v-for="user in users"
              :key="user.id || user.ID"
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

        <div v-if="activeTab === 'polls'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">投票管理</h2>
            <div class="section-actions">
              <span class="section-count">共 {{ pollsTotal }} 条</span>
              <button @click="showCreatePollModal = true" class="mac-btn primary-btn">
                创建投票
              </button>
            </div>
          </div>

          <div class="poll-list">
            <div
              v-for="poll in polls"
              :key="poll.id"
              class="poll-item"
            >
              <div class="poll-item-header">
                <div class="poll-item-title">{{ poll.title }}</div>
                <div class="poll-item-meta">
                  <span>{{ poll.vote_count }} 票</span>
                  <span>{{ new Date(poll.created_at).toLocaleString() }}</span>
                </div>
              </div>
              <div v-if="poll.description" class="poll-item-description">
                {{ poll.description }}
              </div>
              <div class="poll-item-options">
                <span v-for="(option, index) in poll.options" :key="index" class="poll-option-tag">
                  {{ option }}
                </span>
              </div>
              <div class="poll-item-actions">
                <button
                  @click="handleDeletePoll(poll.id)"
                  class="mac-btn danger-btn"
                >
                  删除
                </button>
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

        <div v-if="activeTab === 'claims'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">认领管理</h2>
            <div class="section-actions">
              <span class="section-count">共 {{ claimsTotal }} 条</span>
              <select v-model="claimsStatusFilter" @change="claimsPage = 1; loadClaims()" class="form-input filter-select">
                <option value="">全部状态</option>
                <option value="pending">待处理</option>
                <option value="approved">已通过</option>
                <option value="rejected">已拒绝</option>
              </select>
            </div>
          </div>

          <div class="claim-list">
            <div
              v-for="claim in claims"
              :key="claim.id"
              class="claim-item"
            >
              <div class="claim-media">
                <template v-if="(claim.content.type || claim.content.Type) !== 'text'">
                  <img
                    :src="getImageUrl(claim.content.image, claim.content.thumb_path || claim.content.file_path || claim.content.FilePath, claim.content.type || claim.content.Type)"
                    :alt="(claim.content.type || claim.content.Type) === 'video' ? '视频封面' : '内容图片'"
                    class="claim-thumb"
                    loading="lazy"
                  />
                </template>
                <template v-else>
                  <div class="claim-text-preview">{{ getPreviewText(claim.content.content || claim.content.Content) }}</div>
                </template>
              </div>
              <div class="claim-body">
                <div class="claim-item-header">
                  <span class="claim-content-title">{{ claim.content?.title || claim.content?.Title || '未知内容' }}</span>
                  <span :class="['claim-status', claim.status]">
                    {{ claim.status === 'pending' ? '待处理' : claim.status === 'approved' ? '已通过' : '已拒绝' }}
                  </span>
                </div>
                <div class="claim-item-body">
                  <div class="claim-info">
                    <span class="claim-label">认领者：</span>
                    <span class="claim-value">{{ claim.user?.username || '未知用户' }}</span>
                  </div>
                  <div class="claim-info">
                    <span class="claim-label">认领理由：</span>
                    <span class="claim-value">{{ claim.reason }}</span>
                  </div>
                  <div v-if="claim.remark" class="claim-info">
                    <span class="claim-label">备注：</span>
                    <span class="claim-value">{{ claim.remark }}</span>
                  </div>
                  <div class="claim-info">
                    <span class="claim-label">提交时间：</span>
                    <span class="claim-value">{{ new Date(claim.created_at).toLocaleString() }}</span>
                  </div>
                </div>
                <div v-if="claim.status === 'pending'" class="claim-item-actions">
                  <button @click="handleClaim(claim.id, 'approve')" class="mac-btn primary-btn">通过</button>
                  <button @click="handleClaim(claim.id, 'reject')" class="mac-btn danger-btn">拒绝</button>
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
      </div>
    </div>

    <input
      ref="editImageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleEditImageUpload"
    />
    
    <AdminUploadForm
      :visible="showUploadModal"
      :upload-form="uploadForm"
      :all-tags="allTags"
      :file-preview="filePreview"
      :upload-progress="uploadProgress"
      :is-uploading="isUploading"
      @close="showUploadModal = false; filePreview = ''"
      @submit="handleUpload"
      @insert-markdown="insertMarkdown"
      @trigger-image-upload="triggerImageUpload"
      @handle-image-upload="handleImageUpload"
      @handle-file-change="handleFileChange"
      @add-tag="addTag"
      @remove-tag="removeTag"
      @toggle-tag="toggleTag"
      @clear-preview="clearFilePreview"
    />
    
    <AdminEditModal
      :visible="isEditModal"
      :edit-form="editForm"
      :all-tags="allTags"
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

      <!-- 创建投票模态框 -->
      <div
        v-if="showCreatePollModal"
        class="modal-overlay"
        @click.self="showCreatePollModal = false"
      >
        <div class="modal-content poll-modal">
          <div class="modal-header">
            <h3>创建投票</h3>
            <button @click="showCreatePollModal = false" class="modal-close">×</button>
          </div>
          <div class="modal-body">
            <div class="form-group">
              <label>投票标题</label>
              <input
                v-model="createPollForm.title"
                type="text"
                placeholder="请输入投票标题"
                class="form-input"
              />
            </div>
            <div class="form-group">
              <label>投票描述</label>
              <textarea
                v-model="createPollForm.description"
                placeholder="请输入投票描述（可选）"
                class="form-textarea"
                rows="3"
              />
            </div>
            <div class="form-group">
              <label>投票选项</label>
              <div class="options-list">
                <div
                  v-for="(option, index) in createPollForm.options"
                  :key="index"
                  class="option-item"
                >
                  <input
                    v-model="createPollForm.options[index]"
                    type="text"
                    placeholder="请输入选项内容"
                    class="form-input"
                  />
                  <button
                    v-if="createPollForm.options.length > 2"
                    @click="removePollOption(index)"
                    class="option-remove"
                  >
                    ×
                  </button>
                </div>
              </div>
              <button @click="addPollOption" class="add-option-btn mac-btn secondary-btn">
                + 添加选项
              </button>
            </div>
          </div>
          <div class="modal-footer">
            <button @click="showCreatePollModal = false" class="mac-btn">取消</button>
            <button @click="handleCreatePoll" class="mac-btn primary-btn">创建</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.admin-container {
  min-height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.admin-window {
  width: 100%;
  max-width: 1400px;
  min-height: 80vh;
  overflow: hidden;
  background: rgba(255, 255, 255, 0.75);
  border-radius: 12px;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.08),
    0 2px 8px rgba(0, 0, 0, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.mac-title-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 10px 16px;
  background: linear-gradient(180deg, rgba(0,0,0,0.08) 0%, rgba(0,0,0,0.02) 100%);
  border-radius: 12px 12px 0 0;
}

.mac-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.mac-dot-red {
  background: #ff5f57;
}

.mac-dot-yellow {
  background: #febc2e;
}

.mac-dot-green {
  background: #28c840;
}

.window-title {
  margin-left: 16px;
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.admin-content {
  padding: 24px;
  background: rgba(255, 255, 255, 0.75);
}

.message-bar {
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.message-bar.success {
  background: #ecfdf5;
  color: #059669;
}

.message-bar.error {
  background: #fef2f2;
  color: #dc2626;
}

.message-close {
  cursor: pointer;
  font-size: 18px;
  font-weight: bold;
}

.nav-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.nav-tab {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  color: #666;
  cursor: pointer;
  transition: all 0.2s ease;
}

.nav-tab:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.nav-tab.active {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  font-weight: 500;
  border-color: rgba(59, 130, 246, 0.3);
}

.tab-content {
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.section-count {
  font-size: 13px;
  color: #888;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: 16px;
}

/* 公共按钮样式 */
.mac-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mac-btn:hover {
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.primary-btn {
  background: rgba(59, 130, 246, 0.95);
  color: white;
  border-color: rgba(59, 130, 246, 0.95);
}

.primary-btn:hover {
  background: rgba(37, 99, 235, 0.95);
  color: white;
  border-color: rgba(37, 99, 235, 0.95);
}

.danger-btn {
  background: rgba(239, 68, 68, 0.95);
  color: white;
  border-color: rgba(239, 68, 68, 0.95);
}

.danger-btn:hover {
  background: rgba(220, 38, 38, 0.95);
  color: white;
  border-color: rgba(220, 38, 38, 0.95);
}

.secondary-btn {
  background: rgba(255, 255, 255, 0.95);
  color: #666;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  padding: 6px 14px;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.secondary-btn:hover {
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.secondary-btn svg {
  width: 14px;
  height: 14px;
}

.content-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.user-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.poll-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.poll-item {
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  padding: 16px;
}

.poll-item-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}

.poll-item-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  flex: 1;
}

.poll-item-meta {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: #888;
  white-space: nowrap;
}

.poll-item-description {
  font-size: 14px;
  color: #666;
  margin-bottom: 12px;
  line-height: 1.5;
}

.poll-item-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.poll-option-tag {
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 12px;
  font-size: 12px;
  color: #3b82f6;
  white-space: nowrap;
}

.poll-item-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

/* 投票创建模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
  max-width: 500px;
  width: 90%;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.poll-modal .modal-content {
  max-width: 600px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  color: #666;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.2s ease;
}

.modal-close:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #333;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #333;
  margin-bottom: 6px;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 10px 12px;
  font-size: 14px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  background: white;
  outline: none;
  transition: all 0.2s ease;
}

.form-input:focus,
.form-textarea:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.options-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.option-item {
  display: flex;
  gap: 8px;
  align-items: center;
}

.option-item .form-input {
  flex: 1;
}

.option-remove {
  padding: 4px 8px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 4px;
  font-size: 14px;
  color: #ef4444;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.option-remove:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.add-option-btn {
  margin-top: 8px;
}

/* 认领管理样式 */
.claim-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.claim-item {
  display: flex;
  gap: 16px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
}

.claim-media {
  width: 200px;
  height: 140px;
  flex-shrink: 0;
  overflow: hidden;
  background: #f8f9fa;
  border-radius: 8px;
}

.claim-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.claim-text-preview {
  width: 100%;
  height: 100%;
  padding: 8px;
  font-size: 12px;
  color: #666;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 5;
  -webkit-box-orient: vertical;
}

.claim-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.claim-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.claim-content-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.claim-status {
  padding: 4px 10px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.claim-status.pending {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.claim-status.approved {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.claim-status.rejected {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.claim-item-body {
  margin-bottom: 12px;
}

.claim-info {
  font-size: 14px;
  color: #666;
  margin-bottom: 4px;
  line-height: 1.5;
}

.claim-label {
  color: #888;
  font-weight: 500;
}

.claim-value {
  color: #333;
}

.claim-item-actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}

.filter-select {
  width: auto;
  min-width: 120px;
}

@media screen and (max-width: 768px) {
  .admin-container {
    padding: 0;
  }

  .admin-window {
    border-radius: 0;
    box-shadow: none;
    min-height: 100vh;
  }

  .mac-title-bar {
    display: none;
  }

  .admin-content {
    padding: 12px;
  }

  .nav-tabs {
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 16px;
    padding-bottom: 12px;
  }

  .nav-tab {
    padding: 6px 12px;
    font-size: 13px;
  }

  .section-header {
    flex-wrap: wrap;
    gap: 8px;
  }

  .section-actions {
    width: 100%;
    justify-content: space-between;
  }

  .secondary-btn {
    padding: 5px 10px;
    font-size: 12px;
  }

  .mac-btn {
    padding: 6px 12px;
    font-size: 13px;
  }

  .claim-item {
    flex-direction: column;
    gap: 12px;
  }

  .claim-media {
    width: 100%;
    height: 200px;
  }
}
</style>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { contentApi, adminApi, pollApi } from '@/api'
import type { Content, User, Poll, CreatePollData, Claim } from '@/types'
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
import AdminUploadForm from '@/components/admin/AdminUploadForm.vue'
import AdminChangeAuthorModal from '@/components/admin/AdminChangeAuthorModal.vue'

const userStore = useUserStore()
const router = useRouter()

const activeTab = ref('my')
const message = ref('')
const showUploadModal = ref(false)

const uploadForm = ref({
  title: '',
    type: 'text' as 'video' | 'image' | 'text' | 'link',
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
      uploadForm.value = { title: '', type: 'text', content: '', url: '', tags: [], file: undefined }
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
}

onMounted(() => {
  if (userStore.isLoggedIn) {
    loadMyContents()
  }
})
</script>

<template>
  <div class="min-h-screen p-2 sm:p-4 md:p-5 lg:p-6 flex justify-center bg-transparent">
    <div class="w-full max-w-6xl bg-white/75 rounded-xl shadow-lg shadow-black/5 border border-white/40 overflow-hidden">
      <div class="flex items-center px-3 sm:px-4 py-2 sm:py-2.5 bg-gradient-to-b from-black/8 to-black/2">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm text-gray-500 font-medium">小泉动漫 - 后台管理</div>
      </div>

      <div class="p-3 sm:p-4 md:p-6">
        <div v-if="message" class="px-3 sm:px-4 py-2 sm:py-3 rounded-lg mb-3 sm:mb-4 flex justify-between items-center" :class="message.includes('成功') ? 'bg-emerald-50/80 text-emerald-600' : 'bg-red-50/80 text-red-600'">
          <span class="text-xs sm:text-sm">{{ message }}</span>
          <span class="text-sm font-bold cursor-pointer hover:opacity-70" @click="message = ''">×</span>
        </div>

        <div class="flex flex-wrap gap-1.5 sm:gap-2 mb-3 sm:mb-4 pb-2 sm:pb-3 border-b border-black/6">
          <button
            v-for="tab in [{key:'my',label:'我的内容'},{key:'pending',label:'审核内容'},{key:'all',label:'所有内容'},{key:'users',label:'用户管理'},{key:'polls',label:'投票管理'},{key:'claims',label:'认领管理'}]"
            :key="tab.key"
            v-show="tab.key === 'my' || userStore.user?.is_admin"
            @click="onTabChange(tab.key)"
            :class="[
              'px-2.5 sm:px-4 py-1.5 sm:py-2 text-xs sm:text-sm rounded-md transition-all',
              activeTab === tab.key 
                ? 'bg-blue-500/10 text-blue-500 font-medium border border-blue-500/30' 
                : 'bg-white/95 text-gray-600 border border-black/10 hover:bg-blue-500/8 hover:text-blue-500 hover:border-blue-500/30'
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
          <div class="flex items-center justify-between">
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">上传内容</h2>
            <button @click="router.push('/upload')" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-blue-500/95 text-white border border-blue-500/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">
              新建上传
            </button>
          </div>
          
          <div class="flex items-center justify-between">
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">我的内容</h2>
            <span class="text-xs sm:text-sm text-gray-500">共 {{ myContentsTotal }} 条</span>
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
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">待审核内容</h2>
            <span class="text-xs sm:text-sm text-gray-500">共 {{ pendingContentsTotal }} 条</span>
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
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">所有内容</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm text-gray-500">共 {{ allContentsTotal }} 条</span>
              <button @click="regenerateAllThumbnails" class="flex items-center gap-1.5 px-2.5 sm:px-4 py-1.5 sm:py-2 bg-white/95 text-gray-600 border border-black/10 rounded-md text-xs sm:text-sm hover:bg-blue-500/8 hover:text-blue-500 hover:border-blue-500/30 transition-all">
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
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">用户管理</h2>
            <span class="text-xs sm:text-sm text-gray-500">共 {{ usersTotal }} 条</span>
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
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">投票管理</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm text-gray-500">共 {{ pollsTotal }} 条</span>
              <button @click="showCreatePollModal = true" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-blue-500/95 text-white border border-blue-500/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">
                创建投票
              </button>
            </div>
          </div>

          <div class="space-y-2 sm:space-y-3">
            <div
              v-for="poll in polls"
              :key="poll.id"
              class="bg-white/95 border border-black/10 rounded-lg p-3 sm:p-4"
            >
              <div class="flex flex-wrap items-start justify-between gap-2">
                <div class="flex-1">
                  <div class="text-base sm:text-lg font-semibold text-gray-900 mb-1">{{ poll.title }}</div>
                  <div class="flex flex-wrap gap-2 text-xs sm:text-sm text-gray-500">
                    <span>{{ poll.vote_count }} 票</span>
                    <span>{{ new Date(poll.created_at).toLocaleString() }}</span>
                  </div>
                </div>
                <button
                  @click="handleDeletePoll(poll.id)"
                  class="px-2 sm:px-3 py-1 sm:py-1.5 bg-red-500/95 text-white border border-red-500/95 rounded-md text-xs sm:text-sm hover:bg-red-700/95 hover:border-red-700/95 transition-all shrink-0"
                >
                  删除
                </button>
              </div>
              <div v-if="poll.description" class="text-xs sm:text-sm text-gray-600 mt-2 sm:mt-3 leading-relaxed">
                {{ poll.description }}
              </div>
              <div class="flex flex-wrap gap-1.5 sm:gap-2 mt-2 sm:mt-3">
                <span v-for="(option, index) in poll.options" :key="index" class="px-2 sm:px-2.5 py-0.5 sm:py-1 bg-blue-500/10 border border-blue-500/30 rounded-full text-xs sm:text-sm text-blue-500">
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
            <h2 class="text-base sm:text-lg font-semibold text-gray-900">认领管理</h2>
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="text-xs sm:text-sm text-gray-500">共 {{ claimsTotal }} 条</span>
              <select v-model="claimsStatusFilter" @change="claimsPage = 1; loadClaims()" class="px-2.5 sm:px-3 py-1.5 sm:py-2 border border-black/10 rounded-md text-xs sm:text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10">
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
              class="bg-white/60 border border-white/36 rounded-xl p-3 sm:p-4 shadow-md shadow-black/5"
            >
              <div class="flex flex-col sm:flex-row gap-3 sm:gap-4">
                <div class="w-full sm:w-48 h-28 sm:h-36 bg-gray-100 rounded-lg overflow-hidden shrink-0">
                  <template v-if="claim.content.type !== 'text' && claim.content.type !== 'link'">
                    <img
                      :src="getImageUrl(claim.content.thumb)"
                      :alt="claim.content.type === 'video' ? '视频封面' : '内容图片'"
                      class="w-full h-full object-cover"
                      loading="lazy"
                    />
                  </template>
                  <template v-else>
                    <div class="w-full h-full p-2 text-xs text-gray-500 overflow-hidden flex items-center justify-center">
                      {{ getPreviewText(claim.content.text) }}
                    </div>
                  </template>
                </div>
                <div class="flex-1 space-y-2">
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <span class="text-base sm:text-lg font-semibold text-gray-900">{{ claim.content?.title || '未知内容' }}</span>
                    <span :class="[
                      'px-2 sm:px-3 py-1 rounded-full text-xs font-medium',
                      claim.status === 'pending' ? 'bg-amber-500/10 text-amber-600 border border-amber-500/30' :
                      claim.status === 'approved' ? 'bg-green-500/10 text-green-600 border border-green-500/30' :
                      'bg-red-500/10 text-red-600 border border-red-500/30'
                    ]">
                      {{ claim.status === 'pending' ? '待处理' : claim.status === 'approved' ? '已通过' : '已拒绝' }}
                    </span>
                  </div>
                  <div class="text-xs sm:text-sm text-gray-600 space-y-1">
                    <div><span class="text-gray-400">认领者：</span>{{ claim.user?.username || '未知用户' }}</div>
                    <div><span class="text-gray-400">认领理由：</span>{{ claim.reason }}</div>
                    <div v-if="claim.remark"><span class="text-gray-400">备注：</span>{{ claim.remark }}</div>
                    <div><span class="text-gray-400">提交时间：</span>{{ new Date(claim.created_at).toLocaleString() }}</div>
                  </div>
                  <div v-if="claim.status === 'pending'" class="flex gap-2 mt-2">
                    <button @click="handleClaim(claim.id, 'approve')" class="flex-1 px-3 py-1.5 bg-blue-500/95 text-white border border-blue-500/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">通过</button>
                    <button @click="handleClaim(claim.id, 'reject')" class="flex-1 px-3 py-1.5 bg-red-500/95 text-white border border-red-500/95 rounded-md text-xs sm:text-sm hover:bg-red-700/95 hover:border-red-700/95 transition-all">拒绝</button>
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

      <div
        v-if="showCreatePollModal"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-3"
        @click.self="showCreatePollModal = false"
      >
        <div class="w-full max-w-md sm:max-w-lg bg-white rounded-xl shadow-xl overflow-hidden">
          <div class="flex items-center justify-between px-4 py-3 border-b border-black/6">
            <h3 class="text-base sm:text-lg font-semibold text-gray-900">创建投票</h3>
            <button @click="showCreatePollModal = false" class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-black/6 text-gray-500 hover:text-gray-800 transition-all">
              <span class="text-xl font-bold">×</span>
            </button>
          </div>
          <div class="p-4 space-y-3">
            <div>
              <label class="block text-xs sm:text-sm font-medium text-gray-700 mb-1.5">投票标题</label>
              <input
                v-model="createPollForm.title"
                type="text"
                placeholder="请输入投票标题"
                class="w-full px-3 py-2 sm:py-2.5 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10"
              />
            </div>
            <div>
              <label class="block text-xs sm:text-sm font-medium text-gray-700 mb-1.5">投票描述</label>
              <textarea
                v-model="createPollForm.description"
                placeholder="请输入投票描述（可选）"
                rows="3"
                class="w-full px-3 py-2 sm:py-2.5 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 resize-none"
              />
            </div>
            <div>
              <label class="block text-xs sm:text-sm font-medium text-gray-700 mb-1.5">投票选项</label>
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
                    class="flex-1 px-3 py-2 sm:py-2.5 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10"
                  />
                  <button
                    v-if="createPollForm.options.length > 2"
                    @click="removePollOption(index)"
                    class="w-8 h-8 flex items-center justify-center bg-red-500/10 border border-red-500/30 rounded-md text-red-500 hover:bg-red-500/20 hover:border-red-500/50 transition-all"
                  >
                    ×
                  </button>
                </div>
              </div>
              <button @click="addPollOption" class="mt-2 px-3 py-1.5 bg-white/95 text-gray-600 border border-black/10 rounded-md text-xs sm:text-sm hover:bg-blue-500/8 hover:text-blue-500 hover:border-blue-500/30 transition-all">
                + 添加选项
              </button>
            </div>
          </div>
          <div class="flex justify-end gap-2 px-4 py-3 border-t border-black/6">
            <button @click="showCreatePollModal = false" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-white/95 text-gray-600 border border-black/10 rounded-md text-xs sm:text-sm hover:bg-blue-500/8 hover:text-blue-500 hover:border-blue-500/30 transition-all">取消</button>
            <button @click="handleCreatePoll" class="px-3 sm:px-4 py-1.5 sm:py-2 bg-blue-500/95 text-white border border-blue-500/95 rounded-md text-xs sm:text-sm hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">创建</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
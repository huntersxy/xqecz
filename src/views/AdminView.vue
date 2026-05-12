<!-- eslint-disable @typescript-eslint/no-unused-vars -->
<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { contentApi, adminApi } from '@/api'
import type { Content, User, ApiResponse } from '@/types'
import { marked } from 'marked'
import { getImageUrl, getPreviewText } from '@/utils'

const userStore = useUserStore()

const activeTab = ref('my')
const message = ref('')

const renderedDetailContent = computed(() => {
  if (!contentDetail.value) return ''
  const text = contentDetail.value.content || contentDetail.value.Content || ''
  let processedText = text.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '[禁止的脚本]')
  processedText = processedText.replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '[禁止的iframe]')
  return marked(processedText) as string
})

const uploadForm = ref({
  title: '',
  type: 'text' as 'video' | 'image' | 'text',
  content: '',
  tags: [] as string[],
  file: undefined as File | undefined
})

const filePreview = ref<string>('')

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

const showDeleteConfirm = ref(false)
const deleteTargetId = ref(0)

const showContentDetail = ref(false)
const contentDetail = ref<Content | null>(null)

const uploadProgress = ref(0)
const isUploading = ref(false)

function insertMarkdown(prefix: string, suffix: string = '') {
  const editTextarea = document.querySelector('.edit-textarea') as HTMLTextAreaElement
  const uploadTextarea = document.querySelector('.upload-textarea') as HTMLTextAreaElement
  const textarea = editTextarea && editTextarea.offsetParent !== null ? editTextarea : uploadTextarea
  if (!textarea) return
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const currentForm = isEditModal.value ? editForm.value : uploadForm.value
  const text = currentForm.content || ''
  const before = text.substring(0, start)
  const selected = text.substring(start, end)
  const after = text.substring(end)
  currentForm.content = before + prefix + selected + suffix + after
  textarea.focus()
  const newPos = start + prefix.length + selected.length + suffix.length
  setTimeout(() => {
    textarea.setSelectionRange(newPos, newPos)
  }, 0)
}

function insertHeading() { insertMarkdown('## ') }
function insertBold() { insertMarkdown('**', '**') }
function insertItalic() { insertMarkdown('*', '*') }
function insertCode() { insertMarkdown('`', '`') }
function insertCodeBlock() { insertMarkdown('\n```\n', '\n```\n') }
function insertLink() { insertMarkdown('[', '](url)') }
function insertList() { insertMarkdown('\n- ') }

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
        const before = editForm.value.content.substring(0, start)
        const after = editForm.value.content.substring(end)
        editForm.value.content = before + `![${file.name}](${res.data.image_url})` + after
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

const tagTargetForm = ref<'upload' | 'edit'>('upload')

function addTag(target: 'upload' | 'edit' = 'upload') {
  tagTargetForm.value = target
  showQuickAddTag.value = true
}

async function handleQuickAddTag() {
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

async function loadMyContents() {
  try {
    const res = await contentApi.myList({ page: myContentsPage.value, page_size: 10 })
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
    const res = await adminApi.pending({ page: pendingContentsPage.value, page_size: 10 })
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
    const res = await adminApi.getAllContent({ page: allContentsPage.value, page_size: 10 })
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
    const res = await adminApi.getUsers({ page: usersPage.value, page_size: 10 })
    if (res.code === 200) {
      users.value = res.data.list
      usersTotal.value = res.data.total
      usersTotalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载用户列表失败', error)
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

    const res = await contentApi.upload({
      ...uploadForm.value,
      user_id: userId
    }, (percent) => {
      uploadProgress.value = percent
    }) as ApiResponse<Content>

    if (res.code === 200) {
      message.value = '上传成功'
      uploadForm.value = {
        title: '',
        type: 'text',
        content: '',
        tags: [],
        file: undefined
      }
      loadMyContents()
    } else {
      message.value = res.message || '上传失败'
    }
  } catch (error: unknown) {
    const messageText = error instanceof Error ? error.message : '网络错误'
    message.value = `上传失败: ${messageText}`
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
            v-for="tab in [{key:'my',label:'我的内容'},{key:'pending',label:'审核内容'},{key:'all',label:'所有内容'},{key:'users',label:'用户管理'}]"
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
          </div>

          <div class="upload-form">
            <div class="form-row">
              <label class="form-label">标题</label>
              <input v-model="uploadForm.title" type="text" class="mac-input" placeholder="请输入标题" />
            </div>

            <div class="form-row">
              <label class="form-label">类型</label>
              <div class="type-buttons">
                <button
                  type="button"
                  @click="uploadForm.type = 'text'"
                  :class="['type-btn', { active: uploadForm.type === 'text' }]"
                >
                  <svg class="type-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                    <polyline points="14 2 14 8 20 8"/>
                    <line x1="16" y1="13" x2="8" y2="13"/>
                    <line x1="16" y1="17" x2="8" y2="17"/>
                  </svg>
                  图文
                </button>
                <button
                  type="button"
                  @click="uploadForm.type = 'image'"
                  :class="['type-btn', { active: uploadForm.type === 'image' }]"
                >
                  <svg class="type-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                    <circle cx="8.5" cy="8.5" r="1.5"/>
                    <polyline points="21 15 16 10 5 21"/>
                  </svg>
                  图片
                </button>
                <button
                  type="button"
                  @click="uploadForm.type = 'video'"
                  :class="['type-btn', { active: uploadForm.type === 'video' }]"
                >
                  <svg class="type-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="23 7 16 12 23 17 23 7"/>
                    <rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
                  </svg>
                  视频
                </button>
              </div>
            </div>

            <div v-if="uploadForm.type === 'text'" class="form-row">
              <label class="form-label">内容</label>
              <div class="md-toolbar">
                <button type="button" @click="insertMarkdown('## ')" class="md-btn" title="标题">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M4 12h16M4 6h16M4 18h10"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('**', '**')" class="md-btn" title="粗体">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
                    <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('*', '*')" class="md-btn" title="斜体">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="19" y1="4" x2="10" y2="4"/>
                    <line x1="14" y1="20" x2="5" y2="20"/>
                    <line x1="15" y1="4" x2="9" y2="20"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('`', '`')" class="md-btn" title="行内代码">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="16 18 22 12 16 6"/>
                    <polyline points="8 6 2 12 8 18"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('\n```\n', '\n```\n')" class="md-btn" title="代码块">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="18" height="18" rx="2"/>
                    <path d="M8 10l4 4-4 4"/>
                    <line x1="14" y1="18" x2="18" y2="18"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('[', '](url)')" class="md-btn" title="链接">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                  </svg>
                </button>
                <button type="button" @click="triggerImageUpload" class="md-btn" title="上传图片">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                    <circle cx="8.5" cy="8.5" r="1.5"/>
                    <polyline points="21 15 16 10 5 21"/>
                  </svg>
                </button>
                <button type="button" @click="insertMarkdown('\n- ')" class="md-btn" title="列表">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="8" y1="6" x2="21" y2="6"/>
                    <line x1="8" y1="12" x2="21" y2="12"/>
                    <line x1="8" y1="18" x2="21" y2="18"/>
                    <line x1="3" y1="6" x2="3.01" y2="6"/>
                    <line x1="3" y1="12" x2="3.01" y2="12"/>
                    <line x1="3" y1="18" x2="3.01" y2="18"/>
                  </svg>
                </button>
              </div>
              <textarea v-model="uploadForm.content" class="mac-input textarea upload-textarea" placeholder="支持 Markdown 语法：# 标题、**粗体**、*斜体*、`代码`、![图片](url)、[链接](url)"></textarea>
            </div>

            <div v-else class="form-row">
              <label class="form-label">文件</label>
              <input type="file" @change="handleFileChange" class="file-input" />
              <div v-if="filePreview" class="file-preview">
                <img :src="filePreview" alt="文件预览" class="preview-image" />
                <button type="button" @click="filePreview = ''; uploadForm.file = undefined" class="preview-remove">移除</button>
              </div>
            </div>

            <div class="form-row">
              <label class="form-label">标签</label>
              <div class="tag-buttons-row">
                <div class="tag-buttons">
                  <span
                    v-for="tag in uploadForm.tags"
                    :key="tag"
                    class="selected-tag"
                  >
                    {{ tag }}
                    <span class="tag-remove" @click="removeTag(tag)">×</span>
                  </span>
                  <button type="button" @click="addTag('upload')" class="quick-add-btn" title="添加标签">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <button @click="handleUpload" class="mac-btn primary-btn" :disabled="isUploading">上传</button>
          <div v-if="isUploading" class="upload-progress">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
            </div>
            <span class="progress-text">{{ uploadProgress }}%</span>
          </div>
          </div>

          <div class="section-header">
            <h2 class="section-title">我的内容</h2>
            <span class="section-count">共 {{ myContentsTotal }} 条</span>
          </div>

          <div class="content-list">
            <div
              v-for="content in myContents"
              :key="content.id || content.ID"
              class="content-item mac-card"
            >
              <div class="item-media">
                <template v-if="(content.type || content.Type) === 'image'">
                  <img
                    :src="getImageUrl(content.image, content.file_path || content.FilePath)"
                    alt="内容图片"
                    class="item-image"
                  />
                </template>
                <template v-else-if="(content.type || content.Type) === 'video'">
                  <img
                    :src="getImageUrl(content.image, content.thumb_path || content.file_path || content.FilePath)"
                    alt="视频封面"
                    class="item-image"
                  />
                </template>
                <template v-else>
                  <div class="item-text-preview">{{ getPreviewText(content.content || content.Content || '') }}</div>
                </template>
              </div>
              <div class="item-info">
                <h3 class="item-title">{{ content.title || content.Title }}</h3>
                <div class="item-tags">
                  <span v-for="tag in (content.tags || content.Tags || [])" :key="tag" class="item-tag">{{ tag }}</span>
                </div>
                <div class="item-meta">
                  <span :class="['type-badge', (content.type || content.Type)]">
                    {{ (content.type || content.Type) === 'video' ? '视频' : (content.type || content.Type) === 'image' ? '图片' : '文字' }}
                  </span>
                  <span :class="['status-badge', (content.audit_status || content.AuditStatus)]">
                    {{ (content.audit_status || content.AuditStatus) === 'approved' ? '已通过' : (content.audit_status || content.AuditStatus) === 'pending' ? '审核中' : '已拒绝' }}
                  </span>
                </div>
                <div class="item-actions">
                  <button @click="openContentDetail(content)" class="action-btn">查看</button>
                  <button @click="openEditModal(content)" class="action-btn">编辑</button>
                  <button v-if="(content.type || content.Type) === 'video'" @click="regenerateThumbnail(content.id || content.ID)" class="action-btn">更新封面</button>
                  <button @click="confirmDelete(content.id || content.ID)" class="action-btn delete-btn">删除</button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="myContents.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <p>暂无内容</p>
          </div>

          <div v-if="myContentsTotalPages > 1" class="pagination-section">
            <button
              @click="myContentsPage = myContentsPage - 1; loadMyContents()"
              :disabled="myContentsPage <= 1"
              class="mac-btn pagination-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 19l-7-7 7-7"/>
              </svg>
              上一页
            </button>
            <span class="pagination-info">第 {{ myContentsPage }} / {{ myContentsTotalPages }} 页</span>
            <button
              @click="myContentsPage = myContentsPage + 1; loadMyContents()"
              :disabled="myContentsPage >= myContentsTotalPages"
              class="mac-btn pagination-btn"
            >
              下一页
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 5l7 7-7 7"/>
              </svg>
            </button>
          </div>
        </div>

        <div v-if="activeTab === 'pending'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">待审核内容</h2>
            <span class="section-count">共 {{ pendingContentsTotal }} 条</span>
          </div>

          <div class="content-list">
            <div
              v-for="content in pendingContents"
              :key="content.id || content.ID"
              class="content-item mac-card"
            >
              <div class="item-media">
                <template v-if="(content.type || content.Type) === 'image'">
                  <img
                    :src="getImageUrl(content.image, content.file_path || content.FilePath)"
                    alt="内容图片"
                    class="item-image"
                  />
                </template>
                <template v-else-if="(content.type || content.Type) === 'video'">
                  <img
                    :src="getImageUrl(content.image, content.thumb_path || content.file_path || content.FilePath)"
                    alt="视频封面"
                    class="item-image"
                  />
                </template>
                <template v-else>
                  <div class="item-text-preview">{{ getPreviewText(content.content || content.Content || '') }}</div>
                </template>
              </div>
              <div class="item-info">
                <h3 class="item-title">{{ content.title || content.Title }}</h3>
                <div class="item-tags">
                  <span v-for="tag in (content.tags || content.Tags || [])" :key="tag" class="item-tag">{{ tag }}</span>
                </div>
                <div class="item-meta">
                  <span :class="['type-badge', (content.type || content.Type)]">
                    {{ (content.type || content.Type) === 'video' ? '视频' : (content.type || content.Type) === 'image' ? '图片' : '文字' }}
                  </span>
                  <span class="meta-item">{{ content.user?.username || content.User?.Username }}</span>
                </div>
                <div class="item-actions">
                  <button @click="openContentDetail(content)" class="action-btn">预览</button>
                  <button @click="handleAudit(content.id || content.ID, 'approved')" class="action-btn approve-btn">通过</button>
                  <button @click="handleAudit(content.id || content.ID, 'rejected')" class="action-btn reject-btn">拒绝</button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="pendingContents.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <p>暂无待审核内容</p>
          </div>

          <div v-if="pendingContentsTotalPages > 1" class="pagination-section">
            <button
              @click="pendingContentsPage = pendingContentsPage - 1; loadPendingContents()"
              :disabled="pendingContentsPage <= 1"
              class="mac-btn pagination-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 19l-7-7 7-7"/>
              </svg>
              上一页
            </button>
            <span class="pagination-info">第 {{ pendingContentsPage }} / {{ pendingContentsTotalPages }} 页</span>
            <button
              @click="pendingContentsPage = pendingContentsPage + 1; loadPendingContents()"
              :disabled="pendingContentsPage >= pendingContentsTotalPages"
              class="mac-btn pagination-btn"
            >
              下一页
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 5l7 7-7 7"/>
              </svg>
            </button>
          </div>
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
            <div
              v-for="content in allContents"
              :key="content.id || content.ID"
              class="content-item mac-card"
            >
              <div class="item-media">
                <template v-if="(content.type || content.Type) === 'image'">
                  <img
                    :src="getImageUrl(content.image, content.file_path || content.FilePath)"
                    alt="内容图片"
                    class="item-image"
                  />
                </template>
                <template v-else-if="(content.type || content.Type) === 'video'">
                  <img
                    :src="getImageUrl(content.image, content.thumb_path || content.file_path || content.FilePath)"
                    alt="视频封面"
                    class="item-image"
                  />
                </template>
                <template v-else>
                  <div class="item-text-preview">{{ getPreviewText(content.content || content.Content || '') }}</div>
                </template>
              </div>
              <div class="item-info">
                <h3 class="item-title">{{ content.title || content.Title }}</h3>
                <div class="item-tags">
                  <span v-for="tag in (content.tags || content.Tags || [])" :key="tag" class="item-tag">{{ tag }}</span>
                </div>
                <div class="item-meta">
                  <span :class="['type-badge', (content.type || content.Type)]">
                    {{ (content.type || content.Type) === 'video' ? '视频' : (content.type || content.Type) === 'image' ? '图片' : '文字' }}
                  </span>
                  <span :class="['status-badge', (content.audit_status || content.AuditStatus)]">
                    {{ (content.audit_status || content.AuditStatus) === 'approved' ? '已通过' : (content.audit_status || content.AuditStatus) === 'pending' ? '审核中' : '已拒绝' }}
                  </span>
                  <span class="meta-item">{{ content.user?.username || content.User?.Username }}</span>
                </div>
                <div class="item-actions">
                  <button @click="openContentDetail(content)" class="action-btn">查看</button>
                  <button @click="openEditModal(content)" class="action-btn">修改</button>
                  <button v-if="(content.type || content.Type) === 'video'" @click="regenerateThumbnail(content.id || content.ID)" class="action-btn">更新封面</button>
                  <button @click="confirmDelete(content.id || content.ID)" class="action-btn delete-btn">删除</button>
                </div>
              </div>
            </div>
          </div>

          <div v-if="allContents.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <p>暂无内容</p>
          </div>

          <div v-if="allContentsTotalPages > 1" class="pagination-section">
            <button
              @click="allContentsPage = allContentsPage - 1; loadAllContents()"
              :disabled="allContentsPage <= 1"
              class="mac-btn pagination-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 19l-7-7 7-7"/>
              </svg>
              上一页
            </button>
            <span class="pagination-info">第 {{ allContentsPage }} / {{ allContentsTotalPages }} 页</span>
            <button
              @click="allContentsPage = allContentsPage + 1; loadAllContents()"
              :disabled="allContentsPage >= allContentsTotalPages"
              class="mac-btn pagination-btn"
            >
              下一页
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 5l7 7-7 7"/>
              </svg>
            </button>
          </div>
        </div>

        <div v-if="activeTab === 'users'" class="tab-content">
          <div class="section-header">
            <h2 class="section-title">用户管理</h2>
            <span class="section-count">共 {{ usersTotal }} 条</span>
          </div>

          <div class="user-list">
            <div
              v-for="user in users"
              :key="user.id || user.ID"
              class="user-item mac-card"
            >
              <div class="user-info">
                <div class="user-avatar">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                    <circle cx="12" cy="7" r="4"/>
                  </svg>
                </div>
                <div class="user-details">
                  <span class="user-name">{{ user.username || user.Username }}</span>
                  <div class="user-status">
                    <span v-if="user.is_admin || user.IsAdmin" class="admin-badge">管理员</span>
                    <span v-if="user.is_banned || user.IsBanned" class="banned-badge">已封禁</span>
                  </div>
                </div>
              </div>
              <div class="user-actions">
                <template v-if="(user.id || user.ID) !== (userStore.user?.id || userStore.user?.ID)">
                  <button
                    @click="handleUpdateUserRole(user.id || user.ID, !(user.is_admin || user.IsAdmin))"
                    class="action-btn"
                  >
                    {{ (user.is_admin || user.IsAdmin) ? '取消管理员' : '设为管理员' }}
                  </button>
                  <template v-if="!(user.is_admin || user.IsAdmin)">
                    <button
                      @click="handleUpdateUserBan(user.id || user.ID, !(user.is_banned || user.IsBanned))"
                      :class="['action-btn', (user.is_banned || user.IsBanned) ? 'unban-btn' : 'ban-btn']"
                    >
                      {{ (user.is_banned || user.IsBanned) ? '解封' : '封禁' }}
                    </button>
                    <button @click="handleDeleteUser(user.id || user.ID)" class="action-btn delete-btn">删除</button>
                  </template>
                </template>
                <span v-else class="current-user-hint">当前用户</span>
              </div>
            </div>
          </div>

          <div v-if="users.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            <p>暂无用户</p>
          </div>

          <div v-if="usersTotalPages > 1" class="pagination-section">
            <button
              @click="usersPage = usersPage - 1; loadUsers()"
              :disabled="usersPage <= 1"
              class="mac-btn pagination-btn"
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 19l-7-7 7-7"/>
              </svg>
              上一页
            </button>
            <span class="pagination-info">第 {{ usersPage }} / {{ usersTotalPages }} 页</span>
            <button
              @click="usersPage = usersPage + 1; loadUsers()"
              :disabled="usersPage >= usersTotalPages"
              class="mac-btn pagination-btn"
            >
              下一页
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 5l7 7-7 7"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="isEditModal" class="modal-overlay" @click.self="isEditModal = false">
      <input
        ref="editImageUploadInput"
        type="file"
        accept="image/jpeg,image/png,image/gif,image/webp"
        style="display: none"
        @change="handleEditImageUpload"
      />
      <div class="modal-content">
        <div class="modal-header">
          <h2>编辑内容</h2>
          <span class="modal-close" @click="isEditModal = false">×</span>
        </div>
        <div class="modal-body">
          <div class="form-row">
            <label class="form-label">标题</label>
            <input v-model="editForm.title" type="text" class="mac-input" />
          </div>
          <div v-if="editForm.type === 'text'" class="form-row">
            <label class="form-label">内容</label>
            <div class="md-toolbar">
              <button type="button" @click="insertMarkdown('## ')" class="md-btn" title="标题">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M4 12h16M4 6h16M4 18h10"/>
                </svg>
              </button>
              <button type="button" @click="insertMarkdown('**', '**')" class="md-btn" title="粗体">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
                  <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
                </svg>
              </button>
              <button type="button" @click="insertMarkdown('*', '*')" class="md-btn" title="斜体">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="19" y1="4" x2="10" y2="4"/>
                  <line x1="14" y1="20" x2="5" y2="20"/>
                  <line x1="15" y1="4" x2="9" y2="20"/>
                </svg>
              </button>
              <button type="button" @click="triggerEditImageUpload" class="md-btn" title="上传图片">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                  <circle cx="8.5" cy="8.5" r="1.5"/>
                  <polyline points="21 15 16 10 5 21"/>
                </svg>
              </button>
            </div>
            <textarea v-model="editForm.content" class="mac-input textarea edit-textarea"></textarea>
          </div>
          <div v-else class="form-row">
            <label class="form-label">文件</label>
            <input type="file" @change="handleEditFileChange" class="file-input" />
            <span v-if="editForm.filePath" class="file-path">{{ editForm.filePath }}</span>
          </div>
          <div class="form-row">
            <label class="form-label">标签</label>
            <div class="tag-buttons-row">
              <div class="tag-buttons">
                <span
                  v-for="tag in editForm.tags"
                  :key="tag"
                  class="selected-tag"
                >
                  {{ tag }}
                  <span class="tag-remove" @click="removeEditTag(tag)">×</span>
                </span>
                <button type="button" @click="addTag('edit')" class="quick-add-btn" title="添加标签">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="12" y1="5" x2="12" y2="19"/>
                    <line x1="5" y1="12" x2="19" y2="12"/>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="isEditModal = false" class="mac-btn">取消</button>
          <button @click="handleUpdate" class="mac-btn primary-btn">保存</button>
        </div>
      </div>
    </div>

    <div v-if="showContentDetail && contentDetail" class="modal-overlay" @click.self="showContentDetail = false">
      <div class="modal-content detail-modal">
        <div class="modal-header">
          <h2>{{ contentDetail.title || contentDetail.Title }}</h2>
          <span class="modal-close" @click="showContentDetail = false">×</span>
        </div>
        <div class="modal-body">
          <div class="detail-content">
            <div class="detail-meta">
              <span :class="['type-badge', (contentDetail.type || contentDetail.Type)]">
                {{ (contentDetail.type || contentDetail.Type) === 'video' ? '视频' : (contentDetail.type || contentDetail.Type) === 'image' ? '图片' : '文字' }}
              </span>
              <span :class="['status-badge', (contentDetail.audit_status || contentDetail.AuditStatus)]">
                {{ (contentDetail.audit_status || contentDetail.AuditStatus) === 'approved' ? '已通过' : (contentDetail.audit_status || contentDetail.AuditStatus) === 'pending' ? '审核中' : '已拒绝' }}
              </span>
              <span class="meta-item">{{ contentDetail.user?.username || contentDetail.User?.Username }}</span>
            </div>
            <div class="detail-tags">
              <span v-for="tag in (contentDetail.tags || contentDetail.Tags || [])" :key="tag" class="detail-tag">{{ tag }}</span>
            </div>
            <div v-if="(contentDetail.type || contentDetail.Type) === 'image'" class="detail-media">
              <img :src="getImageUrl(contentDetail.image, contentDetail.file_path || contentDetail.FilePath)" alt="内容图片" class="detail-image" />
            </div>
            <div v-else-if="(contentDetail.type || contentDetail.Type) === 'video'" class="detail-media">
              <video :src="getImageUrl(undefined, contentDetail.file_path || contentDetail.FilePath)" controls class="detail-video">
                您的浏览器不支持视频播放
              </video>
            </div>
            <div v-else class="detail-text" v-html="renderedDetailContent"></div>
          </div>
        </div>
        <div class="modal-footer">
          <button @click="showContentDetail = false" class="mac-btn">关闭</button>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showQuickAddTag" class="tag-modal-overlay" @click.self="showQuickAddTag = false">
        <div class="tag-modal">
          <div class="tag-modal-header">
            <h3>添加标签</h3>
            <button class="tag-modal-close" @click="showQuickAddTag = false">×</button>
          </div>
          <div class="tag-modal-body">
            <input
              v-model="quickAddTagName"
              type="text"
              class="mac-input"
              placeholder="请输入标签名称"
              @keyup.enter="handleQuickAddTag"
              autofocus
            />
          </div>
          <div class="tag-modal-footer">
            <button @click="showQuickAddTag = false" class="mac-btn">取消</button>
            <button @click="handleQuickAddTag" class="mac-btn primary-btn">确定</button>
          </div>
        </div>
      </div>

      <div v-if="showDeleteConfirm" class="tag-modal-overlay" @click.self="showDeleteConfirm = false">
        <div class="tag-modal">
          <div class="tag-modal-header">
            <h3>确认删除</h3>
            <button class="tag-modal-close" @click="showDeleteConfirm = false">×</button>
          </div>
          <div class="tag-modal-body">
            <p>确定要删除这条内容吗？此操作不可撤销。</p>
          </div>
          <div class="tag-modal-footer">
            <button @click="showDeleteConfirm = false" class="mac-btn">取消</button>
            <button @click="handleDelete" class="mac-btn delete-btn">确定删除</button>
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

.upload-form {
  padding: 20px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  margin-bottom: 24px;
  border: 1px solid rgba(255, 255, 255, 0.36);
}

.upload-progress {
  margin-top: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: rgba(59, 130, 246, 0.15);
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #60a5fa 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
  position: relative;
}

.progress-fill::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.4) 50%,
    transparent 100%
  );
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(100%); }
}

.progress-text {
  font-size: 13px;
  font-weight: 600;
  color: #3b82f6;
  min-width: 45px;
  text-align: right;
}

.primary-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-row {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #333;
  margin-bottom: 8px;
}

.type-buttons {
  display: flex;
  gap: 8px;
}

.type-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.type-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.type-btn.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: white;
}

.type-icon {
  width: 18px;
  height: 18px;
}

.md-toolbar {
  display: flex;
  gap: 4px;
  margin-bottom: 8px;
  padding: 6px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 6px;
}

.md-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  color: #666;
  cursor: pointer;
  transition: all 0.2s ease;
}

.md-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.md-btn:active {
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.1) 0%, rgba(59, 130, 246, 0.05) 100%);
  box-shadow: inset 0 1px 2px rgba(59, 130, 246, 0.1);
}

.md-btn svg {
  width: 16px;
  height: 16px;
}

.textarea {
  width: 100%;
  min-height: 200px;
  resize: vertical;
  font-family: monospace;
  font-size: 14px;
  line-height: 1.6;
  box-sizing: border-box;
  background: white;
}

.file-input {
  padding: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  background: white;
  cursor: pointer;
}

.file-preview {
  margin-top: 12px;
  position: relative;
  display: inline-block;
}

.preview-image {
  max-width: 200px;
  max-height: 200px;
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.preview-remove {
  position: absolute;
  top: -8px;
  right: -8px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #dc2626;
  color: white;
  border: none;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tag-buttons-row {
  margin-top: 8px;
}

.tag-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 20px;
  font-size: 13px;
  color: #3b82f6;
}

.tag-remove {
  cursor: pointer;
  font-weight: bold;
  font-size: 16px;
  line-height: 1;
}

.quick-add-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px dashed rgba(0, 0, 0, 0.2);
  border-radius: 6px;
  color: #888;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-add-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  border-color: rgba(59, 130, 246, 0.3);
  color: #3b82f6;
}

.quick-add-btn svg {
  width: 16px;
  height: 16px;
}

.content-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.content-item {
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

.item-media {
  width: 150px;
  height: 100px;
  flex-shrink: 0;
  overflow: hidden;
  background: #f8f9fa;
  border-radius: 8px;
}

.item-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.item-text-preview {
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

.item-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.item-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 8px 0;
}

.item-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 8px;
}

.item-tag {
  padding: 2px 8px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  font-size: 12px;
  color: #666;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.meta-item {
  font-size: 13px;
  color: #888;
}

.item-actions {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 13px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.action-btn.approve-btn {
  background: rgba(5, 150, 105, 0.1);
  border-color: rgba(5, 150, 105, 0.3);
  color: #059669;
}

.action-btn.approve-btn:hover {
  background: rgba(5, 150, 105, 0.15);
}

.action-btn.reject-btn {
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.3);
  color: #dc2626;
}

.action-btn.reject-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.action-btn.delete-btn {
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.3);
  color: #dc2626;
}

.action-btn.delete-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.action-btn.ban-btn {
  background: rgba(249, 115, 22, 0.1);
  border-color: rgba(249, 115, 22, 0.3);
  color: #f97316;
}

.action-btn.unban-btn {
  background: rgba(5, 150, 105, 0.1);
  border-color: rgba(5, 150, 105, 0.3);
  color: #059669;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #999;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}

.pagination-section {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.pagination-btn {
  display: flex;
  align-items: center;
  gap: 6px;
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination-btn svg {
  width: 14px;
  height: 14px;
}

.pagination-info {
  font-size: 14px;
  color: #666;
}

.user-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.user-avatar svg {
  width: 24px;
  height: 24px;
}

.user-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.user-name {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
}

.user-status {
  display: flex;
  gap: 8px;
}

.admin-badge {
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #3b82f6;
}

.banned-badge {
  padding: 2px 8px;
  background: rgba(220, 38, 38, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #dc2626;
}

.user-actions {
  display: flex;
  gap: 8px;
}

.current-user-hint {
  font-size: 13px;
  color: #888;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
}

.tag-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 99999;
  background: rgba(0, 0, 0, 0.5);
}

.tag-modal {
  width: 90%;
  max-width: 400px;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

.tag-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: linear-gradient(180deg, rgba(0,0,0,0.05) 0%, transparent 100%);
}

.tag-modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.tag-modal-close {
  width: 28px;
  height: 28px;
  border: none;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 50%;
  font-size: 18px;
  color: #666;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.tag-modal-close:hover {
  background: rgba(0, 0, 0, 0.1);
  color: #3b82f6;
}

.tag-modal-body {
  padding: 20px;
}

.tag-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(0, 0, 0, 0.03);
}

.modal-content {
  width: 90%;
  max-width: 600px;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  overflow: hidden;
  position: relative;
  z-index: 10000;
}

.detail-modal {
  max-width: 800px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(180deg, rgba(0,0,0,0.05) 0%, transparent 100%);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.modal-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.modal-close {
  font-size: 24px;
  color: #999;
  cursor: pointer;
  line-height: 1;
}

.modal-close:hover {
  color: #3b82f6;
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.03);
}

.detail-content {
  max-height: 60vh;
  overflow-y: auto;
}

.detail-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 16px;
}

.detail-tag {
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 4px;
  font-size: 13px;
  color: #3b82f6;
}

.detail-media {
  margin-bottom: 16px;
}

.detail-image {
  max-width: 100%;
  border-radius: 8px;
}

.detail-video {
  width: 100%;
  max-height: 400px;
  border-radius: 8px;
}

.detail-text {
  line-height: 1.8;
  color: #333;
}

.detail-text h1,
.detail-text h2,
.detail-text h3 {
  margin-top: 16px;
  margin-bottom: 8px;
}

.detail-text h1 { font-size: 24px; }
.detail-text h2 { font-size: 20px; }
.detail-text h3 { font-size: 16px; }

.detail-text p {
  margin-bottom: 12px;
}

.detail-text ul,
.detail-text ol {
  margin-bottom: 12px;
  padding-left: 24px;
}

.detail-text blockquote {
  border-left: 4px solid #3b82f6;
  padding-left: 12px;
  margin: 12px 0;
  color: #666;
  font-style: italic;
}

.detail-text code {
  background: rgba(0, 0, 0, 0.06);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
  font-size: 14px;
}

.detail-text pre {
  background: #1a1a1a;
  color: #e0e0e0;
  padding: 12px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 12px 0;
}

.detail-text pre code {
  background: none;
  padding: 0;
  color: inherit;
}

.file-path {
  display: block;
  margin-top: 8px;
  font-size: 12px;
  color: #888;
}

@media screen and (max-width: 768px) {
  .admin-container {
    padding: 0;
    min-height: calc(100vh - 60px);
  }

  .admin-window {
    border-radius: 0;
    box-shadow: none;
    min-height: calc(100vh - 60px);
    background: rgba(255, 255, 255, 0.9);
  }

  .mac-title-bar {
    display: none;
  }

  .admin-content {
    padding: 16px;
  }

  .nav-tabs {
    overflow-x: auto;
    flex-wrap: nowrap;
    gap: 6px;
    padding-bottom: 12px;
  }

  .nav-tab {
    padding: 10px 16px;
    font-size: 14px;
    flex-shrink: 0;
  }

  .upload-form {
    padding: 16px;
  }

  .form-row {
    margin-bottom: 14px;
  }

  .form-label {
    font-size: 14px;
    margin-bottom: 6px;
  }

  .type-buttons {
    gap: 6px;
  }

  .type-btn {
    padding: 10px 12px;
    font-size: 13px;
  }

  .md-toolbar {
    gap: 3px;
    padding: 4px;
  }

  .md-btn {
    width: 36px;
    height: 36px;
  }

  .textarea {
    min-height: 150px;
    font-size: 15px;
    padding: 14px;
  }

  .file-input {
    width: 100%;
    padding: 12px;
  }

  .selected-tag {
    padding: 5px 12px;
    font-size: 14px;
  }

  .quick-add-btn {
    width: 36px;
    height: 36px;
  }

  .content-item {
    flex-direction: column;
    gap: 12px;
    padding: 14px;
  }

  .item-media {
    width: 100%;
    height: 150px;
  }

  .item-title {
    font-size: 15px;
  }

  .item-tag {
    padding: 3px 8px;
    font-size: 13px;
  }

  .item-actions {
    flex-wrap: wrap;
    gap: 6px;
  }

  .action-btn {
    padding: 8px 14px;
    font-size: 14px;
    min-width: 70px;
  }

  .section-header {
    margin-bottom: 12px;
  }

  .section-title {
    font-size: 17px;
  }

  .user-item {
    flex-direction: column;
    gap: 12px;
    padding: 14px;
  }

  .user-info {
    width: 100%;
  }

  .user-avatar {
    width: 56px;
    height: 56px;
  }

  .user-avatar svg {
    width: 28px;
    height: 28px;
  }

  .user-name {
    font-size: 16px;
  }

  .user-actions {
    width: 100%;
    flex-wrap: wrap;
    gap: 6px;
  }

  .user-actions .action-btn {
    flex: 1;
    min-width: calc(50% - 3px);
    text-align: center;
  }

  .modal-content {
    width: 95%;
    max-width: none;
    margin: 12px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.98);
  }

  .detail-modal {
    max-width: 95%;
  }

  .modal-body {
    padding: 16px;
  }

  .modal-footer {
    padding: 14px 16px;
    flex-wrap: wrap;
  }

  .modal-footer .mac-btn {
    flex: 1;
    min-width: calc(50% - 6px);
  }

  .pagination-section {
    gap: 10px;
  }

  .pagination-btn {
    min-width: 80px;
    padding: 10px;
    font-size: 14px;
  }

  .pagination-info {
    font-size: 13px;
  }

  .empty-state {
    padding: 30px 16px;
  }

  .empty-icon {
    width: 48px;
    height: 48px;
  }

  .message-bar {
    padding: 12px 14px;
    font-size: 14px;
  }
}

@media screen and (max-width: 480px) {
  .content-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .item-media {
    height: 120px;
  }

  .item-actions {
    flex-direction: column;
  }

  .action-btn {
    width: 100%;
  }
}
</style>


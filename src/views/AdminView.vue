<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { contentApi, adminApi } from '@/api'
import type { Content, User } from '@/types'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { getImageUrl, getPreviewText } from '@/utils'

import AdminContentCard from '@/components/admin/AdminContentCard.vue'
import AdminUserCard from '@/components/admin/AdminUserCard.vue'
import AdminPagination from '@/components/admin/AdminPagination.vue'
import AdminEmptyState from '@/components/admin/AdminEmptyState.vue'
import AdminContentDetailModal from '@/components/admin/AdminContentDetailModal.vue'
import AdminEditModal from '@/components/admin/AdminEditModal.vue'
import AdminDeleteConfirm from '@/components/admin/AdminDeleteConfirm.vue'
import AdminQuickAddTag from '@/components/admin/AdminQuickAddTag.vue'
import AdminUploadForm from '@/components/admin/AdminUploadForm.vue'

const userStore = useUserStore()

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
  return DOMPurify.sanitize(marked(processedText) as string)
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
            <button @click="showUploadModal = true" class="mac-btn primary-btn">
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
              @view="openContentDetail"
              @edit="openEditModal"
              @delete="confirmDelete"
              @regenerate-thumbnail="regenerateThumbnail"
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
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.user-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>

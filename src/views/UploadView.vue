<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { contentApi } from '@/api'
import type { Content } from '@/types'
import AdminUploadForm from '@/components/admin/AdminUploadForm.vue'
import AdminContentCard from '@/components/admin/AdminContentCard.vue'
import AdminContentDetailModal from '@/components/admin/AdminContentDetailModal.vue'
import AdminEditModal from '@/components/admin/AdminEditModal.vue'
import AdminDeleteConfirm from '@/components/admin/AdminDeleteConfirm.vue'
import AdminQuickAddTag from '@/components/admin/AdminQuickAddTag.vue'
import AdminEmptyState from '@/components/admin/AdminEmptyState.vue'
import AdminPagination from '@/components/admin/AdminPagination.vue'

const userStore = useUserStore()

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
  type: 'video' | 'image' | 'text' as const,
  filePath: '',
  tags: [] as string[],
  file: undefined as File | undefined
})
const isEditModal = ref(false)

const myContents = ref<Content[]>([])
const myContentsPage = ref(1)
const myContentsTotal = ref(0)
const myContentsTotalPages = ref(1)

const showQuickAddTag = ref(false)
const quickAddTagName = ref('')
const tagTargetForm = ref<'upload' | 'edit'>('upload')

const showDeleteConfirm = ref(false)
const deleteTargetId = ref(0)

const showContentDetail = ref(false)
const contentDetail = ref<Content | null>(null)

const allTags = computed(() => {
  const tagsSet = new Set<string>()
  myContents.value.forEach(content => {
    const tags = content.tags || content.Tags || []
    tags.forEach(tag => tagsSet.add(String(tag)))
  })
  return Array.from(tagsSet)
})

const imageUploadInput = ref<HTMLInputElement | null>(null)
const editImageUploadInput = ref<HTMLInputElement | null>(null)

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
      loadMyContents()
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
      loadMyContents()
    }
  } catch (error) {
    message.value = '删除失败'
  }
  showDeleteConfirm.value = false
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

onMounted(() => {
  if (userStore.isLoggedIn) {
    loadMyContents()
  }
})
</script>

<template>
  <div class="upload-container">
    <div class="mac-window upload-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">小泉动漫 - 内容上传</div>
      </div>

      <div class="upload-content">
        <div v-if="message" class="message-bar" :class="{ success: message.includes('成功'), error: message.includes('失败') }">
          {{ message }}
          <span class="message-close" @click="message = ''">×</span>
        </div>

        <div class="section-header">
          <h2 class="section-title">上传新内容</h2>
          <button @click="showUploadModal = true" class="mac-btn primary-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
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
            @view="openContentDetail"
            @edit="openEditModal"
            @delete="confirmDelete"
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
    </div>

    <input
      ref="imageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleImageUpload"
    />

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
.upload-container {
  min-height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.upload-window {
  width: 100%;
  max-width: 1000px;
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

.upload-content {
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

.mac-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 8px;
}

.mac-btn:hover {
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.mac-btn svg {
  width: 16px;
  height: 16px;
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

.content-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

@media screen and (max-width: 768px) {
  .upload-container {
    padding: 12px;
  }

  .upload-window {
    border-radius: 12px;
  }

  .upload-content {
    padding: 16px;
  }
}
</style>
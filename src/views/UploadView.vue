<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { contentApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const message = ref('')
const uploadProgress = ref(0)
const isUploading = ref(false)

const uploadForm = ref({
  title: '',
  type: 'image' as 'video' | 'image' | 'text' | 'link',
  content: '',
  url: '',
  tags: [] as string[],
  file: undefined as File | undefined,
  filePath: '',
})

const filePreview = ref<string>('')
const allTags = ref<string[]>([])
const customTagInput = ref('')
const isDragging = ref(false)

const imageUploadInput = ref<HTMLInputElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const agreeUpload = ref(false)

watch(
  () => uploadForm.value.type,
  () => {
    agreeUpload.value = false
    clearFilePreview()
  }
)

function resetForm() {
  agreeUpload.value = false
  uploadForm.value = {
    title: '',
    type: 'image' as 'video' | 'image' | 'text' | 'link',
    content: '',
    url: '',
    tags: [],
    file: undefined,
    filePath: '',
  }
  filePreview.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

function insertMarkdown(prefix: string, suffix: string = '') {
  const textarea = document.querySelector('.upload-textarea') as HTMLTextAreaElement
  if (!textarea) return
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const text = uploadForm.value.content || ''
  uploadForm.value.content =
    text.substring(0, start) + prefix + text.substring(start, end) + suffix + text.substring(end)
  textarea.focus()
  const newPos = start + prefix.length + (end - start) + suffix.length
  setTimeout(() => {
    textarea.setSelectionRange(newPos, newPos)
  }, 0)
}

function triggerImageUpload() {
  imageUploadInput.value?.click()
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
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
  } catch (error) {
    message.value = '上传失败，请稍后重试'
  } finally {
    target.value = ''
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

function removeTag(tag: string) {
  const index = uploadForm.value.tags.indexOf(tag)
  if (index > -1) {
    uploadForm.value.tags.splice(index, 1)
  }
}

function addCustomTag() {
  const tag = customTagInput.value.trim()
  if (!tag) return
  if (uploadForm.value.tags.includes(tag)) {
    message.value = '该标签已存在'
    return
  }
  uploadForm.value.tags.push(tag)
  customTagInput.value = ''
}

function handleCustomTagKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.preventDefault()
    addCustomTag()
  }
}

function handleDragOver(event: DragEvent) {
  event.preventDefault()
  isDragging.value = true
}

function handleDragLeave(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
}

function handleDrop(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
  const files = event.dataTransfer?.files
  if (files && files.length > 0) {
    processFile(files[0])
  }
}

function processFile(file: File) {
  const isImage = uploadForm.value.type === 'image'
  const isVideo = uploadForm.value.type === 'video'

  if (isImage) {
    const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
    if (!validTypes.includes(file.type)) {
      message.value = '只支持 jpg、jpeg、png、gif、webp 格式的图片'
      return
    }
    if (file.size > 10 * 1024 * 1024) {
      message.value = '图片大小不能超过10MB'
      return
    }
  }

  if (isVideo) {
    if (!file.type.startsWith('video/')) {
      message.value = '请上传视频文件'
      return
    }
    if (file.size > 100 * 1024 * 1024) {
      message.value = '视频大小不能超过100MB'
      return
    }
  }

  uploadForm.value.file = file
  const reader = new FileReader()
  reader.onload = (e) => {
    filePreview.value = e.target?.result as string
  }
  reader.readAsDataURL(file)
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    processFile(target.files[0])
  }
}

function clearFilePreview() {
  filePreview.value = ''
  uploadForm.value.file = undefined
  if (fileInputRef.value) fileInputRef.value.value = ''
}

async function loadAllTags() {
  try {
    const res = await contentApi.getTags()
    if (res.code === 200) {
      allTags.value = res.data as string[]
    }
  } catch (error) {
    console.error('加载标签失败', error)
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
      message.value = '上传成功！'
      resetForm()
    } else {
      message.value = res.message || '上传失败'
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } catch (error: any) {
    message.value = `上传失败: ${error.message}`
  } finally {
    isUploading.value = false
    uploadProgress.value = 0
  }
}

onMounted(() => {
  loadAllTags()
})
</script>

<template>
  <div class="upload-container">
    <div class="mac-window upload-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">小泉动漫 - 发布内容</div>
      </div>

      <div class="upload-content">
        <div
          v-if="message"
          class="message-bar"
          :class="{ success: message.includes('成功'), error: message.includes('失败') }"
        >
          {{ message }}
          <span class="message-close" @click="message = ''">×</span>
        </div>

        <div class="form-section">
          <label class="form-label">标题</label>
          <input v-model="uploadForm.title" type="text" class="mac-input" placeholder="输入标题" />
        </div>

        <div class="form-section">
          <label class="form-label">类型</label>
          <div class="radio-group">
            <label class="radio-item">
              <input type="radio" value="text" v-model="uploadForm.type" />
              <span>图文</span>
            </label>
            <label class="radio-item">
              <input type="radio" value="image" v-model="uploadForm.type" />
              <span>图片</span>
            </label>
            <label class="radio-item">
              <input type="radio" value="video" v-model="uploadForm.type" />
              <span>视频</span>
            </label>
            <label class="radio-item">
              <input type="radio" value="link" v-model="uploadForm.type" />
              <span>链接</span>
            </label>
          </div>
        </div>

        <div v-if="uploadForm.type === 'text'" class="form-section">
          <label class="form-label">内容</label>
          <div class="md-toolbar">
            <button type="button" @click="insertMarkdown('## ')" class="md-btn" title="标题">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 12h16M4 6h16M4 18h10" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('**', '**')" class="md-btn" title="粗体">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('*', '*')" class="md-btn" title="斜体">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="19" y1="4" x2="10" y2="4" />
                <line x1="14" y1="20" x2="5" y2="20" />
                <line x1="15" y1="4" x2="9" y2="20" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('- ')" class="md-btn" title="列表">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="8" y1="6" x2="21" y2="6" />
                <line x1="8" y1="12" x2="21" y2="12" />
                <line x1="8" y1="18" x2="21" y2="18" />
                <line x1="3" y1="6" x2="3.01" y2="6" />
                <line x1="3" y1="12" x2="3.01" y2="12" />
                <line x1="3" y1="18" x2="3.01" y2="18" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('`', '`')" class="md-btn" title="代码">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="16 18 22 12 16 6" />
                <polyline points="8 6 2 12 8 18" />
              </svg>
            </button>
            <button type="button" @click="triggerImageUpload" class="md-btn" title="上传图片">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </button>
          </div>
          <textarea
            v-model="uploadForm.content"
            class="mac-input textarea upload-textarea"
            placeholder="支持Markdown格式"
          ></textarea>
        </div>

        <div v-else-if="uploadForm.type !== 'link'" class="form-section">
          <label class="form-label">文件</label>
          <div
            class="file-upload-area"
            :class="{ dragging: isDragging, 'has-file': filePreview }"
            @dragover="handleDragOver"
            @dragleave="handleDragLeave"
            @drop="handleDrop"
            @click="fileInputRef?.click()"
          >
            <div v-if="!filePreview" class="upload-placeholder">
              <div class="upload-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="17 8 12 3 7 8" />
                  <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
              </div>
              <p class="upload-text">
                拖拽{{ uploadForm.type === 'image' ? '图片' : '视频' }}到此处，或<span
                  class="upload-link"
                  >点击上传</span
                >
              </p>
              <p class="upload-hint">
                {{ uploadForm.type === 'image' ? '支持 JPG、PNG、GIF、WebP，最大 10MB' : '支持常见视频格式，最大 100MB' }}
              </p>
            </div>
            <div v-else class="file-preview-wrapper">
              <img v-if="uploadForm.type === 'image'" :src="filePreview" class="preview-image" />
              <video
                v-else-if="uploadForm.type === 'video'"
                :src="filePreview"
                class="preview-video"
                controls
              />
              <button type="button" @click.stop="clearFilePreview" class="clear-preview-btn">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
              <div v-if="uploadForm.file" class="file-info">
                <span class="file-name">{{ uploadForm.file.name }}</span>
                <span class="file-size">{{ (uploadForm.file.size / 1024 / 1024).toFixed(2) }} MB</span>
              </div>
            </div>
          </div>
          <input
            ref="fileInputRef"
            type="file"
            @change="handleFileChange"
            class="hidden-file-input"
            :accept="uploadForm.type === 'image' ? 'image/*' : 'video/*'"
          />
        </div>

        <div v-if="uploadForm.type === 'video'" class="terms-section">
          <label class="terms-label">
            <input type="checkbox" v-model="agreeUpload" />
            <span>自2026年5月14日起，本平台建议使用「链接」类型来代替视频类型，您可以在链接类型中填入您的B站视频链接。若您同意，但您继续上传大于15MB的视频，则视为委托站长通过二创站官方B站账号代为上传，原视频将替换为该B站链接。该官方账号不会产生任何收益，亦不会用于上传其他内容。勾选即表示您已阅读并同意上述条款。</span>
          </label>
        </div>
        <div class="terms-section">
          <div class="terms-text">上传至本站的内容如未另行标注版权信息，默认采用<a href="https://creativecommons.org/licenses/by/4.0/deed.zh-hans" target="_blank" rel="noopener">知识共享署名 4.0 国际许可协议（CC BY 4.0）</a>进行授权。建议您在作品中添加水印或署名以保障自身权益。</div>
        </div>

        <div v-if="uploadForm.type === 'link'" class="form-section">
          <label class="form-label">链接地址</label>
          <input v-model="uploadForm.url" type="url" class="mac-input" placeholder="https://..." />
        </div>

        <div class="form-section">
          <label class="form-label">已选标签</label>
          <div v-if="uploadForm.tags.length > 0" class="selected-tags">
            <span v-for="tag in uploadForm.tags" :key="tag" class="selected-tag">
              {{ tag }}
              <span @click="removeTag(tag)" class="tag-remove">×</span>
            </span>
          </div>
          <div v-else class="empty-tags">点击下方标签进行选择</div>
        </div>

        <div class="form-section">
          <label class="form-label">选择标签</label>
          <div class="custom-tag-input">
            <input
              v-model="customTagInput"
              type="text"
              class="tag-input"
              placeholder="输入自定义标签后按回车"
              @keydown="handleCustomTagKeydown"
            />
            <button @click="addCustomTag" class="add-tag-btn">添加</button>
          </div>
          <div class="all-tags-container">
            <div class="all-tags">
              <span
                v-for="tag in allTags"
                :key="tag"
                :class="['tag-btn', { selected: uploadForm.tags.includes(tag) }]"
                @click="toggleTag(tag)"
                >{{ tag }}</span
              >
            </div>
            <div v-if="allTags.length === 0" class="no-tags">暂无可用标签</div>
          </div>
        </div>

        <div v-if="isUploading" class="form-section">
          <div class="progress-bar-container">
            <div class="progress-bar" :style="{ width: `${uploadProgress}%` }"></div>
            <span class="progress-text">{{ uploadProgress }}%</span>
          </div>
        </div>

        <div class="form-actions">
          <button @click="router.push('/')" class="mac-btn cancel-btn">取消</button>
          <button @click="handleUpload" class="mac-btn primary-btn" :disabled="isUploading || (uploadForm.type === 'video' && uploadForm.file && uploadForm.file.size > 15 * 1024 * 1024 && !agreeUpload)">
            {{ isUploading ? '上传中...' : '发布内容' }}
          </button>
        </div>
      </div>
    </div>

    <input
      ref="imageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleImageUpload"
    />
  </div>
</template>

<style scoped>
.upload-container {
  min-height: calc(100vh - 100px);
  padding: 20px;
  display: flex;
  justify-content: center;
}

.upload-window {
  width: 100%;
  max-width: 800px;
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
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.08) 0%, rgba(0, 0, 0, 0.02) 100%);
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

.terms-section {
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  padding-top: 12px;
  margin-bottom: 4px;
}

.terms-section:last-of-type {
  border-top: none;
  padding-top: 4px;
  margin-bottom: 8px;
}

.terms-label {
  display: flex;
  gap: 8px;
  font-size: 13px;
  color: #666;
  line-height: 1.5;
  cursor: pointer;
  align-items: flex-start;
}

.terms-label input[type="checkbox"] {
  margin-top: 2px;
  flex-shrink: 0;
}

.terms-label:hover {
  color: #333;
}

.terms-text {
  font-size: 13px;
  color: #888;
}

.form-section {
  margin-bottom: 20px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 8px;
}

.mac-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  background: white;
  box-sizing: border-box;
}

.mac-input:focus {
  outline: none;
  border-color: rgba(59, 130, 246, 0.5);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.textarea {
  min-height: 200px;
  resize: vertical;
  font-family: monospace;
  line-height: 1.6;
}

.radio-group {
  display: flex;
  gap: 16px;
}

.radio-item {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 14px;
  color: #333;
}

.radio-item input {
  cursor: pointer;
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
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.md-btn svg {
  width: 16px;
  height: 16px;
}

.file-upload-area {
  border: 2px dashed rgba(0, 0, 0, 0.15);
  border-radius: 12px;
  padding: 32px 24px;
  background: rgba(0, 0, 0, 0.02);
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: center;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.file-upload-area:hover {
  border-color: rgba(59, 130, 246, 0.4);
  background: rgba(59, 130, 246, 0.04);
}

.file-upload-area.dragging {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.08);
  transform: scale(1.01);
}

.file-upload-area.has-file {
  border-style: solid;
  border-color: rgba(59, 130, 246, 0.3);
  background: rgba(59, 130, 246, 0.02);
}

.upload-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.upload-icon {
  width: 48px;
  height: 48px;
  color: #9ca3af;
}

.upload-icon svg {
  width: 100%;
  height: 100%;
}

.upload-text {
  font-size: 14px;
  color: #666;
  margin: 0;
}

.upload-link {
  color: #3b82f6;
  font-weight: 500;
}

.upload-hint {
  font-size: 12px;
  color: #9ca3af;
  margin: 0;
}

.file-preview-wrapper {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.preview-image {
  max-width: 100%;
  max-height: 240px;
  border-radius: 8px;
  object-fit: contain;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.preview-video {
  max-width: 100%;
  max-height: 240px;
  border-radius: 8px;
}

.clear-preview-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 32px;
  height: 32px;
  border: none;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.clear-preview-btn:hover {
  background: rgba(239, 68, 68, 0.9);
  transform: scale(1.1);
}

.clear-preview-btn svg {
  width: 16px;
  height: 16px;
}

.file-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: rgba(59, 130, 246, 0.08);
  border-radius: 8px;
}

.file-name {
  font-size: 13px;
  font-weight: 500;
  color: #333;
}

.file-size {
  font-size: 12px;
  color: #666;
}

.hidden-file-input {
  display: none;
}

.selected-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.selected-tag {
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: 20px;
  font-size: 13px;
  color: #3b82f6;
  display: flex;
  align-items: center;
  gap: 6px;
}

.tag-remove {
  cursor: pointer;
  font-weight: bold;
  opacity: 0.6;
}

.tag-remove:hover {
  opacity: 1;
}

.empty-tags {
  color: #999;
  font-size: 13px;
  padding: 8px 0;
}

.all-tags-container {
  background: rgba(0, 0, 0, 0.02);
  border-radius: 8px;
  padding: 12px;
  max-height: none;
  overflow-y: visible;
}

.all-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-btn {
  padding: 4px 10px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.15);
  border-radius: 20px;
  font-size: 13px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.tag-btn:hover {
  border-color: rgba(59, 130, 246, 0.3);
  color: #3b82f6;
}

.tag-btn.selected {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.3);
  color: #3b82f6;
}

.custom-tag-input {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.tag-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  background: white;
}

.tag-input:focus {
  outline: none;
  border-color: rgba(59, 130, 246, 0.5);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.add-tag-btn {
  padding: 8px 16px;
  background: rgba(59, 130, 246, 0.95);
  color: white;
  border: 1px solid rgba(59, 130, 246, 0.95);
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.add-tag-btn:hover {
  background: rgba(37, 99, 235, 0.95);
  border-color: rgba(37, 99, 235, 0.95);
}

.no-tags {
  color: #999;
  font-size: 13px;
  text-align: center;
  padding: 16px;
}

.progress-bar-container {
  position: relative;
  width: 100%;
  height: 20px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 10px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #8b5cf6);
  border-radius: 10px;
  transition: width 0.3s ease;
}

.progress-text {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 500;
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.mac-btn {
  padding: 10px 20px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
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

.mac-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.primary-btn {
  background: rgba(59, 130, 246, 0.95);
  color: white;
  border-color: rgba(59, 130, 246, 0.95);
}

.primary-btn:hover:not(:disabled) {
  background: rgba(37, 99, 235, 0.95);
  color: white;
  border-color: rgba(37, 99, 235, 0.95);
}

.cancel-btn {
  background: rgba(255, 255, 255, 0.95);
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

  .file-upload-area {
    padding: 24px 16px;
    min-height: 150px;
  }

  .form-actions {
    flex-direction: column;
  }

  .mac-btn {
    width: 100%;
  }
}
</style>

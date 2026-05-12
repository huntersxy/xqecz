<script setup lang="ts">
interface UploadForm {
  title: string
  type: 'video' | 'image' | 'text'
  content: string
  tags: string[]
  file?: File
}

interface Props {
  uploadForm: UploadForm
  filePreview: string
  uploadProgress: number
  isUploading: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  insertMarkdown: [prefix: string, suffix?: string]
  triggerImageUpload: []
  handleImageUpload: [e: Event]
  handleFileChange: [e: Event]
  handleUpload: []
  addTag: [type: string]
  removeTag: [tag: string]
  clearPreview: []
}>()
</script>

<template>
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
        <button type="button" @click="$emit('insertMarkdown', '## ')" class="md-btn" title="标题">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M4 12h16M4 6h16M4 18h10"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '**', '**')" class="md-btn" title="粗体">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
            <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '*', '*')" class="md-btn" title="斜体">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="19" y1="4" x2="10" y2="4"/>
            <line x1="14" y1="20" x2="5" y2="20"/>
            <line x1="15" y1="4" x2="9" y2="20"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '`', '`')" class="md-btn" title="行内代码">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="16 18 22 12 16 6"/>
            <polyline points="8 6 2 12 8 18"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '\n```\n', '\n```\n')" class="md-btn" title="代码块">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <path d="M8 10l4 4-4 4"/>
            <line x1="14" y1="18" x2="18" y2="18"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '[', '](url)')" class="md-btn" title="链接">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
          </svg>
        </button>
        <button type="button" @click="$emit('triggerImageUpload')" class="md-btn" title="上传图片">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
        </button>
        <button type="button" @click="$emit('insertMarkdown', '\n- ')" class="md-btn" title="列表">
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
      <input type="file" @change="$emit('handleFileChange', $event)" class="file-input" />
      <div v-if="filePreview" class="file-preview">
        <img :src="filePreview" alt="文件预览" class="preview-image" />
        <button type="button" @click="$emit('clearPreview')" class="preview-remove">移除</button>
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
            <span class="tag-remove" @click="$emit('removeTag', tag)">×</span>
          </span>
          <button type="button" @click="$emit('addTag', 'upload')" class="quick-add-btn" title="添加标签">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19"/>
              <line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <button @click="$emit('handleUpload')" class="mac-btn primary-btn" :disabled="isUploading">上传</button>
    <div v-if="isUploading" class="upload-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: uploadProgress + '%' }"></div>
      </div>
      <span class="progress-text">{{ uploadProgress }}%</span>
    </div>
  </div>
</template>

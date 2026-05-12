<script setup lang="ts">
interface EditForm {
  id: number
  title: string
  content: string
  type: 'video' | 'image' | 'text'
  filePath: string
  tags: string[]
  file?: File
}

interface Props {
  visible: boolean
  editForm: EditForm
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  save: []
  insertMarkdown: [prefix: string, suffix?: string]
  triggerImageUpload: []
  handleImageUpload: [e: Event]
  handleFileChange: [e: Event]
  addTag: [type: string]
  removeTag: [tag: string]
}>()
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <input
      ref="editImageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="$emit('handleImageUpload', $event)"
    />
    <div class="modal-content">
      <div class="modal-header">
        <h2>编辑内容</h2>
        <span class="modal-close" @click="$emit('close')">×</span>
      </div>
      <div class="modal-body">
        <div class="form-row">
          <label class="form-label">标题</label>
          <input v-model="editForm.title" type="text" class="mac-input" />
        </div>
        <div v-if="editForm.type === 'text'" class="form-row">
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
            <button type="button" @click="$emit('triggerImageUpload')" class="md-btn" title="上传图片">
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
          <input type="file" @change="$emit('handleFileChange', $event)" class="file-input" />
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
                <span class="tag-remove" @click="$emit('removeTag', tag)">×</span>
              </span>
              <button type="button" @click="$emit('addTag', 'edit')" class="quick-add-btn" title="添加标签">
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
        <button @click="$emit('close')" class="mac-btn">取消</button>
        <button @click="$emit('save')" class="mac-btn primary-btn">保存</button>
      </div>
    </div>
  </div>
</template>

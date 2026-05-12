<script setup lang="ts">
import { computed } from 'vue';

interface UploadForm {
  title: string;
  content: string;
  type: 'video' | 'image' | 'text';
  filePath: string;
  tags: string[];
  file?: File;
}

interface Props {
  visible: boolean;
  uploadForm: UploadForm;
  allTags: string[];
  filePreview?: string;
  uploadProgress?: number;
  isUploading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  allTags: () => [],
  filePreview: '',
  uploadProgress: 0,
  isUploading: false,
});

const mergedTags = computed(() => {
  const tagSet = new Set([...props.allTags, ...props.uploadForm.tags]);
  return Array.from(tagSet);
});

const emit = defineEmits<{
  close: [];
  submit: [];
  insertMarkdown: [prefix: string, suffix?: string];
  triggerImageUpload: [];
  handleImageUpload: [e: Event];
  handleFileChange: [e: Event];
  addTag: [target: string];
  removeTag: [tag: string];
  toggleTag: [tag: string];
  clearPreview: [];
}>();
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>上传内容</h2>
        <span class="modal-close" @click="$emit('close')">×</span>
      </div>
      <div class="modal-body">
        <div class="form-row">
          <label class="form-label">标题</label>
          <input v-model="uploadForm.title" type="text" class="mac-input" placeholder="输入标题" />
        </div>
        <div class="form-row">
          <label class="form-label">类型</label>
          <div class="radio-group">
            <label class="radio-item">
              <input type="radio" value="text" v-model="uploadForm.type" />
              <span>文字</span>
            </label>
            <label class="radio-item">
              <input type="radio" value="image" v-model="uploadForm.type" />
              <span>图片</span>
            </label>
            <label class="radio-item">
              <input type="radio" value="video" v-model="uploadForm.type" />
              <span>视频</span>
            </label>
          </div>
        </div>
        <div v-if="uploadForm.type === 'text'" class="form-row">
          <label class="form-label">内容</label>
          <div class="md-toolbar">
            <button type="button" @click="$emit('insertMarkdown', '## ')" class="md-btn" title="标题">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 12h16M4 6h16M4 18h10" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '**', '**')" class="md-btn" title="粗体">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '*', '*')" class="md-btn" title="斜体">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="19" y1="4" x2="10" y2="4" />
                <line x1="14" y1="20" x2="5" y2="20" />
                <line x1="15" y1="4" x2="9" y2="20" />
              </svg>
            </button>
            <button type="button" @click="$emit('triggerImageUpload')" class="md-btn" title="上传图片">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </button>
          </div>
          <textarea v-model="uploadForm.content" class="mac-input textarea upload-textarea" placeholder="支持Markdown"></textarea>
        </div>
        <div v-else class="form-row">
          <label class="form-label">文件</label>
          <input type="file" @change="$emit('handleFileChange', $event)" class="file-input" :accept="uploadForm.type === 'image' ? 'image/*' : 'video/*'" />
          <div v-if="filePreview" class="file-preview">
            <img v-if="uploadForm.type === 'image'" :src="filePreview" class="preview-image" />
            <video v-else-if="uploadForm.type === 'video'" :src="filePreview" class="preview-video" controls />
            <button type="button" @click="$emit('clearPreview')" class="clear-preview-btn">×</button>
          </div>
        </div>
        <div class="form-row">
          <label class="form-label">标签</label>
          <div class="tag-buttons-row">
            <div class="tag-buttons">
              <span v-for="tag in mergedTags" :key="tag"
                :class="['tag-btn', { selected: uploadForm.tags.includes(tag) }]"
                @click="$emit('toggleTag', tag)">{{ tag }}</span>
              <button type="button" @click="$emit('addTag', 'upload')" class="quick-add-btn" title="添加标签">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>
          </div>
        </div>
        <div v-if="isUploading" class="form-row">
          <div class="progress-bar-container">
            <div class="progress-bar" :style="{ width: `${uploadProgress}%` }"></div>
            <span class="progress-text">{{ uploadProgress }}%</span>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button @click="$emit('close')" class="mac-btn" :disabled="isUploading">取消</button>
        <button @click="$emit('submit')" class="mac-btn primary-btn" :disabled="isUploading">
          {{ isUploading ? '上传中...' : '提交' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
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

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.05) 0%, transparent 100%);
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
  position: relative;
  margin-top: 12px;
  border-radius: 8px;
  overflow: hidden;
}

.preview-image {
  width: 100%;
  max-height: 300px;
  object-fit: contain;
}

.preview-video {
  width: 100%;
  max-height: 300px;
}

.clear-preview-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border: none;
  background: rgba(0, 0, 0, 0.6);
  color: white;
  border-radius: 50%;
  cursor: pointer;
  font-size: 18px;
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

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.03);
}

@media screen and (max-width: 768px) {
  .modal-content {
    width: 95%;
    max-width: none;
    margin: 12px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.98);
  }

  .modal-body {
    padding: 16px;
  }

  .modal-footer {
    padding: 14px 16px;
    flex-wrap: wrap;
  }
}
</style>
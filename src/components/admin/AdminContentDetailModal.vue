<script setup lang="ts">
import { computed } from 'vue';
import { getImageUrl, renderMarkdown } from '@/utils';
import LazyImage from '@/components/LazyImage.vue';
import type { Content } from '@/types';

interface Props {
  content: Content | null;
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  close: [];
}>();

if (!props.content) {
  console.warn('AdminContentDetailModal: content is null');
}

const contentType = props.content?.type || props.content?.Type || 'text';
const auditStatus = props.content?.audit_status || props.content?.AuditStatus || 'pending';
const tags = props.content?.tags || props.content?.Tags || [];
const title = props.content?.title || props.content?.Title || '';
const contentText = props.content?.content || props.content?.Content || '';
const authorName = props.content?.user?.username || props.content?.User?.Username || '';

const renderedContent = computed(() => {
  return renderMarkdown(contentText);
});
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content detail-modal">
      <div class="modal-header">
        <h2>{{ title }}</h2>
        <span class="modal-close" @click="$emit('close')">×</span>
      </div>
      <div class="modal-body">
        <div class="detail-content">
          <div class="detail-meta">
            <span :class="['type-badge', contentType]">
              {{ contentType === 'video' ? '视频' : contentType === 'image' ? '图片' : '文字' }}
            </span>
            <span :class="['status-badge', auditStatus]">
              {{ auditStatus === 'approved' ? '已通过' : auditStatus === 'pending' ? '审核中' : '已拒绝' }}
            </span>
            <span class="meta-item">{{ authorName }}</span>
          </div>
          <div class="detail-tags">
            <span v-for="tag in tags" :key="tag" class="detail-tag">{{ tag }}</span>
          </div>
          <div v-if="contentType === 'image'" class="detail-media">
            <LazyImage :src="getImageUrl(content.image, content.file_path || content.FilePath)" alt="内容图片" class="detail-image" />
          </div>
          <div v-else-if="contentType === 'video'" class="detail-media">
            <video :src="getImageUrl(undefined, content.file_path || content.FilePath)" controls class="detail-video">
              您的浏览器不支持视频播放
            </video>
          </div>
          <div v-else class="detail-text" v-html="renderedContent"></div>
        </div>
      </div>
      <div class="modal-footer">
        <button @click="$emit('close')" class="mac-btn">关闭</button>
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

.detail-modal {
  max-width: 800px;
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

.type-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.type-badge.text {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.type-badge.image {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.type-badge.video {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
}

.status-badge.approved {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.status-badge.pending {
  background: rgba(251, 191, 36, 0.1);
  color: #fbbf24;
}

.status-badge.rejected {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.meta-item {
  font-size: 13px;
  color: #888;
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

.detail-text h1 {
  font-size: 24px;
}

.detail-text h2 {
  font-size: 20px;
}

.detail-text h3 {
  font-size: 16px;
}

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

@media screen and (max-width: 768px) {
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
}
</style>
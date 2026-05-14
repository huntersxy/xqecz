<script setup lang="ts">
import { getImageUrl, getPreviewText } from '@/utils';
import type { Content } from '@/types';

interface Props {
  content: Content;
  showActions?: boolean;
  showAuditActions?: boolean;
  showAuthor?: boolean;
  showRegenerateThumbnail?: boolean;
  showChangeAuthor?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true,
  showAuditActions: false,
  showAuthor: false,
  showRegenerateThumbnail: false,
  showChangeAuthor: false,
});

const emit = defineEmits<{
  view: [content: Content];
  edit: [content: Content];
  delete: [id: number];
  regenerateThumbnail: [id: number];
  audit: [id: number, status: 'approved' | 'rejected'];
  changeAuthor: [content: Content];
}>();

const contentType = props.content.type || 'text';
const tags = props.content.tags || [];
const title = props.content.title || '';
const contentText = props.content.text || '';
const authorName = props.content.user?.username || '';
const viewCount = props.content.view_count || 0;

const contentId = props.content.id || 0;

const handleView = () => emit('view', props.content);
const handleEdit = () => emit('edit', props.content);
const handleDelete = () => emit('delete', contentId);
const handleRegenerateThumbnail = () => emit('regenerateThumbnail', contentId);
const handleApprove = () => emit('audit', contentId, 'approved');
const handleReject = () => emit('audit', contentId, 'rejected');
const handleChangeAuthor = () => emit('changeAuthor', props.content);
</script>

<template>
  <div class="content-item mac-card">
    <div class="item-media">
        <template v-if="contentType !== 'text'">
          <img
            :src="getImageUrl(content.thumb)"
            :alt="contentType === 'video' ? '视频封面' : '内容图片'"
            class="item-image"
            loading="lazy"
          />
        </template>
      <template v-else>
        <div class="item-text-preview">{{ getPreviewText(contentText) }}</div>
      </template>
    </div>
    <div class="item-info">
      <h3 class="item-title">{{ title }}</h3>
      <div class="item-tags">
        <span v-for="tag in tags" :key="tag" class="item-tag">{{ tag }}</span>
      </div>
      <div class="item-meta">
        <span :class="['type-badge', contentType]">
          {{ contentType === 'video' ? '视频' : contentType === 'image' ? '图片' : contentType === 'link' ? '链接' : '文字' }}
        </span>
        <span class="meta-item">{{ viewCount }} 次浏览</span>
        <span v-if="showAuthor" class="meta-item">{{ authorName }}</span>
      </div>
      <div v-if="showActions || showAuditActions" class="item-actions">
        <template v-if="showActions">
          <button @click="handleView" class="action-btn">查看</button>
          <button @click="handleEdit" class="action-btn">编辑</button>
          <button
            v-if="showRegenerateThumbnail && contentType === 'video'"
            @click="handleRegenerateThumbnail"
            class="action-btn"
          >
            更新封面
          </button>
          <button v-if="showChangeAuthor" @click="handleChangeAuthor" class="action-btn">修改作者</button>
          <button @click="handleDelete" class="action-btn delete-btn">删除</button>
        </template>
        <template v-if="showAuditActions">
          <button @click="handleApprove" class="action-btn approve-btn">通过</button>
          <button @click="handleReject" class="action-btn reject-btn">拒绝</button>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.content-item {
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.item-media {
  width: 100%;
  height: 160px;
  overflow: hidden;
  background: #f8f9fa;
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
  padding: 12px;
}

.item-title {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 8px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.type-badge.link {
  background: rgba(139, 92, 246, 0.1);
  color: #7c3aed;
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

@media screen and (max-width: 768px) {
  .item-media {
    height: 150px;
  }

  .item-info {
    padding: 10px;
  }

  .item-title {
    font-size: 14px;
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
}
</style>
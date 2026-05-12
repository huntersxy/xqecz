<script setup lang="ts">
import { getImageUrl, getPreviewText } from '@/utils'
import type { Content } from '@/types'

interface Props {
  content: Content
  showActions?: boolean
  showAuditActions?: boolean
  showAuthor?: boolean
  showRegenerateThumbnail?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showActions: true,
  showAuditActions: false,
  showAuthor: false,
  showRegenerateThumbnail: false,
})

const emit = defineEmits<{
  view: [content: Content]
  edit: [content: Content]
  delete: [id: number]
  regenerateThumbnail: [id: number]
  audit: [id: number, status: 'approved' | 'rejected']
}>()

const contentType = props.content.type || props.content.Type || 'text'
const auditStatus = props.content.audit_status || props.content.AuditStatus || 'pending'
const tags = props.content.tags || props.content.Tags || []
const title = props.content.title || props.content.Title || ''
const contentText = props.content.content || props.content.Content || ''
const authorName = props.content.user?.username || props.content.User?.Username || ''

const contentId = props.content.id || props.content.ID || 0

const handleView = () => emit('view', props.content)
const handleEdit = () => emit('edit', props.content)
const handleDelete = () => emit('delete', contentId)
const handleRegenerateThumbnail = () => emit('regenerateThumbnail', contentId)
const handleApprove = () => emit('audit', contentId, 'approved')
const handleReject = () => emit('audit', contentId, 'rejected')
</script>

<template>
  <div class="content-item mac-card">
    <div class="item-media">
      <template v-if="contentType === 'image'">
        <img
          :src="getImageUrl(content.image, content.file_path || content.FilePath)"
          alt="内容图片"
          class="item-image"
        />
      </template>
      <template v-else-if="contentType === 'video'">
        <img
          :src="getImageUrl(content.image, content.thumb_path || content.file_path || content.FilePath)"
          alt="视频封面"
          class="item-image"
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
          {{ contentType === 'video' ? '视频' : contentType === 'image' ? '图片' : '文字' }}
        </span>
        <span v-if="auditStatus" :class="['status-badge', auditStatus]">
          {{ auditStatus === 'approved' ? '已通过' : auditStatus === 'pending' ? '审核中' : '已拒绝' }}
        </span>
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
          <button @click="handleDelete" class="action-btn delete-btn">删除</button>
        </template>
        <template v-if="showAuditActions">
          <button @click="handleView" class="action-btn">预览</button>
          <button @click="handleApprove" class="action-btn approve-btn">通过</button>
          <button @click="handleReject" class="action-btn reject-btn">拒绝</button>
        </template>
      </div>
    </div>
  </div>
</template>

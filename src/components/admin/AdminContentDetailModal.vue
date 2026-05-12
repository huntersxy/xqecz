<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { getImageUrl } from '@/utils'
import type { Content } from '@/types'

interface Props {
  content: Content
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
}>()

const contentType = props.content.type || props.content.Type || 'text'
const auditStatus = props.content.audit_status || props.content.AuditStatus || 'pending'
const tags = props.content.tags || props.content.Tags || []
const title = props.content.title || props.content.Title || ''
const contentText = props.content.content || props.content.Content || ''
const authorName = props.content.user?.username || props.content.User?.Username || ''

const renderedContent = computed(() => {
  return DOMPurify.sanitize(marked(contentText) as string)
})
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
            <img :src="getImageUrl(content.image, content.file_path || content.FilePath)" alt="内容图片" class="detail-image" />
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

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

function onImageLoad(e: Event) {
  const img = e.target as HTMLImageElement
  if (img.naturalHeight > img.naturalWidth * 1.2) {
    img.style.objectPosition = '50% 8%'
  }
}

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
  <div class="flex flex-col bg-[var(--theme-card-bg)] border border-white/30 rounded-xl shadow-md overflow-hidden">
    <div class="relative w-full pt-[75%] bg-[var(--theme-hover-bg)] overflow-hidden">
      <template v-if="contentType !== 'text'">
        <img
          :src="getImageUrl(content.thumb)"
          :alt="contentType === 'video' ? '视频封面' : '内容图片'"
          class="absolute top-0 left-0 w-full h-full object-cover"
          loading="lazy"
          @load="onImageLoad"
        />
      </template>
      <template v-else>
        <div class="absolute top-0 left-0 w-full h-full p-2 text-sm theme-text-secondary overflow-hidden line-clamp-5">
          {{ getPreviewText(contentText) }}
        </div>
      </template>
    </div>
    <div class="flex-1 flex flex-col justify-between p-3">
      <h3 class="text-[15px] font-semibold theme-text mb-2 truncate">{{ title }}</h3>
      <div class="flex flex-wrap gap-1 mb-2">
        <span v-for="tag in tags" :key="tag" class="px-2 py-0.5 bg-[var(--theme-hover-bg)] rounded text-xs theme-text-secondary">{{ tag }}</span>
      </div>
      <div class="flex items-center gap-2 mb-2">
        <span v-if="contentType === 'text'" class="px-2 py-0.5 rounded text-xs bg-[var(--theme-primary)]/10 text-[var(--theme-primary)]">文字</span>
        <span v-else-if="contentType === 'image'" class="px-2 py-0.5 rounded text-xs bg-[var(--theme-success)]/10 text-[var(--theme-success)]">图片</span>
        <span v-else-if="contentType === 'video'" class="px-2 py-0.5 rounded text-xs bg-[var(--theme-danger)]/10 text-[var(--theme-danger)]">视频</span>
        <span v-else-if="contentType === 'link'" class="px-2 py-0.5 rounded text-xs bg-[var(--theme-secondary)]/10 text-[var(--theme-secondary)]">链接</span>
        <span class="text-[13px] theme-text-secondary">{{ viewCount }} 次浏览</span>
        <span v-if="showAuthor" class="text-[13px] theme-text-secondary">{{ authorName }}</span>
      </div>
      <div v-if="showActions || showAuditActions" class="flex flex-wrap gap-2">
        <template v-if="showActions">
          <button @click="handleView" class="px-3 py-1.5 bg-[var(--theme-surface)] border border-gray-200 rounded-md text-[13px] theme-text hover:text-[var(--theme-primary)] hover:border-blue-300 transition-colors">查看</button>
          <button @click="handleEdit" class="px-3 py-1.5 bg-[var(--theme-surface)] border border-gray-200 rounded-md text-[13px] theme-text hover:text-[var(--theme-primary)] hover:border-blue-300 transition-colors">编辑</button>
          <button v-if="showRegenerateThumbnail && contentType === 'video'" @click="handleRegenerateThumbnail" class="px-3 py-1.5 bg-[var(--theme-surface)] border border-gray-200 rounded-md text-[13px] theme-text hover:text-[var(--theme-primary)] hover:border-blue-300 transition-colors">更新封面</button>
          <button v-if="showChangeAuthor" @click="handleChangeAuthor" class="px-3 py-1.5 bg-[var(--theme-surface)] border border-gray-200 rounded-md text-[13px] theme-text hover:text-[var(--theme-primary)] hover:border-blue-300 transition-colors">修改作者</button>
          <button @click="handleDelete" class="px-3 py-1.5 bg-[var(--theme-danger)]/10/50 border border-red-200 rounded-md text-[13px] text-[var(--theme-danger)] hover:bg-[var(--theme-danger)]/10 transition-colors">删除</button>
        </template>
        <template v-if="showAuditActions">
          <button @click="handleApprove" class="px-3 py-1.5 bg-[var(--theme-success)]/10/50 border border-emerald-200 rounded-md text-[13px] text-[var(--theme-success)] hover:bg-[var(--theme-success)]/10 transition-colors">通过</button>
          <button @click="handleReject" class="px-3 py-1.5 bg-[var(--theme-danger)]/10/50 border border-red-200 rounded-md text-[13px] text-[var(--theme-danger)] hover:bg-[var(--theme-danger)]/10 transition-colors">拒绝</button>
        </template>
      </div>
    </div>
  </div>
</template>

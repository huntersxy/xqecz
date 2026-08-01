<script setup lang="ts">
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { formatTime } from '@/utils'
import {
  IconUser, IconMore, IconReply, IconDelete, IconExclamationCircle,
} from '@arco-design/web-vue/es/icon'
import type { Comment } from '@/types'

interface Props {
  comment: Comment
  replyTarget: Comment | null
  level?: number
}

const props = withDefaults(defineProps<Props>(), {
  level: 0,
})

const emit = defineEmits<{
  'select-reply': [comment: Comment]
  'delete-comment': [id: number]
  'report-comment': [comment: Comment]
}>()

const userStore = useUserStore()

// 仅本人或管理员可删除/举报
const canManage = computed(() =>
  userStore.isLoggedIn &&
  (userStore.user?.is_admin || props.comment.user_id === userStore.user?.id),
)

function onMenuSelect(value: string | number | Record<string, unknown> | undefined) {
  if (value === 'delete') emit('delete-comment', props.comment.id)
  if (value === 'report') emit('report-comment', props.comment)
}
</script>

<template>
  <div class="cd-comment">
    <a-avatar :size="36" class="cd-comment-avatar">
      <IconUser />
    </a-avatar>

    <div class="cd-comment-main">
      <div class="cd-comment-head">
        <span class="cd-comment-username">{{ comment.user?.username }}</span>
        <span class="cd-comment-time">{{ formatTime(comment.created_at) }}</span>

        <a-dropdown v-if="canManage" trigger="click" position="br" @select="onMenuSelect">
          <a-button type="text" size="small" class="cd-comment-more" aria-label="更多操作">
            <IconMore />
          </a-button>
          <template #content>
            <a-doption value="delete">
              <template #icon><IconDelete /></template>
              删除
            </a-doption>
            <a-doption value="report">
              <template #icon><IconExclamationCircle /></template>
              举报
            </a-doption>
          </template>
        </a-dropdown>
      </div>

      <div class="cd-comment-body">
        <div v-if="comment.parent" class="cd-comment-quote">
          <span class="cd-comment-quote-user">{{ comment.parent.user?.username }}: </span>{{ comment.parent.text }}
        </div>
        <span>{{ comment.text }}</span>
      </div>

      <div class="cd-comment-actions">
        <a-button
          v-if="userStore.isLoggedIn"
          type="text"
          size="small"
          class="cd-comment-reply"
          @click="emit('select-reply', comment)"
        >
          <IconReply />
          回复
        </a-button>
      </div>

      <div v-if="comment.replies && comment.replies.length > 0" class="cd-comment-replies">
        <CommentItem
          v-for="reply in comment.replies"
          :key="reply.id"
          :comment="reply"
          :reply-target="replyTarget"
          :level="level + 1"
          @select-reply="(c) => emit('select-reply', c)"
          @delete-comment="(id) => emit('delete-comment', id)"
          @report-comment="(c) => emit('report-comment', c)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.cd-comment {
  display: flex;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 10px;
  background: var(--color-card);
  border: 1px solid var(--color-border);
}

.cd-comment-avatar {
  flex-shrink: 0;
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  color: var(--color-primary);
}

.cd-comment-main {
  flex: 1;
  min-width: 0;
}

.cd-comment-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.cd-comment-username {
  font-weight: 600;
  font-size: 13px;
  color: var(--color-text);
}

.cd-comment-time {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.cd-comment-more {
  margin-left: auto;
  color: var(--color-text-secondary);
}

.cd-comment-body {
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text);
}

.cd-comment-quote {
  margin-bottom: 4px;
  padding: 6px 8px;
  border-left: 2px solid var(--color-primary);
  border-radius: 6px;
  background: var(--color-hover);
  font-size: 12px;
  color: var(--color-text-secondary);
}

.cd-comment-quote-user {
  font-weight: 500;
}

.cd-comment-reply {
  color: var(--color-text-secondary);
}

.cd-comment-reply:hover {
  color: var(--color-primary);
}

.cd-comment-replies {
  margin-top: 8px;
  padding-left: 10px;
  border-left: 2px solid color-mix(in srgb, var(--color-primary) 20%, transparent);
}
</style>

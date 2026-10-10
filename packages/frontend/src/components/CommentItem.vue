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
  <div class="cd-comment" :class="{ 'is-reply-target': replyTarget?.id === comment.id }">
    <a-avatar :size="36" class="cd-comment-avatar">
      <IconUser />
    </a-avatar>

    <div class="cd-comment-main">
      <div class="cd-comment-head">
        <span class="cd-comment-username">{{ comment.user?.username }}</span>
        <span class="cd-comment-time">{{ formatTime(comment.created_at) }}</span>

        <a-dropdown v-if="canManage" trigger="click" position="br" popup-container=".cd-root" @select="onMenuSelect">
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
  position: relative; padding: 13px 14px;
  border-radius: 13px;
  background: var(--creative-paper);
  border: 1px solid var(--creative-line);
  box-shadow: 0 7px 22px color-mix(in srgb, var(--creative-accent) 4%, transparent);
  transition: border-color .18s, box-shadow .18s, transform .18s;
}

.cd-comment:hover { border-color: color-mix(in srgb, var(--creative-accent) 18%, var(--creative-line)); box-shadow: 0 10px 26px color-mix(in srgb, var(--creative-accent) 7%, transparent); }

.cd-comment-avatar {
  flex-shrink: 0; background: linear-gradient(145deg, var(--creative-soft), var(--creative-paper));
  border: 1px solid color-mix(in srgb, var(--creative-accent) 14%, var(--creative-line)); color: var(--creative-accent);
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
  font-weight: 650;
  font-size: 13px;
  color: var(--creative-ink);
}

.cd-comment-time {
  font-size: 11px;
  color: var(--creative-muted);
}

.cd-comment-more {
  margin-left: auto; width: 28px; height: 28px; border-radius: 50%; color: var(--creative-muted);
}
.cd-comment-more:hover { color: var(--creative-accent); background: var(--creative-soft); }

.cd-comment-body {
  font-size: 13px;
  line-height: 1.7;
  color: var(--creative-ink);
}

.cd-comment-quote {
  margin-bottom: 7px; padding: 7px 9px;
  border-left: 2px solid color-mix(in srgb, var(--creative-accent) 55%, transparent);
  border-radius: 7px; background: var(--creative-soft);
  font-size: 11px;
  color: var(--creative-muted);
}

.cd-comment-quote-user {
  font-weight: 500;
}

.cd-comment-actions { margin-top: 8px; }
.cd-comment-reply {
  height: 27px; padding-inline: 8px; border-radius: 999px; color: var(--creative-muted);
}
.cd-comment-reply:hover { color: var(--creative-accent); background: var(--creative-soft); }

.cd-comment-replies {
  margin-top: 10px; padding: 2px 0 0 10px;
  border-left: 1px solid color-mix(in srgb, var(--creative-accent) 20%, transparent);
}
.cd-comment.is-reply-target { border-color: var(--creative-accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.cd-comment-body { overflow-wrap: anywhere; }
</style>

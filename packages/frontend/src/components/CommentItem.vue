<script setup lang="ts">
import { computed } from 'vue'
import { useUserStore } from '@/stores/user'
import { formatTime } from '@/utils'
import UserAvatar from '@/components/UserAvatar.vue'
import {
  IconMore, IconReply, IconDelete, IconExclamationCircle,
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
  <div class="cd-comment" :class="{ 'is-reply-target': replyTarget?.id === comment.id, 'cd-comment-child': level > 0 }">
    <UserAvatar :src="comment.user?.avatar_url" :username="comment.user?.username" :size="level > 0 ? 26 : 32" class="cd-comment-avatar" />

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
  display: flex; align-items: flex-start; gap: 10px; position: relative;
  padding: 18px 0; border-bottom: 1px solid var(--creative-line);
}
.cd-comment:first-child { padding-top: 0; }
.cd-comment:last-child { border-bottom: 0; padding-bottom: 0; }
.cd-comment-avatar { flex-shrink: 0; background: var(--creative-soft); border: 1px solid var(--creative-line); color: var(--creative-accent); }
.cd-comment-main { flex: 1; min-width: 0; }
.cd-comment-head { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 3px 8px; margin-bottom: 9px; }
.cd-comment-username { font-size: 13px; font-weight: 650; color: var(--creative-ink); overflow-wrap: anywhere; }
.cd-comment-time { grid-column: 1; grid-row: 2; font-size: 10px; line-height: 1.5; color: var(--creative-muted); }
.cd-comment-more { grid-column: 2; grid-row: 1 / 3; width: 26px; height: 26px; border-radius: 7px; color: var(--creative-muted); }
.cd-comment-more:hover { color: var(--creative-accent); background: var(--creative-soft); }
.cd-comment-body { font-size: 13px; line-height: 1.8; color: var(--creative-ink); overflow-wrap: anywhere; white-space: pre-wrap; }
.cd-comment-quote { margin-bottom: 8px; padding: 6px 9px; border-left: 2px solid color-mix(in srgb, var(--creative-accent) 45%, var(--creative-line)); border-radius: 0 6px 6px 0; background: var(--creative-soft); font-size: 11px; line-height: 1.6; color: var(--creative-muted); max-height: 5.2em; overflow-y: auto; }
.cd-comment-quote-user { font-weight: 500; color: var(--creative-accent); }
.cd-comment-actions { display: flex; margin-top: 6px; }
.cd-comment-reply { height: 26px; padding-inline: 0; border-radius: 6px; color: var(--creative-muted); font-size: 11px; }
.cd-comment-reply:hover { color: var(--creative-accent); background: var(--creative-soft); }
.cd-comment-replies { display: flex; flex-direction: column; gap: 14px; margin-top: 14px; padding-left: 12px; border-left: 2px solid var(--creative-line); }
.cd-comment-child { padding: 0; border: 0; gap: 8px; }
.cd-comment-child .cd-comment-head { margin-bottom: 6px; }
.cd-comment-child .cd-comment-username { font-size: 12px; }
.cd-comment.is-reply-target > .cd-comment-main > .cd-comment-head .cd-comment-username { color: var(--creative-accent); }
.cd-comment.is-reply-target > .cd-comment-avatar { border-color: var(--creative-accent); box-shadow: 0 0 0 3px var(--creative-soft); }
</style>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { formatTime } from '@/utils'
import UserAvatar from '@/components/UserAvatar.vue'
import SiteIcon from '@/components/SiteIcon.vue'
import type { Comment } from '@/types'

const props = withDefaults(defineProps<{
  comment: Comment
  replyTarget: Comment | null
  level?: number
}>(), { level: 0 })

const emit = defineEmits<{
  'select-reply': [comment: Comment]
  'delete-comment': [id: number]
  'report-comment': [comment: Comment]
}>()

const userStore = useUserStore()

const canManage = computed(() =>
  userStore.isLoggedIn &&
  (userStore.user?.is_admin || props.comment.user_id === userStore.user?.id),
)

const confirming = ref(false)
function onDelete() {
  if (!confirming.value) { confirming.value = true; return }
  confirming.value = false
  emit('delete-comment', props.comment.id)
}
</script>

<template>
  <div class="cd-comment" :class="{ 'is-reply-target': replyTarget?.id === comment.id, 'cd-comment-child': level > 0 }">
    <UserAvatar :src="comment.user?.avatar_url" :username="comment.user?.username" :size="level > 0 ? 26 : 32" class="cd-comment-avatar" />

    <div class="cd-comment-main">
      <div class="cd-comment-head">
        <span class="cd-comment-username">{{ comment.user?.username }}</span>
        <span class="cd-comment-time">{{ formatTime(comment.created_at) }}</span>
      </div>

      <div class="cd-comment-body">
        <div v-if="comment.parent" class="cd-comment-quote">
          <span class="cd-comment-quote-user">{{ comment.parent.user?.username }}: </span>{{ comment.parent.text }}
        </div>
        <span>{{ comment.text }}</span>
      </div>

      <div class="cd-comment-actions">
        <button v-if="userStore.isLoggedIn" type="button" class="cd-comment-act cd-comment-reply" @click="emit('select-reply', comment)">
          <SiteIcon name="reply" />回复
        </button>
        <template v-if="canManage">
          <button type="button" class="cd-comment-act cd-comment-delete" :class="{ 'is-danger': confirming }" @click="onDelete">
            <SiteIcon name="trash" />{{ confirming ? '确认删除？' : '删除' }}
          </button>
          <button v-if="confirming" type="button" class="cd-comment-act" @click="confirming = false">取消</button>
          <button v-else type="button" class="cd-comment-act cd-comment-report" @click="emit('report-comment', comment)">
            <SiteIcon name="flag" />举报
          </button>
        </template>
      </div>

      <div v-if="comment.replies && comment.replies.length > 0" class="cd-comment-replies">
        <CommentNode
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
.cd-comment { display: flex; align-items: flex-start; gap: 10px; padding: 18px 0; border-bottom: 1px solid var(--creative-line); }
.cd-comment:first-child { padding-top: 0; }
.cd-comment:last-child { border-bottom: 0; padding-bottom: 0; }
.cd-comment-avatar { flex-shrink: 0; background: var(--creative-soft); border: 1px solid var(--creative-line); color: var(--creative-accent); }
.cd-comment-main { flex: 1; min-width: 0; }
.cd-comment-head { display: flex; flex-direction: column; gap: 3px; margin-bottom: 9px; }
.cd-comment-username { font-size: 13px; font-weight: 650; color: var(--creative-ink); overflow-wrap: anywhere; }
.cd-comment-time { font-size: 10px; line-height: 1.5; color: var(--creative-muted); }
.cd-comment-body { font-size: 13px; line-height: 1.8; color: var(--creative-ink); overflow-wrap: anywhere; white-space: pre-wrap; }
.cd-comment-quote { margin-bottom: 8px; padding: 6px 9px; max-height: 5.2em; overflow-y: auto; border-left: 2px solid color-mix(in srgb, var(--creative-accent) 45%, var(--creative-line)); border-radius: 0 6px 6px 0; background: var(--creative-soft); font-size: 11px; line-height: 1.6; color: var(--creative-muted); }
.cd-comment-quote-user { font-weight: 500; color: var(--creative-accent); }
.cd-comment-actions { display: flex; flex-wrap: wrap; gap: 4px 12px; margin-top: 6px; }
.cd-comment-act { display: inline-flex; align-items: center; gap: 4px; height: 26px; padding: 0; border: 0; background: transparent; color: var(--creative-muted); font: inherit; font-size: 11px; cursor: pointer; transition: color .18s; }
.cd-comment-act:hover { color: var(--creative-accent); }
.cd-comment-act.is-danger { color: #c0392b; font-weight: 600; }
.cd-comment-replies { display: flex; flex-direction: column; gap: 14px; margin-top: 14px; padding-left: 12px; border-left: 2px solid var(--creative-line); }
.cd-comment-child { padding: 0; border: 0; gap: 8px; }
.cd-comment-child .cd-comment-head { margin-bottom: 6px; }
.cd-comment-child .cd-comment-username { font-size: 12px; }
.cd-comment.is-reply-target > .cd-comment-main > .cd-comment-head .cd-comment-username { color: var(--creative-accent); }
.cd-comment.is-reply-target > .cd-comment-avatar { border-color: var(--creative-accent); box-shadow: 0 0 0 3px var(--creative-soft); }
</style>

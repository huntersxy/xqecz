<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { IconArrowRight, IconSend } from '@arco-design/web-vue/es/icon'
import { Message } from '@arco-design/web-vue'
import { commentApi } from '@/api'
import { useConfirm } from '@/composables/useToast'
import CommentItem from '@/components/CommentItem.vue'
import type { Comment } from '@/types'

const props = defineProps<{ contentId: number; isLoggedIn: boolean }>()
const emit = defineEmits<{
  'report-comment': [comment: Comment]
}>()

const { confirm } = useConfirm()

const comments = ref<Comment[]>([])
const commentText = ref('')
const replyTarget = ref<Comment | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)
const totalComments = ref(0)
const totalPages = ref(1)

async function loadComments(page: number = 1) {
  try {
    const id = props.contentId
    const res = await commentApi.list(id, page, pageSize.value)
    if (res.code === 200) {
      currentPage.value = page
      comments.value = res.data.list
      totalComments.value = res.data.total
      totalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载评论失败:', error)
  }
}

async function submitComment() {
  if (!commentText.value.trim()) {
    Message.warning('请输入评论内容')
    return
  }
  try {
    const id = props.contentId
    const res = await commentApi.add(
      id,
      commentText.value.trim(),
      replyTarget.value?.id || undefined,
    )
    if (res.code === 200) {
      commentText.value = ''
      replyTarget.value = null
      Message.success('评论成功')
      await loadComments(1)
    } else {
      Message.error(res.message || '评论失败')
    }
  } catch {
    Message.error('评论失败')
  }
}

function cancelReply() {
  replyTarget.value = null
  commentText.value = ''
}

async function deleteComment(commentId: number) {
  const confirmed = await confirm('确定要删除这条评论吗？')
  if (!confirmed) return
  try {
    const res = await commentApi.delete(commentId)
    if (res.code === 200) {
      Message.success('删除成功')
      await loadComments(currentPage.value)
    } else {
      Message.error(res.message || '删除失败')
    }
  } catch {
    Message.error('删除失败')
  }
}

function openReport(comment: Comment) {
  emit('report-comment', comment)
}

defineExpose({ loadComments })
</script>

<template>
  <div class="cd-section cd-comments-section">
    <div class="cd-section-head">
      <span class="cd-section-title">评论 ({{ totalComments }})</span>
      <a-pagination
        v-if="totalPages > 1"
        size="small"
        :current="currentPage"
        :total="totalComments"
        :page-size="pageSize"
        simple
        @change="(page: number) => loadComments(page)"
      />
    </div>

    <!-- 评论输入 -->
    <div v-if="isLoggedIn" class="cd-comment-input-wrap">
      <div v-if="replyTarget" class="cd-reply-hint">
        <span><b>正在回复</b> {{ replyTarget.user?.username }}</span>
        <button type="button" @click="cancelReply">取消回复</button>
      </div>
      <a-textarea
        v-model="commentText"
        class="cd-comment-textarea"
        placeholder="写下你的评论... (Ctrl+Enter 发送)"
        :auto-size="{ minRows: 3, maxRows: 6 }"
        @keyup.ctrl.enter="submitComment"
      />
      <a-button type="primary" size="small" class="cd-comment-submit" @click="submitComment">
        <IconSend />
        <span>发表评论</span>
      </a-button>
    </div>
    <div v-else class="cd-login-prompt">
      <div class="cd-login-copy">
        <span class="cd-login-kicker">想留下你的想法？</span>
        <span>登录后即可参与评论</span>
      </div>
      <RouterLink to="/login" class="cd-login-link">
        <span>去登录</span>
        <IconArrowRight />
      </RouterLink>
    </div>

    <!-- 评论列表 -->
    <div v-if="comments.length > 0" class="cd-comment-list">
      <template v-for="comment in comments" :key="comment.id">
        <CommentItem
          :comment="comment"
          :reply-target="replyTarget"
          :level="0"
          @select-reply="replyTarget = $event"
          @delete-comment="deleteComment($event)"
          @report-comment="openReport($event)"
        />
      </template>
    </div>
    <div v-else class="cd-comment-empty">
      <p>暂无评论，快来发表第一条评论吧</p>
    </div>

  </div>
</template>

<style scoped>
.cd-section {
  display: flex; flex-direction: column; gap: 0.5rem;
  padding: 0; background: var(--color-surface);
  border: 0; border-radius: 0;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title {
  font-size: 0.75rem; font-weight: 600; color: var(--color-text);
  letter-spacing: 0.05em; text-transform: uppercase;
}
.cd-comments-section { flex: 1; min-height: 200px; gap: 14px; }
.cd-comment-input-wrap { display: flex; flex-direction: column; gap: 0.625rem; }
.cd-reply-hint {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0.5rem 0.7rem; background: var(--creative-soft); border: 1px solid color-mix(in srgb, var(--creative-accent) 16%, var(--creative-line));
  border-radius: 9px; font-size: 0.75rem; color: var(--creative-muted);
  box-shadow: 0 5px 16px color-mix(in srgb, var(--creative-accent) 5%, transparent);
}
.cd-comment-textarea {
  width: 100%;
}
.cd-comment-textarea :deep(.arco-textarea-wrapper),
:deep(.cd-comment-textarea.arco-textarea-wrapper) {
  border: 1px solid var(--creative-line); border-radius: 13px; background: var(--creative-canvas);
  box-shadow: inset 0 1px 0 rgb(255 255 255 / 0.72); transition: border-color .18s, box-shadow .18s, background .18s;
}
.cd-comment-textarea :deep(.arco-textarea-wrapper:hover),
:deep(.cd-comment-textarea.arco-textarea-wrapper:hover),
.cd-comment-textarea :deep(.arco-textarea-wrapper.arco-textarea-focus),
:deep(.cd-comment-textarea.arco-textarea-wrapper.arco-textarea-focus) {
  border-color: color-mix(in srgb, var(--creative-accent) 48%, var(--creative-line));
  background: var(--creative-paper); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 8%, transparent);
}
.cd-comment-submit {
  align-self: flex-end; display: inline-flex; align-items: center; gap: 6px;
}

.cd-login-prompt {
  display: flex; align-items: center; gap: 0.625rem;
  position: relative; overflow: hidden; justify-content: space-between;
  padding: 0.72rem 0.8rem 0.72rem 0.95rem; background: linear-gradient(135deg, var(--creative-canvas), var(--creative-soft));
  border: 1px solid var(--creative-line); border-radius: 13px; font-size: 0.8125rem; color: var(--creative-muted);
  box-shadow: 0 8px 22px color-mix(in srgb, var(--creative-accent) 5%, transparent);
}
.cd-login-prompt::after { content: '✦'; position: absolute; right: 88px; top: -8px; color: color-mix(in srgb, var(--creative-accent) 32%, transparent); font-size: 22px; transform: rotate(14deg); }
.cd-login-copy { display: flex; flex-direction: column; gap: 2px; }
.cd-login-kicker { color: var(--creative-ink); font-weight: 600; }
.cd-login-link {
  position: relative; z-index: 1; display: inline-flex; align-items: center; gap: 5px; flex-shrink: 0;
  padding: 6px 10px 6px 12px; border: 1px solid color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line));
  border-radius: 999px; color: var(--creative-accent); font-size: 12px; font-weight: 600; text-decoration: none;
  background: rgb(255 255 255 / 0.7); transition: background .18s, color .18s, border-color .18s, transform .18s;
}
.cd-login-link:hover { color: #fff; background: var(--creative-accent); border-color: var(--creative-accent); transform: translateY(-1px); }

.cd-comment-list { display: flex; flex-direction: column; gap: 0.5rem; }
.cd-comment-empty { text-align: center; padding: 1.5rem 0; font-size: 0.8125rem; color: var(--color-text-secondary); }
.cd-comment-submit { border-radius: 999px; padding-inline: 16px; }
.cd-comment-empty::before { content: '✧'; display: block; color: var(--creative-accent); font-size: 24px; margin-bottom: 8px; opacity: .6; }
.cd-comment-empty { line-height: 1.8; font-size: 11px; }
</style>

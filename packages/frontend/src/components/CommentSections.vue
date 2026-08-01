<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
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
        <span>回复 {{ replyTarget.user?.username }}:</span>
        <button type="button" @click="cancelReply">取消</button>
      </div>
      <a-textarea
        v-model="commentText"
        class="cd-comment-textarea"
        placeholder="写下你的评论... (Ctrl+Enter 发送)"
        :auto-size="{ minRows: 3, maxRows: 6 }"
        @keyup.ctrl.enter="submitComment"
      />
      <a-button type="primary" size="small" class="cd-comment-submit" @click="submitComment">发表评论</a-button>
    </div>
    <div v-else class="cd-login-prompt">
      <span>请先登录以发表评论</span>
      <a-button type="primary" size="small">
        <RouterLink to="/login" class="cd-login-link">登录</RouterLink>
      </a-button>
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
  padding: 0.75rem; background: var(--color-surface);
  border: 1px solid var(--color-border); border-radius: 10px;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title {
  font-size: 0.75rem; font-weight: 600; color: var(--color-text-secondary);
  letter-spacing: 0.05em; text-transform: uppercase;
}
.cd-comments-section { flex: 1; min-height: 200px; }
.cd-comment-input-wrap { display: flex; flex-direction: column; gap: 0.375rem; }
.cd-reply-hint {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0.375rem 0.625rem; background: color-mix(in srgb, var(--color-primary) 10%, transparent);
  border-radius: 6px; font-size: 0.75rem; color: var(--color-primary);
}
.cd-comment-textarea {
  width: 100%;
}
.cd-comment-submit {
  align-self: flex-end;
}

.cd-login-prompt {
  display: flex; align-items: center; gap: 0.625rem;
  padding: 0.625rem; background: var(--color-hover); border-radius: 8px;
  font-size: 0.8125rem; color: var(--color-text-secondary);
}
.cd-login-link {
  color: inherit;
  text-decoration: none;
}

.cd-comment-list { display: flex; flex-direction: column; gap: 0.5rem; }
.cd-comment-empty { text-align: center; padding: 1.5rem 0; font-size: 0.8125rem; color: var(--color-text-secondary); }
</style>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'
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

const route = useRoute()
const { confirm } = useConfirm()

const comments = ref<Comment[]>([])
const commentText = ref('')
const replyTarget = ref<Comment | null>(null)
const menuTarget = ref<number | null>(null)
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

async function goToPrevPage() {
  if (currentPage.value <= 1) return
  await loadComments(currentPage.value - 1)
}

async function goToNextPage() {
  if (currentPage.value >= totalPages.value) return
  await loadComments(currentPage.value + 1)
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
  menuTarget.value = null
}

function closeMenu() {
  menuTarget.value = null
}

defineExpose({ loadComments })
</script>

<template>
  <div class="cd-section cd-comments-section">
    <div class="cd-section-head">
      <span class="cd-section-title">评论 ({{ totalComments }})</span>
      <div v-if="totalPages > 1" class="cd-comment-pager">
        <button v-if="currentPage > 1" type="button" @click="goToPrevPage">‹</button>
        <span>{{ currentPage }}/{{ totalPages }}</span>
        <button v-if="currentPage < totalPages" type="button" @click="goToNextPage">›</button>
      </div>
    </div>

    <!-- 评论输入 -->
    <div v-if="isLoggedIn" class="cd-comment-input-wrap">
      <div v-if="replyTarget" class="cd-reply-hint">
        <span>回复 {{ replyTarget.user?.username }}:</span>
        <button type="button" @click="cancelReply">取消</button>
      </div>
      <textarea
        v-model="commentText"
        class="cd-comment-textarea"
        placeholder="写下你的评论... (Ctrl+Enter 发送)"
        @keyup.enter.ctrl="submitComment"
      ></textarea>
      <button class="cd-comment-submit" type="button" @click="submitComment">发表评论</button>
    </div>
    <div v-else class="cd-login-prompt">
      <span>请先登录以发表评论</span>
      <RouterLink to="/login" class="cd-login-link">登录</RouterLink>
    </div>

    <!-- 评论列表 -->
    <div v-if="comments.length > 0" class="cd-comment-list">
      <template v-for="comment in comments" :key="comment.id">
        <CommentItem
          :comment="comment"
          :reply-target="replyTarget"
          :menu-target="menuTarget"
          :level="0"
          @select-reply="replyTarget = $event"
          @toggle-menu="menuTarget = $event"
          @delete-comment="deleteComment($event)"
          @report-comment="openReport($event)"
        />
      </template>
    </div>
    <div v-else class="cd-comment-empty">
      <p>暂无评论，快来发表第一条评论吧</p>
    </div>

    <!-- 菜单遮罩 -->
    <Teleport to="body">
      <div v-if="menuTarget" class="cd-menu-overlay" @click="closeMenu"></div>
    </Teleport>
  </div>
</template>

<style scoped>
.cd-section {
  display: flex; flex-direction: column; gap: 0.5rem;
  padding: 0.75rem; background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 10px;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title {
  font-size: 0.75rem; font-weight: 600; color: var(--theme-text-secondary);
  letter-spacing: 0.05em; text-transform: uppercase;
}
.cd-comments-section { flex: 1; min-height: 200px; }
.cd-comment-pager { display: flex; align-items: center; gap: 0.375rem; font-size: 0.6875rem; color: var(--theme-text-secondary); }
.cd-comment-pager button {
  background: var(--theme-hover-bg); border: none; border-radius: 4px;
  padding: 2px 6px; cursor: pointer; color: var(--theme-text-secondary);
}
.cd-comment-pager button:hover { color: var(--theme-primary); }

.cd-comment-input-wrap { display: flex; flex-direction: column; gap: 0.375rem; }
.cd-reply-hint {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0.375rem 0.625rem; background: color-mix(in srgb, var(--theme-primary) 10%, transparent);
  border-radius: 6px; font-size: 0.75rem; color: var(--theme-primary);
}
.cd-reply-hint button { background: none; border: none; color: var(--theme-text-secondary); cursor: pointer; font-size: 0.75rem; }
.cd-comment-textarea {
  width: 100%; min-height: 64px; padding: 0.5rem 0.625rem;
  border: 1px solid var(--theme-card-border); border-radius: 8px;
  background: var(--theme-bg-color); color: var(--theme-text);
  font-size: 0.8125rem; resize: vertical; font-family: inherit;
}
.cd-comment-textarea:focus { outline: none; border-color: var(--theme-primary); }
.cd-comment-submit {
  align-self: flex-end; padding: 0.375rem 1rem;
  background: var(--theme-primary); color: var(--theme-on-primary);
  border: none; border-radius: 8px; font-size: 0.8125rem; font-weight: 600;
  cursor: pointer; transition: filter 0.15s;
}
.cd-comment-submit:hover { filter: brightness(0.92); }

.cd-login-prompt {
  display: flex; align-items: center; gap: 0.625rem;
  padding: 0.625rem; background: var(--theme-hover-bg); border-radius: 8px;
  font-size: 0.8125rem; color: var(--theme-text-secondary);
}
.cd-login-link {
  padding: 0.25rem 0.75rem; background: var(--theme-primary); color: var(--theme-on-primary);
  border-radius: 6px; font-size: 0.75rem; font-weight: 600; text-decoration: none;
}

.cd-comment-list { display: flex; flex-direction: column; gap: 0.5rem; }
.cd-comment-empty { text-align: center; padding: 1.5rem 0; font-size: 0.8125rem; color: var(--theme-text-secondary); }

.cd-menu-overlay { position: fixed; inset: 0; z-index: 90; }
</style>

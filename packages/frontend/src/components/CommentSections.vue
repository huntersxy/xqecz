<script setup lang="ts">
import { ref, watch } from 'vue'
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
const submitting = ref(false)
const loading = ref(false)
const loadError = ref(false)
let loadSeq = 0
const replyTarget = ref<Comment | null>(null)
const currentPage = ref(1)
const pageSize = ref(20)
const totalComments = ref(0)
const totalPages = ref(1)

async function loadComments(page: number = 1) {
  const seq = ++loadSeq
  const id = props.contentId
  if (!id) return
  loading.value = true
  loadError.value = false
  try {
    const res = await commentApi.list(id, page, pageSize.value)
    if (seq !== loadSeq || id !== props.contentId) return
    if (res.code === 200) {
      currentPage.value = page
      comments.value = res.data.list
      totalComments.value = res.data.total
      totalPages.value = res.data.total_page
    } else {
      loadError.value = true
    }
  } catch {
    if (seq === loadSeq) loadError.value = true
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

watch(() => props.contentId, () => {
  comments.value = []
  replyTarget.value = null
  commentText.value = ''
  currentPage.value = 1
  totalComments.value = 0
  totalPages.value = 1
  loadComments()
}, { immediate: true })

async function submitComment() {
  if (submitting.value) return
  if (!commentText.value.trim()) {
    Message.warning('请输入评论内容')
    return
  }
  submitting.value = true
  const targetPage = replyTarget.value ? currentPage.value : 1
  const id = props.contentId
  try {
    const res = await commentApi.add(
      id,
      commentText.value.trim(),
      replyTarget.value?.id || undefined,
    )
    if (id !== props.contentId) return
    if (res.code === 200) {
      commentText.value = ''
      replyTarget.value = null
      Message.success('评论成功')
      await loadComments(targetPage)
    } else {
      Message.error(res.message || '评论失败')
    }
  } catch {
    Message.error('评论失败')
  } finally {
    submitting.value = false
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
  <div class="cd-comments-section">
    <div class="cd-section-head">
      <span class="cd-section-title">评论 <span class="cd-comment-count">{{ totalComments }}</span></span>
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
      <Transition name="cd-reply-state">
        <div v-if="replyTarget" class="cd-reply-hint">
          <span><b>正在回复</b> {{ replyTarget.user?.username }}</span>
          <button type="button" :disabled="submitting" @click="cancelReply">取消回复</button>
        </div>
      </Transition>
      <a-textarea
        v-model="commentText"
        class="cd-comment-textarea"
        :disabled="submitting"
        aria-label="评论内容"
        placeholder="聊聊这份作品，留下你的想法…"
        :auto-size="{ minRows: 3, maxRows: 6 }"
        @keyup.ctrl.enter="submitComment"
      />
      <div class="cd-comment-input-footer">
        <span class="cd-comment-shortcut">Ctrl + Enter 发送</span>
        <a-button type="primary" size="small" class="cd-comment-submit" :loading="submitting" @click="submitComment">
          <IconSend />
          <span>{{ replyTarget ? '发送回复' : '发表评论' }}</span>
        </a-button>
      </div>
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
    <div v-if="loading" class="cd-comment-status" role="status"><a-spin :loading="true" :size="16" /> 正在加载评论…</div>
    <div v-else-if="loadError" class="cd-comment-status" role="status">评论加载失败<button type="button" @click="loadComments(currentPage)">重试</button></div>
    <div v-if="comments.length > 0" class="cd-comment-list">
      <template v-for="comment in comments" :key="comment.id">
        <CommentItem
          :comment="comment"
          :reply-target="replyTarget"
          :level="0"
          @select-reply="!submitting && (replyTarget = $event)"
          @delete-comment="deleteComment($event)"
          @report-comment="openReport($event)"
        />
      </template>
    </div>
    <div v-else-if="!loading && !loadError" class="cd-comment-empty">
      <p>暂无评论，快来发表第一条评论吧</p>
    </div>

  </div>
</template>

<style scoped>
.cd-comments-section {
  display: flex; flex-direction: column; flex-shrink: 0; gap: 20px;
  padding: 0 0 8px; background: transparent; border: 0;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; }
.cd-section-title { display: inline-flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 650; color: var(--creative-ink); }
.cd-comment-count { display: inline-flex; align-items: center; justify-content: center; min-width: 24px; height: 22px; padding: 0 7px; border-radius: 7px; background: var(--creative-soft); color: var(--creative-accent); font-size: 11px; font-weight: 600; }
.cd-comment-input-wrap {
  display: flex; flex-direction: column; gap: 10px; padding: 12px;
  border: 1px solid var(--creative-line); border-radius: 14px; background: var(--creative-canvas);
  transition: border-color .18s, box-shadow .18s;
}
.cd-comment-input-wrap:focus-within { border-color: color-mix(in srgb, var(--creative-accent) 55%, var(--creative-line)); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.cd-reply-hint { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 10px; background: var(--creative-soft); border-radius: 8px; font-size: 12px; color: var(--creative-muted); }
.cd-reply-hint > span { min-width: 0; overflow-wrap: anywhere; }
.cd-reply-hint b { color: var(--creative-accent); font-weight: 500; }
.cd-reply-hint button { flex-shrink: 0; padding: 0; border: 0; background: transparent; color: var(--creative-muted); font: inherit; cursor: pointer; }
.cd-reply-hint button:hover { color: var(--creative-accent); }
.cd-comment-textarea { width: 100%; }
:deep(.cd-comment-textarea.arco-textarea-wrapper),
:deep(.cd-comment-textarea.arco-textarea-wrapper:hover),
:deep(.cd-comment-textarea.arco-textarea-wrapper.arco-textarea-focus) { border: 0; background: transparent; box-shadow: none; }
.cd-comment-textarea :deep(.arco-textarea) { padding: 2px; font-size: 13px; line-height: 1.7; color: var(--creative-ink); resize: none; }
.cd-comment-textarea :deep(.arco-textarea::placeholder) { color: var(--creative-muted); }
.cd-comment-input-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.cd-comment-shortcut { font-size: 10px; color: var(--creative-muted); }
.cd-comment-submit { display: inline-flex; align-items: center; flex-shrink: 0; gap: 6px; height: 30px; border-radius: 9px; padding-inline: 12px; }

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
  background: var(--creative-paper); transition: background .18s, color .18s, border-color .18s, transform .18s;
}
.cd-login-link:hover { color: #fff; background: var(--creative-accent); border-color: var(--creative-accent); transform: translateY(-1px); }

.cd-comment-list { display: flex; flex-direction: column; gap: 0; }
.cd-comment-empty { padding: 28px 12px; border: 1px dashed var(--creative-line); border-radius: 12px; text-align: center; line-height: 1.8; font-size: 12px; color: var(--creative-muted); }
.cd-comment-empty p { margin: 8px 0 0; }
.cd-comment-empty::before { content: '✧'; display: block; color: var(--creative-accent); font-size: 24px; opacity: .6; }
.cd-comment-status { display: flex; align-items: center; gap: 8px; padding: 12px 0; color: var(--creative-muted); font-size: 11px; }
.cd-comment-status button { border: 0; padding: 3px 8px; border-radius: 999px; background: var(--creative-soft); color: var(--creative-accent); cursor: pointer; }
.cd-reply-state-enter-active, .cd-reply-state-leave-active { transition: opacity .16s ease, transform .16s ease; }
.cd-reply-state-enter-from, .cd-reply-state-leave-to { opacity: 0; transform: translateY(-3px); }
@media (max-width: 768px) { .cd-comment-shortcut { display: none; } .cd-comment-input-footer { justify-content: flex-end; } }
</style>

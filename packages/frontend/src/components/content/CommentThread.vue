<script setup lang="ts">
import { ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { commentApi } from '@/api'
import { toast } from '@/composables/useToast'
import CommentNode from './CommentNode.vue'
import SiteIcon from '@/components/SiteIcon.vue'
import type { Comment } from '@/types'

const props = defineProps<{ contentId: number; isLoggedIn: boolean }>()
const emit = defineEmits<{ 'report-comment': [comment: Comment] }>()

const comments = ref<Comment[]>([])
const commentText = ref('')
const submitting = ref(false)
const loading = ref(false)
const loadError = ref(false)
let loadSeq = 0
const replyTarget = ref<Comment | null>(null)
const currentPage = ref(1)
const pageSize = 20
const totalComments = ref(0)
const totalPages = ref(1)

async function loadComments(page: number = 1) {
  const seq = ++loadSeq
  const id = props.contentId
  if (!id) return
  loading.value = true
  loadError.value = false
  try {
    const res = await commentApi.list(id, page, pageSize)
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
  void loadComments()
}, { immediate: true })

async function submitComment() {
  if (submitting.value) return
  if (!commentText.value.trim()) {
    toast.warning('请输入评论内容')
    return
  }
  submitting.value = true
  const targetPage = replyTarget.value ? currentPage.value : 1
  const id = props.contentId
  try {
    const res = await commentApi.add(id, commentText.value.trim(), replyTarget.value?.id || undefined)
    if (id !== props.contentId) return
    if (res.code === 200) {
      commentText.value = ''
      replyTarget.value = null
      toast.success('评论成功')
      await loadComments(targetPage)
    } else {
      toast.error(res.message || '评论失败')
    }
  } catch {
    toast.error('评论失败')
  } finally {
    submitting.value = false
  }
}

function cancelReply() {
  replyTarget.value = null
  commentText.value = ''
}

async function deleteComment(commentId: number) {
  try {
    const res = await commentApi.delete(commentId)
    if (res.code === 200) {
      toast.success('删除成功')
      await loadComments(currentPage.value)
    } else {
      toast.error(res.message || '删除失败')
    }
  } catch {
    toast.error('删除失败')
  }
}

defineExpose({ loadComments })
</script>

<template>
  <div class="cd-comments-section">
    <div class="cd-comments-head">
      <span class="cd-comments-title">评论 <span class="cd-comment-count">{{ totalComments }}</span></span>
      <nav v-if="totalPages > 1" class="cd-pager" aria-label="评论分页">
        <button type="button" class="cd-pager-btn" aria-label="上一页" :disabled="currentPage <= 1 || loading" @click="loadComments(currentPage - 1)">
          <SiteIcon name="chevron-left" />
        </button>
        <span class="cd-pager-info">{{ currentPage }} / {{ totalPages }}</span>
        <button type="button" class="cd-pager-btn" aria-label="下一页" :disabled="currentPage >= totalPages || loading" @click="loadComments(currentPage + 1)">
          <SiteIcon name="chevron-right" />
        </button>
      </nav>
    </div>

    <div v-if="isLoggedIn" class="cd-comment-input-wrap">
      <Transition name="cd-reply-state">
        <div v-if="replyTarget" class="cd-reply-hint">
          <span><b>正在回复</b> {{ replyTarget.user?.username }}</span>
          <button type="button" :disabled="submitting" @click="cancelReply">取消回复</button>
        </div>
      </Transition>
      <textarea
        v-model="commentText"
        class="cd-comment-textarea"
        :disabled="submitting"
        aria-label="评论内容"
        placeholder="对这颗星球说点什么…"
        rows="3"
        @keyup.ctrl.enter="submitComment"
      ></textarea>
      <div class="cd-comment-input-footer">
        <span class="cd-comment-shortcut">Ctrl + Enter 发送</span>
        <button type="button" class="cd-comment-submit" :disabled="submitting" :aria-busy="submitting" @click="submitComment">
          <SiteIcon name="send" />
          <span>{{ replyTarget ? '发送回复' : '发表评论' }}</span>
        </button>
      </div>
    </div>
    <div v-else class="cd-login-prompt">
      <div class="cd-login-copy">
        <span class="cd-login-kicker">想对这颗星球说点什么？</span>
        <span>登录后即可参与评论</span>
      </div>
      <RouterLink to="/login" class="cd-login-link">
        <span>去登录</span>
        <SiteIcon name="chevron-right" />
      </RouterLink>
    </div>

    <div v-if="loading" class="cd-comment-status" role="status"><span class="cd-spin" aria-hidden="true"></span>正在接收信号…</div>
    <div v-else-if="loadError" class="cd-comment-status" role="status">评论加载失败<button type="button" @click="loadComments(currentPage)">重试</button></div>
    <div v-if="comments.length > 0" class="cd-comment-list">
      <CommentNode
        v-for="comment in comments"
        :key="comment.id"
        :comment="comment"
        :reply-target="replyTarget"
        :level="0"
        @select-reply="!submitting && (replyTarget = $event)"
        @delete-comment="deleteComment($event)"
        @report-comment="emit('report-comment', $event)"
      />
    </div>
    <div v-else-if="!loading && !loadError" class="cd-comment-empty">
      <p>还没有人造访，来留下第一条信号吧</p>
    </div>
  </div>
</template>

<style scoped>
.cd-comments-section { display: flex; flex-direction: column; flex-shrink: 0; gap: 20px; padding: 0 0 8px; }
.cd-comments-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 10px; }
.cd-comments-title { display: inline-flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 650; color: var(--creative-ink); }
.cd-comment-count { display: inline-flex; align-items: center; justify-content: center; min-width: 24px; height: 22px; padding: 0 7px; border-radius: 7px; background: var(--creative-soft); color: var(--creative-accent); font-size: 11px; font-weight: 600; }
.cd-pager { display: inline-flex; align-items: center; gap: 6px; color: var(--creative-muted); font-size: 11px; font-variant-numeric: tabular-nums; }
.cd-pager-btn { display: grid; place-items: center; width: 26px; height: 26px; border: 1px solid var(--creative-line); border-radius: 7px; background: transparent; color: var(--creative-muted); cursor: pointer; }
.cd-pager-btn:hover:not(:disabled) { color: var(--creative-accent); border-color: var(--creative-accent); }
.cd-pager-btn:disabled { opacity: .45; cursor: not-allowed; }

.cd-comment-input-wrap { display: flex; flex-direction: column; gap: 10px; padding: 12px; border: 1px solid var(--creative-line); border-radius: 14px; background: var(--creative-canvas); transition: border-color .18s, box-shadow .18s; }
.cd-comment-input-wrap:focus-within { border-color: color-mix(in srgb, var(--creative-accent) 55%, var(--creative-line)); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 8%, transparent); }
.cd-reply-hint { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 10px; border-radius: 8px; background: var(--creative-soft); font-size: 12px; color: var(--creative-muted); }
.cd-reply-hint > span { min-width: 0; overflow-wrap: anywhere; }
.cd-reply-hint b { color: var(--creative-accent); font-weight: 500; }
.cd-reply-hint button { flex-shrink: 0; padding: 0; border: 0; background: transparent; color: var(--creative-muted); font: inherit; cursor: pointer; }
.cd-reply-hint button:hover { color: var(--creative-accent); }
.cd-comment-textarea { width: 100%; min-height: 4.8em; max-height: 14em; padding: 2px; border: 0; outline: 0; background: transparent; color: var(--creative-ink); font: inherit; font-size: 13px; line-height: 1.7; resize: none; field-sizing: content; }
.cd-comment-textarea::placeholder { color: var(--creative-muted); }
.cd-comment-input-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.cd-comment-shortcut { font-size: 10px; color: var(--creative-muted); }
.cd-comment-submit { display: inline-flex; align-items: center; flex-shrink: 0; gap: 6px; height: 30px; padding: 0 12px; border: 0; border-radius: 9px; background: var(--creative-accent); color: var(--color-on-primary, #fff); font: inherit; font-size: 12px; cursor: pointer; transition: background .18s, opacity .18s; }
.cd-comment-submit:hover:not(:disabled) { background: var(--creative-accent-hover); }
.cd-comment-submit:disabled { opacity: .6; cursor: progress; }

.cd-login-prompt { position: relative; display: flex; align-items: center; justify-content: space-between; gap: .625rem; overflow: hidden; padding: .72rem .8rem .72rem .95rem; border: 1px solid var(--creative-line); border-radius: 13px; background: linear-gradient(135deg, var(--creative-canvas), var(--creative-soft)); font-size: .8125rem; color: var(--creative-muted); }
.cd-login-copy { display: flex; flex-direction: column; gap: 2px; }
.cd-login-kicker { color: var(--creative-ink); font-weight: 600; }
.cd-login-link { display: inline-flex; align-items: center; gap: 5px; flex-shrink: 0; padding: 6px 10px 6px 12px; border: 1px solid color-mix(in srgb, var(--creative-accent) 28%, var(--creative-line)); border-radius: 999px; background: var(--creative-paper); color: var(--creative-accent); font-size: 12px; font-weight: 600; text-decoration: none; transition: background .18s, color .18s, border-color .18s; }
.cd-login-link:hover { color: #fff; background: var(--creative-accent); border-color: var(--creative-accent); }

.cd-comment-list { display: flex; flex-direction: column; }
.cd-comment-empty { padding: 28px 12px; border: 1px dashed var(--creative-line); border-radius: 12px; text-align: center; line-height: 1.8; font-size: 12px; color: var(--creative-muted); }
.cd-comment-empty p { margin: 8px 0 0; }
.cd-comment-empty::before { content: '✧'; display: block; color: var(--creative-accent); font-size: 24px; opacity: .6; }
.cd-comment-status { display: flex; align-items: center; gap: 8px; padding: 12px 0; color: var(--creative-muted); font-size: 11px; }
.cd-comment-status button { padding: 3px 8px; border: 0; border-radius: 999px; background: var(--creative-soft); color: var(--creative-accent); cursor: pointer; }
.cd-spin { width: 14px; height: 14px; border: 2px solid var(--creative-line); border-top-color: var(--creative-accent); border-radius: 50%; animation: cd-spin .8s linear infinite; }
@keyframes cd-spin { to { transform: rotate(360deg); } }
.cd-reply-state-enter-active, .cd-reply-state-leave-active { transition: opacity .16s ease, transform .16s ease; }
.cd-reply-state-enter-from, .cd-reply-state-leave-to { opacity: 0; transform: translateY(-3px); }
@media (max-width: 768px) { .cd-comment-shortcut { display: none; } .cd-comment-input-footer { justify-content: flex-end; } }
</style>

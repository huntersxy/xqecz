<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RouterLink } from 'vue-router'
import { contentApi, commentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import type { Content, Comment } from '@/types'
import { renderMarkdown } from '@/utils'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const content = ref<Content | null>(null)
const comments = ref<Comment[]>([])
const commentText = ref('')
const replyTarget = ref<Comment | null>(null)
const reportTarget = ref<Comment | null>(null)
const reportReason = ref('')
const message = ref('')

const showClaimModal = ref(false)
const claimReason = ref('')
const isSubmittingClaim = ref(false)

const renderedContent = computed(() => {
  if (!content.value) return ''
  const text = content.value.text || ''
  return renderMarkdown(text)
})

function getImageUrl(image?: string): string {
  if (image) {
    let url = image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
    if (url.startsWith('/')) {
      url = `https://xqapi.xiey.work${url}`
    }
    return url
  }
  return ''
}

function formatTime(ts: number): string {
  if (!ts) return ''
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

async function loadContent() {
  try {
    const id = Number(route.params.id)
    const res = await contentApi.detail(id)
    if (res.code === 200) {
      content.value = res.data
    } else {
      message.value = res.message
    }
  } catch {
    message.value = '加载内容失败'
  }
}

async function loadComments() {
  try {
    const id = Number(route.params.id)
    const res = await commentApi.list(id)
    if (res.code === 200) {
      comments.value = res.data
    }
  } catch (error) {
    console.error('加载评论失败:', error)
  }
}

async function submitComment() {
  if (!commentText.value.trim()) {
    message.value = '请输入评论内容'
    return
  }

  try {
    const id = Number(route.params.id)
    const res = await commentApi.add(id, commentText.value.trim(), replyTarget.value?.id || undefined)
    if (res.code === 200) {
      commentText.value = ''
      replyTarget.value = null
      message.value = '评论成功'
      await loadComments()
    } else {
      message.value = res.message || '评论失败'
    }
  } catch {
    message.value = '评论失败'
  }
}

function cancelReply() {
  replyTarget.value = null
  commentText.value = ''
}

async function deleteComment(commentId: number) {
  if (!confirm('确定要删除这条评论吗？')) return

  try {
    const res = await commentApi.delete(commentId)
    if (res.code === 200) {
      message.value = '删除成功'
      await loadComments()
    } else {
      message.value = res.message || '删除失败'
    }
  } catch {
    message.value = '删除失败'
  }
}

function openReport(comment: Comment) {
  reportTarget.value = comment
  reportReason.value = ''
}

async function submitReport() {
  if (!reportTarget.value) return

  try {
    const res = await commentApi.report(reportTarget.value.id, reportReason.value || undefined)
    if (res.code === 200) {
      message.value = '举报成功，管理员将尽快处理'
      reportTarget.value = null
      reportReason.value = ''
    } else {
      message.value = res.message || '举报失败'
    }
  } catch {
    message.value = '举报失败'
  }
}

function goBack() {
  router.back()
}

async function submitClaim() {
  if (!claimReason.value.trim()) {
    message.value = '请输入认领理由'
    return
  }
  if (!content.value) return

  try {
    isSubmittingClaim.value = true
    const res = await contentApi.submitClaim(
      content.value.id || 0,
      claimReason.value.trim()
    )
    if (res.code === 200) {
      message.value = '认领申请已提交，请等待管理员审核'
      showClaimModal.value = false
      claimReason.value = ''
    } else {
      message.value = res.message || '认领申请提交失败'
    }
  } catch {
    message.value = '认领申请提交失败'
  } finally {
    isSubmittingClaim.value = false
  }
}

onMounted(() => {
  userStore.checkAuth()
  loadContent()
  loadComments()
})
</script>

<template>
  <div class="detail-container">
    <div class="mac-window detail-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">内容详情</div>
      </div>

      <div class="detail-content">
        <div v-if="message" class="message-bar error">
          {{ message }}
          <span class="message-close" @click="message = ''">×</span>
        </div>

        <div v-if="content" class="content-detail">
          <div class="detail-header">
            <button @click="goBack" class="back-btn mac-btn small-btn">
              <svg class="back-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 12H5"/>
                <polyline points="12 19 5 12 12 5"/>
              </svg>
              返回
            </button>
          </div>

          <div class="content-main">
            <div class="content-header">
              <h1 class="content-title">{{ content.title }}</h1>
              <div class="content-badges">
                <span :class="['type-badge', content.type]">
                  {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : content.type === 'link' ? '链接' : '文字' }}
                </span>
                <span :class="['status-badge', content.audit_status]">
                  {{ content.audit_status === 'approved' ? '已通过' : content.audit_status === 'pending' ? '审核中' : '已拒绝' }}
                </span>
              </div>
            </div>

            <div class="content-meta">
              <div class="meta-item">
                <svg class="meta-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
                <span>{{ content.user?.username }}</span>
                <button
                  @click="userStore.isLoggedIn ? showClaimModal = true : router.push('/login')"
                  class="claim-link"
                >
                  认领内容
                </button>
              </div>
              <div v-if="(content.tags || []).length > 0" class="meta-tags">
                <svg class="meta-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/>
                </svg>
                <span v-for="tag in (content.tags || [])" :key="tag" class="meta-tag">{{ tag }}</span>
              </div>
            </div>

            <div class="content-timeline">
              <span class="timeline-item">
                <svg class="timeline-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <polyline points="12 6 12 12 16 14"/>
                </svg>
                {{ formatTime(content.created_at) }}
              </span>
              <span v-if="content.updated_at !== content.created_at" class="timeline-item">
                <svg class="timeline-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="18 21 23 21 23 15"/>
                  <path d="M20.49 9h-5.99M14.51 21h-5.99M9 9H3m3 12H3"/>
                </svg>
                {{ formatTime(content.updated_at) }}
              </span>
            </div>

            <div class="content-body">
              <div v-if="content.type === 'text'" class="text-content" v-html="renderedContent"></div>
              <div v-else-if="content.type === 'image'" class="image-content">
                <img :src="getImageUrl(content.img)" alt="内容图片" class="content-image" />
              </div>
              <div v-else-if="content.type === 'video'" class="video-content">
                <video controls class="content-video">
                  <source :src="getImageUrl(content.video)" />
                  您的浏览器不支持视频播放。
                </video>
              </div>
              <div v-else-if="content.type === 'link'" class="text-content" v-html="renderedContent"></div>
            </div>
          </div>

          <div class="comments-section">
            <div class="comments-header">
              <h3>评论 ({{ comments.length }})</h3>
            </div>

            <div v-if="userStore.isLoggedIn" class="comment-input-area">
              <div v-if="replyTarget" class="reply-indicator">
                <span>回复 {{ replyTarget.user?.username }}:</span>
                <button @click="cancelReply" class="cancel-reply-btn">取消</button>
              </div>
              <textarea
                v-model="commentText"
                class="comment-input"
                placeholder="写下你的评论..."
                @keyup.enter.ctrl="submitComment"
              ></textarea>
              <div class="comment-actions">
                <span class="hint">Ctrl + Enter 发送</span>
                <button @click="submitComment" class="mac-btn primary-btn">发表评论</button>
              </div>
            </div>

            <div v-else class="login-prompt">
              <p>请先登录以发表评论</p>
              <RouterLink to="/login" class="mac-btn">登录</RouterLink>
            </div>

            <div v-if="comments.length > 0" class="comments-list">
              <div v-for="comment in comments" :key="comment.id" class="comment-item">
                <div class="comment-avatar">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                    <circle cx="12" cy="7" r="4"/>
                  </svg>
                </div>
                <div class="comment-content">
                  <div class="comment-header">
                    <span class="comment-author">{{ comment.user?.username }}</span>
                    <span class="comment-id">#{{ comment.id }}</span>
                    <span class="comment-time">{{ formatTime(comment.created_at) }}</span>
                    <div class="comment-actions">
                      <button v-if="userStore.isLoggedIn" @click="replyTarget = comment" class="action-btn reply-btn">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M11 5H6a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2v-5m-1.414-9.414a2 2 0 1 1 2.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                        </svg>
                        回复
                      </button>
                      <button v-if="userStore.isLoggedIn && (userStore.user?.is_admin || comment.user_id === userStore.user?.id)" @click="deleteComment(comment.id)" class="action-btn delete-btn">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M3 6h18"/>
                          <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                          <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                        </svg>
                        删除
                      </button>
                      <button v-if="userStore.isLoggedIn" @click="openReport(comment)" class="action-btn report-btn">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                        </svg>
                        举报
                      </button>
                    </div>
                  </div>
                  <p class="comment-text">{{ comment.text }}</p>

                  <div v-if="comment.replies && comment.replies.length > 0" class="replies-list">
                    <div v-for="reply in comment.replies" :key="reply.id" class="reply-item">
                      <div class="reply-avatar">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                          <circle cx="12" cy="7" r="4"/>
                        </svg>
                      </div>
                      <div class="reply-content">
                        <div class="reply-header">
                          <span class="reply-author">{{ reply.user?.username }}</span>
                          <span class="reply-time">{{ formatTime(reply.created_at) }}</span>
                          <div class="reply-actions">
                            <button v-if="userStore.isLoggedIn" @click="replyTarget = reply" class="action-btn reply-btn">
                              回复
                            </button>
                            <button v-if="userStore.isLoggedIn && (userStore.user?.is_admin || reply.user_id === userStore.user?.id)" @click="deleteComment(reply.id)" class="action-btn delete-btn">
                              删除
                            </button>
                            <button v-if="userStore.isLoggedIn" @click="openReport(reply)" class="action-btn report-btn">
                              举报
                            </button>
                          </div>
                        </div>
                        <p class="reply-text"><span class="reply-to">{{ reply.parent?.user?.username }}:</span> {{ reply.text }}</p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div v-else class="empty-comments">
              <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
              <p>暂无评论，快来发表第一条评论吧</p>
            </div>
          </div>
        </div>

        <div v-else class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 20h-8l-4-4H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2z"/>
            <polyline points="14 2 14 8 20 8"/>
            <line x1="9" y1="15" x2="15" y2="15"/>
          </svg>
          <p>加载中...</p>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="reportTarget" class="report-modal-overlay" @click.self="reportTarget = null">
        <div class="report-modal">
          <div class="report-modal-header">
            <h3>举报评论</h3>
            <button @click="reportTarget = null" class="modal-close">×</button>
          </div>
          <div class="report-modal-body">
            <p>您正在举报以下评论：</p>
            <p class="report-content">{{ reportTarget.text }}</p>
            <label class="report-label">举报原因（可选）</label>
            <input v-model="reportReason" type="text" class="report-input" placeholder="请输入举报原因" />
          </div>
          <div class="report-modal-footer">
            <button @click="reportTarget = null" class="mac-btn">取消</button>
            <button @click="submitReport" class="mac-btn primary-btn">确认举报</button>
          </div>
        </div>
      </div>

      <div v-if="showClaimModal" class="claim-modal-overlay" @click.self="showClaimModal = false">
        <div class="claim-modal">
          <div class="claim-modal-header">
            <h3>认领此内容</h3>
            <button @click="showClaimModal = false" class="modal-close">×</button>
          </div>
          <div class="claim-modal-body">
            <p class="claim-desc">请提供认领理由，管理员将在审核后决定是否将此内容转移给您。</p>
            <label class="claim-label">认领理由 <span class="required">*</span></label>
            <textarea
              v-model="claimReason"
              class="claim-textarea"
              placeholder="请详细说明您认为此内容应归属于您的原因..."
              :disabled="isSubmittingClaim"
            ></textarea>
          </div>
          <div class="claim-modal-footer">
            <button @click="showClaimModal = false" class="mac-btn" :disabled="isSubmittingClaim">取消</button>
            <button @click="submitClaim" class="mac-btn primary-btn" :disabled="isSubmittingClaim">
              {{ isSubmittingClaim ? '提交中...' : '提交申请' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.detail-container {
  min-height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.detail-window {
  width: 100%;
  max-width: 800px;
  min-height: calc(100vh - 40px);
  overflow: hidden;
  background: rgba(255, 255, 255, 0.75);
  border-radius: 12px;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.08),
    0 2px 8px rgba(0, 0, 0, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.mac-title-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 10px 16px;
  background: linear-gradient(180deg, rgba(0,0,0,0.08) 0%, rgba(0,0,0,0.02) 100%);
  border-radius: 12px 12px 0 0;
}

.mac-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.mac-dot-red {
  background: #ff5f57;
}

.mac-dot-yellow {
  background: #febc2e;
}

.mac-dot-green {
  background: #28c840;
}

.window-title {
  margin-left: 16px;
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.detail-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.message-bar {
  padding: 10px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.message-bar.error {
  background: #fee2e2;
  color: #dc2626;
}

.message-close {
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}

.content-detail {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.detail-header {
  margin-bottom: 20px;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(0, 0, 0, 0.05);
}

.back-btn:hover {
  background: rgba(0, 0, 0, 0.1);
  color: #3b82f6;
}

.back-icon {
  width: 16px;
  height: 16px;
}

.content-main {
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  padding: 24px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.36);
  margin-bottom: 16px;
}

.content-header {
  margin-bottom: 16px;
}

.content-title {
  font-size: 22px;
  font-weight: 700;
  color: #1a1a1a;
  margin: 0 0 10px 0;
  line-height: 1.3;
}

.content-badges {
  display: flex;
  gap: 8px;
}

.type-badge,
.status-badge {
  padding: 4px 10px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
}

.type-badge.video {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.type-badge.link {
  background: rgba(139, 92, 246, 0.1);
  color: #7c3aed;
}

.type-badge.image {
  background: #ecfdf5;
  color: #059669;
}

.type-badge.text {
  background: #eff6ff;
  color: #2563eb;
}

.status-badge.approved {
  background: #ecfdf5;
  color: #059669;
}

.status-badge.pending {
  background: #fef3c7;
  color: #d97706;
}

.status-badge.rejected {
  background: #fef2f2;
  color: #dc2626;
}

.content-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #666;
}

.meta-tags {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #666;
}

.meta-tag {
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 4px;
  color: #3b82f6;
  font-size: 12px;
}

.meta-icon {
  width: 16px;
  height: 16px;
}

.content-timeline {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 16px;
  font-size: 12px;
  color: #999;
}

.timeline-item {
  display: flex;
  align-items: center;
  gap: 6px;
}

.timeline-icon {
  width: 14px;
  height: 14px;
}

.claim-link {
  background: none;
  border: none;
  padding: 0;
  margin: 0;
  font: inherit;
  font-size: 12px;
  color: #aaa;
  cursor: pointer;
  text-decoration: underline;
  text-underline-offset: 2px;
  text-decoration-style: dotted;
  margin-left: 6px;
  transition: color 0.2s;
}

.claim-link:hover {
  color: #3b82f6;
}

.content-body {
}

.text-content {
  font-size: 15px;
  line-height: 1.8;
  color: #333;
}

.text-content :deep(h1) {
  font-size: 22px;
  font-weight: 700;
  margin: 18px 0 10px;
}

.text-content :deep(h2) {
  font-size: 18px;
  font-weight: 600;
  margin: 16px 0 8px;
}

.text-content :deep(h3) {
  font-size: 16px;
  font-weight: 600;
  margin: 14px 0 8px;
}

.text-content :deep(p) {
  margin: 0 0 10px;
}

.text-content :deep(a) {
  color: #3b82f6;
  text-decoration: none;
}

.text-content :deep(a:hover) {
  text-decoration: underline;
}

.text-content :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 10px 0;
}

.image-content {
  display: flex;
  justify-content: center;
}

.content-image {
  width: 100%;
  border-radius: 10px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.video-content {
  display: flex;
  justify-content: center;
}

.content-video {
  width: 100%;
  max-width: 700px;
  max-height: 450px;
  border-radius: 10px;
  background: #000;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: #999;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}

.empty-state p {
  margin: 0;
}

.comments-section {
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  padding: 20px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.36);
}

.comments-header {
  margin-bottom: 16px;
}

.comments-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.comment-input-area {
  margin-bottom: 16px;
}

.reply-indicator {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 6px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #3b82f6;
}

.cancel-reply-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  font-size: 14px;
}

.cancel-reply-btn:hover {
  color: #3b82f6;
}

.comment-input {
  width: 100%;
  min-height: 70px;
  padding: 12px 14px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  resize: vertical;
  box-sizing: border-box;
  background: white;
}

.comment-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.comment-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 10px;
}

.hint {
  font-size: 12px;
  color: #999;
}

.login-prompt {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 8px;
  margin-bottom: 16px;
}

.login-prompt p {
  margin: 0;
  color: #666;
}

.comments-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.comment-item {
  display: flex;
  gap: 12px;
  padding: 14px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.36);
}

.comment-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.comment-avatar svg {
  width: 18px;
  height: 18px;
  color: #3b82f6;
}

.comment-content {
  flex: 1;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.comment-author {
  font-weight: 600;
  color: #1a1a1a;
  font-size: 14px;
}

.comment-id {
  font-size: 12px;
  color: #666;
  margin: 0 8px;
}

.comment-time {
  font-size: 12px;
  color: #999;
}

.comment-actions {
  margin-left: auto;
  display: flex;
  gap: 6px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  padding: 6px 10px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  color: #666;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  transform: translateY(-1px);
}

.action-btn svg {
  width: 14px;
  height: 14px;
}

.reply-btn {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.reply-btn:hover {
  background: rgba(59, 130, 246, 0.15);
}

.delete-btn {
  background: rgba(220, 38, 38, 0.1);
  color: #dc2626;
}

.delete-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.report-btn {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.report-btn:hover {
  background: rgba(245, 158, 11, 0.15);
}

.comment-text {
  margin: 0;
  font-size: 14px;
  color: #333;
  line-height: 1.6;
}

.replies-list {
  margin-top: 10px;
  padding-left: 16px;
  border-left: 2px solid rgba(59, 130, 246, 0.2);
}

.reply-item {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.reply-item:last-child {
  margin-bottom: 0;
}

.reply-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: rgba(59, 130, 246, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.reply-avatar svg {
  width: 14px;
  height: 14px;
  color: #3b82f6;
}

.reply-content {
  flex: 1;
}

.reply-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 3px;
}

.reply-author {
  font-weight: 500;
  color: #333;
  font-size: 12px;
}

.reply-time {
  font-size: 11px;
  color: #999;
}

.reply-actions {
  margin-left: auto;
  display: flex;
  gap: 4px;
}

.reply-actions .action-btn {
  padding: 2px 6px;
  font-size: 11px;
}

.reply-text {
  margin: 0;
  font-size: 13px;
  color: #444;
  line-height: 1.5;
}

.reply-to {
  color: #3b82f6;
  font-weight: 500;
}

.empty-comments {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 30px;
  color: #999;
}

.empty-comments .empty-icon {
  width: 40px;
  height: 40px;
}

.empty-comments p {
  margin: 10px 0 0 0;
  font-size: 14px;
}

.report-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  z-index: 9999;
}

.report-modal {
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
  box-shadow:
    0 20px 60px rgba(0, 0, 0, 0.3);
}

.report-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: linear-gradient(180deg, rgba(0,0,0,0.05) 0%, transparent 100%);
}

.report-modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
  line-height: 1;
}

.modal-close:hover {
  color: #3b82f6;
}

.report-modal-body {
  padding: 20px;
}

.report-modal-body p {
  margin: 0 0 10px 0;
  font-size: 14px;
  color: #333;
}

.report-content {
  padding: 12px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 6px;
  font-style: italic;
  color: #666 !important;
}

.report-label {
  display: block;
  margin-bottom: 8px;
  font-size: 13px;
  color: #666;
}

.report-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  box-sizing: border-box;
  background: white;
}

.report-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.report-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(0, 0, 0, 0.03);
}

.claim-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  z-index: 9999;
}

.claim-modal {
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  width: 90%;
  max-width: 450px;
  overflow: hidden;
  box-shadow:
    0 20px 60px rgba(0, 0, 0, 0.3);
}

.claim-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: linear-gradient(180deg, rgba(59, 130, 246, 0.05) 0%, transparent 100%);
}

.claim-modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.claim-modal-body {
  padding: 20px;
}

.claim-desc {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: #666;
  line-height: 1.6;
}

.claim-label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.claim-label .required {
  color: #ef4444;
}

.claim-textarea {
  width: 100%;
  min-height: 120px;
  padding: 12px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.6;
  resize: vertical;
  box-sizing: border-box;
  background: white;
}

.claim-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.claim-textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.claim-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(0, 0, 0, 0.03);
}

@media screen and (max-width: 768px) {
  .detail-container {
    padding: 0;
    min-height: calc(100vh - 60px);
  }

  .detail-window {
    border-radius: 0;
    box-shadow: none;
    min-height: calc(100vh - 60px);
    background: rgba(255, 255, 255, 0.9);
  }

  .mac-title-bar {
    display: none;
  }

  .detail-content {
    padding: 16px;
  }

  .content-main {
    padding: 16px;
    border-radius: 10px;
  }

  .content-title {
    font-size: 20px;
  }

  .content-meta {
    gap: 12px;
    padding-bottom: 12px;
    margin-bottom: 12px;
  }

  .meta-item {
    font-size: 14px;
  }

  .meta-icon {
    width: 18px;
    height: 18px;
  }

  .content-timeline {
    gap: 12px;
    font-size: 13px;
  }

  .timeline-icon {
    width: 16px;
    height: 16px;
  }

  .content-body {
    margin-top: 12px;
  }

  .text-content {
    font-size: 16px;
    line-height: 1.8;
  }

  .text-content :deep(h1) {
    font-size: 20px;
  }

  .text-content :deep(h2) {
    font-size: 18px;
  }

  .text-content :deep(h3) {
    font-size: 16px;
  }

  .content-image {
    width: 90%;
    max-height: none;
  }

  .content-video {
    width: 100%;
    max-height: 300px;
  }

  .comments-section {
    padding: 16px;
    border-radius: 10px;
  }

  .comment-input {
    min-height: 80px;
    padding: 14px;
    font-size: 15px;
  }

  .comment-actions {
    flex-direction: column;
    gap: 10px;
    align-items: stretch;
  }

  .comment-actions .mac-btn {
    width: 100%;
  }

  .hint {
    display: none;
  }

  .comment-item {
    flex-direction: row;
    gap: 10px;
    padding: 12px;
  }

  .comment-avatar {
    width: 36px;
    height: 36px;
  }

  .comment-avatar svg {
    width: 18px;
    height: 18px;
  }

  .comment-header {
    flex-wrap: wrap;
    gap: 6px;
    align-items: flex-start;
  }

  .comment-author {
    font-size: 14px;
  }

  .comment-id {
    display: none;
  }

  .comment-time {
    font-size: 11px;
  }

  .comment-actions {
    margin-left: 0;
    width: 100%;
    margin-top: 4px;
  }

  .comment-text {
    font-size: 14px;
    line-height: 1.6;
  }

  .action-btn {
    padding: 6px 10px;
    font-size: 12px;
    flex: 1;
  }

  .replies-list {
    padding-left: 10px;
    margin-top: 8px;
  }

  .reply-item {
    flex-direction: row;
    gap: 8px;
    padding: 8px 0;
  }

  .reply-avatar {
    width: 28px;
    height: 28px;
  }

  .reply-avatar svg {
    width: 14px;
    height: 14px;
  }

  .reply-header {
    flex-wrap: wrap;
    gap: 4px;
  }

  .reply-actions {
    width: 100%;
    margin-top: 4px;
  }

  .reply-text {
    font-size: 13px;
  }

  .login-prompt {
    flex-direction: column;
    gap: 10px;
    padding: 14px;
    text-align: center;
  }

  .login-prompt p {
    margin: 0;
    font-size: 14px;
  }

  .login-prompt .mac-btn {
    width: 100%;
  }

  .back-btn {
    padding: 10px 16px;
    font-size: 14px;
  }

  .back-icon {
    width: 18px;
    height: 18px;
  }

  .message-bar {
    padding: 12px 14px;
    font-size: 14px;
  }

  .report-modal {
    width: 95%;
    max-width: none;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.98);
  }

  .report-modal-body {
    padding: 16px;
  }

  .report-input {
    padding: 12px 14px;
    font-size: 15px;
  }

  .claim-modal {
    width: 95%;
    max-width: none;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.98);
  }

  .claim-modal-body {
    padding: 16px;
  }

  .claim-textarea {
    min-height: 140px;
    font-size: 15px;
  }
}

@media screen and (max-width: 480px) {
  .content-title {
    font-size: 18px;
  }

  .type-badge,
  .status-badge {
    padding: 3px 8px;
    font-size: 12px;
  }

  .content-image {
    width: 90%;
    max-height: none;
  }

  .content-video {
    max-height: 220px;
  }

  .comments-section {
    padding: 12px;
  }

  .comment-item {
    padding: 10px;
    gap: 8px;
  }

  .comment-avatar {
    width: 32px;
    height: 32px;
  }

  .comment-avatar svg {
    width: 16px;
    height: 16px;
  }

  .comment-author {
    font-size: 13px;
  }

  .comment-time {
    font-size: 10px;
  }

  .comment-text {
    font-size: 13px;
  }

  .action-btn {
    padding: 5px 8px;
    font-size: 11px;
  }

  .replies-list {
    padding-left: 8px;
    margin-top: 6px;
  }

  .reply-item {
    gap: 6px;
    padding: 6px 0;
  }

  .reply-avatar {
    width: 24px;
    height: 24px;
  }

  .reply-avatar svg {
    width: 12px;
    height: 12px;
  }

  .reply-text {
    font-size: 12px;
  }

  .reply-author {
    font-size: 11px;
  }

  .reply-time {
    font-size: 10px;
  }

  .reply-actions .action-btn {
    padding: 3px 6px;
    font-size: 10px;
  }
}
</style>

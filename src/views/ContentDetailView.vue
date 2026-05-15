<script setup lang="ts">
import { ref, computed, onMounted, defineComponent, h } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { RouterLink } from 'vue-router'
import { contentApi, commentApi } from '@/api'
import { useUserStore } from '@/stores/user'
import type { Content, Comment } from '@/types'
import { renderMarkdown } from '@/utils'

const CommentItem = defineComponent({
  name: 'CommentItem',
  props: {
    comment: { type: Object as () => Comment, required: true },
    replyTarget: { type: Object as () => Comment | null, required: true },
    menuTarget: { type: Number, required: true },
    level: { type: Number, default: 0 }
  },
  emits: ['select-reply', 'toggle-menu', 'delete-comment', 'report-comment'],
  setup(props, { emit }) {
    const userStore = useUserStore()

    const formatTime = (ts: number | string) => {
      if (!ts) return ''
      const timestamp = typeof ts === 'string' ? parseInt(ts) : ts
      return new Date(timestamp * 1000).toLocaleString('zh-CN')
    }

    return () => h('div', { class: 'flex gap-2 sm:gap-3 p-2 sm:p-3 bg-white/60 rounded-lg border border-white/36' }, [
      h('div', { class: 'w-9 h-9 sm:w-10 sm:h-10 rounded-full bg-blue-500/10 flex items-center justify-center shrink-0' }, [
        h('svg', { 
          class: 'w-4 h-4 sm:w-5 sm:h-5 text-blue-500', 
          viewBox: '0 0 24 24', 
          fill: 'none', 
          stroke: 'currentColor', 
          'stroke-width': '2'
        }, [
          h('path', { d: 'M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2' }),
          h('circle', { cx: '12', cy: '7', r: '4' })
        ])
      ]),
      h('div', { class: 'flex-1 relative' }, [
        h('div', { class: 'flex items-center gap-2 sm:gap-3 mb-1 sm:mb-2' }, [
          h('span', { class: 'font-semibold text-sm text-gray-900' }, props.comment.user?.username),
          h('span', { class: 'text-xs text-gray-400' }, formatTime(props.comment.created_at)),
          userStore.isLoggedIn && (userStore.user?.is_admin || props.comment.user_id === userStore.user?.id) ? h('button', {
            class: 'ml-auto p-1 text-gray-400 hover:text-gray-700 transition-colors',
            onClick: (e: Event) => {
              e.stopPropagation()
              emit('toggle-menu', props.comment.id)
            }
          }, [
            h('svg', {
              class: 'w-4 h-4',
              viewBox: '0 0 24 24',
              fill: 'none',
              stroke: 'currentColor',
              'stroke-width': '2'
            }, [
              h('circle', { cx: '12', cy: '12', r: '1' }),
              h('circle', { cx: '19', cy: '12', r: '1' }),
              h('circle', { cx: '5', cy: '12', r: '1' })
            ])
          ]) : null
        ]),
        h('div', { class: 'text-xs sm:text-sm text-gray-700 leading-relaxed' }, [
          props.comment.parent ? h('div', { class: 'mb-1 p-1.5 bg-gray-100 rounded text-xs text-gray-500 border-l-2 border-blue-500' }, [
            h('span', { class: 'font-medium text-gray-600' }, props.comment.parent.user?.username + ': '),
            props.comment.parent.text
          ]) : null,
          h('span', props.comment.text)
        ]),
        h('div', { class: 'mt-1 sm:mt-2' }, [
          userStore.isLoggedIn ? h('button', {
            class: 'flex items-center gap-1 text-xs text-gray-400 hover:text-blue-500 transition-colors',
            onClick: () => emit('select-reply', props.comment)
          }, [
            h('svg', {
              class: 'w-3.5 h-3.5',
              viewBox: '0 0 24 24',
              fill: 'none',
              stroke: 'currentColor',
              'stroke-width': '2'
            }, [
              h('path', { d: 'M11 5H6a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h11a2 2 0 0 0 2-2v-5m-1.414-9.414a2 2 0 1 1 2.828 2.828L11.828 15H9v-2.828l8.586-8.586z' })
            ]),
            h('span', '回复')
          ]) : null
        ]),
        
        props.menuTarget === props.comment.id ? h('div', {
          class: 'absolute right-0 top-0 bg-white rounded-lg shadow-lg shadow-black/12 p-1 min-w-[120px] z-10'
        }, [
          h('button', {
            class: 'flex items-center gap-2 w-full px-3 py-2 text-sm text-red-500 hover:bg-red-50 rounded',
            onClick: () => {
              emit('delete-comment', props.comment.id)
              emit('toggle-menu', null)
            }
          }, [
            h('svg', {
              class: 'w-4 h-4',
              viewBox: '0 0 24 24',
              fill: 'none',
              stroke: 'currentColor',
              'stroke-width': '2'
            }, [
              h('path', { d: 'M3 6h18' }),
              h('path', { d: 'M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6' }),
              h('path', { d: 'M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2' })
            ]),
            '删除'
          ]),
          h('button', {
            class: 'flex items-center gap-2 w-full px-3 py-2 text-sm text-amber-500 hover:bg-amber-50 rounded',
            onClick: () => {
              emit('report-comment', props.comment)
              emit('toggle-menu', null)
            }
          }, [
            h('svg', {
              class: 'w-4 h-4',
              viewBox: '0 0 24 24',
              fill: 'none',
              stroke: 'currentColor',
              'stroke-width': '2'
            }, [
              h('path', { d: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z' })
            ]),
            '举报'
          ])
        ]) : null,

        props.comment.replies && props.comment.replies.length > 0 ? h('div', {
          class: 'mt-2 sm:mt-3 pl-3 sm:pl-4 border-l-2 border-blue-500/20'
        }, props.comment.replies.map(reply => 
          h(CommentItem, {
            key: reply.id,
            comment: reply,
            replyTarget: props.replyTarget,
            menuTarget: props.menuTarget,
            level: props.level + 1,
            onSelectReply: (c: Comment) => emit('select-reply', c),
            onToggleMenu: (id: number) => emit('toggle-menu', id),
            onDeleteComment: (id: number) => emit('delete-comment', id),
            onReportComment: (c: Comment) => emit('report-comment', c)
          })
        )) : null
      ])
    ])
  }
})

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

const menuTarget = ref<number | null>(null)

const currentPage = ref(1)
const pageSize = ref(20)
const totalComments = ref(0)
const totalPages = ref(1)

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

function toggleMenu(id: number) {
  if (menuTarget.value === id) {
    menuTarget.value = null
  } else {
    menuTarget.value = id
  }
}

function closeMenu() {
  menuTarget.value = null
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

async function loadComments(page: number = 1) {
  try {
    const id = Number(route.params.id)
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
      await loadComments(1)
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
      await loadComments(currentPage.value)
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
  menuTarget.value = null
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
  <div class="min-h-screen p-2 sm:p-5 flex justify-center">
    <div class="w-full max-w-[800px] overflow-hidden bg-white/75 rounded-xl shadow-lg shadow-black/5 border border-white/40">
      <div class="flex items-center px-3 py-2 sm:px-4 sm:py-2.5 bg-gradient-to-b from-black/8 to-black/2">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm text-gray-500 font-medium">内容详情</div>
      </div>

      <div class="p-4 sm:p-6">
        <div v-if="message" class="px-3 py-2 sm:px-4 sm:py-3 rounded-lg mb-3 sm:mb-4 flex justify-between items-center" :class="{ 'bg-red-100 text-red-600': message.includes('失败') || message.includes('请'), 'bg-emerald-100 text-emerald-600': message.includes('成功') }">
          <span class="text-sm">{{ message }}</span>
          <span class="text-lg font-bold cursor-pointer" @click="message = ''">×</span>
        </div>

        <div v-if="content" class="animate-[fadeIn_0.3s_ease]">
          <div class="mb-3 sm:mb-5">
            <button @click="goBack" class="flex items-center gap-1.5 px-3 py-2 sm:px-4 sm:py-2.5 bg-black/5 rounded-lg text-sm text-gray-700 hover:bg-black/10 hover:text-blue-500 transition-all">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 12H5"></path>
                <polyline points="12 19 5 12 12 5"></polyline>
              </svg>
              <span class="hidden sm:inline">返回</span>
            </button>
          </div>

          <div class="bg-white/60 rounded-xl p-4 sm:p-6 mb-3 sm:mb-4 shadow-md shadow-black/8 border border-white/36">
            <div class="mb-3 sm:mb-4">
              <h1 class="text-lg sm:text-xl font-bold text-gray-900 mb-2 sm:mb-3 leading-relaxed">{{ content.title }}</h1>
              <div class="flex gap-2 sm:gap-3">
                <span :class="[content.type === 'video' ? 'bg-red-100 text-red-600' : content.type === 'image' ? 'bg-emerald-100 text-emerald-600' : content.type === 'link' ? 'bg-violet-100 text-violet-600' : 'bg-blue-100 text-blue-600']" class="px-2 py-0.5 sm:px-3 sm:py-1 rounded-full text-xs sm:text-sm font-medium">
                  {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : content.type === 'link' ? '链接' : '文字' }}
                </span>
                <span :class="[content.audit_status === 'approved' ? 'bg-emerald-100 text-emerald-600' : content.audit_status === 'pending' ? 'bg-amber-100 text-amber-600' : 'bg-red-100 text-red-600']" class="px-2 py-0.5 sm:px-3 sm:py-1 rounded-full text-xs sm:text-sm font-medium">
                  {{ content.audit_status === 'approved' ? '已通过' : content.audit_status === 'pending' ? '审核中' : '已拒绝' }}
                </span>
              </div>
            </div>

            <div class="flex flex-wrap gap-3 sm:gap-4 mb-3 sm:mb-4 pb-3 sm:pb-4 border-b border-black/6">
              <div class="flex items-center gap-1.5 text-xs sm:text-sm text-gray-600">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                  <circle cx="12" cy="7" r="4"></circle>
                </svg>
                <span>{{ content.user?.username }}</span>
                <button
                  @click="userStore.isLoggedIn ? showClaimModal = true : router.push('/login')"
                  class="text-gray-400 hover:text-blue-500 text-xs sm:text-sm underline underline-offset-2 decoration-dotted ml-1 sm:ml-2"
                >
                  认领内容
                </button>
              </div>
              <div v-if="(content.tags || []).length > 0" class="flex items-center gap-1.5 text-xs sm:text-sm text-gray-600">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path>
                </svg>
                <span v-for="tag in (content.tags || [])" :key="tag" class="px-1.5 py-0.5 bg-blue-500/10 rounded text-blue-500 text-xs">{{ tag }}</span>
              </div>
            </div>

            <div class="flex flex-wrap gap-3 sm:gap-4 mb-4 sm:mb-5 text-xs sm:text-sm text-gray-400">
              <span class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                {{ formatTime(content.created_at) }}
              </span>
              <span v-if="content.updated_at !== content.created_at" class="flex items-center gap-1.5">
                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"></polyline>
                  <polyline points="18 21 23 21 23 15"></polyline>
                  <path d="M20.49 9h-5.99M14.51 21h-5.99M9 9H3m3 12H3"></path>
                </svg>
                {{ formatTime(content.updated_at) }}
              </span>
            </div>

            <div>
              <div v-if="content.type === 'text'" class="text-gray-800 text-sm sm:text-base leading-relaxed" v-html="renderedContent"></div>
              <div v-else-if="content.type === 'image'" class="flex justify-center">
                <img :src="getImageUrl(content.img)" alt="内容图片" class="w-full sm:w-[90%] rounded-xl shadow-lg shadow-black/10">
              </div>
              <div v-else-if="content.type === 'video'" class="flex justify-center">
                <video controls class="w-full sm:w-[90%] max-h-[300px] sm:max-h-[450px] rounded-xl bg-black">
                  <source :src="getImageUrl(content.video)">
                  您的浏览器不支持视频播放。
                </video>
              </div>
              <div v-else-if="content.type === 'link'" class="text-gray-800 text-sm sm:text-base leading-relaxed" v-html="renderedContent"></div>
            </div>
          </div>

          <div class="bg-white/60 rounded-xl p-3 sm:p-4 shadow-md shadow-black/8 border border-white/36">
            <div class="mb-3 sm:mb-4 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
              <h3 class="text-base sm:text-lg font-semibold text-gray-900">评论 ({{ totalComments }})</h3>
              <div class="flex items-center gap-2">
                <button
                  v-if="currentPage > 1"
                  @click="goToPrevPage"
                  class="px-3 py-1 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded transition-colors"
                >
                  上一页
                </button>
                <span class="text-sm text-gray-500">
                  第 {{ currentPage }} / {{ totalPages }} 页
                </span>
                <button
                  v-if="currentPage < totalPages"
                  @click="goToNextPage"
                  class="px-3 py-1 text-sm text-gray-600 bg-gray-100 hover:bg-gray-200 rounded transition-colors"
                >
                  下一页
                </button>
              </div>
            </div>

            <div v-if="userStore.isLoggedIn" class="mb-3 sm:mb-4">
              <div v-if="replyTarget" class="flex justify-between items-center px-3 py-2 bg-blue-500/10 rounded-lg mb-2 text-sm text-blue-500">
                <span>回复 {{ replyTarget.user?.username }}:</span>
                <button @click="cancelReply" class="text-gray-500 hover:text-blue-500 font-medium">取消</button>
              </div>
              <textarea
                v-model="commentText"
                class="w-full min-h-[70px] sm:min-h-[80px] px-3 py-2 sm:px-4 sm:py-3 border border-black/10 rounded-lg bg-white text-sm resize-vertical focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10"
                placeholder="写下你的评论..."
                @keyup.enter.ctrl="submitComment"
              ></textarea>
              <div class="flex flex-col sm:flex-row justify-between items-center gap-2 sm:gap-4 mt-2">
                <span class="text-xs text-gray-400 hidden sm:block">Ctrl + Enter 发送</span>
                <button @click="submitComment" class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 transition-all">发表评论</button>
              </div>
            </div>

            <div v-else class="flex flex-col sm:flex-row items-center gap-3 sm:gap-4 p-3 sm:p-4 bg-black/3 rounded-lg mb-3 sm:mb-4">
              <p class="text-sm text-gray-600">请先登录以发表评论</p>
              <RouterLink to="/login" class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium text-center hover:bg-blue-600 transition-all">登录</RouterLink>
            </div>

            <div v-if="comments.length > 0" class="space-y-2 sm:space-y-3">
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

            <div v-else class="flex flex-col items-center py-8 sm:py-10 text-gray-400">
              <svg class="w-10 h-10 sm:w-12 sm:h-12 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path>
              </svg>
              <p class="text-sm">暂无评论，快来发表第一条评论吧</p>
            </div>
          </div>
        </div>

        <div v-else class="flex flex-col items-center py-12 sm:py-20 text-gray-400">
          <svg class="w-16 h-16 mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 20h-8l-4-4H4a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
            <line x1="9" y1="15" x2="15" y2="15"></line>
          </svg>
          <p class="text-sm">加载中...</p>
        </div>
      </div>

      <Teleport to="body">
        <div v-if="reportTarget" class="fixed inset-0 flex items-center justify-center bg-black/50 z-[9999]" @click.self="reportTarget = null">
          <div class="w-[90%] sm:w-[400px] bg-white/98 rounded-xl shadow-2xl overflow-hidden">
            <div class="flex justify-between items-center px-4 py-3 border-b border-black/6 bg-gradient-to-b from-black/5 to-transparent">
              <h3 class="font-semibold text-gray-900">举报评论</h3>
              <button @click="reportTarget = null" class="text-gray-400 hover:text-blue-500 text-xl leading-none">×</button>
            </div>
            <div class="p-4">
              <p class="text-sm text-gray-700 mb-2">您正在举报以下评论：</p>
              <p class="px-3 py-2 bg-black/3 rounded-lg text-sm text-gray-500 italic mb-3">{{ reportTarget.text }}</p>
              <label class="block text-sm text-gray-600 mb-2">举报原因（可选）</label>
              <input v-model="reportReason" type="text" class="w-full px-3 py-2 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10" placeholder="请输入举报原因">
            </div>
            <div class="flex justify-end gap-3 px-4 py-3 border-t border-black/6 bg-black/3">
              <button @click="reportTarget = null" class="px-4 py-2 text-sm text-gray-700 hover:text-blue-500 transition-colors">取消</button>
              <button @click="submitReport" class="px-4 py-2 bg-blue-500 text-white text-sm font-medium rounded-lg hover:bg-blue-600 transition-colors">确认举报</button>
            </div>
          </div>
        </div>

        <div v-if="showClaimModal" class="fixed inset-0 flex items-center justify-center bg-black/50 z-[9999]" @click.self="showClaimModal = false">
          <div class="w-[90%] sm:w-[450px] bg-white/98 rounded-xl shadow-2xl overflow-hidden">
            <div class="flex justify-between items-center px-4 py-3 border-b border-black/6 bg-gradient-to-b from-blue-500/5 to-transparent">
              <h3 class="font-semibold text-gray-900">认领此内容</h3>
              <button @click="showClaimModal = false" class="text-gray-400 hover:text-blue-500 text-xl leading-none">×</button>
            </div>
            <div class="p-4">
              <p class="text-sm text-gray-600 mb-4 leading-relaxed">请提供认领理由，管理员将在审核后决定是否将此内容转移给您。</p>
              <label class="block text-sm font-medium text-gray-700 mb-2">认领理由 <span class="text-red-500">*</span></label>
              <textarea
                v-model="claimReason"
                class="w-full min-h-[120px] px-3 py-2 border border-black/10 rounded-lg text-sm bg-white resize-vertical focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 disabled:opacity-60"
                placeholder="请详细说明您认为此内容应归属于您的原因..."
                :disabled="isSubmittingClaim"
              ></textarea>
            </div>
            <div class="flex justify-end gap-3 px-4 py-3 border-t border-black/6 bg-black/3">
              <button @click="showClaimModal = false" class="px-4 py-2 text-sm text-gray-700 hover:text-blue-500 transition-colors" :disabled="isSubmittingClaim">取消</button>
              <button @click="submitClaim" class="px-4 py-2 bg-blue-500 text-white text-sm font-medium rounded-lg hover:bg-blue-600 transition-colors disabled:opacity-50" :disabled="isSubmittingClaim">
                {{ isSubmittingClaim ? '提交中...' : '提交申请' }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="menuTarget" class="fixed inset-0 z-[90]" @click="closeMenu"></div>
      </Teleport>
    </div>
  </div>
</template>

<style>
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>

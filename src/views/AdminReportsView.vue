<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { commentApi } from '@/api'
import type { CommentReport } from '@/types'

const router = useRouter()
const reports = ref<CommentReport[]>([])
const message = ref('')
const showDeleteConfirm = ref(false)
const deleteTargetId = ref(0)
const deleteReportId = ref(0)

async function loadReports() {
  try {
    const res = await commentApi.getReports()
    if (res.code === 200) {
      reports.value = res.data
    } else {
      message.value = res.message
    }
  } catch {
    message.value = '加载举报列表失败'
  }
}

async function handleReport(reportId: number) {
  try {
    const res = await commentApi.handleReport(reportId)
    if (res.code === 200) {
      message.value = '处理成功'
      await loadReports()
    } else {
      message.value = res.message || '处理失败'
    }
  } catch {
    message.value = '处理失败'
  }
}

function confirmDelete(commentId: number, reportId: number) {
  deleteTargetId.value = commentId
  deleteReportId.value = reportId
  showDeleteConfirm.value = true
}

async function deleteComment() {
  const commentId = deleteTargetId.value
  const reportId = deleteReportId.value
  
  try {
    const deleteRes = await commentApi.delete(commentId)
    if (deleteRes.code === 200) {
      await commentApi.handleReport(reportId)
      message.value = '删除成功'
      await loadReports()
    } else {
      message.value = deleteRes.message || '删除失败'
    }
  } catch {
    message.value = '删除失败'
  }
  showDeleteConfirm.value = false
}

function goBack() {
  router.back()
}

onMounted(() => {
  loadReports()
})
</script>

<template>
  <div class="min-h-screen p-2 sm:p-5 flex justify-center">
    <div class="w-full max-w-[900px] overflow-hidden bg-[var(--theme-surface)] rounded-xl shadow-lg shadow-black/5 border border-[var(--theme-card-border)]">
      <div class="flex items-center px-3 py-2 sm:px-4 sm:py-2.5 theme-header-bg">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm theme-text-secondary font-medium">举报管理</div>
      </div>

      <div class="p-4 sm:p-6">
        <div v-if="message" class="px-3 py-2 sm:px-4 sm:py-3 rounded-lg mb-3 sm:mb-4 flex justify-between items-center bg-[var(--theme-success)]/10 text-[var(--theme-success)]">
          <span class="text-sm">{{ message }}</span>
          <span class="text-lg font-bold cursor-pointer" @click="message = ''">×</span>
        </div>

        <div class="flex flex-col sm:flex-row items-start sm:items-center gap-3 sm:gap-4 mb-4 sm:mb-6">
          <button @click="goBack" class="flex items-center gap-1.5 px-3 py-2 sm:px-4 sm:py-2.5 bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-lg text-sm theme-text hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)] transition-all">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5"/>
              <polyline points="12 19 5 12 12 5"/>
            </svg>
            <span>返回</span>
          </button>
          <h2 class="text-lg sm:text-xl font-semibold theme-text">举报列表</h2>
        </div>

        <div v-if="reports.length > 0" class="space-y-3 sm:space-y-4">
          <div v-for="report in reports" :key="report.id" class="bg-[var(--theme-card-bg)] rounded-xl p-3 sm:p-4 shadow-md shadow-black/8 border border-[var(--theme-card-border)]">
            <div class="flex flex-wrap items-center gap-2 sm:gap-3 mb-3 sm:mb-4">
              <span class="font-semibold text-[var(--theme-primary)] text-sm">#{{ report.id }}</span>
              <span class="text-xs sm:text-sm theme-text-secondary">{{ report.created_at }}</span>
              <span :class="[report.handled ? 'bg-[var(--theme-success)]/10 text-[var(--theme-success)]' : 'bg-[var(--theme-warning)]/10 text-[var(--theme-warning)]']" class="px-2 py-0.5 sm:px-3 sm:py-1 rounded-full text-xs font-medium">
                {{ report.handled ? '已处理' : '待处理' }}
              </span>
            </div>
            
            <div class="pb-3 sm:pb-4 border-b border-[var(--theme-card-border)] mb-3 sm:mb-4">
              <p class="text-xs sm:text-sm theme-text-secondary mb-1">
                <span class="font-medium theme-text-secondary">举报原因：</span>
                {{ report.reason || '其他' }}
              </p>
              <p class="text-xs sm:text-sm theme-text-secondary italic mb-1">
                <span class="font-medium theme-text-secondary">被举报内容：</span>
                {{ report.Comment?.text }}
              </p>
              <p class="text-xs sm:text-sm theme-text-secondary">
                <span class="font-medium theme-text-secondary">举报人：</span>
                {{ report.User?.username }}
              </p>
            </div>

            <div class="flex flex-col sm:flex-row justify-end gap-2 sm:gap-3">
              <button 
                v-if="!report.handled" 
                @click="handleReport(report.id)" 
                class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-[var(--theme-primary)] text-white rounded-lg text-sm font-medium hover:brightness-90 transition-all"
              >
                标记已处理
              </button>
              <button 
                @click="confirmDelete(report.comment_id, report.id)" 
                class="w-full sm:w-auto px-4 sm:px-5 py-2 bg-[var(--theme-danger)]/50/10 text-[var(--theme-danger)] border border-red-500/30 rounded-lg text-sm font-medium hover:bg-[var(--theme-danger)]/50/15 transition-all"
              >
                删除评论
              </button>
            </div>
          </div>
        </div>

        <div v-else class="flex flex-col items-center py-8 sm:py-12 theme-text-secondary">
          <svg class="w-12 h-12 sm:w-16 sm:h-16 mb-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p class="text-sm">暂无举报</p>
        </div>
      </div>

      <Teleport to="body">
        <div v-if="showDeleteConfirm" class="fixed inset-0 flex items-center justify-center bg-[var(--theme-hover-bg)]0 z-[9999]" @click.self="showDeleteConfirm = false">
          <div class="w-[90%] sm:w-[400px] bg-[var(--theme-surface)] rounded-xl shadow-2xl overflow-hidden">
            <div class="flex justify-between items-center px-4 py-3 border-b border-[var(--theme-card-border)] bg-gradient-to-b from-black/5 to-transparent">
              <h3 class="font-semibold theme-text">确认删除</h3>
              <button @click="showDeleteConfirm = false" class="theme-text-secondary hover:text-[var(--theme-primary)] text-xl leading-none">×</button>
            </div>
            <div class="p-4">
              <p class="text-sm theme-text">确定要删除这条评论吗？此操作不可撤销。</p>
            </div>
            <div class="flex justify-end gap-3 px-4 py-3 border-t border-[var(--theme-card-border)] bg-[var(--theme-hover-bg)]">
              <button @click="showDeleteConfirm = false" class="px-4 py-2 text-sm theme-text hover:text-[var(--theme-primary)] transition-colors">取消</button>
              <button @click="deleteComment" class="px-4 py-2 bg-[var(--theme-danger)]/50 text-white text-sm font-medium rounded-lg hover:bg-red-600 transition-colors">确定删除</button>
            </div>
          </div>
        </div>
      </Teleport>
    </div>
  </div>
</template>
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
  <div class="reports-container">
    <div class="mac-window reports-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">举报管理</div>
      </div>

      <div class="reports-content">
        <div v-if="message" class="message-bar">
          {{ message }}
          <span class="message-close" @click="message = ''">×</span>
        </div>

        <div class="reports-header">
          <button @click="goBack" class="back-btn mac-btn small-btn">
            <svg class="back-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5"/>
              <polyline points="12 19 5 12 12 5"/>
            </svg>
            返回
          </button>
          <h2>举报列表</h2>
        </div>

        <div v-if="reports.length > 0" class="reports-list">
          <div v-for="report in reports" :key="report.id" class="report-item">
            <div class="report-info">
              <div class="report-header">
                <span class="report-id">#{{ report.id }}</span>
                <span class="report-time">{{ report.created_at }}</span>
                <span :class="['report-status', report.handled ? 'handled' : 'pending']">
                  {{ report.handled ? '已处理' : '待处理' }}
                </span>
              </div>
              
              <div class="report-content">
                <p class="report-reason">
                  <span class="label">举报原因：</span>
                  {{ report.reason || '其他' }}
                </p>
                <p class="report-comment">
                  <span class="label">被举报内容：</span>
                  {{ report.Comment?.text }}
                </p>
                <p class="report-reporter">
                  <span class="label">举报人：</span>
                  {{ report.User?.username }}
                </p>
              </div>
            </div>

            <div class="report-actions">
              <button 
                v-if="!report.handled" 
                @click="handleReport(report.id)" 
                class="mac-btn"
              >
                标记已处理
              </button>
              <button 
                @click="confirmDelete(report.comment_id, report.id)" 
                class="mac-btn delete-btn"
              >
                删除评论
              </button>
            </div>
          </div>
        </div>

        <div v-else class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
          </svg>
          <p>暂无举报</p>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showDeleteConfirm" class="tag-modal-overlay" @click.self="showDeleteConfirm = false">
        <div class="tag-modal">
          <div class="tag-modal-header">
            <h3>确认删除</h3>
            <button class="tag-modal-close" @click="showDeleteConfirm = false">×</button>
          </div>
          <div class="tag-modal-body">
            <p>确定要删除这条评论吗？此操作不可撤销。</p>
          </div>
          <div class="tag-modal-footer">
            <button @click="showDeleteConfirm = false" class="mac-btn">取消</button>
            <button @click="deleteComment" class="mac-btn delete-btn">确定删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.reports-container {
  min-height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.reports-window {
  width: 100%;
  max-width: 900px;
  min-height: 600px;
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

.reports-content {
  padding: 24px;
  background: rgba(255, 255, 255, 0.75);
}

.message-bar {
  padding: 10px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #ecfdf5;
  color: #059669;
}

.message-close {
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}

.reports-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.reports-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  padding: 10px 16px;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.back-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.back-icon {
  width: 16px;
  height: 16px;
}

.reports-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.report-item {
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  padding: 20px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
}

.report-info {
  flex: 1;
}

.report-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.report-id {
  font-weight: 600;
  color: #3b82f6;
}

.report-time {
  font-size: 13px;
  color: #999;
}

.report-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  margin-left: auto;
}

.report-status.pending {
  background: #fef3c7;
  color: #d97706;
}

.report-status.handled {
  background: #ecfdf5;
  color: #059669;
}

.report-content {
  padding-bottom: 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.report-content p {
  margin: 8px 0;
  font-size: 14px;
  color: #333;
}

.report-content .label {
  color: #999;
  font-weight: 500;
}

.report-comment {
  font-style: italic;
  color: #666 !important;
}

.report-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 16px;
}

.delete-btn {
  background: rgba(220, 38, 38, 0.1);
  color: #dc2626;
  border: 1px solid rgba(220, 38, 38, 0.3);
}

.delete-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
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

.tag-modal-overlay {
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

.tag-modal {
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.tag-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: linear-gradient(180deg, rgba(0,0,0,0.05) 0%, transparent 100%);
}

.tag-modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.tag-modal-close {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #999;
  line-height: 1;
  transition: color 0.2s ease;
}

.tag-modal-close:hover {
  color: #3b82f6;
}

.tag-modal-body {
  padding: 20px;
}

.tag-modal-body p {
  margin: 0;
  font-size: 14px;
  color: #333;
}

.tag-modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(0, 0, 0, 0.03);
}

@media screen and (max-width: 768px) {
  .reports-container {
    padding: 0;
    min-height: calc(100vh - 60px);
  }

  .reports-window {
    border-radius: 0;
    box-shadow: none;
    min-height: calc(100vh - 60px);
    background: rgba(255, 255, 255, 0.9);
  }

  .mac-title-bar {
    display: none;
  }

  .reports-content {
    padding: 16px;
  }

  .reports-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .reports-header h2 {
    font-size: 18px;
  }

  .back-btn {
    padding: 10px 16px;
    font-size: 14px;
  }

  .report-item {
    padding: 16px;
  }

  .report-header {
    flex-wrap: wrap;
    gap: 8px;
  }

  .report-status {
    margin-left: 0;
  }

  .report-content p {
    font-size: 14px;
  }

  .report-actions {
    flex-direction: column;
    gap: 10px;
    margin-top: 12px;
  }

  .report-actions .mac-btn {
    width: 100%;
    padding: 12px;
    font-size: 15px;
  }

  .empty-state {
    padding: 40px 16px;
  }

  .empty-icon {
    width: 48px;
    height: 48px;
  }

  .message-bar {
    padding: 12px 14px;
    font-size: 14px;
  }

  .tag-modal {
    width: 95%;
    max-width: none;
    border-radius: 12px;
  }
}
</style>

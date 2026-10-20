<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { IconCheck, IconDelete, IconExclamationCircle } from '@arco-design/web-vue/es/icon'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import { useAdminStore } from '@/stores/admin'
import { formatTime, prefersReducedMotion } from '@/utils'
import { useConfirm } from '@/composables/useToast'

const admin = useAdminStore()

const showSkeleton = computed(() => admin.reportsLoading && admin.reports.length === 0)
const reportSummary = computed(() => `共 ${admin.reports.length} 条举报`)

const LEAVE_MS = 280
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
let disposed = false

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

async function load() {
  await admin.loadReports()
}

function scheduleLeave(id: number, refresh: () => unknown) {
  if (disposed) return
  if (prefersReducedMotion()) {
    void refresh()
    return
  }
  if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
  timers.add(
    setTimeout(() => {
      void (async () => {
        try {
          await refresh()
        } catch {
        } finally {
          leavingIds.value = leavingIds.value.filter(item => item !== id)
        }
      })()
    }, LEAVE_MS),
  )
}

async function doHandle(id: number) {
  if (await admin.handleReport(id)) scheduleLeave(id, load)
}

async function doDelete(commentId: number, reportId: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('删除该评论？删除后不可恢复。')
  if (!ok) return
  if (await admin.deleteComment(commentId)) {
    await admin.handleReport(reportId)
    scheduleLeave(reportId, load)
  }
}

onMounted(load)

onBeforeUnmount(() => {
  disposed = true
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
})
</script>

<template>
  <AdminPanel title="举报管理" desc="审核用户对评论的举报；删除评论不可恢复。">
    <template #toolbar>
      <AdminDataToolbar :summary="reportSummary" :loading="admin.reportsLoading" refreshable @refresh="load" />
    </template>

    <AdminSkeleton v-if="showSkeleton" variant="cards" :rows="6" label="正在加载举报" />

    <AdminEmpty
      v-else-if="admin.reports.length === 0"
      title="暂无举报"
      desc="举报会出现在这里；处理后可删除违规评论。"
    />

    <div v-else class="adm-card-grid report-grid">
      <article
        v-for="record in admin.reports"
        :key="record.id"
        class="adm-card"
        :class="{ 'is-leaving': isLeaving(record.id) }"
      >
        <div class="adm-card-media is-placeholder is-quiet">
          <div class="report-glyph">
            <span class="report-glyph-icon" aria-hidden="true"><IconExclamationCircle /></span>
            <span class="report-glyph-text">举报 #{{ record.id }}</span>
          </div>
          <span class="adm-card-badge">
            <AdminStatus
              :type="record.handled ? 'success' : 'warning'"
              :label="record.handled ? '已处理' : '待处理'"
            />
          </span>
        </div>

        <div class="adm-card-body">
          <h3 class="adm-card-title" :title="record.reason || '其他'">{{ record.reason || '其他' }}</h3>
          <div class="adm-card-meta">
            <span>举报人 {{ record.User?.username || '未知' }}</span>
            <span class="report-time">{{ formatTime(record.created_at) }}</span>
          </div>
          <p v-if="record.Comment?.text" class="report-quote" :title="record.Comment?.text">
            {{ record.Comment?.text }}
          </p>
        </div>

        <div class="adm-card-actions report-actions">
          <a-button
            v-if="!record.handled"
            type="primary"
            size="small"
            aria-label="标记已处理"
            @click="doHandle(record.id)"
          >
            <template #icon><IconCheck /></template>
            处理
          </a-button>
          <a-button
            status="danger"
            size="small"
            aria-label="删除评论"
            @click="doDelete(record.comment_id, record.id)"
          >
            <template #icon><IconDelete /></template>
            删除
          </a-button>
        </div>
      </article>
    </div>
  </AdminPanel>
</template>

<style scoped lang="scss">
:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.report-grid {
  padding: var(--adm-gap-lg, 16px);
}

.report-glyph {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.report-glyph-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--adm-danger, rgb(var(--danger-6))) 12%, transparent);
  color: var(--adm-danger, rgb(var(--danger-6)));
  font-size: 24px;
}

.report-glyph-text {
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: 0.08em;
}

.report-time {
  white-space: nowrap;
}

.report-quote {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  padding-left: 10px;
  border-left: 2px solid var(--adm-line, var(--admin-border));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-sm, 13px);
  font-style: italic;
  line-height: 1.5;
  word-break: break-word;
}

.report-actions {
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .report-grid {
    padding: var(--adm-gap-md, 12px);
  }

  .report-glyph-icon {
    width: 44px;
    height: 44px;
    font-size: 20px;
  }
}
</style>

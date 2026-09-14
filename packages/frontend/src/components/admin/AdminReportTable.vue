<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import { ACTION_COL, STATUS_COL, TIME_COL } from './adminColumns'
import { Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { useMediaQuery } from '@vueuse/core'
import { IconCheck, IconDelete, IconRefresh } from '@arco-design/web-vue/es/icon'
import { useConfirm } from '@/composables/useToast'

const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')

const columns: TableColumnData[] = [
  { title: 'ID', dataIndex: 'id', width: 64, align: 'center' },
  { title: '举报原因', slotName: 'reason', minWidth: 140 },
  { title: '被举报内容', slotName: 'content', minWidth: 180 },
  { title: '举报人', slotName: 'reporter', width: 110 },
  { ...STATUS_COL },
  { ...TIME_COL },
  { ...ACTION_COL },
]

function doHandle(id: number) {
  admin.handleReport(id).then(ok => { if (ok) admin.loadReports() })
}

async function doDelete(commentId: number, reportId: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除？不可撤销。')
  if (!ok) return
  if (await admin.deleteComment(commentId)) { await admin.handleReport(reportId); admin.loadReports() }
}

onMounted(() => admin.loadReports())
</script>

<template>
  <AdminPanel title="举报管理" :desc="`共 ${admin.reports.length} 条举报`">
    <template #actions>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="admin.reportsLoading" @click="admin.loadReports()">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="admin.reportsLoading">
      <a-empty v-if="admin.reports.length === 0" description="暂无举报" />
      <template v-else-if="!isMobile">
        <a-table :columns="columns" :data="admin.reports" :pagination="false" row-key="id">
          <template #reason="{ record }"><span class="admin-cell-2">{{ record.reason || '其他' }}</span></template>
          <template #content="{ record }">
            <span class="report-quote">{{ record.Comment?.text }}</span>
          </template>
          <template #reporter="{ record }"><span class="admin-cell-2">{{ record.User?.username }}</span></template>
          <template #status="{ record }">
            <AdminStatus :type="record.handled ? 'success' : 'warning'" :label="record.handled ? '已处理' : '待处理'" />
          </template>
          <template #time="{ record }"><span class="admin-cell-3">{{ record.created_at }}</span></template>
          <template #actions="{ record }">
            <div class="admin-action-group">
              <Tooltip v-if="!record.handled" title="标记已处理">
                <a-button class="admin-icon-btn is-primary" type="text" size="small" @click="doHandle(record.id)"><IconCheck /></a-button>
              </Tooltip>
              <Tooltip title="删除评论">
                <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="doDelete(record.comment_id, record.id)"><IconDelete /></a-button>
              </Tooltip>
            </div>
          </template>
        </a-table>
      </template>

      <template v-else>
        <div class="admin-mobile-list">
          <div v-for="record in admin.reports" :key="record.id" class="admin-mobile-card">
            <div class="admin-cell-title mb-1">{{ record.reason || '其他' }}</div>
            <div class="report-quote mb-1.5">{{ record.Comment?.text }}</div>
            <div class="flex items-center gap-2 flex-wrap">
              <span class="admin-cell-3">{{ record.User?.username }}</span>
              <AdminStatus :type="record.handled ? 'success' : 'warning'" :label="record.handled ? '已处理' : '待处理'" />
              <span class="admin-cell-3">{{ record.created_at }}</span>
            </div>
            <div class="admin-mobile-actions">
              <a-button v-if="!record.handled" class="admin-icon-btn is-primary" type="text" size="small" @click="doHandle(record.id)"><IconCheck /></a-button>
              <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="doDelete(record.comment_id, record.id)"><IconDelete /></a-button>
            </div>
          </div>
        </div>
      </template>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 被举报评论：引用样式（左侧细条 + 斜体）
.report-quote {
  display: inline-block;
  padding-left: 10px;
  border-left: 2px solid $admin-border;
  font-size: 13px;
  font-style: italic;
  color: $admin-text-2;
  line-height: 1.5;
}
</style>

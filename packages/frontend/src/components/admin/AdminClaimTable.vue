<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import MediaImage from '@/components/MediaImage.vue'
import { ACTION_COL, CONTENT_COL, STATUS_COL, CLAIMER_COL, REASON_COL } from './adminColumns'
import { Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { useMediaQuery } from '@vueuse/core'
import { IconCheck, IconClose, IconRefresh } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')
const statusFilter = ref('')

const columns: TableColumnData[] = [
  { ...CONTENT_COL },
  { ...CLAIMER_COL },
  { ...REASON_COL },
  { ...STATUS_COL },
  { ...ACTION_COL },
]

const statusMap: Record<string, { type: 'warning' | 'success' | 'danger'; label: string }> = {
  pending: { type: 'warning', label: '待处理' },
  approved: { type: 'success', label: '已通过' },
  rejected: { type: 'danger', label: '已拒绝' },
}

async function load() { admin.loadClaims(1, statusFilter.value || undefined) }
async function handleClaim(id: number, action: 'approve' | 'reject') { if (await admin.handleClaim(id, action)) load() }
function onTableChange(current: number) { admin.loadClaims(current, statusFilter.value || undefined) }

onMounted(load)
</script>

<template>
  <AdminPanel title="认领管理" :desc="`共 ${admin.claims.total} 条认领申请`">
    <template #actions>
      <a-select v-model="statusFilter" size="small" class="claim-status-select" @change="load">
        <a-option value="">全部状态</a-option>
        <a-option value="pending">待处理</a-option>
        <a-option value="approved">已通过</a-option>
        <a-option value="rejected">已拒绝</a-option>
      </a-select>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="admin.claims.loading" @click="load">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="admin.claims.loading">
      <template v-if="!isMobile">
        <a-table :columns="columns" :data="admin.claims.list" :pagination="{ current: admin.claims.page, pageSize: admin.claims.pageSize, total: admin.claims.total }" row-key="id" @page-change="onTableChange">
          <template #content="{ record }">
            <div class="flex items-center gap-3">
              <div v-if="record.content?.thumb" class="claim-thumb">
                <MediaImage :src="record.content?.thumb" :preview="false" loading="lazy" alt="" />
              </div>
              <Tooltip :title="record.content?.title || '未知内容'">
                <span class="claim-title admin-cell-title">{{ record.content?.title || '未知内容' }}</span>
              </Tooltip>
            </div>
          </template>
          <template #claimer="{ record }"><span class="admin-cell-2">{{ record.user?.username }}</span></template>
          <template #reason="{ record }"><span class="admin-cell-2">{{ record.reason }}</span></template>
          <template #status="{ record }">
            <AdminStatus :type="statusMap[record.status]?.type" :label="statusMap[record.status]?.label" />
          </template>
          <template #actions="{ record }">
            <div v-if="record.status === 'pending'" class="admin-action-group">
              <Tooltip title="通过">
                <a-button class="admin-icon-btn is-primary" type="text" size="small" @click="handleClaim(record.id, 'approve')"><IconCheck /></a-button>
              </Tooltip>
              <Tooltip title="拒绝">
                <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleClaim(record.id, 'reject')"><IconClose /></a-button>
              </Tooltip>
            </div>
            <span v-else class="admin-cell-3">已处理</span>
          </template>
        </a-table>
      </template>

      <template v-else>
        <div class="admin-mobile-list">
          <div v-for="record in admin.claims.list" :key="record.id" class="admin-mobile-card">
            <div class="claim-title admin-cell-title mb-1">{{ record.content?.title || '未知内容' }}</div>
            <div class="flex items-center gap-2 flex-wrap">
              <span class="admin-cell-3">{{ record.user?.username }}</span>
              <AdminStatus :type="statusMap[record.status]?.type" :label="statusMap[record.status]?.label" />
            </div>
            <div v-if="record.reason" class="admin-cell-2 mt-1.5">{{ record.reason }}</div>
            <div v-if="record.status === 'pending'" class="admin-mobile-actions">
              <a-button class="admin-icon-btn is-primary" type="text" size="small" @click="handleClaim(record.id, 'approve')"><IconCheck /></a-button>
              <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleClaim(record.id, 'reject')"><IconClose /></a-button>
            </div>
          </div>
          <a-empty v-if="admin.claims.list.length === 0" description="暂无认领" />
        </div>
        <div v-if="admin.claims.totalPages > 1" class="admin-mobile-pagination">
          <a-pagination :current="admin.claims.page" :total="admin.claims.total" :page-size="admin.claims.pageSize" size="small" @change="(current: number) => admin.loadClaims(current, statusFilter || undefined)" />
        </div>
      </template>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.claim-thumb {
  @include content-thumb(48px, 36px);
}

.claim-title {
  @include ellipsis;
  max-width: 200px;
  display: inline-block;
}

.claim-status-select {
  width: 120px;
}
</style>

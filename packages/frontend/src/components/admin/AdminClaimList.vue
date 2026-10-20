<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { IconLink } from '@arco-design/web-vue/es/icon'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import MediaImage from '@/components/MediaImage.vue'
import { useAdminStore } from '@/stores/admin'
import { formatTime, prefersReducedMotion } from '@/utils'

const admin = useAdminStore()

const statusFilter = ref('')

const statusMap: Record<string, { type: 'success' | 'warning' | 'danger'; label: string }> = {
  pending: { type: 'warning', label: '待处理' },
  approved: { type: 'success', label: '已通过' },
  rejected: { type: 'danger', label: '已拒绝' },
}

// 认领内容的封面：优先缩略图，没有缩略图退回原图，两者皆空才显示占位图标。
function claimThumb(content?: { thumb?: string; img?: string } | null) {
  return content?.thumb || content?.img || ''
}

// 首屏加载才出骨架；翻页与刷新保留已有卡片，避免整块内容闪烁
const showSkeleton = computed(() => admin.claims.loading && admin.claims.list.length === 0)
const claimSummary = computed(() => `共 ${admin.claims.total} 条认领申请`)

// M8：审核后让卡片先淡出 280ms 再刷新；命中 reduced-motion 时直达终态
const LEAVE_MS = 280
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
let disposed = false

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

async function load() {
  await admin.loadClaims(1, statusFilter.value || undefined)
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
          // 失败提示由 store 统一负责，这里只保证卡片能恢复
        } finally {
          // 刷新成功时卡片已不在列表里；失败时清掉标记，卡片回到可见状态
          leavingIds.value = leavingIds.value.filter(item => item !== id)
        }
      })()
    }, LEAVE_MS),
  )
}

async function handleClaim(id: number, action: 'approve' | 'reject') {
  if (await admin.handleClaim(id, action)) scheduleLeave(id, load)
}

function onTableChange(current: number) {
  admin.loadClaims(current, statusFilter.value || undefined)
}

onMounted(load)

onBeforeUnmount(() => {
  disposed = true
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
})
</script>

<template>
  <AdminPanel title="认领管理" desc="审核用户对内容的认领申请；拒绝时需要填写原因。">
    <template #toolbar>
      <AdminDataToolbar
        :summary="claimSummary"
        :loading="admin.claims.loading"
        refreshable
        @refresh="load"
      >
        <a-select
          v-model="statusFilter"
          size="small"
          class="claim-status-select"
          aria-label="按状态筛选认领"
          @change="load"
        >
          <a-option value="">全部状态</a-option>
          <a-option value="pending">待处理</a-option>
          <a-option value="approved">已通过</a-option>
          <a-option value="rejected">已拒绝</a-option>
        </a-select>
      </AdminDataToolbar>
    </template>

    <AdminSkeleton v-if="showSkeleton" variant="cards" :rows="6" label="正在加载认领申请" />

    <AdminEmpty
      v-else-if="admin.claims.list.length === 0"
      :title="statusFilter ? '没有符合条件的认领' : '暂无认领'"
      :desc="statusFilter ? '换个状态筛选看看。' : '用户认领内容后会出现在这里。'"
    />

    <template v-else>
      <div class="adm-card-grid claim-grid">
        <article
          v-for="record in admin.claims.list"
          :key="record.id"
          class="adm-card"
          :class="{ 'is-leaving': isLeaving(record.id) }"
        >
          <div class="adm-card-media" :class="{ 'is-placeholder': !claimThumb(record.content) }">
            <MediaImage
              v-if="claimThumb(record.content)"
              class="claim-media-img"
              :src="claimThumb(record.content)"
              :preview="false"
              alt=""
            />
            <div v-else class="claim-glyph is-quiet">
              <span class="claim-glyph-icon" aria-hidden="true"><IconLink /></span>
              <span class="claim-glyph-text">认领申请</span>
            </div>
            <span class="adm-card-badge">
              <AdminStatus :type="statusMap[record.status]?.type" :label="statusMap[record.status]?.label" />
            </span>
          </div>

          <div class="adm-card-body">
            <h3 class="adm-card-title" :title="record.content?.title || '未知内容'">
              {{ record.content?.title || '未知内容' }}
            </h3>
            <div class="adm-card-meta">
              <span>申请人 {{ record.user?.username || '未知' }}</span>
              <span class="claim-time">{{ formatTime(record.created_at) }}</span>
            </div>
            <p v-if="record.reason" class="claim-reason" :title="record.reason">{{ record.reason }}</p>
          </div>

          <div class="adm-card-actions claim-actions">
            <template v-if="record.status === 'pending'">
              <a-button
                type="primary"
                size="small"
                aria-label="通过认领"
                @click="handleClaim(record.id, 'approve')"
              >
                通过
              </a-button>
              <a-button
                status="danger"
                size="small"
                aria-label="拒绝认领"
                @click="handleClaim(record.id, 'reject')"
              >
                拒绝
              </a-button>
            </template>
            <span v-else class="claim-done">已处理</span>
          </div>
        </article>
      </div>

      <div v-if="admin.claims.totalPages > 1" class="claim-pagination">
        <a-pagination
          :current="admin.claims.page"
          :total="admin.claims.total"
          :page-size="admin.claims.pageSize"
          size="small"
          @change="onTableChange"
        />
      </div>
    </template>
  </AdminPanel>
</template>

<style scoped lang="scss">
// 工具条与表格页保持一致的零内边距
:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.claim-status-select {
  width: 120px;
}

// 卡片网格与骨架的 16px 内边距对齐；认领卡带封面后可收窄，宽屏排三列避免大片留白
.claim-grid {
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  padding: var(--adm-gap-lg, 16px);
}

// MediaImage 渲染成 a-image（外层 .arco-image + 内层 img），全局 .adm-card-media > img 命不中，
// 这里把封面铺满媒体区；尺寸与占位风格仍由全局样式决定。
.claim-media-img {
  display: block;
  width: 100%;
  height: 100%;

  :deep(.arco-image-img) {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.claim-glyph {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.claim-glyph-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--adm-accent, var(--admin-primary)) 12%, transparent);
  color: var(--adm-accent, var(--admin-primary));
  font-size: 24px;
}

.claim-glyph-text {
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: 0.08em;
}

.claim-time {
  white-space: nowrap;
}

// 申请理由：两行截断，完整文本放在 title 属性里
.claim-reason {
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

.claim-actions {
  flex-wrap: wrap;
}

.claim-done {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.claim-pagination {
  display: flex;
  justify-content: center;
  padding: 0 var(--adm-gap-lg, 16px) var(--adm-gap-lg, 16px);
}

@media (max-width: 768px) {
  .claim-grid {
    padding: var(--adm-gap-md, 12px);
  }

  .claim-glyph-icon {
    width: 44px;
    height: 44px;
    font-size: 20px;
  }
}
</style>

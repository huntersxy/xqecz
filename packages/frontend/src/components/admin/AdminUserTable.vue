<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import UserAvatar from '@/components/UserAvatar.vue'
import AdminPanel from './AdminPanel.vue'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminStatus from './AdminStatus.vue'
import { ACTION_COL, ROLE_COL, STATUS_COL, USERNAME_COL } from './adminColumns'
import { Tooltip, type TableColumnData, type TableData } from '@arco-design/web-vue'
import { useMediaQuery } from '@vueuse/core'
import { useConfirm } from '@/composables/useToast'
import { prefersReducedMotion } from '@/utils'
import { IconDelete, IconUserGroup, IconLock, IconUnlock } from '@arco-design/web-vue/es/icon'

const userStore = useUserStore()
const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')

// M8：删除成功后先让该行退场 280ms，再刷新列表；刷新失败则行恢复。
const LEAVE_MS = 280
const ENTER_MS = 900
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
const enterRows = ref(false)
const firstListLoaded = ref(false)
let disposed = false

const columns: TableColumnData[] = [
  { ...USERNAME_COL, minWidth: 140 },
  { ...ROLE_COL, width: 104 },
  { ...STATUS_COL, width: 96 },
  { ...ACTION_COL, width: 156 },
]

const summary = computed(() => `共 ${admin.users.total} 位用户`)
const showSkeleton = computed(() => admin.users.loading && admin.users.list.length === 0)

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

function rowClass(record: TableData) {
  return isLeaving(Number(record.id)) ? 'is-leaving' : ''
}

function trackTimer(timer: ReturnType<typeof setTimeout>) {
  timers.add(timer)
  return timer
}

function refresh() {
  void admin.loadUsers(admin.users.page)
}

function scheduleLeave(id: number) {
  if (disposed) return
  if (prefersReducedMotion()) {
    void admin.loadUsers(admin.users.page)
    return
  }
  if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
  trackTimer(setTimeout(() => {
    void (async () => {
      await admin.loadUsers(admin.users.page)
      leavingIds.value = leavingIds.value.filter(item => item !== id)
    })()
  }, LEAVE_MS))
}

function onTableChange(current: number) {
  void admin.loadUsers(current)
}

async function doRole(id: number, isAdmin: boolean) {
  const { confirm } = useConfirm()
  const ok = await confirm(isAdmin ? '确定取消该用户的管理员权限？' : '确定将该用户设为管理员？')
  if (!ok) return
  if (await admin.updateUserRole(id, !isAdmin)) refresh()
}

async function doBan(id: number, isBanned: boolean) {
  const { confirm } = useConfirm()
  const ok = await confirm(isBanned ? '确定解封该用户？' : '确定封禁该用户？')
  if (!ok) return
  if (await admin.updateUserBan(id, !isBanned)) refresh()
}

async function doDelete(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该用户？此操作不可撤销。')
  if (!ok) return
  if (await admin.deleteUser(id)) scheduleLeave(id)
}

// 首次数据到达时前 12 行错峰淡入，后续刷新不重播。
watch(() => admin.users.list.length, (len) => {
  if (firstListLoaded.value || len === 0) return
  firstListLoaded.value = true
  if (prefersReducedMotion()) return
  enterRows.value = true
  trackTimer(setTimeout(() => { enterRows.value = false }, ENTER_MS))
})

function clearTimers() {
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
}

onMounted(() => { void admin.loadUsers() })
onBeforeUnmount(() => {
  disposed = true
  clearTimers()
})
</script>

<template>
  <AdminPanel title="用户管理" desc="管理账号角色、封禁状态与账号删除">
    <template #toolbar>
      <AdminDataToolbar
        :summary="summary"
        :loading="admin.users.loading"
        refreshable
        @refresh="refresh"
      />
    </template>

    <AdminSkeleton
      v-if="showSkeleton"
      :variant="isMobile ? 'cards' : 'rows'"
      :rows="isMobile ? 3 : 6"
      :columns="4"
      label="正在加载用户"
    />

    <AdminEmpty
      v-else-if="!admin.users.list.length"
      title="暂无用户"
      desc="新注册用户会出现在这里"
      :compact="isMobile"
    />

    <div v-else-if="!isMobile" class="user-table" :class="{ 'is-entering': enterRows }">
      <a-spin :loading="admin.users.loading">
        <a-table
          :columns="columns"
          :data="admin.users.list"
          :row-class="rowClass"
          :pagination="{ current: admin.users.page, pageSize: admin.users.pageSize, total: admin.users.total }"
          row-key="id"
          @page-change="onTableChange"
        >
          <template #username="{ record }">
            <div class="user-cell">
              <UserAvatar :size="28" class="user-avatar" :email="record.email" :username="record.username" />
              <span class="user-name admin-cell-title">{{ record.username }}</span>
            </div>
          </template>

          <template #role="{ record }">
            <AdminStatus
              :type="record.is_admin ? 'info' : 'neutral'"
              :label="record.is_admin ? '管理员' : '普通用户'"
            />
          </template>

          <template #status="{ record }">
            <AdminStatus
              :type="record.is_banned ? 'danger' : 'success'"
              :label="record.is_banned ? '已封禁' : '正常'"
            />
          </template>

          <template #actions="{ record }">
            <div v-if="record.id !== userStore.user?.id" class="admin-action-group">
              <Tooltip :title="record.is_admin ? '取消管理员' : '设为管理员'">
                <a-button
                  class="admin-icon-btn"
                  type="text"
                  size="small"
                  :aria-label="record.is_admin ? '取消管理员' : '设为管理员'"
                  @click="doRole(record.id, !!record.is_admin)"
                >
                  <IconUserGroup />
                </a-button>
              </Tooltip>
              <template v-if="!record.is_admin">
                <Tooltip :title="record.is_banned ? '解封' : '封禁'">
                  <a-button
                    class="admin-icon-btn is-danger"
                    type="text"
                    size="small"
                    :aria-label="record.is_banned ? '解封' : '封禁'"
                    @click="doBan(record.id, !!record.is_banned)"
                  >
                    <IconLock v-if="!record.is_banned" /><IconUnlock v-else />
                  </a-button>
                </Tooltip>
                <Tooltip title="删除">
                  <a-button
                    class="admin-icon-btn is-danger"
                    type="text"
                    size="small"
                    aria-label="删除"
                    @click="doDelete(record.id)"
                  >
                    <IconDelete />
                  </a-button>
                </Tooltip>
              </template>
            </div>
            <span v-else class="admin-cell-3">当前用户</span>
          </template>
        </a-table>
      </a-spin>
    </div>

    <template v-else>
      <div class="admin-mobile-list" :class="{ 'is-entering': enterRows }">
        <div
          v-for="record in admin.users.list"
          :key="record.id"
          class="admin-mobile-card"
          :class="{ 'is-leaving': isLeaving(record.id) }"
        >
          <div class="mobile-user-head">
            <UserAvatar :size="28" class="user-avatar" :email="record.email" :username="record.username" />
            <div class="mobile-user-main">
              <div class="mobile-user-name admin-cell-title">{{ record.username }}</div>
              <div class="mobile-user-meta">
                <AdminStatus
                  :type="record.is_admin ? 'info' : 'neutral'"
                  :label="record.is_admin ? '管理员' : '普通用户'"
                />
                <AdminStatus
                  :type="record.is_banned ? 'danger' : 'success'"
                  :label="record.is_banned ? '已封禁' : '正常'"
                />
              </div>
            </div>
          </div>
          <div v-if="record.id !== userStore.user?.id" class="admin-mobile-actions">
            <Tooltip :title="record.is_admin ? '取消管理员' : '设为管理员'">
              <a-button
                class="admin-icon-btn"
                type="text"
                size="small"
                :aria-label="record.is_admin ? '取消管理员' : '设为管理员'"
                @click="doRole(record.id, !!record.is_admin)"
              >
                <IconUserGroup />
              </a-button>
            </Tooltip>
            <template v-if="!record.is_admin">
              <Tooltip :title="record.is_banned ? '解封' : '封禁'">
                <a-button
                  class="admin-icon-btn is-danger"
                  type="text"
                  size="small"
                  :aria-label="record.is_banned ? '解封' : '封禁'"
                  @click="doBan(record.id, !!record.is_banned)"
                >
                  <IconLock v-if="!record.is_banned" /><IconUnlock v-else />
                </a-button>
              </Tooltip>
              <Tooltip title="删除">
                <a-button
                  class="admin-icon-btn is-danger"
                  type="text"
                  size="small"
                  aria-label="删除"
                  @click="doDelete(record.id)"
                >
                  <IconDelete />
                </a-button>
              </Tooltip>
            </template>
          </div>
          <span v-else class="admin-cell-3 mobile-self">当前用户</span>
        </div>
      </div>
      <div v-if="admin.users.totalPages > 1" class="admin-mobile-pagination">
        <a-pagination
          :current="admin.users.page"
          :total="admin.users.total"
          :page-size="admin.users.pageSize"
          size="small"
          @change="(current: number) => admin.loadUsers(current)"
        />
      </div>
    </template>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 桌面图标按钮按 §6.2 放大到 36px；宽度已按 3 个按钮预留（操作列 156px）。
.user-table {
  :deep(.arco-btn.admin-icon-btn) {
    width: var(--adm-touch, 36px);
    height: var(--adm-touch, 36px);
    padding: 0;
  }

  :deep(.arco-table-td) {
    vertical-align: middle;
  }
}

.user-avatar {
  background: $admin-primary-soft;
  color: $admin-primary;
  flex-shrink: 0;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.user-name {
  @include ellipsis;
  display: block;
}

.mobile-user-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.mobile-user-main {
  flex: 1;
  min-width: 0;
}

.mobile-user-name {
  @include ellipsis;
  margin-bottom: 4px;
}

.mobile-user-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mobile-self {
  display: block;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid $admin-border-soft;
}

// M8 进场：首次数据到达后前 12 行错峰淡入，后续刷新不重播。
.user-table.is-entering :deep(.arco-table-body .arco-table-tr:nth-child(-n + 12)),
.admin-mobile-list.is-entering :deep(.admin-mobile-card:nth-child(-n + 12)) {
  animation: adm-rise var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1)) backwards;
}

@for $i from 1 through 12 {
  .user-table.is-entering :deep(.arco-table-body .arco-table-tr:nth-child(#{$i})),
  .admin-mobile-list.is-entering :deep(.admin-mobile-card:nth-child(#{$i})) {
    animation-delay: calc(var(--adm-stagger, 40ms) * #{$i - 1});
  }
}

@media (prefers-reduced-motion: reduce) {
  .user-table.is-entering :deep(.arco-table-body .arco-table-tr),
  .admin-mobile-list.is-entering :deep(.admin-mobile-card) {
    animation: none;
  }
}
</style>

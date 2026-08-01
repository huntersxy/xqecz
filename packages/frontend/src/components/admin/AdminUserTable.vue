<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import { ACTION_COL } from './adminColumns'
import { getAvatarUrl } from '@/utils'
import { IconUser, IconDelete, IconUserGroup, IconLock, IconUnlock, IconRefresh } from '@arco-design/web-vue/es/icon'
import { Tag, Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { useMediaQuery } from '@vueuse/core'
import { useConfirm } from '@/composables/useToast'

const userStore = useUserStore()
const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')

const columns: TableColumnData[] = [
  { title: '用户名', slotName: 'username', minWidth: 160 },
  { title: '角色', slotName: 'role', width: 110, align: 'center' },
  { title: '状态', slotName: 'status', width: 110, align: 'center' },
  { ...ACTION_COL },
]

function onTableChange(current: number) {
  admin.loadUsers(current)
}

async function doRole(id: number, isAdmin: boolean) {
  const { confirm } = useConfirm()
  const ok = await confirm(isAdmin ? '确定取消该用户的管理员权限？' : '确定将该用户设为管理员？')
  if (!ok) return
  if (await admin.updateUserRole(id, !isAdmin)) admin.loadUsers(admin.users.page)
}

async function doBan(id: number, isBanned: boolean) {
  const { confirm } = useConfirm()
  const ok = await confirm(isBanned ? '确定解封该用户？' : '确定封禁该用户？')
  if (!ok) return
  if (await admin.updateUserBan(id, !isBanned)) admin.loadUsers(admin.users.page)
}

async function doDelete(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该用户？此操作不可撤销。')
  if (!ok) return
  if (await admin.deleteUser(id)) admin.loadUsers(admin.users.page)
}

onMounted(() => admin.loadUsers())
</script>

<template>
  <AdminPanel title="用户管理" :desc="`共 ${admin.users.total} 位用户`">
    <template #actions>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="admin.users.loading" @click="admin.loadUsers(admin.users.page)">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="admin.users.loading">
      <template v-if="!isMobile">
        <a-table :columns="columns" :data="admin.users.list" :pagination="{ current: admin.users.page, pageSize: admin.users.pageSize, total: admin.users.total }" row-key="id" @page-change="onTableChange">
          <template #username="{ record }">
            <div class="flex items-center gap-2.5">
              <a-avatar :size="28" class="user-avatar" :image-url="record.email ? getAvatarUrl(record.email) : ''">
                <IconUser v-if="!record.email" />
                <template #error><IconUser /></template>
              </a-avatar>
              <span class="admin-cell-title">{{ record.username }}</span>
            </div>
          </template>
          <template #role="{ record }">
            <Tag :color="record.is_admin ? 'arcoblue' : 'gray'" :bordered="false" class="admin-tag-inline">{{ record.is_admin ? '管理员' : '普通用户' }}</Tag>
          </template>
          <template #status="{ record }">
            <AdminStatus :type="record.is_banned ? 'danger' : 'success'" :label="record.is_banned ? '已封禁' : '正常'" />
          </template>
          <template #actions="{ record }">
            <div v-if="record.id !== userStore.user?.id" class="admin-action-group">
              <Tooltip :title="record.is_admin ? '取消管理员' : '设为管理员'">
                <a-button class="admin-icon-btn" type="text" size="small" @click="doRole(record.id, record.is_admin)"><IconUserGroup /></a-button>
              </Tooltip>
              <template v-if="!record.is_admin">
                <Tooltip :title="record.is_banned ? '解封' : '封禁'">
                  <a-button class="admin-icon-btn" type="text" size="small" @click="doBan(record.id, record.is_banned)">
                    <IconLock v-if="!record.is_banned" /><IconUnlock v-else />
                  </a-button>
                </Tooltip>
                <Tooltip title="删除">
                  <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="doDelete(record.id)"><IconDelete /></a-button>
                </Tooltip>
              </template>
            </div>
            <span v-else class="admin-cell-3">当前用户</span>
          </template>
        </a-table>
      </template>

      <template v-else>
        <div class="admin-mobile-list">
          <div v-for="record in admin.users.list" :key="record.id" class="admin-mobile-card">
            <div class="flex items-center gap-2 flex-wrap">
              <a-avatar :size="24" class="user-avatar" :image-url="record.email ? getAvatarUrl(record.email) : ''">
                <IconUser v-if="!record.email" />
                <template #error><IconUser /></template>
              </a-avatar>
              <span class="admin-cell-title">{{ record.username }}</span>
              <Tag :color="record.is_admin ? 'arcoblue' : 'gray'" size="small" :bordered="false" class="admin-tag-inline">{{ record.is_admin ? '管理员' : '普通用户' }}</Tag>
              <AdminStatus v-if="record.is_banned" type="danger" label="已封禁" />
            </div>
            <div v-if="record.id !== userStore.user?.id" class="admin-mobile-actions">
              <Tooltip :title="record.is_admin ? '取消管理员' : '设为管理员'">
                <a-button class="admin-icon-btn" type="text" size="small" @click="doRole(record.id, !!record.is_admin)"><IconUserGroup /></a-button>
              </Tooltip>
              <template v-if="!record.is_admin">
                <Tooltip :title="record.is_banned ? '解封' : '封禁'">
                  <a-button class="admin-icon-btn" type="text" size="small" @click="doBan(record.id, !!record.is_banned)">
                    <IconLock v-if="!record.is_banned" /><IconUnlock v-else />
                  </a-button>
                </Tooltip>
                <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="doDelete(record.id)"><IconDelete /></a-button>
              </template>
            </div>
            <span v-else class="admin-cell-3 mt-2 block">当前用户</span>
          </div>
          <a-empty v-if="admin.users.list.length === 0" description="暂无用户" />
        </div>
        <div v-if="admin.users.totalPages > 1" class="admin-mobile-pagination">
          <a-pagination :current="admin.users.page" :total="admin.users.total" :page-size="admin.users.pageSize" size="small" @change="(current: number) => admin.loadUsers(current)" />
        </div>
      </template>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.user-avatar {
  background: $admin-primary-soft;
  color: $admin-primary;
  flex-shrink: 0;
}
</style>

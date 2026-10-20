<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import UserAvatar from '@/components/UserAvatar.vue'
import AdminPanel from './AdminPanel.vue'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminEmpty from './AdminEmpty.vue'
import { useConfirm } from '@/composables/useToast'
import { formatTime, prefersReducedMotion } from '@/utils'
import { IconDelete, IconLock, IconUnlock, IconUserGroup } from '@arco-design/web-vue/es/icon'
import type { User } from '@/types'

const userStore = useUserStore()
const admin = useAdminStore()

const summary = computed(() => `共 ${admin.users.total} 位用户`)
const showSkeleton = computed(() => admin.users.loading && admin.users.list.length === 0)

const pageAdmins = computed(() => admin.users.list.filter(u => u.is_admin).length)
const pageBanned = computed(() => admin.users.list.filter(u => u.is_banned).length)

type UserFilter = 'all' | 'admin' | 'banned'
const filter = ref<UserFilter>('all')
const filterOptions: { value: UserFilter; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'admin', label: '管理员' },
  { value: 'banned', label: '已封禁' },
]

const visibleUsers = computed(() => {
  if (filter.value === 'admin') return admin.users.list.filter(u => u.is_admin)
  if (filter.value === 'banned') return admin.users.list.filter(u => u.is_banned)
  return admin.users.list
})
const filterEmpty = computed(() => admin.users.list.length > 0 && visibleUsers.value.length === 0)

function isSelf(user: User) {
  return user.id === userStore.user?.id
}

function refresh() {
  void admin.loadUsers(admin.users.page)
}

function onPageChange(current: number) {
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

const LEAVE_MS = 280
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
let disposed = false

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

function clearLeaving(id: number) {
  leavingIds.value = leavingIds.value.filter(item => item !== id)
}

async function commitDelete(id: number) {
  const ok = await admin.deleteUser(id)
  if (!ok) {
    clearLeaving(id)
    return
  }
  await admin.loadUsers(admin.users.page)
  if (admin.users.list.length === 0 && admin.users.page > 1) {
    await admin.loadUsers(admin.users.page - 1)
  }
  clearLeaving(id)
}

async function doDelete(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该用户？此操作不可撤销。')
  if (!ok) return
  if (prefersReducedMotion()) {
    await commitDelete(id)
    return
  }
  if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
  timers.add(setTimeout(() => {
    if (disposed) return
    void commitDelete(id)
  }, LEAVE_MS))
}

onMounted(() => { void admin.loadUsers() })

onBeforeUnmount(() => {
  disposed = true
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
})
</script>

<template>
  <AdminPanel title="用户管理" desc="管理账号角色、封禁状态与账号删除" flow>
    <template #toolbar>
      <AdminDataToolbar
        :summary="summary"
        :loading="admin.users.loading"
        refreshable
        @refresh="refresh"
      >
        <div class="user-filter" role="group" aria-label="筛选当前页用户">
          <span class="user-filter-label">当前页</span>
          <button
            v-for="opt in filterOptions"
            :key="opt.value"
            type="button"
            class="user-filter-btn"
            :class="{ 'is-active': filter === opt.value }"
            :aria-pressed="filter === opt.value"
            @click="filter = opt.value"
          >{{ opt.label }}</button>
        </div>
      </AdminDataToolbar>
    </template>

    <AdminSkeleton v-if="showSkeleton" variant="cards" :rows="6" label="正在加载用户" />

    <AdminEmpty
      v-else-if="!admin.users.list.length"
      title="暂无用户"
      desc="新注册用户会出现在这里"
    />

    <div v-else class="adm-board user-board">
      <div class="adm-board-main">
        <div class="adm-stat-grid user-stats">
          <div class="adm-stat">
            <span class="adm-stat-label">总用户数</span>
            <span class="adm-stat-value">{{ admin.users.total }}</span>
            <span class="adm-stat-hint">全站注册账号</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">管理员</span>
            <span class="adm-stat-value">{{ pageAdmins }}</span>
            <span class="adm-stat-hint">当前页 · 共 {{ admin.users.list.length }} 位用户</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">已封禁</span>
            <span class="adm-stat-value">{{ pageBanned }}</span>
            <span class="adm-stat-hint">当前页 · 已封禁账号</span>
          </div>
        </div>

        <AdminEmpty
          v-if="filterEmpty"
          compact
          title="当前页没有符合条件的用户"
          desc="换个筛选条件，或翻到其它页继续查看。"
        >
          <template #actions>
            <a-button size="small" @click="filter = 'all'">查看全部</a-button>
          </template>
        </AdminEmpty>

        <div v-else class="adm-card-grid is-dense user-grid">
          <article
            v-for="u in visibleUsers"
            :key="u.id"
            class="adm-card user-card"
            :class="{ 'is-leaving': isLeaving(u.id) }"
          >
            <div class="adm-card-content">
              <header class="user-head">
                <UserAvatar :size="40" class="user-avatar" :email="u.email" :username="u.username" />
                <div class="user-identity">
                  <h3 class="adm-card-title user-name" :title="u.username">{{ u.username }}</h3>
                  <span class="user-uid">#{{ u.id }}</span>
                </div>
              </header>

              <div class="user-badges">
                <span class="adm-card-badge" :class="u.is_admin ? 'is-role-admin' : 'is-role-plain'">
                  {{ u.is_admin ? '管理员' : '普通用户' }}
                </span>
                <span v-if="u.is_banned" class="adm-card-badge is-banned">已封禁</span>
                <span v-if="isSelf(u)" class="adm-card-badge is-self">自己</span>
              </div>

              <dl class="adm-kv">
                <div class="adm-kv-row">
                  <dt>邮箱</dt>
                  <dd :title="u.email || ''">{{ u.email || '未填写' }}</dd>
                </div>
                <div class="adm-kv-row">
                  <dt>注册</dt>
                  <dd>{{ u.created_at ? formatTime(u.created_at) : '未知' }}</dd>
                </div>
              </dl>
            </div>

            <footer class="adm-card-foot">
              <template v-if="!isSelf(u)">
                <a-tooltip :content="u.is_admin ? '取消管理员' : '设为管理员'">
                  <a-button
                    class="admin-icon-btn user-action"
                    type="text"
                    size="small"
                    :aria-label="u.is_admin ? '取消管理员' : '设为管理员'"
                    @click="doRole(u.id, !!u.is_admin)"
                  >
                    <IconUserGroup />
                  </a-button>
                </a-tooltip>
                <template v-if="!u.is_admin">
                  <a-tooltip :content="u.is_banned ? '解封' : '封禁'">
                    <a-button
                      class="admin-icon-btn user-action is-danger"
                      type="text"
                      size="small"
                      :aria-label="u.is_banned ? '解封' : '封禁'"
                      @click="doBan(u.id, !!u.is_banned)"
                    >
                      <IconLock v-if="!u.is_banned" />
                      <IconUnlock v-else />
                    </a-button>
                  </a-tooltip>
                  <a-tooltip content="删除">
                    <a-button
                      class="admin-icon-btn user-action is-danger"
                      type="text"
                      size="small"
                      aria-label="删除"
                      @click="doDelete(u.id)"
                    >
                      <IconDelete />
                    </a-button>
                  </a-tooltip>
                </template>
              </template>
              <span v-else class="user-self-hint">当前用户，不可操作</span>
            </footer>
          </article>
        </div>

        <div v-if="admin.users.totalPages > 1" class="user-pagination">
          <a-pagination
            :current="admin.users.page"
            :total="admin.users.total"
            :page-size="admin.users.pageSize"
            size="small"
            @change="onPageChange"
          />
        </div>
      </div>

      <aside class="adm-board-aside" aria-label="用户概览">
        <section class="adm-card user-side">
          <div class="adm-card-content">
            <h3 class="user-side-title">分页与操作</h3>
            <dl class="adm-kv">
              <div class="adm-kv-row">
                <dt>当前页</dt>
                <dd>{{ admin.users.page }} / {{ Math.max(1, admin.users.totalPages) }}</dd>
              </div>
              <div class="adm-kv-row">
                <dt>每页</dt>
                <dd>{{ admin.users.pageSize }} 位</dd>
              </div>
              <div class="adm-kv-row">
                <dt>本页管理员</dt>
                <dd>{{ pageAdmins }}</dd>
              </div>
            </dl>
            <ul class="user-rules">
              <li>自己的账号不显示操作按钮。</li>
              <li>管理员不可封禁或删除，需先取消管理员。</li>
              <li>删除后不可恢复。</li>
            </ul>
          </div>
        </section>
      </aside>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.user-filter {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.user-filter-label {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: .04em;
}

.user-filter-btn {
  min-height: 28px;
  padding: 0 10px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: transparent;
  color: var(--adm-muted, var(--admin-text-2));
  font-family: inherit;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  cursor: pointer;
  transition:
    background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1));
  @include focus-ring(2px);

  &:hover {
    border-color: var(--adm-accent, var(--admin-primary));
    color: var(--adm-ink, var(--admin-text));
  }

  &.is-active {
    border-color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    color: var(--adm-accent, var(--admin-primary));
    font-weight: 600;
  }
}

.user-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.user-avatar {
  flex-shrink: 0;
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  color: var(--adm-accent, var(--admin-primary));
}

.user-identity {
  flex: 1;
  min-width: 0;
}

.user-card .user-name {
  display: block;
  overflow: hidden;
  color: var(--adm-ink, var(--admin-text));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-uid {
  display: block;
  margin-top: 2px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.user-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.user-card .adm-card-badge {
  position: static;
  padding: 2px 8px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  white-space: nowrap;

  &.is-role-admin {
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    color: var(--adm-accent, var(--admin-primary));
  }

  &.is-banned {
    background: color-mix(in srgb, var(--adm-danger, var(--admin-danger)) 14%, transparent);
    color: var(--adm-danger, var(--admin-danger));
  }

  &.is-self {
    border: 1px solid var(--adm-line, var(--admin-border-soft));
    background: transparent;
    color: var(--adm-muted, var(--admin-text-3));
  }
}

.user-card .adm-card-foot {
  gap: 6px;
}

.user-card .arco-btn.user-action {
  width: var(--adm-touch, 36px);
  height: var(--adm-touch, 36px);
  padding: 0;
}

.user-self-hint {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.user-pagination {
  display: flex;
  justify-content: center;
  padding: var(--adm-gap-md, 12px) 0 var(--adm-gap-xs, 4px);
}

.user-side-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.user-rules {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 10px 0 0;
  border-top: 1px solid var(--admin-border-soft);
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  list-style: none;

  li {
    position: relative;
    padding-left: 12px;
  }

  li::before {
    content: '';
    position: absolute;
    top: .62em;
    left: 0;
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: var(--adm-line, var(--admin-border));
  }
}

@media (max-width: 768px) {
  .user-filter-btn {
    min-height: var(--adm-touch, 44px);
    padding: 0 14px;
  }

  .user-head {
    gap: 8px;
  }

  .adm-card-grid.user-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

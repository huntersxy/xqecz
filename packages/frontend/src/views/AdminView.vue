<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import type { AdminNavGroup } from '@/components/admin/AdminNav.vue'
import {
  IconDashboard, IconFile, IconEye, IconHome, IconUserGroup, IconBarChart,
  IconLink, IconExclamationCircle, IconUpload, IconLock, IconMenu,
} from '@arco-design/web-vue/es/icon'
import AdminDashboard from '@/components/admin/AdminDashboard.vue'

const userStore = useUserStore()
const admin = useAdminStore()
const mobileMenuOpen = ref(false)

const navGroups = computed<AdminNavGroup[]>(() => {
  const groups: AdminNavGroup[] = []
  if (userStore.user?.is_admin) {
    groups.push({
      label: '总览',
      items: [
        { key: 'dashboard', title: '仪表盘', icon: IconDashboard },
      ],
    })
  }
  groups.push({
      label: '内容',
      items: [
        { key: 'my', title: '我的内容', icon: IconFile },
        { key: 'upload', title: '上传内容', icon: IconUpload },
      ],
    })
  if (userStore.user?.is_admin) {
    groups.push({
      label: '管理',
      items: [
        { key: 'pending', title: '审核内容', icon: IconEye, badge: admin.pendingCounts.content || undefined },
        { key: 'all', title: '所有内容', icon: IconHome },
        { key: 'users', title: '用户管理', icon: IconUserGroup },
        { key: 'polls', title: '投票管理', icon: IconBarChart },
        { key: 'claims', title: '认领管理', icon: IconLink, badge: admin.pendingCounts.claims || undefined },
        { key: 'reports', title: '举报管理', icon: IconExclamationCircle, badge: admin.pendingCounts.reports || undefined },
      ],
    })
  }
  groups.push({
    label: '系统',
    items: [{ key: 'api-keys', title: 'API 密钥', icon: IconLock }],
  })
  return groups
})

const sectionTitle = computed(() => {
  for (const g of navGroups.value) {
    const hit = g.items.find((i) => i.key === admin.activeTab)
    if (hit) return hit.title
  }
  return ''
})

function onMenuSelect(key: string) {
  admin.activeTab = key
  mobileMenuOpen.value = false
}

// 切换页签后刷新角标（审批/认领/举报处理完后数字自动校正）
watch(() => admin.activeTab, () => {
  if (userStore.user?.is_admin) admin.loadPendingCounts()
})

onMounted(() => {
  admin.loadTags()
  if (userStore.user?.is_admin) {
    admin.activeTab = 'dashboard'
    admin.loadPendingCounts()
  }
})
</script>

<template>
  <div class="admin-shell admin-theme">
    <aside class="admin-side">
      <AdminNav
        :groups="navGroups"
        :active="admin.activeTab"
        :username="userStore.user?.username"
        :email="userStore.user?.email"
        :is-admin="!!userStore.user?.is_admin"
        @select="onMenuSelect"
      />
    </aside>

    <div class="admin-main">
      <header class="admin-topbar">
        <button type="button" class="admin-topbar-menu-btn" aria-label="打开导航" @click="mobileMenuOpen = true">
          <IconMenu />
        </button>
        <nav class="admin-crumb" aria-label="breadcrumb">
          <span class="admin-crumb-root">管理控制台</span>
          <span class="admin-crumb-sep">/</span>
          <span class="admin-crumb-current">{{ sectionTitle }}</span>
        </nav>
      </header>

      <main class="admin-content">
        <transition name="admin-page" mode="out-in">
          <div :key="admin.activeTab" class="admin-page">
            <AdminDashboard v-if="admin.activeTab === 'dashboard'" @select="onMenuSelect" />
            <AdminContentTable v-else-if="admin.activeTab === 'my'" mode="my" />
            <AdminUploadPanel v-else-if="admin.activeTab === 'upload'" />
            <AdminContentTable v-else-if="admin.activeTab === 'pending' && userStore.user?.is_admin" mode="pending" />
            <AdminContentTable v-else-if="admin.activeTab === 'all' && userStore.user?.is_admin" mode="all" />
            <AdminUserTable v-else-if="admin.activeTab === 'users' && userStore.user?.is_admin" />
            <AdminPollPanel v-else-if="admin.activeTab === 'polls' && userStore.user?.is_admin" />
            <AdminClaimTable v-else-if="admin.activeTab === 'claims' && userStore.user?.is_admin" />
            <AdminReportTable v-else-if="admin.activeTab === 'reports' && userStore.user?.is_admin" />
            <AdminApiKeys v-else-if="admin.activeTab === 'api-keys'" />
            <a-result
              v-else
              status="403"
              title="该页面仅管理员可见"
              subtitle="你当前账号没有访问此管理功能的权限。"
            />
          </div>
        </transition>
      </main>
    </div>

    <a-drawer
      v-model:visible="mobileMenuOpen"
      placement="left"
      :closable="false"
      :width="264"
      class="admin-theme admin-mobile-drawer"
    >
      <AdminNav
        :groups="navGroups"
        :active="admin.activeTab"
        :username="userStore.user?.username"
        :email="userStore.user?.email"
        :is-admin="!!userStore.user?.is_admin"
        @select="onMenuSelect"
      />
    </a-drawer>

    <AdminContentDrawer />
  </div>
</template>

<!-- admin 设计令牌：非 scoped，配合 .admin-theme 类作用于外壳与 teleport 出去的抽屉/弹层；
     类名 admin-* 仅后台使用，不影响其他页面 -->
<style lang="scss">
.admin-theme {
  --admin-text: var(--color-text-1);
  --admin-text-2: var(--color-text-2);
  --admin-text-3: var(--color-text-3);
  --admin-border: var(--color-border-2);
  --admin-border-soft: var(--color-border-1);
  --admin-fill: var(--color-fill-2);
  --admin-surface: var(--color-bg-2);
  --admin-primary: rgb(var(--primary-6));
  --admin-primary-soft: color-mix(in srgb, rgb(var(--primary-6)) 10%, transparent);
  --admin-danger: rgb(var(--danger-6));
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 10%, transparent);
  --admin-radius: 14px;
  --admin-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 24px -12px rgba(15, 23, 42, 0.08);
}

body[arco-theme='dark'] .admin-theme {
  --admin-primary-soft: color-mix(in srgb, rgb(var(--primary-6)) 22%, transparent);
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 20%, transparent);
  --admin-shadow: 0 1px 2px rgba(0, 0, 0, 0.24);
}

.admin-mobile-drawer .arco-drawer-body {
  padding: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>

<style lang="scss" scoped>
.admin-shell {
  display: flex;
  align-items: flex-start;
  min-height: calc(100vh - 56px);
  background: var(--color-bg-1);
  color: var(--admin-text);
}

// ── 侧边栏 ──
.admin-side {
  width: 240px;
  flex-shrink: 0;
  position: sticky;
  top: 56px;
  height: calc(100vh - 56px);
  background: var(--admin-surface);
  border-right: 1px solid var(--admin-border-soft);
}

// ── 主列 ──
.admin-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.admin-topbar {
  position: sticky;
  top: 56px;
  z-index: 40;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  background: color-mix(in srgb, var(--color-bg-1) 80%, transparent);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.admin-topbar-menu-btn {
  display: none;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  font-size: 18px;
  color: var(--admin-text-2);
  cursor: pointer;
  transition: background 0.15s ease;

  &:hover {
    background: var(--admin-fill);
  }
}

.admin-crumb {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  min-width: 0;
}

.admin-crumb-root {
  color: var(--admin-text-3);
}

.admin-crumb-sep {
  color: var(--admin-text-3);
  opacity: 0.6;
}

.admin-crumb-current {
  color: var(--admin-text);
  font-weight: 500;
}

.admin-content {
  flex: 1;
  padding: 24px;
}

.admin-page {
  max-width: 1120px;
  margin: 0 auto;
}

// ── 页签切换过渡 ──
.admin-page-enter-active {
  transition: opacity 0.22s ease, transform 0.22s ease;
}

.admin-page-leave-active {
  transition: opacity 0.12s ease;
}

.admin-page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.admin-page-leave-to {
  opacity: 0;
}

@media (max-width: 768px) {
  .admin-side {
    display: none;
  }

  .admin-topbar {
    padding: 0 16px;
  }

  .admin-topbar-menu-btn {
    display: inline-flex;
  }

  .admin-content {
    padding: 16px 12px;
  }
}
</style>

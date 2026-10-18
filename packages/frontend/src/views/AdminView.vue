<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import type { AdminNavGroup } from '@/components/admin/AdminNav.vue'
import {
  IconDashboard, IconFile, IconEye, IconHome, IconUserGroup, IconBarChart,
  IconLink, IconExclamationCircle, IconUpload, IconLock, IconMenu, IconBook,
  IconCode,
} from '@arco-design/web-vue/es/icon'
import AdminDashboard from '@/components/admin/AdminDashboard.vue'
import AdminApiDocs from '@/components/admin/AdminApiDocs.vue'
import AdminVisionSettings from '@/components/admin/AdminVisionSettings.vue'
import AdminAI from '@/components/admin/AdminAI.vue'
import AdminLogsTerminal from '@/components/admin/AdminLogsTerminal.vue'

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
        { key: 'vision', title: '视觉模型', icon: IconEye },
        { key: 'ai', title: 'AI 创作管理', icon: IconEye },
      ],
    })
  }
  groups.push({
    label: '系统',
    items: [
      { key: 'api-keys', title: 'API 密钥', icon: IconLock },
      { key: 'api-docs', title: 'API 文档', icon: IconBook },
      ...(userStore.user?.is_admin ? [{ key: 'logs', title: '运行日志', icon: IconCode }] : []),
    ],
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
  <div class="admin-shell admin-theme creative-theme">
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
          <span class="admin-crumb-root">{{ userStore.user?.is_admin ? '管理控制台' : '创作工作台' }}</span>
          <span class="admin-crumb-sep">/</span>
          <span class="admin-crumb-current">{{ sectionTitle }}</span>
        </nav>
      </header>

      <main class="admin-content">
        <div class="admin-intro">
          <div>
            <span class="admin-intro-label">YOUR CREATIVE SPACE <span aria-hidden="true">✦</span></span>
            <h1>{{ userStore.user?.is_admin ? '照料每一份灵感' : '让灵感慢慢生长' }}</h1>
            <p>{{ userStore.user?.is_admin ? '整理作品，回应创作者，让这个小宇宙有序生长。' : '收藏创作的足迹，在这里整理与分享你的作品。' }}</p>
          </div>
          <svg class="admin-intro-art" viewBox="0 0 160 100" fill="none" aria-hidden="true">
            <ellipse cx="80" cy="52" rx="60" ry="20" transform="rotate(-22 80 52)" />
            <circle cx="80" cy="52" r="29" />
            <path d="M78 15v10m-5-5h10M130 64v14m-7-7h14M31 27v8m-4-4h8" />
            <circle cx="37" cy="72" r="3" />
          </svg>
        </div>
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
            <AdminVisionSettings v-else-if="admin.activeTab === 'vision' && userStore.user?.is_admin" />
            <AdminAI v-else-if="admin.activeTab === 'ai' && userStore.user?.is_admin" />
            <AdminLogsTerminal v-else-if="admin.activeTab === 'logs' && userStore.user?.is_admin" />
            <AdminApiKeys v-else-if="admin.activeTab === 'api-keys'" />
            <AdminApiDocs v-else-if="admin.activeTab === 'api-docs'" />
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
      class="admin-theme creative-theme admin-mobile-drawer"
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
  --admin-text: var(--creative-ink);
  --admin-text-2: var(--creative-muted);
  --admin-text-3: var(--creative-muted);
  --admin-border: var(--creative-line);
  --admin-border-soft: color-mix(in srgb, var(--creative-line) 70%, var(--admin-surface));
  --admin-fill: var(--creative-soft);
  --admin-surface: color-mix(in srgb, var(--creative-paper) 35%, white);
  --admin-canvas: color-mix(in srgb, var(--creative-paper) 65%, var(--creative-soft));
  --admin-primary: var(--creative-accent);
  --admin-primary-soft: var(--creative-soft);
  --admin-danger: rgb(var(--danger-6));
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 10%, transparent);
  --admin-radius: 18px;
  --admin-shadow: var(--creative-shadow);
  // Arco 表单、表格与弹层使用同一语义色；状态色继续保留成功/警告/危险的含义。
  --color-bg-1: var(--admin-canvas);
  --color-bg-2: var(--admin-surface);
  --color-bg-3: var(--admin-surface);
  --color-fill-1: var(--admin-canvas);
  --color-fill-2: var(--creative-soft);
  --color-border-2: var(--creative-line);
  --color-border-1: var(--admin-border-soft);
  --color-text-1: var(--creative-ink);
  --color-text-2: var(--creative-muted);
  --color-text-3: var(--creative-muted);
}

body[arco-theme='dark'] .admin-theme {
  --admin-surface: var(--creative-paper);
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 20%, transparent);
  --admin-shadow: 0 1px 2px rgba(0, 0, 0, 0.24);
}

.admin-theme .arco-btn { border-radius: 999px; }
.admin-theme .arco-pagination-item { border-radius: 50%; }
.admin-theme .arco-input-wrapper,
.admin-theme .arco-textarea-wrapper,
.admin-theme .arco-select-view,
.admin-theme .arco-input-tag { border-radius: 10px; }
.admin-theme .arco-drawer,
.admin-theme .arco-modal { background: var(--admin-surface); }

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
  min-height: calc(100vh - 64px);
  background: var(--admin-canvas);
  color: var(--admin-text);
}

// ── 侧边栏 ──
.admin-side {
  width: 248px;
  flex-shrink: 0;
  position: sticky;
  top: 64px;
  height: calc(100vh - 64px);
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
  top: 64px;
  z-index: 40;
  height: 52px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 24px;
  background: color-mix(in srgb, var(--admin-canvas) 96%, transparent);
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
  padding: 28px 32px 48px;
}

.admin-page {
  max-width: 1120px;
  margin: 0 auto;
}

.admin-intro { max-width: 1120px; margin: 2px auto 30px; display: flex; align-items: center; justify-content: space-between; gap: 20px; }
.admin-intro-label { color: var(--admin-primary); font-size: 10px; letter-spacing: .18em; }
.admin-intro-label span { margin-left: 8px; }
.admin-intro h1 { margin: 10px 0 8px; font-family: var(--creative-title-font); font-size: clamp(24px, 2.2vw, 34px); font-weight: 600; letter-spacing: .03em; }
.admin-intro p { color: var(--admin-text-2); font-size: 13px; line-height: 1.8; }
.admin-intro-art { width: 150px; flex-shrink: 0; stroke: var(--admin-primary); stroke-width: 1.2; opacity: .55; pointer-events: none; }

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
    top: 60px;
    padding: 0 16px;
  }

  .admin-topbar-menu-btn {
    display: inline-flex;
  }

  .admin-content {
    padding: 16px 12px;
  }
  .admin-intro { margin: 8px 4px 22px; }
  .admin-intro-art { width: 72px; }
  .admin-intro-label { font-size: 9px; letter-spacing: .1em; }
  .admin-shell { min-height: calc(100vh - 60px); }
}
</style>

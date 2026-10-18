<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import type { AdminNavGroup } from '@/components/admin/AdminNav.vue'
import {
  IconDashboard, IconFile, IconEye, IconHome, IconUserGroup, IconBarChart,
  IconLink, IconExclamationCircle, IconUpload, IconLock, IconBook,
  IconCode,
} from '@arco-design/web-vue/es/icon'
import AdminDashboard from '@/components/admin/AdminDashboard.vue'
import AdminApiDocs from '@/components/admin/AdminApiDocs.vue'
import AdminVisionSettings from '@/components/admin/AdminVisionSettings.vue'
import AdminAI from '@/components/admin/AdminAI.vue'
import AdminLogsTerminal from '@/components/admin/AdminLogsTerminal.vue'

const userStore = useUserStore()
const admin = useAdminStore()

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

function onMenuSelect(key: string) {
  admin.activeTab = key
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
  <div class="admin-theme creative-theme admin-workspace">
    <AdminNav :groups="navGroups" :active="admin.activeTab" :is-admin="!!userStore.user?.is_admin" @select="onMenuSelect" />
    <section class="admin-workspace-content" aria-label="管理工作区">
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
    </section>
    <AdminContentDrawer />
  </div>
</template>

<!-- Admin 令牌在主题根与 Teleport 弹层上共享首页的纸感色板；业务状态色仍由 Arco 语义色提供。 -->
<style lang="scss">
.admin-theme.creative-theme {
  --creative-paper: var(--site-paper);
  --creative-canvas: var(--site-canvas);
  --creative-soft: var(--site-soft);
  --creative-ink: var(--site-ink);
  --creative-muted: var(--site-muted);
  --creative-line: var(--site-line);
  --creative-accent: var(--site-accent);
  --creative-accent-hover: var(--site-accent-hover);
  --creative-title-font: var(--site-title-font);
  --creative-shadow: 0 12px 32px rgba(32, 27, 28, .1);
  --primary-1: 244, 240, 239;
  --primary-2: 233, 227, 227;
  --primary-3: 209, 200, 201;
  --primary-4: 165, 151, 153;
  --primary-5: 85, 72, 74;
  --primary-6: var(--site-action-rgb);
  --primary-7: 26, 22, 23;
  --primary-8: 20, 17, 18;
  --primary-9: 14, 12, 12;
  --primary-10: 7, 6, 6;
  --color-primary: var(--site-action);
  --color-primary-hover: var(--site-action-hover);
  --color-primary-light-1: var(--site-accent-soft);
  --color-primary-light-2: color-mix(in srgb, var(--site-accent) 16%, var(--site-paper));
  --color-primary-light-3: color-mix(in srgb, var(--site-accent) 24%, var(--site-paper));
  --color-primary-light-4: color-mix(in srgb, var(--site-accent) 32%, var(--site-paper));
  --color-on-primary: var(--site-on-action);
  --color-bg: var(--site-canvas);
  --color-surface: var(--site-paper);
  --color-card: var(--site-paper);
  --color-card-hover: var(--site-paper);
  --color-header-bg: var(--site-paper);
  --color-border: var(--site-line);
  --color-hover: var(--site-soft);
  --color-placeholder: var(--site-soft);
  --color-text: var(--site-ink);
  --color-text-secondary: var(--site-muted);
  --color-text-tertiary: var(--site-muted);
  --admin-text: var(--site-ink);
  --admin-text-2: var(--site-muted);
  --admin-text-3: var(--site-muted);
  --admin-border: var(--site-line);
  --admin-border-soft: color-mix(in srgb, var(--site-line) 72%, var(--site-paper));
  --admin-fill: var(--site-soft);
  --admin-surface: var(--site-paper);
  --admin-canvas: var(--site-canvas);
  --admin-primary: var(--site-accent);
  --admin-primary-soft: var(--site-accent-soft);
  --admin-danger: rgb(var(--danger-6));
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 10%, transparent);
  --admin-radius: 16px;
  --admin-shadow: 0 12px 32px rgba(32, 27, 28, .1);
  // 覆盖 Arco 的蓝色页面默认值；危险、警告、成功仍分别使用其语义状态色。
  --color-bg-1: var(--site-canvas);
  --color-bg-2: var(--site-paper);
  --color-bg-3: var(--admin-surface);
  --color-fill-1: var(--site-canvas);
  --color-fill-2: var(--site-soft);
  --color-border-2: var(--site-line);
  --color-border-1: var(--admin-border-soft);
  --color-text-1: var(--site-ink);
  --color-text-2: var(--site-muted);
  --color-text-3: var(--site-muted);
}

body[arco-theme='dark'] .admin-theme.creative-theme {
  --creative-paper: var(--site-paper);
  --creative-canvas: var(--site-canvas);
  --creative-soft: var(--site-soft);
  --creative-ink: var(--site-ink);
  --creative-muted: var(--site-muted);
  --creative-line: var(--site-line);
  --creative-accent: var(--site-accent);
  --creative-accent-hover: var(--site-accent-hover);
  --creative-title-font: var(--site-title-font);
  --creative-shadow: 0 12px 40px rgba(0, 0, 0, .24);
  --primary-1: 74, 48, 58;
  --primary-2: 103, 66, 81;
  --primary-3: 153, 91, 112;
  --primary-4: 186, 130, 146;
  --primary-5: 216, 152, 168;
  --primary-6: var(--site-action-rgb);
  --primary-7: 246, 188, 200;
  --primary-8: 250, 210, 218;
  --primary-9: 253, 229, 235;
  --primary-10: 255, 242, 246;
  --admin-danger-soft: color-mix(in srgb, rgb(var(--danger-6)) 20%, transparent);
  --admin-shadow: 0 12px 32px rgba(0, 0, 0, .24);
}

.admin-theme .arco-btn { border-radius: 999px; }
.admin-theme .arco-pagination-item { border-radius: 50%; }
.admin-theme .arco-input-wrapper,
.admin-theme .arco-textarea-wrapper,
.admin-theme .arco-select-view,
.admin-theme .arco-input-tag { border-radius: 10px; }
.admin-theme .arco-input-wrapper:focus-within,
.admin-theme .arco-textarea-wrapper:focus-within,
.admin-theme .arco-select-view-focus { border-color: var(--site-accent); }
.admin-theme .arco-drawer,
.admin-theme .arco-modal { background: var(--admin-surface); color: var(--admin-text); }
.admin-theme .admin-tag-accent {
  background: var(--site-accent-soft);
  border-color: transparent;
  color: var(--site-accent);
}
.admin-theme ::selection { background: var(--site-accent-soft); color: var(--site-accent); }
.admin-theme input,
.admin-theme textarea { caret-color: var(--site-accent); }
.admin-theme { scrollbar-color: var(--site-muted) transparent; }

</style>

<style lang="scss" scoped>
.admin-workspace {
  width: 100%;
  color: var(--admin-text);
}

.admin-workspace-content {
  min-width: 0;
  padding-top: 24px;
}

.admin-page {
  width: 100%;
  min-width: 0;
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

@media (prefers-reduced-motion: reduce) {
  .admin-page-enter-active,
  .admin-page-leave-active { transition: none; }
  .admin-page-enter-from,
  .admin-page-leave-to { transform: none; }
}
</style>

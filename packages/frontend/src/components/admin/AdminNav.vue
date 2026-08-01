<script lang="ts">
// 导航数据结构（供 AdminView 组装）
export interface AdminNavItem {
  key: string
  title: string
  icon: Component
  badge?: number
}

export interface AdminNavGroup {
  label: string
  items: AdminNavItem[]
}
</script>

<script setup lang="ts">
// 后台侧边导航：品牌区 + 分组菜单（带待办角标）+ 底部用户卡
// 同时用于桌面侧边栏与移动端抽屉
import type { Component } from 'vue'
import { IconUser, IconLeft, IconDashboard } from '@arco-design/web-vue/es/icon'
import { getAvatarUrl } from '@/utils'

interface Props {
  groups: AdminNavGroup[]
  active: string
  username?: string
  email?: string
  isAdmin?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  select: [key: string]
}>()
</script>

<template>
  <div class="admin-nav">
    <div class="admin-nav-brand">
      <div class="admin-nav-mark">
        <IconDashboard />
      </div>
      <div class="admin-nav-brand-text">
        <strong>管理控制台</strong>
        <span>Admin Console</span>
      </div>
    </div>

    <div class="admin-nav-scroll">
      <div v-for="group in groups" :key="group.label" class="admin-nav-group">
        <div class="admin-nav-group-label">{{ group.label }}</div>
        <button
          v-for="item in group.items"
          :key="item.key"
          type="button"
          class="admin-nav-item"
          :class="{ 'is-active': active === item.key }"
          @click="emit('select', item.key)"
        >
          <component :is="item.icon" class="admin-nav-icon" />
          <span class="admin-nav-item-label">{{ item.title }}</span>
          <span v-if="item.badge" class="admin-nav-badge">{{ item.badge > 99 ? '99+' : item.badge }}</span>
        </button>
      </div>
    </div>

    <div class="admin-nav-foot">
      <div class="admin-nav-user">
        <a-avatar :size="32" class="admin-nav-avatar" :image-url="email ? getAvatarUrl(email) : ''">
          <IconUser v-if="!email" />
        </a-avatar>
        <div class="admin-nav-user-meta">
          <span class="admin-nav-username">{{ username || '未登录' }}</span>
          <span class="admin-nav-role">{{ isAdmin ? '管理员' : '成员' }}</span>
        </div>
      </div>
      <RouterLink to="/" class="admin-nav-home">
        <IconLeft />
        <span>返回首页</span>
      </RouterLink>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.admin-nav {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

// ── 品牌区 ──
.admin-nav-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px 16px;
}

.admin-nav-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  font-size: 18px;
  color: #fff;
  background: linear-gradient(135deg, rgb(var(--primary-5)), rgb(var(--primary-7)));
  box-shadow: 0 4px 12px color-mix(in srgb, rgb(var(--primary-6)) 35%, transparent);
  flex-shrink: 0;
}

.admin-nav-brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
  min-width: 0;

  strong {
    font-size: 14px;
    font-weight: 600;
    color: var(--admin-text);
    letter-spacing: -0.01em;
  }

  span {
    font-size: 11px;
    color: var(--admin-text-3);
    letter-spacing: 0.04em;
  }
}

// ── 分组菜单 ──
.admin-nav-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 12px 12px;
}

.admin-nav-group + .admin-nav-group {
  margin-top: 20px;
}

.admin-nav-group-label {
  padding: 0 10px 6px;
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--admin-text-3);
}

.admin-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  margin-top: 2px;
  border: none;
  border-radius: 9px;
  background: transparent;
  font-size: 13.5px;
  color: var(--admin-text-2);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s ease, color 0.15s ease;

  &:hover {
    background: var(--admin-fill);
    color: var(--admin-text);
  }

  &.is-active {
    background: var(--admin-primary-soft);
    color: var(--admin-primary);
    font-weight: 500;
  }
}

.admin-nav-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.admin-nav-item-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-nav-badge {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  background: var(--admin-danger-soft);
  color: var(--admin-danger);
  flex-shrink: 0;
}

// ── 底部用户卡 ──
.admin-nav-foot {
  padding: 12px;
  border-top: 1px solid var(--admin-border-soft);
}

.admin-nav-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--admin-fill);
}

.admin-nav-avatar {
  background: var(--admin-primary-soft);
  color: var(--admin-primary);
  flex-shrink: 0;
}

.admin-nav-user-meta {
  display: flex;
  flex-direction: column;
  line-height: 1.35;
  min-width: 0;
}

.admin-nav-username {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-nav-role {
  font-size: 11px;
  color: var(--admin-text-3);
}

.admin-nav-home {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 9px;
  font-size: 13px;
  color: var(--admin-text-3);
  text-decoration: none;
  transition: background 0.15s ease, color 0.15s ease;

  &:hover {
    background: var(--admin-fill);
    color: var(--admin-text);
  }
}
</style>

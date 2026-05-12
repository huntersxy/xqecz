<script setup lang="ts">
import type { User } from '@/types'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()

interface Props {
  user: User
}

const props = defineProps<Props>()

const emit = defineEmits<{
  updateRole: [id: number, isAdmin: boolean]
  updateBan: [id: number, isBanned: boolean]
  delete: [id: number]
}>()

const userId = props.user.id || props.user.ID || 0
const isAdmin = props.user.is_admin || props.user.IsAdmin || false
const isBanned = props.user.is_banned || props.user.IsBanned || false
const username = props.user.username || props.user.Username || ''
const isCurrentUser = userId === (userStore.user?.id || userStore.user?.ID)

const handleUpdateRole = () => emit('updateRole', userId, !isAdmin)
const handleUpdateBan = () => emit('updateBan', userId, !isBanned)
const handleDelete = () => emit('delete', userId)
</script>

<template>
  <div class="user-item mac-card">
    <div class="user-info">
      <div class="user-avatar">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
      </div>
      <div class="user-details">
        <span class="user-name">{{ username }}</span>
        <div class="user-status">
          <span v-if="isAdmin" class="admin-badge">管理员</span>
          <span v-if="isBanned" class="banned-badge">已封禁</span>
        </div>
      </div>
    </div>
    <div class="user-actions">
      <template v-if="!isCurrentUser">
        <button @click="handleUpdateRole" class="action-btn">
          {{ isAdmin ? '取消管理员' : '设为管理员' }}
        </button>
        <template v-if="!isAdmin">
          <button @click="handleUpdateBan" :class="['action-btn', isBanned ? 'unban-btn' : 'ban-btn']">
            {{ isBanned ? '解封' : '封禁' }}
          </button>
          <button @click="handleDelete" class="action-btn delete-btn">删除</button>
        </template>
      </template>
      <span v-else class="current-user-hint">当前用户</span>
    </div>
  </div>
</template>

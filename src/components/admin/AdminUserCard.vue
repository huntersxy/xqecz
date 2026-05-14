<script setup lang="ts">
import type { User } from '@/types';
import { useUserStore } from '@/stores/user';

const userStore = useUserStore();

interface Props {
  user: User;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  updateRole: [id: number, isAdmin: boolean];
  updateBan: [id: number, isBanned: boolean];
  delete: [id: number];
}>();

const userId = props.user.id || 0;
const isAdmin = props.user.is_admin || false;
const isBanned = props.user.is_banned || false;
const username = props.user.username || '';
const isCurrentUser = userId === userStore.user?.id;

const handleUpdateRole = () => emit('updateRole', userId, !isAdmin);
const handleUpdateBan = () => emit('updateBan', userId, !isBanned);
const handleDelete = () => emit('delete', userId);
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

<style scoped>
.user-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
}

.user-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

.user-avatar svg {
  width: 28px;
  height: 28px;
}

.user-details {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.user-name {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
}

.user-status {
  display: flex;
  gap: 8px;
}

.admin-badge {
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #3b82f6;
}

.banned-badge {
  padding: 2px 8px;
  background: rgba(220, 38, 38, 0.1);
  border-radius: 4px;
  font-size: 11px;
  color: #dc2626;
}

.user-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: center;
}

.action-btn {
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 13px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.action-btn.ban-btn {
  background: rgba(249, 115, 22, 0.1);
  border-color: rgba(249, 115, 22, 0.3);
  color: #f97316;
}

.action-btn.unban-btn {
  background: rgba(5, 150, 105, 0.1);
  border-color: rgba(5, 150, 105, 0.3);
  color: #059669;
}

.action-btn.delete-btn {
  background: rgba(220, 38, 38, 0.1);
  border-color: rgba(220, 38, 38, 0.3);
  color: #dc2626;
}

.action-btn.delete-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.current-user-hint {
  font-size: 13px;
  color: #888;
}

@media screen and (max-width: 768px) {
  .user-item {
    padding: 16px;
  }
}
</style>
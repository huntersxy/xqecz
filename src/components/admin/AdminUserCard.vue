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
  <div class="flex flex-col items-center gap-3 p-5 bg-[var(--theme-card-bg)] border border-white/30 rounded-xl shadow-md">
    <div class="flex flex-col items-center gap-2">
      <div class="w-14 h-14 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-full flex items-center justify-center text-white">
        <svg class="w-7 h-7" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
          <circle cx="12" cy="7" r="4"/>
        </svg>
      </div>
      <div class="flex flex-col items-center gap-1">
        <span class="text-[15px] font-semibold theme-text">{{ username }}</span>
        <div class="flex gap-2">
          <span v-if="isAdmin" class="px-2 py-0.5 bg-[var(--theme-primary)]/10 rounded text-[11px] text-[var(--theme-primary)]">管理员</span>
          <span v-if="isBanned" class="px-2 py-0.5 bg-[var(--theme-danger)]/10 rounded text-[11px] text-[var(--theme-danger)]">已封禁</span>
        </div>
      </div>
    </div>
    <div class="flex gap-2 flex-wrap justify-center">
      <template v-if="!isCurrentUser">
        <button @click="handleUpdateRole" class="px-3 py-1.5 bg-[var(--theme-surface)] border border-gray-200 rounded-md text-[13px] theme-text hover:text-[var(--theme-primary)] hover:border-blue-300 transition-colors">
          {{ isAdmin ? '取消管理员' : '设为管理员' }}
        </button>
        <template v-if="!isAdmin">
          <button @click="handleUpdateBan" :class="['px-3 py-1.5 border rounded-md text-[13px] transition-colors', isBanned ? 'bg-orange-100/50 border-orange-200 text-orange-600 hover:bg-orange-100' : 'bg-[var(--theme-warning)]/10/50 border-amber-200 text-[var(--theme-warning)] hover:bg-[var(--theme-warning)]/10']">
            {{ isBanned ? '解封' : '封禁' }}
          </button>
          <button @click="handleDelete" class="px-3 py-1.5 bg-[var(--theme-danger)]/10/50 border border-red-200 rounded-md text-[13px] text-[var(--theme-danger)] hover:bg-[var(--theme-danger)]/10 transition-colors">删除</button>
        </template>
      </template>
      <span v-else class="text-[13px] theme-text-secondary">当前用户</span>
    </div>
  </div>
</template>

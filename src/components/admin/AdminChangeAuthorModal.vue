<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue';
import type { User } from '@/types';
import { adminApi } from '@/api';

interface Props {
  visible: boolean;
  contentId: number;
  currentAuthorName: string;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  close: [];
  success: [];
}>();

const selectedUserId = ref<number | null>(null);
const users = ref<User[]>([]);
const loading = ref(false);
const isUpdating = ref(false);
const message = ref('');
const searchKeyword = ref('');
const searchInput = ref<HTMLInputElement | null>(null);

const filteredUsers = computed(() => {
  if (!searchKeyword.value.trim()) {
    return users.value;
  }
  const keyword = searchKeyword.value.toLowerCase();
  return users.value.filter(user => {
    const username = (user.username || user.Username || '').toLowerCase();
    return username.includes(keyword);
  });
});

async function loadUsers() {
  try {
    loading.value = true;
    const res = await adminApi.getUsers({ page: 1, page_size: 1000 });
    if (res.code === 200) {
      users.value = res.data.list;
    }
  } catch (error) {
    console.error('加载用户失败', error);
    message.value = '加载用户列表失败';
  } finally {
    loading.value = false;
  }
}

async function handleUpdateAuthor() {
  if (!selectedUserId.value) {
    message.value = '请选择新作者';
    return;
  }

  try {
    isUpdating.value = true;
    const res = await adminApi.updateContentAuthor(props.contentId, selectedUserId.value);
    if (res.code === 200) {
      message.value = '更新成功';
      setTimeout(() => {
        emit('success');
        emit('close');
      }, 500);
    } else {
      message.value = res.message || '更新失败';
    }
  } catch (error) {
    console.error('更新作者失败', error);
    message.value = '更新失败';
  } finally {
    isUpdating.value = false;
  }
}

function handleSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.preventDefault();
  }
}

onMounted(() => {
  if (props.visible) {
    loadUsers();
  }
});

watch(
  () => props.visible,
  (newVal) => {
    if (newVal) {
      loadUsers();
      selectedUserId.value = null;
      message.value = '';
      searchKeyword.value = '';
      setTimeout(() => {
        searchInput.value?.focus();
      }, 100);
    }
  }
);
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>修改上传者</h2>
        <span class="modal-close" @click="$emit('close')">×</span>
      </div>
      <div class="modal-body">
        <div class="form-row">
          <label class="form-label">当前作者</label>
          <div class="current-author">{{ currentAuthorName }}</div>
        </div>
        <div class="form-row">
          <label class="form-label">选择新作者</label>
          <div class="search-box">
            <input
              ref="searchInput"
              v-model="searchKeyword"
              type="text"
              class="search-input"
              placeholder="搜索用户名..."
              @keydown="handleSearchKeydown"
            />
            <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </div>
          <div v-if="loading" class="loading-state">加载中...</div>
          <div v-else class="user-select-wrapper">
            <div class="user-select">
              <div
                v-for="user in filteredUsers"
                :key="user.id || user.ID"
                :class="['user-option', { selected: selectedUserId === (user.id || user.ID) }]"
                @click="selectedUserId = user.id || user.ID"
              >
                <div class="user-info">
                  <span class="username">{{ user.username || user.Username }}</span>
                  <span class="user-role" v-if="user.is_admin || user.IsAdmin">管理员</span>
                </div>
              </div>
              <div v-if="filteredUsers.length === 0" class="no-results">
                未找到匹配的用户
              </div>
            </div>
          </div>
          <div class="user-count">
            共 {{ filteredUsers.length }} 位用户
            <span v-if="searchKeyword">（筛选自 {{ users.length }} 位）</span>
          </div>
        </div>
        <div v-if="message" class="message" :class="{ error: message.includes('失败'), success: message.includes('成功') }">
          {{ message }}
        </div>
      </div>
      <div class="modal-footer">
        <button @click="$emit('close')" class="mac-btn" :disabled="isUpdating">取消</button>
        <button @click="handleUpdateAuthor" class="mac-btn primary-btn" :disabled="isUpdating || !selectedUserId">
          {{ isUpdating ? '更新中...' : '确认更新' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
}

.modal-content {
  width: 90%;
  max-width: 500px;
  max-height: 85vh;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  position: relative;
  z-index: 10000;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.05) 0%, transparent 100%);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.modal-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.modal-close {
  font-size: 24px;
  color: #999;
  cursor: pointer;
  line-height: 1;
}

.modal-close:hover {
  color: #3b82f6;
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.form-row {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: #333;
  margin-bottom: 8px;
}

.current-author {
  padding: 10px 12px;
  background: rgba(59, 130, 246, 0.05);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 8px;
  font-size: 14px;
  color: #1a1a1a;
}

.search-box {
  position: relative;
  margin-bottom: 12px;
}

.search-input {
  width: 100%;
  padding: 10px 12px 10px 36px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  background: white;
  box-sizing: border-box;
}

.search-input:focus {
  outline: none;
  border-color: rgba(59, 130, 246, 0.5);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: #999;
  pointer-events: none;
}

.loading-state {
  padding: 20px;
  text-align: center;
  color: #999;
}

.user-select-wrapper {
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  overflow: hidden;
}

.user-select {
  max-height: 280px;
  overflow-y: scroll;
  -webkit-overflow-scrolling: touch;
}

.user-select::-webkit-scrollbar {
  width: 8px;
}

.user-select::-webkit-scrollbar-track {
  background: rgba(0, 0, 0, 0.02);
}

.user-select::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.15);
  border-radius: 4px;
}

.user-select::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.25);
}

.user-option {
  padding: 12px 16px;
  cursor: pointer;
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  transition: all 0.2s ease;
  background: white;
}

.user-option:last-child {
  border-bottom: none;
}

.user-option:hover {
  background: rgba(59, 130, 246, 0.08);
}

.user-option.selected {
  background: rgba(59, 130, 246, 0.15);
  border-left: 3px solid #3b82f6;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.username {
  font-size: 14px;
  color: #1a1a1a;
  font-weight: 500;
}

.user-role {
  font-size: 12px;
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
  border-radius: 4px;
}

.no-results {
  padding: 20px;
  text-align: center;
  color: #999;
  font-size: 14px;
}

.user-count {
  margin-top: 8px;
  font-size: 12px;
  color: #888;
  text-align: right;
}

.message {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 13px;
  margin-top: 8px;
}

.message.success {
  background: rgba(16, 185, 129, 0.1);
  color: #059669;
}

.message.error {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.03);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
}

.mac-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mac-btn:hover {
  background: rgba(59, 130, 246, 0.08);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.mac-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.primary-btn {
  background: rgba(59, 130, 246, 0.95);
  color: white;
  border-color: rgba(59, 130, 246, 0.95);
}

.primary-btn:hover:not(:disabled) {
  background: rgba(37, 99, 235, 0.95);
  color: white;
  border-color: rgba(37, 99, 235, 0.95);
}

@media screen and (max-width: 768px) {
  .modal-content {
    width: 95%;
    max-width: none;
    margin: 12px;
    max-height: 80vh;
  }

  .modal-body {
    padding: 16px;
  }

  .modal-footer {
    padding: 14px 16px;
    flex-wrap: wrap;
  }

  .user-select {
    max-height: 200px;
  }
}
</style>

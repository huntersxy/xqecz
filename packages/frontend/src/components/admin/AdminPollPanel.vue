<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import { useConfirm } from '@/composables/useToast'
import { Tooltip } from '@arco-design/web-vue'
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()

async function confirmDeletePoll(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该投票？')
  if (!ok) return
  await admin.deletePoll(id)
  admin.loadPolls()
}

onMounted(() => admin.loadPolls())
</script>

<template>
  <AdminPanel title="投票管理" :desc="`共 ${admin.polls.length} 条投票`">
    <template #actions>
      <a-button type="primary" size="small" @click="admin.showCreatePollModal = true">
        <IconPlus /> 创建投票
      </a-button>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="admin.pollsLoading" @click="admin.loadPolls()">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="admin.pollsLoading">
      <div class="poll-list">
        <a-empty v-if="admin.polls.length === 0 && !admin.pollsLoading" description="暂无投票" />
        <div v-for="p in admin.polls" :key="p.id" class="poll-item">
          <div class="poll-main">
            <div class="poll-head">
              <span class="poll-title">{{ p.title }}</span>
              <span class="poll-meta">{{ p.vote_count }} 票 · {{ new Date(p.created_at).toLocaleString() }}</span>
            </div>
            <div v-if="p.description" class="poll-desc">{{ p.description }}</div>
            <div class="poll-options">
              <span v-for="(o, i) in p.options" :key="i" class="poll-chip">{{ o }}</span>
            </div>
          </div>
          <Tooltip title="删除投票">
            <a-button class="admin-icon-btn is-danger poll-delete" type="text" size="small" @click="confirmDeletePoll(p.id)">
              <IconDelete />
            </a-button>
          </Tooltip>
        </div>
      </div>
    </a-spin>
  </AdminPanel>

  <a-modal v-model:visible="admin.showCreatePollModal" title="创建投票" @ok="admin.createPoll">
    <a-form layout="vertical" :model="admin.createPollForm">
      <a-form-item label="投票标题" required><a-input v-model="admin.createPollForm.title" placeholder="请输入投票标题" /></a-form-item>
      <a-form-item label="投票描述"><a-textarea v-model="admin.createPollForm.description" :auto-size="{ minRows: 3, maxRows: 6 }" placeholder="可选描述" /></a-form-item>
      <a-form-item label="投票选项" required>
        <div class="w-full space-y-2">
          <div v-for="(o, i) in admin.createPollForm.options" :key="i" class="flex gap-2 items-center">
            <a-input v-model="admin.createPollForm.options[i]" placeholder="选项内容" class="flex-1" />
            <a-button v-if="admin.createPollForm.options.length > 2" status="danger" shape="circle" size="small" @click="admin.removePollOption(i)"><IconDelete /></a-button>
          </div>
        </div>
        <a-button class="mt-2" size="small" @click="admin.addPollOption"><IconPlus /> 添加选项</a-button>
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.poll-list {
  display: flex;
  flex-direction: column;
}

.poll-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 18px 24px;
  border-bottom: 1px solid $admin-border-soft;
  transition: background 0.15s ease;

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background: $admin-fill;
  }
}

.poll-main {
  flex: 1;
  min-width: 0;
}

.poll-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

.poll-title {
  font-size: 14px;
  font-weight: 600;
  color: $admin-text;
}

.poll-meta {
  font-size: 12px;
  color: $admin-text-3;
  font-variant-numeric: tabular-nums;
}

.poll-desc {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.6;
  color: $admin-text-2;
}

.poll-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.poll-chip {
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  background: $admin-fill;
  color: $admin-text-2;
  border: 1px solid $admin-border-soft;
}

.poll-delete {
  flex-shrink: 0;
  margin-top: 2px;
}
</style>

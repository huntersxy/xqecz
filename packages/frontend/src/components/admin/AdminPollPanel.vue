<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import { useConfirm } from '@/composables/useToast'
import { Tooltip } from '@arco-design/web-vue'
import { IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()

// 首屏加载才出骨架；已有数据时刷新只让工具条的按钮转圈，列表保持可读
const showSkeleton = computed(() => admin.pollsLoading && admin.polls.length === 0)
const pollSummary = computed(() => `共 ${admin.polls.length} 条投票`)

async function confirmDeletePoll(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该投票？删除后不可恢复。')
  if (!ok) return
  await admin.deletePoll(id)
  admin.loadPolls()
}

onMounted(() => admin.loadPolls())
</script>

<template>
  <AdminPanel title="投票管理" desc="管理前台投票广场的投票；删除后不可恢复。">
    <template #actions>
      <RouterLink to="/polls" class="poll-public-link">前台投票广场</RouterLink>
      <a-button type="primary" size="small" @click="admin.showCreatePollModal = true">
        <IconPlus /> 创建投票
      </a-button>
    </template>

    <template #toolbar>
      <AdminDataToolbar :summary="pollSummary" :loading="admin.pollsLoading" refreshable @refresh="admin.loadPolls()" />
    </template>

    <AdminSkeleton v-if="showSkeleton" variant="rows" :rows="5" :columns="3" label="正在加载投票" />

    <AdminEmpty
      v-else-if="admin.polls.length === 0"
      title="暂无投票"
      desc="创建第一个投票，让观众参与选择。"
    >
      <template #actions>
        <a-button type="primary" size="small" @click="admin.showCreatePollModal = true">
          <IconPlus /> 创建投票
        </a-button>
      </template>
    </AdminEmpty>

    <div v-else class="poll-list adm-stagger">
      <div v-for="p in admin.polls" :key="p.id" class="poll-item">
        <div class="poll-main">
          <div class="poll-head">
            <RouterLink :to="`/polls/${p.id}`" class="poll-title">{{ p.title }}</RouterLink>
            <span class="poll-meta">{{ p.vote_count }} 票 · {{ new Date(p.created_at).toLocaleString() }}</span>
          </div>
          <div v-if="p.description" class="poll-desc">{{ p.description }}</div>
          <div v-if="p.options.length" class="poll-options">
            <span v-for="(o, i) in p.options" :key="i" class="poll-chip">{{ o }}</span>
          </div>
        </div>
        <Tooltip title="删除投票">
          <a-button class="admin-icon-btn is-danger poll-delete" type="text" size="small" aria-label="删除投票" @click="confirmDeletePoll(p.id)">
            <IconDelete />
          </a-button>
        </Tooltip>
      </div>
    </div>
  </AdminPanel>

  <a-modal class="admin-theme creative-theme creative-dialog" v-model:visible="admin.showCreatePollModal" title="创建投票" @ok="admin.createPoll">
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

// 工具条自身有上下留白，外层 .admin-panel-toolbar 也有一层，抵消掉避免双倍
:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.poll-list {
  display: flex;
  flex-direction: column;
}

.poll-item {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-md, 12px);
  padding: 16px var(--adm-gap-xl, 24px);
  border-bottom: 1px solid var(--admin-border-soft);
  transition: background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, ease);

  &:last-child {
    border-bottom: none;
  }

  &:hover {
    background: var(--adm-soft, var(--admin-fill));
  }
}

.poll-main {
  flex: 1;
  min-width: 0;
}

.poll-head {
  display: flex;
  align-items: baseline;
  gap: var(--adm-gap-sm, 8px);
  flex-wrap: wrap;
}

.poll-title {
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
  text-decoration: none;
  border-radius: 4px;
  @include focus-ring(2px);

  &:hover {
    color: var(--adm-accent, var(--creative-accent));
  }
}

.poll-public-link {
  padding: 8px 2px;
  border-radius: 4px;
  color: var(--adm-accent, var(--creative-accent));
  font-size: var(--adm-font-sm, 13px);
  text-decoration: none;
  @include focus-ring(2px);

  &:hover {
    text-decoration: underline;
    text-underline-offset: 3px;
  }
}

.poll-meta {
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
  @include numeric;
}

.poll-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  max-width: 72ch;
  margin-top: 6px;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-2));
}

.poll-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.poll-chip {
  padding: 3px 10px;
  border-radius: var(--adm-r-pill, 999px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
}

// §6.2：桌面图标按钮 36px；手机档由 AdminPanel 的 44px 规则接管
.poll-delete {
  flex-shrink: 0;
  width: var(--adm-touch, 36px);
  height: var(--adm-touch, 36px);
}

@media (max-width: 768px) {
  .poll-item {
    padding: 14px var(--adm-gap-lg, 16px);
  }

  .poll-head {
    flex-direction: column;
    gap: 4px;
  }
}
</style>

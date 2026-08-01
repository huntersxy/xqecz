<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { toast, useConfirm } from '@/composables/useToast'
import { apiKeyApi } from '@/api'
import type { ApiKey, ApiKeyCreated } from '@/types'
import { Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-vue/es/icon'
import AdminApiKeysDocs from './AdminApiKeysDocs.vue'

const loading = ref(false)
const keys = ref<ApiKey[]>([])
const showCreateModal = ref(false)
const showKeyModal = ref(false)
const createdKey = ref<ApiKeyCreated | null>(null)

// Create form
const createForm = ref({ name: '', permissions: ['upload'] as string[] })
const allPermissions = [
  { label: '上传内容', value: 'upload' },
  { label: '读取内容', value: 'read' },
  { label: '删除内容', value: 'delete' },
]

const columns: TableColumnData[] = [
  { title: '名称', dataIndex: 'name', minWidth: 140 },
  { title: '前缀', slotName: 'prefix', width: 130 },
  { title: '权限', slotName: 'permissions', width: 170 },
  { title: '状态', slotName: 'status', width: 90 },
  { title: '最后使用', slotName: 'lastUsed', width: 160 },
  { title: '创建时间', slotName: 'createdAt', width: 160 },
  { title: '操作', slotName: 'actions', width: 100, align: 'center' },
]

async function loadKeys() {
  loading.value = true
  try {
    const res = await apiKeyApi.list()
    if (res.code === 200) {
      keys.value = res.data.list
    }
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.name.trim()) {
    toast.warning('请输入名称')
    return
  }
  if (createForm.value.permissions.length === 0) {
    toast.warning('至少选择一个权限')
    return
  }

  const res = await apiKeyApi.create({
    name: createForm.value.name.trim(),
    permissions: createForm.value.permissions,
  })
  if (res.code === 200) {
    createdKey.value = res.data as ApiKeyCreated
    showCreateModal.value = false
    showKeyModal.value = true
    createForm.value = { name: '', permissions: ['upload'] }
    loadKeys()
  }
}

async function handleDelete(id: number, name: string) {
  const { confirm } = useConfirm()
  const ok = await confirm(`确定要撤销 "${name}" 吗？撤销后使用该 Key 的应用将立即无法访问。`)
  if (!ok) return
  await apiKeyApi.delete(id)
  toast.success('已撤销')
  loadKeys()
}

async function toggleActive(key: ApiKey) {
  await apiKeyApi.update(key.id, { is_active: !key.is_active })
  toast.success(key.is_active ? '已禁用' : '已启用')
  loadKeys()
}

function copyKey() {
  if (createdKey.value?.key) {
    navigator.clipboard.writeText(createdKey.value.key)
    toast.success('已复制到剪贴板')
  }
}

function formatDate(ts: number | null) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

function permLabel(p: string) {
  const map: Record<string, string> = { upload: '上传', read: '读取', delete: '删除' }
  return map[p] || p
}

function permColor(p: string) {
  const map: Record<string, string> = { upload: 'arcoblue', read: 'green', delete: 'red' }
  return map[p] || 'gray'
}

onMounted(loadKeys)
</script>

<template>
  <AdminPanel title="API 密钥" desc="用于第三方应用接入；完整密钥仅在创建时展示一次，请妥善保存。">
    <template #actions>
      <a-button type="primary" size="small" @click="showCreateModal = true">
        <IconPlus /> 新建密钥
      </a-button>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="loading" @click="loadKeys">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="loading">
      <a-table
        :columns="columns"
        :data="keys"
        :pagination="false"
        row-key="id"
        :scroll="{ x: 860 }"
      >
        <template #prefix="{ record }">
          <code class="admin-mono">{{ record.key_prefix }}…</code>
        </template>
        <template #permissions="{ record }">
          <a-tag v-for="p in record.permissions" :key="p" :color="permColor(p)" :bordered="false" size="small" class="admin-tag-inline">
            {{ permLabel(p) }}
          </a-tag>
        </template>
        <template #status="{ record }">
          <AdminStatus :type="record.is_active ? 'success' : 'neutral'" :label="record.is_active ? '启用' : '禁用'" />
        </template>
        <template #lastUsed="{ record }">
          <span class="admin-cell-3">{{ formatDate(record.last_used_at) }}</span>
        </template>
        <template #createdAt="{ record }">
          <span class="admin-cell-3">{{ formatDate(record.created_at) }}</span>
        </template>
        <template #actions="{ record }">
          <div class="admin-action-group">
            <a-switch
              size="small"
              :model-value="record.is_active"
              @change="toggleActive(record)"
            />
            <Tooltip title="撤销密钥">
              <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleDelete(record.id, record.name)">
                <IconDelete />
              </a-button>
            </Tooltip>
          </div>
        </template>
      </a-table>
    </a-spin>

    <!-- API 使用说明与多语言示例 -->
    <AdminApiKeysDocs />

    <!-- 新建弹窗 -->
    <a-modal v-model:visible="showCreateModal" title="新建 API 密钥" @ok="handleCreate" ok-text="创建" cancel-text="取消">
      <a-form layout="vertical" :model="createForm">
        <a-form-item label="名称" required>
          <a-input v-model="createForm.name" placeholder="例如：我的上传工具" :maxlength="100" />
        </a-form-item>
        <a-form-item label="权限" required>
          <a-checkbox-group v-model="createForm.permissions" :options="(allPermissions as any)" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 显示完整 Key 弹窗 -->
    <a-modal v-model:visible="showKeyModal" title="API 密钥已创建" :footer="false">
      <div class="mb-3">
        <p class="text-sm mb-2">请保存以下密钥，<strong>关闭后将无法再次查看</strong>：</p>
        <div class="key-reveal">
          <code class="key-reveal-code">{{ createdKey?.key }}</code>
          <a-button size="small" @click="copyKey">复制</a-button>
        </div>
      </div>
      <div class="key-hint">
        <p>在请求头中添加：</p>
        <code>X-API-Key: {{ createdKey?.key }}</code>
      </div>
    </a-modal>
  </AdminPanel>
</template>

<style lang="scss" scoped>
// 弹窗内容 teleport 至 body，不在 .admin-theme 作用域内，此处只用全局 Arco token
.key-reveal {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  border-radius: 10px;
  background: var(--color-fill-2);
}

.key-reveal-code {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  word-break: break-all;
  user-select: all;
}

.key-hint {
  font-size: 12px;
  color: var(--color-text-3);

  code {
    display: inline-block;
    margin-top: 4px;
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--color-fill-2);
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    word-break: break-all;
  }
}
</style>

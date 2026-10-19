<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { toast, useConfirm } from '@/composables/useToast'
import { apiKeyApi } from '@/api'
import type { ApiKey, ApiKeyCreated } from '@/types'
import { Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { IconCheck, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import { useMediaQuery } from '@vueuse/core'

const loading = ref(false)
const keys = ref<ApiKey[]>([])
const showCreateModal = ref(false)
const showKeyModal = ref(false)
const createdKey = ref<ApiKeyCreated | null>(null)
const creating = ref(false)

// ≤768 走卡片列表，与其余后台表格页一致
const isMobile = useMediaQuery('(max-width: 768px)')

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

const activeCount = computed(() => keys.value.filter((k) => k.is_active).length)
// 工具区计数：数字走 tabular-nums，空列表给明确文案
const summary = computed(() => (keys.value.length ? `共 ${keys.value.length} 个密钥，启用 ${activeCount.value} 个` : '暂无密钥'))

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
  if (creating.value) return
  if (!createForm.value.name.trim()) {
    toast.warning('请输入名称')
    return
  }
  if (createForm.value.permissions.length === 0) {
    toast.warning('至少选择一个权限')
    return
  }

  creating.value = true
  try {
    const res = await apiKeyApi.create({
      name: createForm.value.name.trim(),
      permissions: createForm.value.permissions,
    })
    if (res.code === 200) {
      createdKey.value = res.data as ApiKeyCreated
      showCreateModal.value = false
      showKeyModal.value = true
      copied.value = false
      createForm.value = { name: '', permissions: ['upload'] }
      loadKeys()
    }
  } catch {
    toast.error('创建失败，请稍后重试')
  } finally {
    creating.value = false
  }
}

async function handleDelete(id: number, name: string) {
  const { confirm } = useConfirm()
  const ok = await confirm(`确定要撤销 "${name}" 吗？撤销后使用该 Key 的应用将立即无法访问。`)
  if (!ok) return
  try {
    await apiKeyApi.delete(id)
    toast.success('已撤销')
    loadKeys()
  } catch {
    toast.error('撤销失败，请稍后重试')
  }
}

async function toggleActive(key: ApiKey) {
  try {
    await apiKeyApi.update(key.id, { is_active: !key.is_active })
    toast.success(key.is_active ? '已禁用' : '已启用')
    loadKeys()
  } catch {
    toast.error('状态切换失败，请稍后重试')
  }
}

// 一次性密钥只在创建弹层里出现一次，复制后给按钮一个短暂的完成态
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

function flashCopied() {
  copied.value = true
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copied.value = false
    copiedTimer = null
  }, 1600)
}

async function copyKey() {
  const key = createdKey.value?.key
  if (!key) return
  try {
    if (!navigator.clipboard) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(key)
    toast.success('已复制到剪贴板')
    flashCopied()
  } catch {
    toast.warning('复制失败，请手动选择密钥')
  }
}

watch(showKeyModal, (visible) => {
  if (!visible) copied.value = false
})

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = null
})

function formatDate(ts: number | null) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN')
}

function permLabel(p: string) {
  const map: Record<string, string> = { upload: '上传', read: '读取', delete: '删除' }
  return map[p] || p
}

function permColor(p: string) {
  const map: Record<string, string> = { read: 'green', delete: 'red' }
  if (p === 'upload') return undefined
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
    </template>

    <template #toolbar>
      <AdminDataToolbar :summary="summary" :loading="loading" refreshable @refresh="loadKeys" />
    </template>

    <!-- 首屏加载 -->
    <AdminSkeleton v-if="loading && !keys.length" variant="rows" :rows="5" :columns="5" label="正在加载密钥" />

    <!-- 空态 -->
    <AdminEmpty
      v-else-if="!keys.length"
      title="还没有 API 密钥"
      desc="创建一把密钥，第三方应用带上 X-API-Key 请求头即可调用内容接口。"
    >
      <template #actions>
        <a-button type="primary" size="small" @click="showCreateModal = true">
          <IconPlus /> 新建密钥
        </a-button>
      </template>
    </AdminEmpty>

    <!-- 手机档卡片列表 -->
    <div v-else-if="isMobile" class="admin-mobile-list keys-mobile">
      <article v-for="key in keys" :key="key.id" class="admin-mobile-card keys-mobile-card">
        <header class="keys-mobile-head">
          <span class="keys-mobile-name">{{ key.name }}</span>
          <AdminStatus :type="key.is_active ? 'success' : 'neutral'" :label="key.is_active ? '启用' : '禁用'" />
        </header>
        <code class="admin-mono keys-mobile-prefix">{{ key.key_prefix }}…</code>
        <div class="keys-mobile-perms">
          <a-tag
            v-for="p in key.permissions"
            :key="p"
            :color="permColor(p)"
            :bordered="false"
            size="small"
            :class="['admin-tag-inline', { 'admin-tag-accent': p === 'upload' }]"
          >{{ permLabel(p) }}</a-tag>
        </div>
        <dl class="keys-mobile-meta">
          <div class="keys-mobile-meta-row">
            <dt>最后使用</dt>
            <dd class="admin-cell-3">{{ formatDate(key.last_used_at) }}</dd>
          </div>
          <div class="keys-mobile-meta-row">
            <dt>创建时间</dt>
            <dd class="admin-cell-3">{{ formatDate(key.created_at) }}</dd>
          </div>
        </dl>
        <div class="admin-mobile-actions">
          <a-switch size="small" :model-value="key.is_active" @change="toggleActive(key)" />
          <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleDelete(key.id, key.name)">
            <IconDelete /> 撤销密钥
          </a-button>
        </div>
      </article>
    </div>

    <!-- 桌面档表格 -->
    <a-table
      v-else
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
        <a-tag v-for="p in record.permissions" :key="p" :color="permColor(p)" :bordered="false" size="small" :class="['admin-tag-inline', { 'admin-tag-accent': p === 'upload' }]">
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

    <!-- 新建弹窗 -->
    <a-modal class="admin-theme creative-theme creative-dialog" v-model:visible="showCreateModal" title="新建 API 密钥" :ok-loading="creating" @ok="handleCreate" ok-text="创建" cancel-text="取消">
      <a-form layout="vertical" :model="createForm">
        <a-form-item label="名称" required>
          <a-input v-model="createForm.name" placeholder="例如：我的上传工具" :maxlength="100" />
        </a-form-item>
        <a-form-item label="权限" required>
          <a-checkbox-group v-model="createForm.permissions" :options="(allPermissions as any)" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 显示完整 Key 弹窗：关闭后不可再次查看 -->
    <a-modal class="admin-theme creative-theme creative-dialog" v-model:visible="showKeyModal" title="API 密钥已创建" :footer="false">
      <div class="key-reveal-block">
        <p class="key-reveal-tip">请保存以下密钥，<strong>关闭后将无法再次查看</strong>：</p>
        <div class="key-reveal" :class="{ 'is-copied': copied }">
          <code class="key-reveal-code">{{ createdKey?.key }}</code>
          <a-button size="small" :class="['key-copy-btn', { 'is-copied': copied }]" @click="copyKey">
            <IconCheck v-if="copied" class="key-copy-icon" />
            {{ copied ? '已复制' : '复制' }}
          </a-button>
        </div>
        <p class="key-reveal-feedback" aria-live="polite">{{ copied ? '完整密钥已复制到剪贴板' : '' }}</p>
      </div>
      <div class="key-hint">
        <p>在请求头中添加：</p>
        <code>X-API-Key: {{ createdKey?.key }}</code>
      </div>
    </a-modal>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 手机档卡片
.keys-mobile-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-sm, 8px);
}

.keys-mobile-name {
  min-width: 0;
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
  @include ellipsis;
}

.keys-mobile-prefix {
  display: block;
  margin-top: 6px;
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted, var(--admin-text-3));
  word-break: break-all;
}

.keys-mobile-perms {
  display: flex;
  flex-wrap: wrap;
  gap: var(--adm-gap-xs, 4px);
  margin-top: var(--adm-gap-sm, 8px);
}

.keys-mobile-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: var(--adm-gap-md, 12px) 0 0;
}

.keys-mobile-meta-row {
  display: flex;
  justify-content: space-between;
  gap: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-xs, 12px);

  dt {
    color: var(--adm-muted, var(--admin-text-3));
  }

  dd {
    margin: 0;
    color: var(--adm-ink, var(--admin-text));
  }
}

.keys-mobile :deep(.arco-btn) {
  min-height: var(--adm-touch, 44px);
}

// 密钥弹层内容 teleport 至 body；--adm-* 挂在 body 上，依旧可用
.key-reveal-tip {
  margin: 0 0 var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.6;
  color: var(--adm-ink, var(--admin-text));
}

.key-reveal {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--admin-border));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--color-fill-2));
  transition: border-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  &.is-copied {
    border-color: var(--adm-success, var(--color-success-6));
  }
}

.key-reveal-code {
  flex: 1;
  min-width: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-ink, var(--admin-text));
  word-break: break-all;
  user-select: all;
}

.key-copy-btn {
  flex: none;
  min-height: var(--adm-touch, 36px);

  &.is-copied {
    color: var(--adm-success, var(--color-success-6));
    border-color: var(--adm-success, var(--color-success-6));
  }
}

// M9 同款一次性反馈：scale 1 → 1.15 → 1，非无限循环
.key-copy-icon {
  animation: key-copied-pop var(--adm-dur-slow, 240ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1)) 1;
}

@keyframes key-copied-pop {
  0% { transform: scale(1); }
  55% { transform: scale(1.15); }
  100% { transform: scale(1); }
}

.key-reveal-feedback {
  min-height: 18px;
  margin: 6px 0 0;
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-success, var(--color-success-6));
}

.key-hint {
  margin-top: var(--adm-gap-lg, 16px);
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--color-text-3));

  p {
    margin: 0;
  }

  code {
    display: inline-block;
    margin-top: var(--adm-gap-xs, 4px);
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--adm-soft, var(--color-fill-2));
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    color: var(--adm-ink, var(--admin-text));
    word-break: break-all;
  }
}

@media (prefers-reduced-motion: reduce) {
  .key-copy-icon {
    animation: none;
  }
}
</style>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { IconCheck, IconCopy, IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'
import { apiKeyApi } from '@/api'
import { toast, useConfirm } from '@/composables/useToast'
import { formatTime } from '@/utils'
import type { ApiKey, ApiKeyCreated } from '@/types'

const loading = ref(false)
const keys = ref<ApiKey[]>([])
const showCreateModal = ref(false)
const showKeyModal = ref(false)
const createdKey = ref<ApiKeyCreated | null>(null)
const creating = ref(false)
const togglingIds = ref<number[]>([])
const leavingIds = ref<number[]>([])

const createForm = ref({ name: '', permissions: ['upload'] as string[] })
const allPermissions = [
  { label: '上传内容', value: 'upload' },
  { label: '读取内容', value: 'read' },
  { label: '删除内容', value: 'delete' },
]

const showSkeleton = computed(() => loading.value && keys.value.length === 0)

const activeCount = computed(() => keys.value.filter(key => key.is_active).length)
const inactiveCount = computed(() => keys.value.length - activeCount.value)

const latestUsed = computed<ApiKey | null>(() => keys.value.reduce<ApiKey | null>((best, key) => {
  if (!key.last_used_at) return best
  if (!best || key.last_used_at > (best.last_used_at ?? 0)) return key
  return best
}, null))
const latestUsedText = computed(() => {
  const used = latestUsed.value
  return used?.last_used_at ? relativeTime(used.last_used_at) : '从未使用'
})
const latestUsedTitle = computed(() => {
  const used = latestUsed.value
  return used?.last_used_at ? fullTime(used.last_used_at) : ''
})

const summary = computed(() => (keys.value.length ? `共 ${keys.value.length} 个密钥，启用 ${activeCount.value} 个` : '暂无密钥'))

function toDate(ts: number | string | null): Date | null {
  if (!ts) return null
  const date = typeof ts === 'string' && /^\d{4}-/.test(ts)
    ? new Date(ts)
    : new Date((typeof ts === 'string' ? Number.parseInt(ts, 10) : ts) * 1000)
  return Number.isNaN(date.getTime()) ? null : date
}

function relativeTime(ts: number | string | null) {
  const date = toDate(ts)
  if (!date) return '从未使用'
  const diff = Date.now() - date.getTime()
  if (diff < 60_000) return '刚刚'
  const minutes = Math.floor(diff / 60_000)
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  const months = Math.floor(days / 30)
  if (months < 12) return `${months} 个月前`
  return `${Math.floor(months / 12)} 年前`
}

function fullTime(ts: number | string | null) {
  if (!ts) return '—'
  return formatTime(ts) || '—'
}

function permLabel(p: string) {
  const map: Record<string, string> = { upload: '上传', read: '读取', delete: '删除' }
  return map[p] || p
}

function permTone(p: string) {
  if (p === 'upload') return 'is-upload'
  if (p === 'delete') return 'is-delete'
  return ''
}

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

function isToggling(id: number) {
  return togglingIds.value.includes(id)
}

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

function openCreate() {
  showCreateModal.value = true
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
      void loadKeys()
    }
  } catch {
    toast.error('创建失败，请稍后重试')
  } finally {
    creating.value = false
  }
}

async function handleDelete(id: number, name: string) {
  if (isLeaving(id)) return
  const { confirm } = useConfirm()
  const ok = await confirm(`确定要撤销 "${name}" 吗？撤销后使用该 Key 的应用将立即无法访问。`)
  if (!ok) return

  leavingIds.value = [...leavingIds.value, id]
  try {
    await apiKeyApi.delete(id)
    toast.success('已撤销')
    await loadKeys()
  } catch {
    toast.error('撤销失败，请稍后重试')
  } finally {
    leavingIds.value = leavingIds.value.filter(item => item !== id)
  }
}

async function toggleActive(key: ApiKey) {
  if (isToggling(key.id)) return
  togglingIds.value = [...togglingIds.value, key.id]
  try {
    await apiKeyApi.update(key.id, { is_active: !key.is_active })
    toast.success(key.is_active ? '已禁用' : '已启用')
    await loadKeys()
  } catch {
    toast.error('状态切换失败，请稍后重试')
  } finally {
    togglingIds.value = togglingIds.value.filter(item => item !== key.id)
  }
}

const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const copiedPrefixId = ref<number | null>(null)
let prefixTimer: ReturnType<typeof setTimeout> | null = null

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

async function copyPrefix(key: ApiKey) {
  try {
    if (!navigator.clipboard) throw new Error('clipboard unavailable')
    await navigator.clipboard.writeText(key.key_prefix)
    toast.success('前缀已复制')
    copiedPrefixId.value = key.id
    if (prefixTimer) clearTimeout(prefixTimer)
    prefixTimer = setTimeout(() => {
      copiedPrefixId.value = null
      prefixTimer = null
    }, 1600)
  } catch {
    toast.warning('复制失败，请手动选择前缀')
  }
}

watch(showKeyModal, (visible) => {
  if (!visible) copied.value = false
})

onMounted(() => {
  void loadKeys()
})

onBeforeUnmount(() => {
  if (copiedTimer) clearTimeout(copiedTimer)
  if (prefixTimer) clearTimeout(prefixTimer)
  copiedTimer = null
  prefixTimer = null
})
</script>

<template>
  <AdminPanel title="API 密钥" desc="用于第三方应用接入；完整密钥仅在创建时展示一次，请妥善保存。">
    <template #actions>
      <a-button type="primary" size="small" @click="openCreate">
        <IconPlus /> 新建密钥
      </a-button>
    </template>

    <template #toolbar>
      <AdminDataToolbar :summary="summary" :loading="loading" refreshable @refresh="loadKeys" />
    </template>

    <AdminSkeleton v-if="showSkeleton" variant="cards" :rows="4" label="正在加载密钥" />

    <AdminEmpty
      v-else-if="keys.length === 0"
      title="还没有 API 密钥"
      desc="创建一把密钥，第三方应用带上 X-API-Key 请求头即可调用内容接口。"
    >
      <template #actions>
        <a-button type="primary" size="small" @click="openCreate">
          <IconPlus /> 新建密钥
        </a-button>
      </template>
    </AdminEmpty>

    <div v-else class="adm-board key-board">
      <div class="adm-board-main">
        <div class="adm-stat-grid">
          <div class="adm-stat">
            <span class="adm-stat-label">密钥总数</span>
            <span class="adm-stat-value">{{ keys.length }}</span>
            <span class="adm-stat-hint">全部 API 密钥</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">启用中</span>
            <span class="adm-stat-value">{{ activeCount }}</span>
            <span class="adm-stat-hint">{{ inactiveCount ? `${inactiveCount} 个已停用` : '全部可用' }}</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">最近使用</span>
            <span class="adm-stat-value" :title="latestUsedTitle">{{ latestUsedText }}</span>
            <span class="adm-stat-hint">{{ latestUsed ? latestUsed.name : '还没有调用记录' }}</span>
          </div>
        </div>

        <div class="key-list adm-card-grid is-wide">
          <article
            v-for="key in keys"
            :key="key.id"
            class="adm-card key-card"
            :class="{ 'is-off': !key.is_active, 'is-leaving': isLeaving(key.id) }"
          >
            <div class="adm-card-content">
              <header class="key-head">
                <div class="key-ident">
                  <h3 class="adm-card-title key-name" :title="key.name">{{ key.name }}</h3>
                  <AdminStatus
                    :type="key.is_active ? 'success' : 'neutral'"
                    :label="key.is_active ? '启用' : '已停用'"
                  />
                </div>
                <a-switch
                  class="key-switch"
                  size="small"
                  :model-value="key.is_active"
                  :loading="isToggling(key.id)"
                  :aria-label="`${key.is_active ? '停用' : '启用'} ${key.name}`"
                  @change="toggleActive(key)"
                />
              </header>

              <code class="admin-mono key-prefix" :title="`${key.key_prefix}…`">{{ key.key_prefix }}…</code>

              <ul v-if="key.permissions.length" class="key-perms" aria-label="权限">
                <li
                  v-for="p in key.permissions"
                  :key="p"
                  class="key-perm-chip"
                  :class="permTone(p)"
                >{{ permLabel(p) }}</li>
              </ul>
              <p v-else class="key-perms-empty">未分配权限</p>

              <dl class="adm-kv key-meta">
                <div class="adm-kv-row">
                  <dt>最后使用</dt>
                  <dd :title="key.last_used_at ? fullTime(key.last_used_at) : '尚无调用记录'">
                    {{ key.last_used_at ? relativeTime(key.last_used_at) : '从未使用' }}
                  </dd>
                </div>
                <div class="adm-kv-row">
                  <dt>创建时间</dt>
                  <dd :title="fullTime(key.created_at)">{{ fullTime(key.created_at) }}</dd>
                </div>
              </dl>
            </div>

            <footer class="adm-card-foot key-foot">
              <a-button class="key-prefix-copy" size="small" @click="copyPrefix(key)">
                <IconCheck v-if="copiedPrefixId === key.id" />
                <IconCopy v-else />
                {{ copiedPrefixId === key.id ? '已复制' : '复制前缀' }}
              </a-button>
              <a-button
                class="admin-icon-btn is-danger key-revoke is-push"
                type="text"
                size="small"
                :disabled="isLeaving(key.id)"
                aria-label="撤销"
                title="撤销"
                @click="handleDelete(key.id, key.name)"
              >
                <IconDelete />
              </a-button>
            </footer>
          </article>
        </div>
      </div>

      <aside class="adm-board-aside" aria-label="接入说明">
        <section class="adm-card key-side">
          <div class="adm-card-content">
            <h3 class="key-side-title">怎么接</h3>
            <p class="key-side-desc">第三方应用在请求头里带上这把密钥，就可以调用内容接口：</p>
            <pre class="key-side-code"><code>X-API-Key: xq_xxx</code></pre>
            <p class="key-side-note">
              权限按需勾选：<strong>上传</strong>写内容、<strong>读取</strong>查内容、<strong>删除</strong>移除内容，没勾的接口会直接拒绝。
            </p>
            <p class="key-side-foot">完整密钥只在创建时显示一次，列表里只保留前缀。</p>
          </div>
        </section>
      </aside>
    </div>

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

:deep(.admin-panel-toolbar .admin-data-toolbar) {
  padding: 0;
}

.key-head {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-sm, 8px);
  min-width: 0;
}

.key-ident {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.key-name {
  margin: 0;
  @include ellipsis;
}

.key-switch {
  flex: 0 0 auto;
  margin-top: 2px;
}

.key-prefix {
  display: block;
  max-width: 100%;
  overflow: hidden;
  color: var(--adm-muted, var(--admin-text-2));
  text-overflow: ellipsis;
  white-space: nowrap;
  user-select: all;
}

.key-perms {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.key-perm-chip {
  padding: 2px 9px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;

  &.is-upload {
    border-color: transparent;
    background: var(--adm-accent-soft, var(--admin-fill));
    color: var(--adm-accent, var(--admin-primary));
  }

  &.is-delete {
    border-color: transparent;
    color: var(--adm-danger, rgb(var(--danger-6)));
  }
}

.key-perms-empty {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

.key-meta {
  margin-top: 2px;
}

.key-card.is-off .key-name {
  color: var(--adm-muted, var(--admin-text-2));
}

.key-foot :deep(.admin-icon-btn.is-danger) {
  flex: 0 0 auto;
  width: var(--adm-touch, 36px);
  height: var(--adm-touch, 36px);
}

.key-side-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.key-side-desc {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.7;
}

.key-side-code {
  overflow-x: auto;
  margin: 0;
  padding: 10px 12px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-ink, var(--admin-text));
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  white-space: nowrap;

  code {
    font: inherit;
  }
}

.key-side-note {
  margin: 0;
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;

  strong {
    color: var(--adm-ink, var(--admin-text));
    font-weight: 600;
  }
}

.key-side-foot {
  margin: 0;
  padding-top: var(--adm-gap-sm, 8px);
  border-top: 1px solid var(--admin-border-soft);
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
}

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

@media (max-width: 768px) {
  .key-board .key-list {
    grid-template-columns: minmax(0, 1fr);
  }

  .key-foot :deep(.arco-btn) {
    min-height: var(--adm-touch, 44px);
  }

  .key-prefix-copy {
    flex: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .key-copy-icon {
    animation: none;
  }
}
</style>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import type { AdminLogEntry, AdminLogLevel } from '@/types'
import AdminPanel from './AdminPanel.vue'
import { IconCode, IconCopy, IconPause, IconPlayArrow, IconRefresh } from '@arco-design/web-vue/es/icon'

type NormalizedLogEntry = Omit<AdminLogEntry, 'level'> & { level: AdminLogLevel }

const POLL_INTERVAL_MS = 5000
const LOG_LIMIT = 200
const levels: Array<{ label: string; value: 'all' | AdminLogLevel }> = [
  { label: '全部级别', value: 'all' },
  { label: '错误', value: 'error' },
  { label: '警告', value: 'warn' },
  { label: '信息', value: 'info' },
]

const level = ref<'all' | AdminLogLevel>('all')
const entries = ref<NormalizedLogEntry[]>([])
const loading = ref(false)
const refreshing = ref(false)
const autoRefresh = ref(true)
const errorMessage = ref('')
const lastUpdated = ref<string | null>(null)
const terminal = ref<HTMLElement | null>(null)
const nextCursor = ref<number | undefined>()
let pollTimer: ReturnType<typeof setTimeout> | undefined
let requestController: AbortController | undefined
let disposed = false

const connectionLabel = computed(() => {
  if (loading.value && entries.value.length === 0) return '连接中'
  if (errorMessage.value) return '连接异常'
  return autoRefresh.value ? '实时跟随' : '已暂停'
})

const connectionState = computed(() => {
  if (errorMessage.value) return 'error'
  if (loading.value && entries.value.length === 0) return 'loading'
  return autoRefresh.value ? 'live' : 'paused'
})

const visibleEntries = computed(() => entries.value)

function clearPoll() {
  if (pollTimer !== undefined) {
    clearTimeout(pollTimer)
    pollTimer = undefined
  }
}

function schedulePoll() {
  clearPoll()
  if (!autoRefresh.value || disposed) return
  pollTimer = setTimeout(async () => {
    pollTimer = undefined
    await load(true)
    schedulePoll()
  }, POLL_INTERVAL_MS)
}

async function keepTerminalAtEnd() {
  if (!autoRefresh.value) return
  await nextTick()
  const el = terminal.value
  if (el) el.scrollTop = el.scrollHeight
}

function normalizeLevel(value: string): AdminLogLevel {
  const normalized = value.toLowerCase()
  if (normalized === 'error' || normalized === 'erro') return 'error'
  if (normalized === 'warn' || normalized === 'warning') return 'warn'
  if (normalized === 'debug') return 'debug'
  return 'info'
}

function normalizeEntry(entry: AdminLogEntry): NormalizedLogEntry {
  return {
    ...entry,
    level: normalizeLevel(entry.level),
    // line is the already formatted, server-side redacted record. Keep message
    // as the readable fallback for older responses without line.
    message: entry.line || entry.message,
  }
}

async function load(silent = false) {
  requestController?.abort()
  const controller = new AbortController()
  requestController = controller
  if (silent) refreshing.value = true
  else loading.value = true
  errorMessage.value = ''
  try {
    const response = await adminApi.logs({
      limit: LOG_LIMIT,
      level: level.value === 'all' ? undefined : level.value,
      after: silent ? nextCursor.value : undefined,
    }, controller.signal)
    if (controller.signal.aborted || disposed) return
    const incoming = response.data.entries.map(normalizeEntry)
    if (silent && nextCursor.value !== undefined) {
      entries.value = [...entries.value, ...incoming].slice(-LOG_LIMIT)
    } else {
      entries.value = incoming.slice(-LOG_LIMIT)
    }
    nextCursor.value = response.data.next_cursor
    lastUpdated.value = response.data.server_time || new Date().toISOString()
    await keepTerminalAtEnd()
  } catch (error: unknown) {
    if (controller.signal.aborted || disposed) return
    // 保留旧日志，让临时网络问题不会把排障现场清空；错误状态在终端顶部明确提示。
    errorMessage.value = error instanceof Error ? error.message : '日志读取失败'
  } finally {
    if (requestController === controller) {
      loading.value = false
      refreshing.value = false
      requestController = undefined
    }
  }
}

function setAutoRefresh(enabled: boolean) {
  autoRefresh.value = enabled
  if (enabled) {
    void keepTerminalAtEnd()
    schedulePoll()
  } else {
    clearPoll()
  }
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', { hour12: false })
}

function levelLabel(value: AdminLogLevel) {
  return value === 'error' ? 'ERRO' : value === 'warn' ? 'WARN' : value === 'debug' ? 'DEBU' : 'INFO'
}

function entryKey(entry: AdminLogEntry, index: number) {
  return entry.id ?? `${entry.time}-${index}`
}

async function copyLogs() {
  const text = visibleEntries.value
    .map((entry) => entry.line || `[${entry.time} ${levelLabel(entry.level)}] ${entry.message}`)
    .join('\n')
  if (!text) {
    toast.warning('当前没有可复制的日志')
    return
  }
  try {
    await navigator.clipboard?.writeText(text)
    toast.success('日志已复制')
  } catch {
    toast.warning('剪贴板不可用，请直接选择日志文本')
  }
}

watch(level, () => {
  nextCursor.value = undefined
  entries.value = []
  void load()
  schedulePoll()
})

onMounted(() => {
  void load()
  schedulePoll()
})

onBeforeUnmount(() => {
  disposed = true
  requestController?.abort()
  requestController = undefined
  clearPoll()
})
</script>

<template>
  <AdminPanel title="运行日志" desc="查看服务端最近的运行记录，便于定位接口、任务与依赖状态。">
    <template #actions>
      <a-select v-model="level" size="small" class="logs-level-select" aria-label="日志级别" :options="levels" />
      <a-button
        class="admin-icon-btn"
        type="text"
        size="small"
        :loading="loading || refreshing"
        aria-label="刷新日志"
        @click="load()"
      >
        <IconRefresh />
      </a-button>
      <a-button
        class="logs-follow-btn"
        :class="{ 'is-paused': !autoRefresh }"
        type="text"
        size="small"
        :aria-pressed="autoRefresh"
        @click="setAutoRefresh(!autoRefresh)"
      >
        <IconPause v-if="autoRefresh" />
        <IconPlayArrow v-else />
        {{ autoRefresh ? '暂停跟随' : '继续跟随' }}
      </a-button>
      <a-button class="admin-icon-btn" type="text" size="small" aria-label="复制日志" @click="copyLogs">
        <IconCopy />
      </a-button>
    </template>

    <div class="admin-logs-terminal">
      <div class="logs-toolbar" role="status" aria-live="polite">
        <div class="logs-status">
          <span class="logs-status-dot" :class="`is-${connectionState}`" aria-hidden="true" />
          <span>{{ connectionLabel }}</span>
          <span class="logs-count">{{ visibleEntries.length }} 条</span>
        </div>
        <span v-if="lastUpdated" class="logs-updated">更新于 {{ formatTime(lastUpdated) }}</span>
      </div>

      <div ref="terminal" class="logs-terminal-window" tabindex="0" aria-label="服务端运行日志">
        <div class="logs-terminal-head" aria-hidden="true">
          <span class="logs-terminal-dots"><i /><i /><i /></span>
          <span class="logs-terminal-title"><IconCode /> server / logs</span>
          <span class="logs-terminal-limit">tail {{ LOG_LIMIT }}</span>
        </div>

        <div v-if="loading && !visibleEntries.length" class="logs-terminal-empty" role="status">
          正在连接日志流…
        </div>
        <div v-else-if="!visibleEntries.length && !errorMessage" class="logs-terminal-empty">
          暂无日志记录
        </div>
        <div v-else class="logs-lines">
          <div v-for="(entry, index) in visibleEntries" :key="entryKey(entry, index)" class="logs-line">
            <span class="logs-line-marker" :class="`is-${entry.level}`" aria-hidden="true">●</span>
            <span class="logs-line-message">{{ entry.line || entry.message }}</span>
          </div>
        </div>
      </div>

      <p v-if="errorMessage" class="logs-error" role="alert">
        {{ errorMessage }}。保留已加载记录，可点击刷新重试。
      </p>
      <p class="logs-hint">日志仅保留在服务进程内存中，并只向管理员开放；自动刷新每 5 秒拉取最近记录。</p>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
.logs-level-select { min-width: 104px; }

.logs-follow-btn {
  color: var(--admin-primary);

  &.is-paused { color: var(--admin-text-2); }
}

.admin-logs-terminal {
  padding: 20px 24px 24px;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 28px;
  margin-bottom: 12px;
  color: var(--admin-text-2);
  font-size: 12px;
}

.logs-status,
.logs-updated {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

.logs-count { color: var(--admin-text-3); font-variant-numeric: tabular-nums; }
.logs-updated { color: var(--admin-text-3); font-variant-numeric: tabular-nums; }

.logs-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--admin-text-3);

  &.is-live { background: rgb(var(--success-6)); box-shadow: 0 0 0 3px color-mix(in srgb, rgb(var(--success-6)) 16%, transparent); }
  &.is-loading { background: rgb(var(--warning-6)); box-shadow: 0 0 0 3px color-mix(in srgb, rgb(var(--warning-6)) 16%, transparent); }
  &.is-error { background: var(--admin-danger); box-shadow: 0 0 0 3px var(--admin-danger-soft); }
}

.logs-terminal-window {
  overflow: auto;
  max-height: min(620px, 62vh);
  min-height: 300px;
  border: 1px solid color-mix(in srgb, var(--creative-ink) 22%, var(--creative-line));
  border-radius: 14px;
  background: color-mix(in srgb, var(--creative-ink) 96%, black);
  color: var(--creative-paper);
  box-shadow: 0 14px 32px color-mix(in srgb, var(--creative-ink) 14%, transparent);
  scrollbar-color: color-mix(in srgb, var(--creative-accent) 56%, transparent) transparent;
}

.logs-terminal-head {
  position: sticky;
  top: 0;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding: 0 14px;
  border-bottom: 1px solid color-mix(in srgb, var(--creative-paper) 12%, transparent);
  background: color-mix(in srgb, var(--creative-ink) 92%, black);
  color: color-mix(in srgb, var(--creative-paper) 72%, transparent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 11px;
}

.logs-terminal-dots { display: inline-flex; gap: 5px; }
.logs-terminal-dots i { width: 7px; height: 7px; border-radius: 50%; background: color-mix(in srgb, var(--creative-paper) 32%, transparent); }
.logs-terminal-dots i:first-child { background: color-mix(in srgb, var(--creative-accent) 92%, white); }
.logs-terminal-title { display: inline-flex; align-items: center; gap: 6px; color: var(--creative-paper); }
.logs-terminal-limit { margin-left: auto; color: color-mix(in srgb, var(--creative-paper) 48%, transparent); }

.logs-lines {
  padding: 10px 14px 16px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.7;
}

.logs-line {
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
  padding: 2px 0;
}

.logs-line-marker { color: color-mix(in srgb, var(--creative-paper) 38%, transparent); font-size: 8px; }
.logs-line-marker.is-error { color: #ff9da9; }
.logs-line-marker.is-warn { color: #f5ca83; }
.logs-line-marker.is-info { color: #a8d8c0; }
.logs-line-marker.is-debug { color: #bdaeca; }
.logs-line-message { color: var(--creative-paper); white-space: pre-wrap; overflow-wrap: anywhere; }

.logs-terminal-empty {
  display: grid;
  min-height: 260px;
  place-items: center;
  padding: 24px;
  color: color-mix(in srgb, var(--creative-paper) 62%, transparent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.logs-error {
  margin: 12px 0 0;
  color: var(--admin-danger);
  font-size: 12px;
}

.logs-hint {
  margin: 12px 0 0;
  color: var(--admin-text-3);
  font-size: 12px;
  line-height: 1.7;
}

@media (max-width: 768px) {
  .admin-logs-terminal { padding: 16px; }
  .logs-toolbar { align-items: flex-start; flex-direction: column; }
  .logs-line { gap: 8px; }
  .logs-terminal-window { max-height: 58vh; }
}
</style>

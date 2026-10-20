<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { adminApi } from '@/api'
import { toast } from '@/composables/useToast'
import { prefersReducedMotion } from '@/utils'
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
const pageHidden = ref(false)
const reduceMotion = ref(false)
let motionQuery: MediaQueryList | undefined
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

const breathPaused = computed(() => pageHidden.value || reduceMotion.value)

const visibleEntries = computed(() => entries.value)

function syncHidden() {
  pageHidden.value = document.visibilityState === 'hidden'
}

function syncMotion(event?: MediaQueryListEvent) {
  reduceMotion.value = event ? event.matches : prefersReducedMotion()
}

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
  syncHidden()
  syncMotion()
  document.addEventListener('visibilitychange', syncHidden)
  motionQuery = globalThis.matchMedia?.('(prefers-reduced-motion: reduce)')
  motionQuery?.addEventListener?.('change', syncMotion)
  void load()
  schedulePoll()
})

onBeforeUnmount(() => {
  disposed = true
  requestController?.abort()
  requestController = undefined
  clearPoll()
  document.removeEventListener('visibilitychange', syncHidden)
  motionQuery?.removeEventListener?.('change', syncMotion)
  motionQuery = undefined
})
</script>

<template>
  <AdminPanel flow title="运行日志" desc="查看服务端最近的运行记录，便于定位接口、任务与依赖状态。">
    <template #actions>
      <span class="logs-live" :class="`is-${connectionState}`">
        <i class="logs-live-dot" :class="{ 'is-breath': connectionState === 'live', 'is-paused': breathPaused }" aria-hidden="true" />
        <span class="logs-live-text">{{ connectionLabel }}</span>
      </span>
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
        <span class="logs-count">{{ visibleEntries.length }} 条</span>
        <span class="logs-toolbar-sep" aria-hidden="true">·</span>
        <span class="logs-toolbar-tip">每 5 秒增量拉取，仅保留最近 {{ LOG_LIMIT }} 条</span>
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
  color: var(--adm-accent, var(--admin-primary));

  &.is-paused { color: var(--adm-muted, var(--admin-text-2)); }
}

.logs-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.6;
  color: var(--adm-muted, var(--admin-text-3));
  white-space: nowrap;

  &.is-live { color: var(--adm-success, rgb(var(--success-6))); }
  &.is-loading { color: var(--adm-warning, rgb(var(--warning-6))); }
  &.is-error { color: var(--adm-danger, var(--admin-danger)); }
}

.logs-live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex: none;
  background: currentColor;
}

.logs-live-dot.is-breath {
  animation: adm-live-breath 2s var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)) infinite;
}

.logs-live-dot.is-paused { animation-play-state: paused; }

@keyframes adm-live-breath {
  0%,
  100% {
    opacity: 1;
    box-shadow: 0 0 0 0 currentColor;
  }

  50% {
    opacity: .55;
    box-shadow: 0 0 0 4px transparent;
  }
}

.admin-logs-terminal {
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-xl, 24px) var(--adm-gap-xl, 24px);
}

.logs-toolbar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  min-height: 28px;
  margin-bottom: var(--adm-gap-md, 12px);
  padding: 6px 0;
  background: var(--adm-surface, var(--admin-surface));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
}

.logs-count {
  color: var(--adm-ink, var(--admin-text));
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
}

.logs-toolbar-sep { color: var(--adm-line, var(--admin-border-soft)); }

.logs-toolbar-tip {
  color: var(--adm-muted, var(--admin-text-3));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.logs-updated {
  margin-left: auto;
  color: var(--adm-muted, var(--admin-text-3));
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  white-space: nowrap;
}

.logs-terminal-window {
  overflow: auto;
  max-height: min(620px, 62vh);
  min-height: 300px;
  border: 1px solid color-mix(in srgb, var(--creative-ink) 22%, var(--creative-line));
  border-radius: var(--adm-r-control, 14px);
  background: color-mix(in srgb, var(--creative-ink) 96%, black);
  color: var(--creative-paper);
  box-shadow: 0 14px 32px color-mix(in srgb, var(--creative-ink) 14%, transparent);
  scrollbar-color: color-mix(in srgb, var(--creative-accent) 56%, transparent) transparent;

  &:focus-visible { outline: 2px solid var(--adm-accent, var(--admin-primary)); outline-offset: 2px; }
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
  animation: adm-line-in var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(.22, 1, .36, 1)) both;
}

@keyframes adm-line-in {
  from { opacity: 0; transform: translateY(2px); }
  to { opacity: 1; transform: none; }
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
  padding: var(--adm-gap-xl, 24px);
  color: color-mix(in srgb, var(--creative-paper) 62%, transparent);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}

.logs-error {
  margin: var(--adm-gap-md, 12px) 0 0;
  color: var(--adm-danger, var(--admin-danger));
  font-size: var(--adm-font-xs, 12px);
}

.logs-hint {
  margin: var(--adm-gap-md, 12px) 0 0;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
}

@media (prefers-reduced-motion: reduce) {
  .logs-live-dot.is-breath,
  .logs-line { animation: none; }
}

@media (max-width: 768px) {
  .admin-logs-terminal { padding: var(--adm-gap-md, 12px); }
  .logs-toolbar { flex-wrap: wrap; }
  .logs-toolbar-tip { display: none; }
  .logs-updated { margin-left: 0; }
  .logs-live { order: -1; }
  .logs-line { gap: 8px; }
  .logs-terminal-window { max-height: 58vh; }
}
</style>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Tooltip } from '@arco-design/web-vue'
import {
  IconCheck,
  IconClockCircle,
  IconExclamationCircle,
  IconEye,
  IconFile,
  IconLink,
  IconRefresh,
  IconTag,
} from '@arco-design/web-vue/es/icon'
import { adminApi } from '@/api'
import { formatTime } from '@/utils'
import { useAdminStore } from '@/stores/admin'
import { useCountUp } from '@/composables/useCountUp'
import type { DashboardStats } from '@/types'
import MediaImage from '@/components/MediaImage.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminStatus from './AdminStatus.vue'

const emit = defineEmits<{ select: [key: string] }>()

const admin = useAdminStore()
const loading = ref(true)
const stats = ref<DashboardStats | null>(null)

const numberFormat = new Intl.NumberFormat('zh-CN')

function formatNumber(value: number) {
  return numberFormat.format(value)
}

function pct(count: number, total: number) {
  return total > 0 ? Math.round((count / total) * 100) : 0
}

async function load(force = false) {
  loading.value = true
  try {
    stats.value = (await adminApi.getDashboard(force)).data
  } catch {
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void load()
})

const pendingItems = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'pending', label: '待审内容', count: s.content.pending, icon: IconFile },
    { key: 'claims', label: '待处理认领', count: s.claims.pending, icon: IconLink },
    { key: 'reports', label: '未处理举报', count: s.reports.unhandled, icon: IconExclamationCircle },
  ]
})

const pendingTotal = computed(() => pendingItems.value.reduce((sum, item) => sum + item.count, 0))

const totalContents = computed(() => stats.value?.content.total ?? 0)
const pendingContents = computed(() => stats.value?.content.pending ?? 0)
const totalUsers = computed(() => stats.value?.users.total ?? 0)
const unhandledReports = computed(() => stats.value?.reports.unhandled ?? 0)
const totalPolls = computed(() => stats.value?.polls.total ?? 0)
const totalViews = computed(() => stats.value?.views ?? 0)

const countedContents = useCountUp(totalContents)
const countedPending = useCountUp(pendingContents)
const countedUsers = useCountUp(totalUsers)
const countedReports = useCountUp(unhandledReports)
const countedPolls = useCountUp(totalPolls)
const countedViews = useCountUp(totalViews)

const kpiCards = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'all', label: '内容总数', count: countedContents.value, sub: '今日 +' + formatNumber(s.content.today), alert: false },
    { key: 'pending', label: '待审核', count: countedPending.value, sub: '需要处理', alert: true },
    { key: 'users', label: '用户总数', count: countedUsers.value, sub: '管理员 ' + formatNumber(s.users.admins), alert: false },
    { key: 'reports', label: '未处理举报', count: countedReports.value, sub: '累计举报 ' + formatNumber(s.reports.total), alert: true },
    { key: 'polls', label: '投票数', count: countedPolls.value, sub: '累计票数 ' + formatNumber(s.polls.votes), alert: false },
    { key: '', label: '总浏览量', count: countedViews.value, sub: '全部内容累计', alert: false },
  ]
})

const auditRows = computed(() => {
  const content = stats.value?.content
  if (!content) return []
  return [
    { label: '已通过', count: content.approved, status: 'success' },
    { label: '待审核', count: content.pending, status: 'warning' },
    { label: '已拒绝', count: content.rejected, status: 'danger' },
  ]
})

const auditTotal = computed(() => auditRows.value.reduce((sum, row) => sum + row.count, 0))
const auditSegments = computed(() => auditRows.value.filter((row) => row.count > 0))
const auditGap = computed(() => Math.max(0, (stats.value?.content.total ?? 0) - auditTotal.value))
const auditSummary = computed(
  () => '内容审核分布：' + auditRows.value.map((row) => row.label + ' ' + row.count + ' 篇').join('，'),
)

const maxTag = computed(() => Math.max(1, ...(stats.value?.topTags.map((tag) => tag.count) ?? [])))

function tagRatio(count: number) {
  return Math.max(count / maxTag.value, 0.04)
}

function auditType(status?: string): 'success' | 'warning' | 'danger' {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  return 'danger'
}

function auditLabel(status?: string) {
  if (status === 'approved') return '已通过'
  if (status === 'pending') return '待审核'
  return '已拒绝'
}

function go(key: string) {
  if (key) emit('select', key)
}
</script>

<template>
  <AdminPanel title="仪表盘" desc="站点数据总览" flat stagger>
    <template #actions>
      <Tooltip title="刷新">
        <a-button
          class="admin-icon-btn adm-dash-refresh"
          type="text"
          size="small"
          :loading="loading"
          aria-label="刷新仪表盘数据"
          @click="load(true)"
        >
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <AdminSkeleton v-if="loading && !stats" variant="cards" :rows="6" label="正在加载仪表盘数据" />

    <AdminEmpty v-else-if="!stats" title="暂无数据" desc="站点统计暂时取不到，点刷新再试一次。">
      <template #actions>
        <a-button type="primary" size="small" @click="load(true)">重新加载</a-button>
      </template>
    </AdminEmpty>

    <template v-else>
      <section class="adm-pending" aria-label="待处理事项">
        <div class="adm-pending-head">
          <span class="adm-pending-title">待处理</span>
          <span class="adm-pending-total" :class="{ 'is-clear': pendingTotal === 0 }">
            {{ pendingTotal > 0 ? '共 ' + formatNumber(pendingTotal) + ' 项' : '全部处理完了' }}
          </span>
        </div>
        <div class="adm-pending-list">
          <button
            v-for="item in pendingItems"
            :key="item.key"
            type="button"
            class="adm-pending-item"
            :class="{ 'is-clear': item.count === 0 }"
            :aria-label="item.label + ' ' + item.count + ' 项，前往' + item.label"
            @click="go(item.key)"
          >
            <span class="adm-pending-icon" aria-hidden="true">
              <component :is="item.count > 0 ? item.icon : IconCheck" />
            </span>
            <span class="adm-pending-label">{{ item.label }}</span>
            <strong
              :key="item.key + '-' + item.count"
              class="adm-pending-count"
            >{{ formatNumber(item.count) }}</strong>
          </button>
        </div>
      </section>

      <div class="adm-kpi-grid">
        <button
          v-for="(card, index) in kpiCards"
          :key="card.label"
          type="button"
          class="adm-kpi"
          :class="{ 'is-alert': card.alert && card.count > 0 }"
          :disabled="!card.key"
          :title="card.key ? undefined : '暂无对应页签'"
          :style="{ '--i': String(index) }"
          @click="go(card.key)"
        >
          <span class="adm-kpi-label">{{ card.label }}</span>
          <strong class="adm-kpi-value">{{ formatNumber(card.count) }}</strong>
          <span class="adm-kpi-sub">{{ card.sub }}</span>
          <span v-if="card.key" class="adm-kpi-arrow" aria-hidden="true">↗</span>
        </button>
      </div>

      <div class="adm-dash-grid">
        <section class="adm-card" aria-label="内容审核状态">
          <header class="adm-card-head">
            <div class="adm-card-head-text">
              <h3>内容审核状态</h3>
              <p>已发布内容与待处理队列</p>
            </div>
            <span class="adm-card-note">{{ formatNumber(stats.content.total) }} 篇</span>
          </header>

          <div v-if="auditTotal > 0" class="adm-audit-bar" role="img" :aria-label="auditSummary">
            <span
              v-for="row in auditSegments"
              :key="row.label"
              class="adm-audit-seg"
              :class="'is-' + row.status"
              :style="{ flexGrow: row.count }"
            ></span>
            <span v-if="auditGap > 0" class="adm-audit-seg is-rest" :style="{ flexGrow: auditGap }"></span>
          </div>
          <p v-else class="adm-audit-none">暂无审核数据</p>

          <ul class="adm-audit-legend">
            <li v-for="row in auditRows" :key="row.label" class="adm-audit-legend-item">
              <span class="adm-audit-dot" :class="'is-' + row.status" aria-hidden="true"></span>
              <span class="adm-audit-legend-label">{{ row.label }}</span>
              <strong>{{ formatNumber(row.count) }}</strong>
              <span class="adm-audit-legend-pct">{{ pct(row.count, stats.content.total) }}%</span>
            </li>
          </ul>

          <p class="adm-card-foot">
            <span>今日新增内容</span>
            <strong>+{{ formatNumber(stats.content.today) }}</strong>
          </p>
        </section>

        <section class="adm-card" aria-label="热门标签">
          <header class="adm-card-head">
            <div class="adm-card-head-text">
              <h3>热门标签</h3>
              <p>按已通过内容的使用次数</p>
            </div>
            <span class="adm-card-note">Top 10</span>
          </header>

          <ol v-if="stats.topTags.length" class="adm-tag-list">
            <li v-for="(tag, index) in stats.topTags" :key="tag.tag" class="adm-tag-row">
              <span class="adm-tag-rank">{{ String(index + 1).padStart(2, '0') }}</span>
              <span class="adm-tag-name">
                <IconTag aria-hidden="true" />
                <span class="adm-tag-text">{{ tag.tag }}</span>
              </span>
              <span class="adm-tag-track" aria-hidden="true">
                <span class="adm-tag-fill" :style="{ '--ratio': tagRatio(tag.count) }"></span>
              </span>
              <strong class="adm-tag-count">{{ formatNumber(tag.count) }}</strong>
            </li>
          </ol>
          <AdminEmpty v-else compact title="暂无标签" desc="内容打好标签后这里会出现使用频次。" />
        </section>
      </div>

      <div class="adm-dash-grid">
        <section class="adm-card" aria-label="最新内容">
          <header class="adm-card-head">
            <div class="adm-card-head-text">
              <h3>最新内容</h3>
              <p>最近提交到站点的作品</p>
            </div>
            <span class="adm-card-note">{{ stats.recentContents.length }} 条</span>
          </header>

          <div v-if="stats.recentContents.length" class="adm-list">
            <button
              v-for="item in stats.recentContents"
              :key="item.id"
              type="button"
              class="adm-list-item adm-content-item"
              :aria-label="'查看内容：' + (item.title || '无标题')"
              @click="admin.openDrawer(item, 'view')"
            >
              <span class="adm-thumb">
                <MediaImage v-if="item.thumb" :src="item.thumb" alt="" />
                <IconFile v-else aria-hidden="true" />
              </span>
              <span class="adm-list-main">
                <span class="adm-list-title">{{ item.title || '无标题' }}</span>
                <span class="adm-list-meta">
                  <span>{{ item.user?.username }}</span>
                  <span class="adm-meta-item"><IconEye aria-hidden="true" />{{ formatNumber(item.view_count) }}</span>
                  <span class="adm-meta-item">
                    <IconClockCircle aria-hidden="true" />{{ formatTime(item.created_at ?? 0) }}
                  </span>
                </span>
              </span>
              <span class="adm-item-status">
                <AdminStatus :type="auditType(item.audit_status)" :label="auditLabel(item.audit_status)" />
              </span>
            </button>
          </div>
          <AdminEmpty v-else compact title="暂无内容" />
        </section>

        <section class="adm-card" aria-label="最新用户">
          <header class="adm-card-head">
            <div class="adm-card-head-text">
              <h3>最新用户</h3>
              <p>最近加入创作社区的成员</p>
            </div>
            <span class="adm-card-note">{{ stats.recentUsers.length }} 位</span>
          </header>

          <div v-if="stats.recentUsers.length" class="adm-list">
            <div v-for="user in stats.recentUsers" :key="user.id" class="adm-list-item">
              <UserAvatar :size="32" :email="user.email" :username="user.username" />
              <span class="adm-list-main">
                <span class="adm-list-title">{{ user.username }}</span>
                <span class="adm-list-meta">
                  <span>{{ user.email || '未绑定邮箱' }}</span>
                  <span class="adm-meta-item">
                    <IconClockCircle aria-hidden="true" />{{ formatTime(user.created_at ?? 0) }}
                  </span>
                </span>
              </span>
              <span class="adm-item-status is-column">
                <AdminStatus v-if="user.is_admin" type="info" label="管理员" />
                <AdminStatus v-if="user.is_banned" type="danger" label="已封禁" />
              </span>
            </div>
          </div>
          <AdminEmpty v-else compact title="暂无用户" />
        </section>
      </div>
    </template>
  </AdminPanel>
</template>

<style lang="scss" scoped>

.adm-pending {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px) var(--adm-gap-lg, 16px);
  padding: var(--adm-gap-md, 12px) var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-soft, var(--admin-fill));
}

.adm-pending-head {
  display: flex;
  align-items: baseline;
  flex: 0 0 auto;
  gap: var(--adm-gap-sm, 8px);
}

.adm-pending-title {
  font-family: var(--site-title-font);
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--adm-ink, var(--admin-text));
}

.adm-pending-total {
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-danger, var(--admin-danger));

  &.is-clear {
    color: var(--adm-muted);
  }
}

.adm-pending-list {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--adm-gap-sm, 8px);
  margin-left: auto;
}

.adm-pending-item {
  display: inline-flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  min-height: var(--adm-touch, 36px);
  padding: 0 var(--adm-gap-md, 12px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-surface, var(--admin-surface));
  color: var(--adm-ink, var(--admin-text));
  font-family: inherit;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.2;
  cursor: pointer;
  transition:
    border-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1)),
    background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1)),
    transform var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  &:hover {
    border-color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
  }

  &:active {
    transform: translateY(1px);
  }

  &:focus-visible {
    outline: 2px solid var(--adm-accent, var(--admin-primary));
    outline-offset: 2px;
  }

  &.is-clear {
    color: var(--adm-muted);
  }
}

.adm-pending-icon {
  display: inline-flex;
  font-size: var(--adm-font-base, 14px);
  color: var(--adm-muted);
}

.adm-pending-count {
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-danger, var(--admin-danger));
  animation: adm-count-bump 240ms var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));

  .adm-pending-item.is-clear & {
    color: var(--adm-muted);
  }
}

@keyframes adm-count-bump {
  0% {
    transform: scale(1);
  }

  45% {
    transform: scale(1.15);
  }

  100% {
    transform: scale(1);
  }
}

.adm-kpi-grid,
.adm-dash-grid {
  margin-top: var(--adm-gap-lg, 16px);
}

.adm-kpi-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: var(--adm-gap-md, 12px);
}

.adm-kpi {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  min-width: 0;
  min-height: calc(var(--adm-touch, 36px) * 2.8);
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
  color: var(--adm-ink, var(--admin-text));
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  animation: adm-rise var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1)) backwards;
  animation-delay: calc(var(--adm-stagger, 40ms) * var(--i, 0));
  transition:
    border-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1)),
    background-color var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  &:not(:disabled):hover {
    border-color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));

    .adm-kpi-value {
      color: var(--adm-accent, var(--admin-primary));
    }

    .adm-kpi-arrow {
      opacity: 1;
      transform: translateX(0);
    }
  }

  &:active:not(:disabled) {
    transform: translateY(1px);
  }

  &:focus-visible {
    outline: 2px solid var(--adm-accent, var(--admin-primary));
    outline-offset: -2px;
  }

  &:disabled {
    cursor: default;
    opacity: 1;
  }
}

.adm-kpi-label {
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.4;
  letter-spacing: 0.02em;
  color: var(--adm-muted);
}

.adm-kpi-value {
  margin-top: 4px;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-xl, 21px);
  font-weight: 600;
  line-height: 1.15;
  letter-spacing: -0.02em;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-ink, var(--admin-text));
  transition: color var(--adm-dur-fast, 120ms) linear;
}

.adm-kpi.is-alert .adm-kpi-value {
  color: var(--adm-danger, var(--admin-danger));
}

.adm-kpi-sub {
  margin-top: auto;
  padding-top: var(--adm-gap-sm, 8px);
  overflow: hidden;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.4;
  color: var(--adm-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-kpi-arrow {
  position: absolute;
  top: var(--adm-gap-md, 12px);
  right: var(--adm-gap-md, 12px);
  font-size: var(--adm-font-xs, 12px);
  line-height: 1;
  color: var(--adm-accent, var(--admin-primary));
  opacity: 0;
  transform: translateX(-2px);
  transition:
    opacity var(--adm-dur-fast, 120ms) linear,
    transform var(--adm-dur-fast, 120ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));
}

.adm-dash-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
  gap: var(--adm-gap-lg, 16px);
}

.adm-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
}

.adm-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
}

.adm-card-head-text {
  min-width: 0;

  h3 {
    margin: 0;
    font-family: var(--site-title-font);
    font-size: var(--adm-font-md, 15px);
    font-weight: 600;
    line-height: 1.3;
    letter-spacing: -0.01em;
    color: var(--adm-ink, var(--admin-text));
  }

  p {
    margin: 3px 0 0;
    font-size: var(--adm-font-xs, 12px);
    line-height: 1.5;
    color: var(--adm-muted);
  }
}

.adm-card-note {
  flex: 0 0 auto;
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-muted);
  white-space: nowrap;
}

.adm-audit-bar {
  display: flex;
  gap: 2px;
  height: 10px;
  margin-top: var(--adm-gap-lg, 16px);
  overflow: hidden;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
}

.adm-audit-seg {
  flex-basis: 0;
  min-width: 2px;
  transform-origin: left center;
  animation: adm-seg-grow var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1)) backwards;

  &.is-success {
    background: var(--adm-success, rgb(var(--success-6)));
  }

  &.is-warning {
    background: var(--adm-warning, rgb(var(--warning-6)));
  }

  &.is-danger {
    background: var(--adm-danger, var(--admin-danger));
  }

  &.is-rest {
    background: var(--adm-line, var(--admin-border-soft));
  }

  &:nth-child(2) {
    animation-delay: calc(var(--adm-stagger, 40ms) * 1);
  }

  &:nth-child(3) {
    animation-delay: calc(var(--adm-stagger, 40ms) * 2);
  }

  &:nth-child(4) {
    animation-delay: calc(var(--adm-stagger, 40ms) * 3);
  }
}

@keyframes adm-seg-grow {
  from {
    transform: scaleX(0);
  }

  to {
    transform: scaleX(1);
  }
}

.adm-audit-none {
  margin: var(--adm-gap-lg, 16px) 0 0;
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-md, 12px);
  border: 1px dashed var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 10px);
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.5;
  color: var(--adm-muted);
  text-align: center;
}

.adm-audit-legend {
  display: grid;
  gap: 6px;
  margin: var(--adm-gap-md, 12px) 0 0;
  padding: 0;
  list-style: none;
}

.adm-audit-legend-item {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr) auto 42px;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-muted);

  strong {
    font-weight: 600;
    font-variant-numeric: var(--adm-numeric, tabular-nums);
    color: var(--adm-ink, var(--admin-text));
    text-align: right;
  }
}

.adm-audit-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;

  &.is-success {
    background: var(--adm-success, rgb(var(--success-6)));
  }

  &.is-warning {
    background: var(--adm-warning, rgb(var(--warning-6)));
  }

  &.is-danger {
    background: var(--adm-danger, var(--admin-danger));
  }
}

.adm-audit-legend-label {
  min-width: 0;
  overflow: hidden;
  color: var(--adm-ink, var(--admin-text));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-audit-legend-pct {
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  text-align: right;
}

.adm-card-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  margin: auto 0 0;
  padding-top: var(--adm-gap-md, 12px);
  border-top: 1px solid var(--adm-line, var(--admin-border-soft));
  font-size: var(--adm-font-xs, 12px);
  color: var(--adm-muted);

  strong {
    font-weight: 600;
    font-variant-numeric: var(--adm-numeric, tabular-nums);
    color: var(--adm-ink, var(--admin-text));
  }
}

.adm-tag-list {
  display: grid;
  gap: 7px;
  margin: var(--adm-gap-md, 12px) 0 0;
  padding: 0;
  list-style: none;
}

.adm-tag-row {
  display: grid;
  grid-template-columns: 20px minmax(0, 96px) minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
}

.adm-tag-rank {
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-muted);
}

.adm-tag-name {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: var(--adm-font-sm, 13px);
  color: var(--adm-ink, var(--admin-text));

  :deep(svg) {
    flex: 0 0 auto;
    font-size: var(--adm-font-xs, 12px);
    color: var(--adm-muted);
  }
}

.adm-tag-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-tag-track {
  position: relative;
  display: block;
  height: 6px;
  overflow: hidden;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
}

.adm-tag-fill {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: inherit;
  background: var(--adm-accent, var(--admin-primary));
  opacity: 0.68;
  transform: scaleX(var(--ratio, 0));
  transform-origin: left center;
  transition: transform var(--adm-dur-slow, 360ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));
}

.adm-tag-count {
  min-width: 28px;
  font-size: var(--adm-font-xs, 12px);
  font-weight: 600;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-muted);
  text-align: right;
}

.adm-list {
  display: grid;
  margin-top: var(--adm-gap-md, 12px);
}

.adm-list-item {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  min-width: 0;
  padding: 8px;
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));

  &:last-child {
    border-bottom: 0;
  }
}

button.adm-list-item {
  border: 0;
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
  background: transparent;
  color: inherit;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  &:hover {
    background: var(--adm-accent-soft, var(--admin-primary-soft));
  }

  &:focus-visible {
    outline: 2px solid var(--adm-accent, var(--admin-primary));
    outline-offset: -2px;
  }

  &:last-child {
    border-bottom: 0;
  }
}

.adm-thumb {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 36px;
  overflow: hidden;
  border-radius: var(--adm-r-control, 10px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted);
  font-size: var(--adm-font-lg, 17px);

  :deep(.arco-image) {
    width: 100%;
    height: 100%;
  }

  :deep(img) {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.adm-list-main {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.adm-list-title {
  overflow: hidden;
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  line-height: 1.35;
  color: var(--adm-ink, var(--admin-text));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.adm-list-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.4;
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  color: var(--adm-muted);
}

.adm-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.adm-item-status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;

  &.is-column {
    flex-direction: column;
    align-items: flex-end;
    gap: 2px;
  }
}

@media (max-width: 820px) {
  .adm-kpi-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .adm-dash-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .adm-pending-list {
    margin-left: 0;
    width: 100%;
  }
}

@media (max-width: 768px) {
  .adm-kpi-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .adm-pending {
    padding: var(--adm-gap-md, 12px);
  }

  .adm-pending-item {
    flex: 1 1 auto;
    justify-content: space-between;
  }

  .adm-card {
    padding: var(--adm-gap-md, 12px);
  }

  .adm-tag-row {
    grid-template-columns: 20px minmax(0, 80px) minmax(0, 1fr) auto;
  }

  .adm-item-status.is-column {
    flex-direction: row;
    align-items: center;
  }
}

@media (prefers-reduced-motion: reduce) {
  .adm-kpi,
  .adm-audit-seg,
  .adm-pending-count {
    animation: none;
  }

  .adm-kpi,
  .adm-kpi-value,
  .adm-kpi-arrow,
  .adm-pending-item,
  .adm-tag-fill,
  button.adm-list-item {
    transition: none;
  }
}
</style>

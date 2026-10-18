<script setup lang="ts">
import UserAvatar from '@/components/UserAvatar.vue'
import { ref, computed, onMounted } from 'vue'
import { adminApi } from '@/api'
import { formatTime } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
import { useAdminStore } from '@/stores/admin'
import AdminPanel from './AdminPanel.vue'
import type { DashboardStats } from '@/types'
import { Tooltip } from '@arco-design/web-vue'
import { IconRefresh, IconFile, IconEye, IconTag, IconClockCircle } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()

const emit = defineEmits<{ select: [key: string] }>()

const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

function pct(n: number, total: number): number {
  if (!total) return 0
  return Math.round((n / total) * 100)
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat('zh-CN').format(value)
}

async function load(force = false) {
  loading.value = true
  try {
    stats.value = (await adminApi.getDashboard(force)).data
  } catch { /* request() 已统一 toast */ } finally {
    loading.value = false
  }
}

onMounted(load)

const statCards = computed(() => {
  const s = stats.value
  if (!s) return []
  return [
    { key: 'all', label: '内容总数', value: s.content.total, sub: `今日 +${s.content.today}` },
    { key: 'pending', label: '待审核', value: s.content.pending, sub: '需要处理' },
    { key: 'users', label: '用户总数', value: s.users.total, sub: `管理员 ${s.users.admins}` },
    { key: 'reports', label: '未处理举报', value: s.reports.unhandled, sub: `累计举报 ${s.reports.total}` },
    { key: 'polls', label: '投票数', value: s.polls.total, sub: `累计票数 ${s.polls.votes}` },
    { key: '', label: '总浏览量', value: s.views, sub: '全部内容累计' },
  ]
})

const auditRows = computed(() => {
  const content = stats.value?.content
  if (!content) return []
  return [
    { label: '已通过', count: content.approved, status: 'success' },
    { label: '待审核', count: content.pending, status: 'warning' },
    { label: '已拒绝', count: content.rejected, status: 'danger' },
  ] as const
})

function go(key: string) {
  if (key) emit('select', key)
}

const maxTag = computed(() => Math.max(1, ...(stats.value?.topTags.map((t) => t.count) ?? [1])))
</script>

<template>
  <AdminPanel title="仪表盘" desc="站点数据总览" flat>
    <template #actions>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="loading" @click="load(true)">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="loading" class="dash-spin">
      <div v-if="!stats" class="dashboard-empty">
        <a-empty description="暂无数据">
          <a-button type="primary" size="small" @click="load(true)">重新加载</a-button>
        </a-empty>
      </div>

      <div v-else class="dashboard">
        <div class="dash-stat-grid" aria-label="站点核心数据">
          <button
            v-for="card in statCards"
            :key="card.label"
            type="button"
            class="dash-stat"
            :disabled="!card.key"
            @click="go(card.key)"
          >
            <span class="dash-stat-label">{{ card.label }}</span>
            <strong>{{ formatNumber(card.value) }}</strong>
            <span class="dash-stat-sub">{{ card.sub }}</span>
          </button>
        </div>

        <div class="dashboard-grid dashboard-overview">
          <section class="dash-section">
            <header class="dash-section-head">
              <div>
                <h3>内容审核状态</h3>
                <p>已发布内容与待处理队列</p>
              </div>
              <span class="dash-section-total">{{ formatNumber(stats.content.total) }} 篇</span>
            </header>
            <div class="dash-audit-list">
              <div v-for="row in auditRows" :key="row.label" class="dash-audit-row">
                <div class="dash-audit-head">
                  <span>{{ row.label }}</span>
                  <strong>{{ formatNumber(row.count) }}</strong>
                </div>
                <div class="dash-progress-track" aria-hidden="true">
                  <span :class="`is-${row.status}`" :style="{ width: `${pct(row.count, stats.content.total)}%` }"></span>
                </div>
              </div>
            </div>
            <footer class="dash-section-foot">
              <span>今日新增内容</span>
              <strong>+{{ formatNumber(stats.content.today) }}</strong>
            </footer>
          </section>

          <section class="dash-section">
            <header class="dash-section-head">
              <div>
                <h3>热门标签</h3>
                <p>按已通过内容的使用次数</p>
              </div>
              <span class="dash-section-total">Top 10</span>
            </header>
            <ol v-if="stats.topTags.length" class="dash-tag-list">
              <li v-for="(tag, index) in stats.topTags" :key="tag.tag" class="dash-tag-row">
                <span class="dash-tag-rank">{{ String(index + 1).padStart(2, '0') }}</span>
                <span class="dash-tag-name"><IconTag aria-hidden="true" />{{ tag.tag }}</span>
                <span class="dash-tag-track" aria-hidden="true">
                  <span :style="{ width: `${Math.round((tag.count / maxTag) * 100)}%` }"></span>
                </span>
                <strong class="dash-tag-count">{{ formatNumber(tag.count) }}</strong>
              </li>
            </ol>
            <a-empty v-else description="暂无标签" />
          </section>
        </div>

        <div class="dashboard-grid dashboard-recent">
          <section class="dash-section">
            <header class="dash-section-head">
              <div>
                <h3>最新内容</h3>
                <p>最近提交到站点的作品</p>
              </div>
              <span class="dash-section-total">{{ stats.recentContents.length }} 条</span>
            </header>
            <div v-if="stats.recentContents.length" class="dash-list">
              <button
                v-for="item in stats.recentContents"
                :key="item.id"
                type="button"
                class="dash-list-item dash-content-item"
                :aria-label="`查看内容：${item.title || '无标题'}`"
                @click="admin.openDrawer(item, 'view')"
              >
                <div v-if="item.thumb" class="content-thumb">
                  <MediaImage :src="item.thumb" :preview="false" alt="" />
                </div>
                <div v-else class="content-thumb content-thumb-text">
                  <IconFile aria-hidden="true" />
                </div>
                <span class="dash-list-main">
                  <strong class="dash-list-title">{{ item.title || '无标题' }}</strong>
                  <span class="dash-list-meta">
                    <span>{{ item.user?.username }}</span>
                    <span aria-hidden="true">·</span>
                    <span><IconEye aria-hidden="true" /> {{ formatNumber(item.view_count) }}</span>
                    <span aria-hidden="true">·</span>
                    <span><IconClockCircle aria-hidden="true" /> {{ formatTime(item.created_at ?? 0) }}</span>
                  </span>
                </span>
                <span
                  class="dash-status-tag"
                  :class="{
                    'is-approved': item.audit_status === 'approved',
                    'is-pending': item.audit_status === 'pending',
                    'is-rejected': item.audit_status === 'rejected',
                  }"
                >
                  {{ item.audit_status === 'approved' ? '已通过' : item.audit_status === 'pending' ? '待审核' : '已拒绝' }}
                </span>
              </button>
            </div>
            <a-empty v-else description="暂无内容" />
          </section>

          <section class="dash-section">
            <header class="dash-section-head">
              <div>
                <h3>最新用户</h3>
                <p>最近加入创作社区的成员</p>
              </div>
              <span class="dash-section-total">{{ stats.recentUsers.length }} 位</span>
            </header>
            <div v-if="stats.recentUsers.length" class="dash-list">
              <div v-for="user in stats.recentUsers" :key="user.id" class="dash-list-item">
                <UserAvatar :size="32" :email="user.email" :username="user.username" />
                <div class="dash-list-main">
                  <div class="dash-user-title">
                    <strong>{{ user.username }}</strong>
                    <span v-if="user.is_admin" class="dash-user-tag is-admin">管理员</span>
                    <span v-if="user.is_banned" class="dash-user-tag is-banned">已封禁</span>
                  </div>
                  <span class="dash-list-meta">
                    <span>{{ user.email || '未绑定邮箱' }}</span>
                    <span aria-hidden="true">·</span>
                    <span><IconClockCircle aria-hidden="true" /> {{ formatTime(user.created_at ?? 0) }}</span>
                  </span>
                </div>
              </div>
            </div>
            <a-empty v-else description="暂无用户" />
          </section>
        </div>
      </div>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.dashboard-empty {
  padding: 48px 0;
}

.dash-spin { width: 100%; }
.dashboard { display: grid; gap: 30px; }

.dash-stat-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 0;
  border-top: 1px solid $admin-border-soft;
  border-bottom: 1px solid $admin-border-soft;
}

.dash-stat {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 5px;
  min-width: 0;
  min-height: 112px;
  padding: 18px 16px 16px;
  border: 0;
  border-right: 1px solid $admin-border-soft;
  background: transparent;
  color: var(--admin-text);
  text-align: left;

  &:last-child { border-right: 0; }
  &:not(:disabled) { cursor: pointer; }
  &:not(:disabled):hover { background: color-mix(in srgb, var(--site-accent-soft) 38%, transparent); }
  &:focus-visible { position: relative; z-index: 1; outline: 2px solid var(--site-accent); outline-offset: -2px; }
  &:disabled { cursor: default; }

  strong {
    color: var(--site-ink);
    font: 600 clamp(23px, 2vw, 30px)/1.05 var(--site-title-font);
    font-variant-numeric: tabular-nums;
    letter-spacing: -.035em;
  }
}

.dash-stat-label { color: var(--site-muted); font-size: 10px; }
.dash-stat-sub { overflow: hidden; max-width: 100%; color: var(--site-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(320px, .85fr);
  gap: 34px;
}

.dash-section { min-width: 0; }

.dash-section-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  padding: 0 0 14px;
  border-bottom: 1px solid $admin-border-soft;

  h3 {
    margin: 0;
    color: var(--site-ink);
    font: 600 19px/1.2 var(--site-title-font);
    letter-spacing: -.025em;
  }

  p { margin: 5px 0 0; color: var(--site-muted); font-size: 10px; line-height: 1.5; }
}

.dash-section-total { flex: 0 0 auto; color: var(--site-muted); font-size: 10px; font-variant-numeric: tabular-nums; }
.dash-audit-list { display: grid; gap: 18px; padding: 20px 0 16px; }
.dash-audit-row { display: grid; gap: 8px; }
.dash-audit-head { display: flex; justify-content: space-between; gap: 12px; color: var(--site-muted); font-size: 11px; }
.dash-audit-head strong { color: var(--site-ink); font-size: 12px; font-variant-numeric: tabular-nums; }
.dash-progress-track,
.dash-tag-track { overflow: hidden; height: 5px; border-radius: 999px; background: var(--site-soft); }
.dash-progress-track > span,
.dash-tag-track > span { display: block; height: 100%; border-radius: inherit; transition: width .24s ease; }
.dash-progress-track .is-success { background: rgb(var(--success-6)); }
.dash-progress-track .is-warning { background: rgb(var(--warning-6)); }
.dash-progress-track .is-danger { background: rgb(var(--danger-6)); }
.dash-section-foot { display: flex; justify-content: space-between; padding-top: 12px; border-top: 1px solid $admin-border-soft; color: var(--site-muted); font-size: 10px; }
.dash-section-foot strong { color: var(--site-accent); font-size: 12px; font-variant-numeric: tabular-nums; }

.dash-tag-list { display: grid; gap: 14px; margin: 0; padding: 18px 0 0; list-style: none; }
.dash-tag-row { display: grid; grid-template-columns: 22px minmax(64px, .8fr) minmax(60px, 1fr) 42px; align-items: center; gap: 9px; min-width: 0; }
.dash-tag-rank { color: var(--site-muted); font-size: 9px; font-variant-numeric: tabular-nums; }
.dash-tag-name { display: flex; align-items: center; gap: 5px; overflow: hidden; color: var(--site-ink); font-size: 10px; white-space: nowrap; }
.dash-tag-name :deep(svg) { flex: 0 0 auto; color: var(--site-accent); font-size: 12px; }
.dash-tag-name { text-overflow: ellipsis; }
.dash-tag-track { height: 4px; }
.dash-tag-track > span { background: var(--site-accent); opacity: .72; }
.dash-tag-count { color: var(--site-muted); font-size: 10px; font-variant-numeric: tabular-nums; text-align: right; }

.dash-list { display: grid; }
.dash-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  padding: 11px 8px;
  border-bottom: 1px solid $admin-border-soft;

  &:last-child { border-bottom: 0; }
}

.dash-content-item {
  width: 100%;
  border-top: 0;
  border-right: 0;
  border-left: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: background-color .16s ease;

  &:hover { background: color-mix(in srgb, var(--site-accent-soft) 38%, transparent); }
  &:focus-visible { outline: 2px solid var(--site-accent); outline-offset: -2px; }
}

.content-thumb {
  display: grid;
  flex: 0 0 54px;
  place-items: center;
  width: 54px;
  height: 42px;
  overflow: hidden;
  border-radius: 6px;
  background: var(--site-soft);
  color: var(--site-muted);
}
.content-thumb :deep(img) { width: 100%; height: 100%; object-fit: cover; }
.content-thumb-text { font-size: 16px; }
.dash-list-main { display: flex; flex: 1; flex-direction: column; gap: 5px; min-width: 0; }
.dash-list-title { overflow: hidden; color: var(--site-ink); font-size: 11px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.dash-list-meta { display: flex; align-items: center; gap: 5px; min-width: 0; overflow: hidden; color: var(--site-muted); font-size: 9px; white-space: nowrap; }
.dash-list-meta > span { overflow: hidden; text-overflow: ellipsis; }
.dash-list-meta :deep(svg) { vertical-align: -1px; font-size: 10px; }
.dash-status-tag,
.dash-user-tag { flex: 0 0 auto; padding: 3px 7px; border-radius: 999px; font-size: 9px; line-height: 1.2; }
.dash-status-tag.is-approved { background: color-mix(in srgb, rgb(var(--success-6)) 12%, var(--site-paper)); color: rgb(var(--success-6)); }
.dash-status-tag.is-pending { background: color-mix(in srgb, rgb(var(--warning-6)) 14%, var(--site-paper)); color: color-mix(in srgb, rgb(var(--warning-6)) 78%, var(--site-ink)); }
.dash-status-tag.is-rejected { background: color-mix(in srgb, rgb(var(--danger-6)) 11%, var(--site-paper)); color: rgb(var(--danger-6)); }
.dash-user-title { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; min-width: 0; }
.dash-user-title strong { overflow: hidden; color: var(--site-ink); font-size: 11px; font-weight: 600; text-overflow: ellipsis; white-space: nowrap; }
.dash-user-tag.is-admin { background: var(--site-accent-soft); color: var(--site-accent); }
.dash-user-tag.is-banned { background: color-mix(in srgb, rgb(var(--danger-6)) 11%, var(--site-paper)); color: rgb(var(--danger-6)); }

@media (max-width: 1100px) {
  .dash-stat-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
  .dash-stat { border-right: 1px solid $admin-border-soft; }
  .dash-stat:nth-child(3n) { border-right: 0; }
  .dash-stat:last-child { border-right: 0; }
  .dashboard-grid { grid-template-columns: minmax(0, 1fr); gap: 30px; }
}

@media (max-width: 620px) {
  .dashboard { gap: 24px; }
  .dash-stat-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .dash-stat { min-height: 98px; padding: 14px 12px; border-right: 1px solid $admin-border-soft; }
  .dash-stat:nth-child(2n) { border-right: 0; }
  .dash-stat:last-child { border-right: 0; }
  .dash-stat strong { font-size: 24px; }
  .dash-section-head h3 { font-size: 18px; }
  .dash-tag-row { grid-template-columns: 20px minmax(56px, .8fr) minmax(40px, 1fr) 36px; gap: 7px; }
  .dash-list-item { gap: 8px; padding-inline: 4px; }
  .content-thumb { flex-basis: 44px; width: 44px; height: 38px; }
  .dash-status-tag { padding-inline: 5px; }
}

@media (prefers-reduced-motion: reduce) {
  .dash-progress-track > span,
  .dash-tag-track > span,
  .dash-content-item { transition: none; }
}
</style>

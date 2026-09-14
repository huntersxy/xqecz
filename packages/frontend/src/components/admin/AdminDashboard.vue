<script setup lang="ts">
import { ref, computed, onMounted, type Component } from 'vue'
import { adminApi } from '@/api'
import { getAvatarUrl, formatTime } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
import { useAdminStore } from '@/stores/admin'
import AdminPanel from './AdminPanel.vue'
import type { DashboardStats } from '@/types'
import { Tag, Tooltip } from '@arco-design/web-vue'
import {
  IconRefresh, IconFile, IconEye, IconUserGroup, IconMessage, IconBarChart,
  IconFire, IconTag, IconUser, IconClockCircle, IconCheckCircle, IconCloseCircle,
} from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()

const emit = defineEmits<{ select: [key: string] }>()

const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

function pct(n: number, total: number): number {
  if (!total) return 0
  return Math.round((n / total) * 100)
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
    { key: 'all', label: '内容总数', value: s.content.total, icon: IconFile, color: 'rgb(var(--primary-6))', sub: `今日 +${s.content.today}` },
    { key: 'pending', label: '待审核', value: s.content.pending, icon: IconEye, color: 'rgb(var(--warning-6))', sub: '内容待审核' },
    { key: 'users', label: '用户总数', value: s.users.total, icon: IconUserGroup, color: 'rgb(var(--success-6))', sub: `管理员 ${s.users.admins}` },
    { key: 'reports', label: '未处理举报', value: s.reports.unhandled, icon: IconMessage, color: 'rgb(var(--danger-6))', sub: `举报共 ${s.reports.total}` },
    { key: 'polls', label: '投票数', value: s.polls.total, icon: IconBarChart, color: 'rgb(var(--arcoblue-6))', sub: `累计票数 ${s.polls.votes}` },
    { key: '', label: '总浏览量', value: s.views, icon: IconFire, color: 'rgb(var(--gold-6))', sub: '全部内容累计' },
  ]
})

function go(key: string) {
  if (key) emit('select', key)
}

interface ProgressRow { label: string; count: number; status?: 'success' | 'warning' | 'danger'; color?: string; icon?: Component }
interface ProgressCard { title: string; footerLabel: string; footerValue: string; rows: ProgressRow[] }

const progressCards = computed<ProgressCard[]>(() => {
  const s = stats.value
  if (!s) return []
  return [
    {
      title: '内容审核状态',
      footerLabel: '今日新增内容',
      footerValue: String(s.content.today),
      rows: [
        { label: '已通过', count: s.content.approved, status: 'success', icon: IconCheckCircle },
        { label: '待审核', count: s.content.pending, status: 'warning', icon: IconEye },
        { label: '已拒绝', count: s.content.rejected, status: 'danger', icon: IconCloseCircle },
      ],
    },
  ]
})

const maxTag = computed(() => Math.max(1, ...(stats.value?.topTags.map((t) => t.count) ?? [1])))
</script>

<template>
  <AdminPanel title="仪表盘" desc="站点数据总览">
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
        <!-- 核心指标 -->
        <a-row :gutter="[16, 16]">
          <a-col v-for="card in statCards" :key="card.label" :xs="12" :sm="12" :md="8" :xl="4">
            <div
              class="dash-stat"
              :class="{ 'is-clickable': card.key }"
              :style="card.key ? { '--stat-color': card.color } : undefined"
              @click="go(card.key)"
            >
              <div class="dash-stat-icon" :style="{ color: card.color }">
                <component :is="card.icon" />
              </div>
              <a-statistic :title="card.label" :value="card.value" :group-separator="true" />
              <span class="dash-stat-sub">{{ card.sub }}</span>
            </div>
          </a-col>
        </a-row>

        <!-- 分布与热门标签 -->
        <a-row :gutter="[16, 16]" class="mt-4">
          <a-col v-for="card in progressCards" :key="card.title" :xs="24" :md="12" :xl="8">
            <a-card :title="card.title" :bordered="false" class="dash-card">
              <div v-for="row in card.rows" :key="row.label" class="dash-progress-row">
                <div class="dash-progress-head">
                  <span class="dash-progress-label">
                    <component v-if="row.icon" :is="row.icon" class="dash-progress-icon" />
                    {{ row.label }}
                  </span>
                  <span class="dash-progress-count">{{ row.count }}</span>
                </div>
                <a-progress
                  :percent="pct(row.count, stats!.content.total)"
                  :status="row.status"
                  :color="row.color"
                  :stroke-width="8"
                  :show-text="false"
                />
              </div>
              <div class="dash-today">
                <span class="admin-cell-3">{{ card.footerLabel }}</span>
                <span class="dash-today-value">{{ card.footerValue }}</span>
              </div>
            </a-card>
          </a-col>

          <a-col :xs="24" :md="24" :xl="8">
            <a-card title="热门标签 Top 10" :bordered="false" class="dash-card">
              <div v-if="stats!.topTags.length" class="dash-tags">
                <div v-for="t in stats!.topTags" :key="t.tag" class="dash-tag-row">
                  <span class="dash-tag-name">
                    <IconTag class="dash-tag-icon" />
                    <span class="dash-tag-text">{{ t.tag }}</span>
                  </span>
                  <span class="dash-tag-count">{{ t.count }}</span>
                  <a-progress :percent="Math.round((t.count / maxTag) * 100)" :stroke-width="6" :show-text="false" class="dash-tag-bar" />
                </div>
              </div>
              <a-empty v-else description="暂无标签" />
            </a-card>
          </a-col>
        </a-row>

        <!-- 最新内容 / 最新用户 -->
        <a-row :gutter="[16, 16]" class="mt-4">
          <a-col :xs="24" :lg="12">
            <a-card title="最新内容" :bordered="false" class="dash-card">
              <div v-if="stats!.recentContents.length" class="dash-list">
                <div
                  v-for="item in stats!.recentContents"
                  :key="item.id"
                  class="dash-list-item"
                  @click="admin.openDrawer(item, 'view')"
                >
                  <div v-if="item.thumb" class="content-thumb">
                    <MediaImage :src="item.thumb" :preview="false" alt="" />
                  </div>
                  <div v-else class="content-thumb content-thumb-text">
                    <IconFile />
                  </div>
                  <div class="dash-list-main">
                    <span class="admin-cell-title dash-list-title">{{ item.title || '无标题' }}</span>
                    <div class="admin-cell-3 dash-list-meta">
                      <span>{{ item.user?.username }}</span>
                      <span class="dash-list-meta-sep">·</span>
                      <span><IconEye /> {{ item.view_count }}</span>
                      <span class="dash-list-meta-sep">·</span>
                      <span><IconClockCircle /> {{ formatTime(item.created_at ?? 0) }}</span>
                    </div>
                  </div>
                  <Tag :bordered="false" size="small" class="dash-list-tag admin-tag-inline">
                    {{ item.audit_status === 'approved' ? '已通过' : item.audit_status === 'pending' ? '待审核' : '已拒绝' }}
                  </Tag>
                </div>
              </div>
              <a-empty v-else description="暂无内容" />
            </a-card>
          </a-col>

          <a-col :xs="24" :lg="12">
            <a-card title="最新用户" :bordered="false" class="dash-card">
              <div v-if="stats!.recentUsers.length" class="dash-list">
                <div v-for="u in stats!.recentUsers" :key="u.id" class="dash-list-item">
                  <a-avatar :size="30" :image-url="u.email ? getAvatarUrl(u.email) : ''">
                    <IconUser v-if="!u.email" />
                    <template #error><IconUser /></template>
                  </a-avatar>
                  <div class="dash-list-main">
                    <div class="flex items-center gap-1.5">
                      <span class="admin-cell-title">{{ u.username }}</span>
                      <Tag v-if="u.is_admin" color="arcoblue" :bordered="false" size="small" class="admin-tag-inline">管理员</Tag>
                      <Tag v-if="u.is_banned" color="red" :bordered="false" size="small" class="admin-tag-inline">已封禁</Tag>
                    </div>
                    <div class="admin-cell-3">
                      {{ u.email || '未绑定邮箱' }}
                      <span class="dash-list-meta-sep">·</span>
                      <IconClockCircle /> {{ formatTime(u.created_at ?? 0) }}
                    </div>
                  </div>
                </div>
              </div>
              <a-empty v-else description="暂无用户" />
            </a-card>
          </a-col>
        </a-row>
      </div>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.dashboard-empty {
  padding: 48px 0;
}

.dash-spin {
  width: 100%;
}

.dash-stat {
  --stat-color: rgb(var(--primary-6));
  position: relative;
  padding: 16px 18px;
  border: 1px solid $admin-border-soft;
  border-radius: $admin-radius;
  background: var(--admin-surface);
  transition: border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease;

  &.is-clickable {
    cursor: pointer;

    &:hover {
      border-color: var(--stat-color);
      transform: translateY(-2px);
      box-shadow: 0 8px 20px -10px var(--stat-color);
    }
  }

  :deep(.arco-statistic-title) {
    font-size: 12px;
    color: $admin-text-3;
  }

  :deep(.arco-statistic-value) {
    font-size: 22px;
    font-weight: 600;
    color: $admin-text;
    font-variant-numeric: tabular-nums;
  }
}

.dash-stat-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 9px;
  font-size: 17px;
  margin-bottom: 10px;
  background: var(--admin-fill);
}

.dash-stat-sub {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  color: $admin-text-3;
  font-variant-numeric: tabular-nums;
}

.dash-card {
  height: 100%;
  background: var(--admin-surface);

  :deep(.arco-card-header) {
    border-bottom: 1px solid $admin-border-soft;
    padding: 14px 18px;
  }

  :deep(.arco-card-header-title) {
    font-size: 14px;
    font-weight: 600;
    color: $admin-text;
  }

  :deep(.arco-card-body) {
    padding: 16px 18px;
  }
}

.dash-progress-row + .dash-progress-row {
  margin-top: 14px;
}

.dash-progress-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.dash-progress-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: $admin-text-2;
}

.dash-progress-icon {
  color: $admin-text-3;
}

.dash-progress-count {
  font-size: 13px;
  font-weight: 600;
  color: $admin-text;
  font-variant-numeric: tabular-nums;
}

.dash-today {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px dashed $admin-border-soft;
}

.dash-today-value {
  font-size: 15px;
  font-weight: 600;
  color: $admin-primary;
  font-variant-numeric: tabular-nums;
}

.dash-tags {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dash-tag-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dash-tag-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 130px;
  min-width: 0;
}

.dash-tag-icon {
  color: $admin-text-3;
  flex-shrink: 0;
}

.dash-tag-text {
  font-size: 13px;
  color: $admin-text-2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-tag-count {
  flex-shrink: 0;
  width: 34px;
  text-align: right;
  font-size: 12px;
  font-weight: 600;
  color: $admin-text;
  font-variant-numeric: tabular-nums;
}

.dash-tag-bar {
  flex: 1;
  min-width: 40px;
}

.dash-list {
  display: flex;
  flex-direction: column;
}

.dash-list-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  cursor: pointer;
  border-radius: 8px;
  transition: background 0.15s ease;

  &:hover {
    background: var(--admin-fill);
  }

  + .dash-list-item {
    border-top: 1px solid $admin-border-soft;
  }
}

.dash-list-main {
  flex: 1;
  min-width: 0;
}

.dash-list-title {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dash-list-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
  overflow: hidden;
  white-space: nowrap;

  span {
    display: inline-flex;
    align-items: center;
    gap: 3px;
  }
}

.dash-list-meta-sep {
  color: $admin-text-3;
  opacity: 0.5;
}

.dash-list-tag {
  flex-shrink: 0;
}

.content-thumb-text {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: $admin-text-3;
}

.content-thumb {
  @include content-thumb(48px, 36px);
}

@media (max-width: 768px) {
  .dash-stat {
    padding: 12px 14px;
  }

  .dash-tag-name {
    flex-basis: 100px;
  }
}
</style>

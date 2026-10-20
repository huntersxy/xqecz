<script setup lang="ts">
// 投票管理：把「一行一条」的列表重排成「概览指标 + 卡片流 + 排行侧栏」。
// 信息层级：票数最大（观众最关心），标题其次，描述与选项做二级信息，删除收进卡片页脚。
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { IconDelete, IconPlus } from '@arco-design/web-vue/es/icon'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminPanel from './AdminPanel.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import { useAdminStore } from '@/stores/admin'
import { useConfirm } from '@/composables/useToast'
import { formatTime, prefersReducedMotion } from '@/utils'
import type { Poll } from '@/types'

const admin = useAdminStore()

// 首屏加载才出骨架；已有数据时刷新只让工具条的按钮转圈，卡片保持可读
const showSkeleton = computed(() => admin.pollsLoading && admin.polls.length === 0)

// 顶部概览：投票数 / 总票数 / 最高票（带对应标题）
const totalVotes = computed(() => admin.polls.reduce((sum, p) => sum + (p.vote_count || 0), 0))
const topPoll = computed<Poll | null>(() =>
  admin.polls.reduce<Poll | null>((best, p) => (!best || p.vote_count > best.vote_count ? p : best), null),
)
const rankedPolls = computed(() => [...admin.polls].sort((a, b) => b.vote_count - a.vote_count).slice(0, 3))
const pollSummary = computed(() => `共 ${admin.polls.length} 条投票 · 累计 ${totalVotes.value} 票`)

function voteShare(count: number) {
  return totalVotes.value > 0 ? Math.round((count / totalVotes.value) * 100) : 0
}

// M8：删除先让卡片淡出 280ms，再调接口并刷新；命中 reduced-motion 时直达终态
const LEAVE_MS = 280
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
let disposed = false

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

function clearLeaving(id: number) {
  leavingIds.value = leavingIds.value.filter(item => item !== id)
}

async function performDelete(id: number) {
  try {
    await admin.deletePoll(id)
  } catch {
    // 失败提示由 store 统一负责
  }
  try {
    await admin.loadPolls()
  } catch {
    // 刷新失败时卡片回到可见状态，不把「正在删除」的样式卡死
  }
  clearLeaving(id)
}

async function confirmDeletePoll(id: number) {
  const { confirm } = useConfirm()
  const ok = await confirm('确定删除该投票？删除后不可恢复。')
  if (!ok) return
  if (prefersReducedMotion()) {
    await performDelete(id)
    return
  }
  if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
  timers.add(
    setTimeout(() => {
      if (disposed) return
      void performDelete(id)
    }, LEAVE_MS),
  )
}

onMounted(() => admin.loadPolls())

onBeforeUnmount(() => {
  disposed = true
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
})
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

    <AdminSkeleton v-if="showSkeleton" variant="cards" :rows="5" label="正在加载投票" />

    <AdminEmpty
      v-else-if="admin.polls.length === 0"
      title="暂无投票"
      desc="创建投票后，观众就能在投票广场参与选择。"
    >
      <template #actions>
        <a-button type="primary" size="small" @click="admin.showCreatePollModal = true">
          <IconPlus /> 创建第一个投票
        </a-button>
      </template>
    </AdminEmpty>

    <div v-else class="adm-board poll-board">
      <div class="adm-board-main">
        <div class="adm-stat-grid poll-stats">
          <div class="adm-stat">
            <span class="adm-stat-label">投票数</span>
            <span class="adm-stat-value">{{ admin.polls.length }}</span>
            <span class="adm-stat-hint">投票广场的全部投票</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">总票数</span>
            <span class="adm-stat-value">{{ totalVotes }}</span>
            <span class="adm-stat-hint">全部投票累计</span>
          </div>
          <div class="adm-stat">
            <span class="adm-stat-label">最高票</span>
            <span class="adm-stat-value">{{ topPoll ? topPoll.vote_count : 0 }}</span>
            <span class="adm-stat-hint" :title="topPoll ? topPoll.title : ''">{{ topPoll ? topPoll.title : '暂无数据' }}</span>
          </div>
        </div>

        <div class="poll-list adm-card-grid is-wide">
          <article
            v-for="p in admin.polls"
            :key="p.id"
            class="adm-card poll-card"
            :class="{ 'is-leaving': isLeaving(p.id) }"
          >
            <div class="adm-card-content">
              <header class="poll-head">
                <p class="poll-count">
                  <span class="poll-count-num" aria-hidden="true">{{ p.vote_count }}</span>
                  <span class="poll-count-unit" aria-hidden="true">票</span>
                  <span class="poll-count-sr">{{ p.vote_count }} 票</span>
                </p>
                <time class="poll-time" :datetime="p.created_at">{{ formatTime(p.created_at) }}</time>
              </header>

              <h3 class="adm-card-title">
                <RouterLink :to="`/polls/${p.id}`" class="poll-title">{{ p.title }}</RouterLink>
              </h3>

              <p v-if="p.description" class="poll-desc" :title="p.description">{{ p.description }}</p>

              <div v-if="p.options.length" class="poll-options">
                <span class="poll-options-hint">{{ p.options.length > 1 ? `${p.options.length} 个选项` : '只能选一个' }}</span>
                <ul class="poll-chips">
                  <li v-for="(o, i) in p.options" :key="i" class="poll-chip">{{ o }}</li>
                </ul>
              </div>
            </div>

            <footer class="adm-card-foot">
              <span class="poll-share">占全部票数 {{ voteShare(p.vote_count) }}%</span>
              <a-button
                class="poll-delete is-push"
                status="danger"
                size="small"
                aria-label="删除投票"
                @click="confirmDeletePoll(p.id)"
              >
                <IconDelete /> 删除
              </a-button>
            </footer>
          </article>
        </div>
      </div>

      <aside class="adm-board-aside" aria-label="投票概览">
        <section class="adm-card poll-side">
          <div class="adm-card-content">
            <h3 class="poll-side-title">票数排行</h3>
            <ol class="poll-rank">
              <li v-for="(item, i) in rankedPolls" :key="item.id" class="poll-rank-row">
                <span class="poll-rank-no">{{ i + 1 }}</span>
                <span class="poll-rank-title" :title="item.title">{{ item.title }}</span>
                <span class="poll-rank-votes">{{ item.vote_count }} 票</span>
              </li>
            </ol>
            <p class="poll-side-hint">票数由前台投票广场实时累计。</p>
          </div>
        </section>
      </aside>
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

// ── 卡片头：大号票数 + 创建时间 ──
.poll-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 4px 10px;
  min-width: 0;
}

.poll-count {
  display: flex;
  align-items: baseline;
  gap: 3px;
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
}

.poll-count-num {
  font-size: 28px;
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -.03em;
  @include numeric;
}

.poll-count-unit {
  margin-left: 2px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-sm, 13px);
}

// 视觉上把票数拆成「数字 + 单位」，读屏与文本断言只认这一份完整文案
.poll-count-sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

.poll-time {
  flex-shrink: 0;
  padding-top: 4px;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.poll-title {
  color: inherit;
  text-decoration: none;
  border-radius: 4px;
  @include focus-ring(2px);

  &:hover {
    color: var(--adm-accent, var(--creative-accent));
    text-decoration: underline;
    text-underline-offset: 3px;
  }
}

// 描述两行截断；没有描述时整段不渲染，不留空行
.poll-desc {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.6;
  word-break: break-word;
}

.poll-options {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.poll-options-hint {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: .04em;
}

.poll-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.poll-chip {
  padding: 3px 10px;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-2));
  font-size: var(--adm-font-xs, 12px);
}

.poll-share {
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

// ── 侧栏：票数排行 ──
.poll-side-title {
  margin: 0;
  color: var(--adm-ink, var(--admin-text));
  font-family: var(--site-title-font);
  font-size: var(--adm-font-base, 14px);
  font-weight: 600;
  letter-spacing: -.01em;
}

.poll-rank {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.poll-rank-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: var(--adm-font-sm, 13px);
}

.poll-rank-no {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-soft, var(--admin-fill));
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.poll-rank-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--adm-ink, var(--admin-text));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.poll-rank-votes {
  flex: 0 0 auto;
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
  @include numeric;
}

.poll-side-hint {
  margin: 0;
  padding-top: 8px;
  border-top: 1px solid var(--admin-border-soft);
  color: var(--adm-muted, var(--admin-text-3));
  font-size: var(--adm-font-xs, 12px);
}

@media (max-width: 768px) {
  .poll-count-num {
    font-size: 24px;
  }

  // 手机档点击区 ≥ 44px
  .poll-delete {
    min-height: var(--adm-touch, 44px);
  }
}
</style>

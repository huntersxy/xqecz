<script setup lang="ts">
// 后台「内容」页签（我的内容 / 所有内容 / 待审核内容）：卡片流呈现。
// 数据加载、审核、删除、抽屉、分页与旧 AdminContentTable 逐条一致，只把表格换成卡片网格（§6.4）。
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import MediaImage from '@/components/MediaImage.vue'
import AdminPanel from './AdminPanel.vue'
import AdminDataToolbar from './AdminDataToolbar.vue'
import AdminSkeleton from './AdminSkeleton.vue'
import AdminEmpty from './AdminEmpty.vue'
import AdminStatus from './AdminStatus.vue'
import { useMediaQuery } from '@vueuse/core'
import { prefersReducedMotion } from '@/utils'
import type { Content } from '@/types'
import {
  IconEdit, IconDelete, IconRefresh, IconCheck, IconClose, IconMore,
} from '@arco-design/web-vue/es/icon'

defineOptions({ name: 'AdminContentList' })

const props = defineProps<{ mode: 'my' | 'all' | 'pending' }>()
const userStore = useUserStore()
const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')

// M8：卡片退场 280ms 后再刷新，延迟逻辑写在组件内，不改 store。
const LEAVE_MS = 280
const leavingIds = ref<number[]>([])
const timers = new Set<ReturnType<typeof setTimeout>>()
let disposed = false

const storeMap = { my: admin.myContent, all: admin.allContent, pending: admin.pendingContent }
const data = computed(() => storeMap[props.mode])

const panelTitle = computed(() =>
  props.mode === 'my' ? '我的内容' : props.mode === 'pending' ? '待审核内容' : '所有内容',
)

// 页头说明只讲这一页做什么；计数放到工具区，避免两处重复（§10 骨架）。
const panelDesc = computed(() => {
  if (props.mode === 'my') return '你发布的内容与审核状态，可编辑或删除'
  if (props.mode === 'pending') return '待审核内容，通过或拒绝后自动移出列表'
  return '站点全部内容，可编辑、删除，或批量生成缩略图'
})

const summary = computed(() =>
  props.mode === 'pending' ? `共 ${data.value.total} 条待审内容` : `共 ${data.value.total} 条内容`,
)

const emptyTitle = computed(() =>
  props.mode === 'pending' ? '没有待审内容' : props.mode === 'my' ? '还没有内容' : '暂无内容',
)
const emptyDesc = computed(() => {
  if (props.mode === 'my') return '发布第一篇创作后，这里会显示审核状态与操作'
  if (props.mode === 'pending') return '新提交的内容会出现在这里，按需通过或拒绝'
  return '站点还没有任何内容'
})

const showSkeleton = computed(() => data.value.loading && data.value.list.length === 0)

function auditType(status?: Content['audit_status']) {
  if (status === 'approved') return 'success'
  return status === 'pending' ? 'warning' : 'danger'
}
const auditLabel: Record<string, string> = { approved: '已通过', pending: '审核中', rejected: '已拒绝' }

// 卡片徽标只在有审核状态时渲染，v-if 之外再兜一层，模板里不必处理 undefined 索引。
function auditText(status?: Content['audit_status']) {
  return status ? auditLabel[status] ?? '' : ''
}

function tagsOf(record: Content) {
  return record.tags ?? []
}

function isLeaving(id: number) {
  return leavingIds.value.includes(id)
}

function trackTimer(timer: ReturnType<typeof setTimeout>) {
  timers.add(timer)
  return timer
}

async function load(page = 1) {
  if (props.mode === 'my') await admin.loadMyContent(page)
  else if (props.mode === 'all') await admin.loadAllContent(page)
  else await admin.loadPendingContent(page)
}

function refresh() {
  void load(data.value.page)
}

// 退场后刷新：刷新结束（成功或失败）都清掉标记——成功时卡片已消失，失败时卡片恢复原样。
function scheduleLeave(id: number, page: number) {
  if (disposed) return
  if (prefersReducedMotion()) {
    void load(page)
    return
  }
  if (!leavingIds.value.includes(id)) leavingIds.value = [...leavingIds.value, id]
  trackTimer(setTimeout(() => {
    void (async () => {
      await load(page)
      leavingIds.value = leavingIds.value.filter(item => item !== id)
    })()
  }, LEAVE_MS))
}

async function handleAudit(id: number, status: 'approved' | 'rejected') {
  if (!userStore.user) return
  const ok = await admin.auditContent(id, status, userStore.user.id)
  if (ok) scheduleLeave(id, data.value.page)
}

function handleDelete(id: number) {
  admin.confirmDelete(id, () => scheduleLeave(id, data.value.page))
}

function onPageChange(current: number) {
  void load(current)
}

// 整卡点击开抽屉；键盘 Enter / 空格等价（§7.5）。
// 卡片底部的按钮冒泡上来的回车不再开抽屉，否则一次回车会既审核又开抽屉。
function onCardKey(event: KeyboardEvent, record: Content) {
  if (event.key !== 'Enter' && event.key !== ' ') return
  const target = event.target as HTMLElement | null
  if (target?.closest('.adm-card-actions')) return
  event.preventDefault()
  admin.openDrawer(record, 'view')
}

function clearTimers() {
  timers.forEach(timer => clearTimeout(timer))
  timers.clear()
}

onMounted(() => { void load() })
onBeforeUnmount(() => {
  disposed = true
  clearTimers()
})
</script>

<template>
  <AdminPanel :title="panelTitle" :desc="panelDesc">
    <template #toolbar>
      <AdminDataToolbar
        :summary="summary"
        :loading="data.loading"
        refreshable
        @refresh="refresh"
      >
        <template #actions>
          <a-button v-if="mode === 'all'" size="small" @click="admin.regenerateAllThumbnails()">
            <IconRefresh /> 批量生成缩略图
          </a-button>
        </template>
      </AdminDataToolbar>
    </template>

    <AdminSkeleton
      v-if="showSkeleton"
      :variant="isMobile ? 'cards' : 'rows'"
      :rows="isMobile ? 3 : 6"
      :columns="4"
      :label="`正在加载${panelTitle}`"
    />

    <AdminEmpty
      v-else-if="!data.list.length"
      :title="emptyTitle"
      :desc="emptyDesc"
      :compact="isMobile || mode === 'pending'"
    />

    <div v-else class="content-list">
      <a-spin :loading="data.loading">
        <div class="adm-card-grid">
          <article
            v-for="record in data.list"
            :key="record.id"
            class="adm-card"
            :class="{ 'is-leaving': isLeaving(record.id) }"
            tabindex="0"
            :aria-label="`查看 ${record.title || '无标题'}`"
            @click="admin.openDrawer(record, 'view')"
            @keydown="onCardKey($event, record)"
          >
            <div class="adm-card-media" :class="{ 'is-placeholder': !record.thumb }">
              <MediaImage
                v-if="record.thumb"
                class="adm-media-img"
                :src="record.thumb"
                :preview="false"
                alt=""
              />
              <template v-else>暂无缩略图</template>
              <span v-if="record.audit_status" class="adm-card-badge">
                <AdminStatus
                  :type="auditType(record.audit_status)"
                  :label="auditText(record.audit_status)"
                />
              </span>
            </div>

            <div class="adm-card-body">
              <h3 class="adm-card-title">{{ record.title || '无标题' }}</h3>
              <div class="adm-card-meta">
                <template v-if="tagsOf(record).length">
                  <a-tag
                    v-for="tag in tagsOf(record).slice(0, 2)"
                    :key="tag"
                    size="small"
                    :bordered="false"
                    class="admin-tag-inline content-tag"
                  >{{ tag }}</a-tag>
                  <a-tooltip
                    v-if="tagsOf(record).length > 2"
                    :content="tagsOf(record).slice(2).join('、')"
                  >
                    <a-tag size="small" :bordered="false" class="admin-tag-inline content-tag">
                      +{{ tagsOf(record).length - 2 }}
                    </a-tag>
                  </a-tooltip>
                </template>
                <span v-if="mode !== 'my'" class="content-author">{{ record.user?.username || '未知' }}</span>
              </div>
            </div>

            <div class="adm-card-actions" @click.stop>
              <template v-if="mode === 'pending'">
                <a-button
                  type="primary"
                  size="small"
                  class="content-action"
                  aria-label="通过"
                  @click="handleAudit(record.id, 'approved')"
                >
                  <template #icon><IconCheck /></template>
                  通过
                </a-button>
                <a-button
                  status="danger"
                  size="small"
                  class="content-action"
                  aria-label="拒绝"
                  @click="handleAudit(record.id, 'rejected')"
                >
                  <template #icon><IconClose /></template>
                  拒绝
                </a-button>
                <a-dropdown trigger="click" position="br">
                  <a-button
                    size="small"
                    class="content-action is-icon is-more"
                    aria-label="更多操作"
                  >
                    <template #icon><IconMore /></template>
                  </a-button>
                  <template #content>
                    <a-doption @click="admin.openDrawer(record, 'edit')">编辑</a-doption>
                    <a-doption class="content-danger" @click="handleDelete(record.id)">删除</a-doption>
                  </template>
                </a-dropdown>
              </template>
              <template v-else>
                <a-button
                  size="small"
                  class="content-action"
                  aria-label="编辑"
                  @click="admin.openDrawer(record, 'edit')"
                >
                  <template #icon><IconEdit /></template>
                  编辑
                </a-button>
                <a-button
                  status="danger"
                  size="small"
                  class="content-action"
                  aria-label="删除"
                  @click="handleDelete(record.id)"
                >
                  <template #icon><IconDelete /></template>
                  删除
                </a-button>
              </template>
            </div>
          </article>
        </div>
      </a-spin>

      <div v-if="data.totalPages > 1" class="content-pagination">
        <a-pagination
          :current="data.page"
          :total="data.total"
          :page-size="data.pageSize"
          size="small"
          @change="onPageChange"
        />
      </div>
    </div>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 卡片网格、卡片本体、状态徽标、媒体区、卡片操作行都由 AdminPanel.vue 的全局块统一定义
// （.adm-card*），这里只补本页特有的间距、按钮尺寸与媒体填充，避免同一套卡片样式各页签各编译一份。
.content-list {
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-xl, 24px);
}

// 整卡可点击：指针反馈与键盘焦点环（§7.5）。布局仍完全取自全局 .adm-card。
.content-list .adm-card {
  cursor: pointer;

  @include focus-ring(1px);
}

// 手机卡片更窄时操作行允许换行，避免按钮被卡片裁切。
.content-list .adm-card-actions {
  flex-wrap: wrap;
}

// MediaImage 渲染成 a-image（外层 .arco-image + 内层 img），全局 .adm-card-media > img 命不中，
// 这里把图片铺满 4:3 媒体区；媒体区尺寸与占位风格仍由全局样式决定。
.adm-media-img {
  display: block;
  width: 100%;
  height: 100%;

  :deep(.arco-image-img) {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.content-tag {
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.content-author {
  @include ellipsis;

  max-width: 100%;
}

// 卡片底部操作：桌面 36px、手机 44px（--adm-touch 已在 768 断点切换）；
// 字号取 13px 而不是 Arco small 的 12px，保证可读性（§6.2）。
.arco-btn.content-action {
  height: var(--adm-touch, 36px);
  padding: 0 var(--adm-gap-md, 12px);
  font-size: var(--adm-font-sm, 13px);
}

.arco-btn.content-action.is-icon {
  width: var(--adm-touch, 36px);
  padding: 0;
}

.content-action.is-more {
  margin-left: auto;
}

.content-pagination {
  display: flex;
  justify-content: center;
  padding: var(--adm-gap-md, 12px) 0 var(--adm-gap-xs, 4px);
}

@media (max-width: 768px) {
  .content-list {
    padding: var(--adm-gap-md, 12px);
  }

  // 两列卡片时 44px 触控目标放不进一行：通过 / 拒绝仍并排，更多操作换到第二行右对齐。
  .arco-btn.content-action {
    padding: 0 10px;
  }
}
</style>

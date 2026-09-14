<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import { useAdminStore } from '@/stores/admin'
import MediaImage from '@/components/MediaImage.vue'
import { ACTION_COL } from './adminColumns'
import { Tag, Tooltip, type TableColumnData } from '@arco-design/web-vue'
import { useMediaQuery } from '@vueuse/core'
import type { Content } from '@/types'
import {
  IconEdit, IconDelete, IconRefresh, IconCheck, IconClose,
} from '@arco-design/web-vue/es/icon'

const props = defineProps<{ mode: 'my' | 'all' | 'pending' }>()
const userStore = useUserStore()
const admin = useAdminStore()
const isMobile = useMediaQuery('(max-width: 768px)')

const storeMap = { my: admin.myContent, all: admin.allContent, pending: admin.pendingContent }
const data = computed(() => storeMap[props.mode])

const panelTitle = computed(() =>
  props.mode === 'my' ? '我的内容' : props.mode === 'pending' ? '待审核内容' : '所有内容',
)

const columns = computed<TableColumnData[]>(() => [
  { title: '内容', slotName: 'content', minWidth: 220 },
  { title: '标签', slotName: 'tags', width: 180 },
  { title: '作者', slotName: 'author', minWidth: 80 },
  { ...ACTION_COL },
])

async function load(page = 1) {
  if (props.mode === 'my') admin.loadMyContent(page)
  else if (props.mode === 'all') admin.loadAllContent(page)
  else admin.loadPendingContent(page)
}
async function handleAudit(id: number, status: 'approved' | 'rejected') {
  if (!userStore.user) return
  if (await admin.auditContent(id, status, userStore.user.id)) load(data.value.page)
}
function handleDelete(id: number) { admin.confirmDelete(id, () => load(data.value.page)) }
function onTableChange(current: number) { load(current) }

onMounted(load)
</script>

<template>
  <AdminPanel :title="panelTitle" :desc="`共 ${data.total} 条内容`">
    <template #actions>
      <a-button v-if="mode === 'all'" size="small" @click="admin.regenerateAllThumbnails()">
        <IconRefresh /> 批量生成缩略图
      </a-button>
      <Tooltip title="刷新">
        <a-button class="admin-icon-btn" type="text" size="small" :loading="data.loading" @click="load(data.page)">
          <IconRefresh />
        </a-button>
      </Tooltip>
    </template>

    <a-spin :loading="data.loading">
      <template v-if="!isMobile">
        <a-table
          :columns="columns"
          :data="data.list"
          :pagination="{ current: data.page, pageSize: data.pageSize, total: data.total }"
          row-key="id"
          @page-change="onTableChange"
        >
          <template #content="{ record }">
            <div class="flex items-center gap-3">
              <div v-if="record.thumb" class="content-thumb">
                <MediaImage :src="record.thumb" :preview="false" alt="" />
              </div>
              <Tooltip :title="record.title || '无标题'">
                <span class="content-title admin-cell-title">{{ record.title || '无标题' }}</span>
              </Tooltip>
            </div>
          </template>
          <template #author="{ record }">
            <span class="admin-cell-2">{{ record.user?.username }}</span>
          </template>
          <template #tags="{ record }">
            <div class="flex flex-wrap gap-1">
              <Tag v-for="tag in (record.tags || []).slice(0, 3)" :key="tag" :bordered="false" class="admin-tag-inline">{{ tag }}</Tag>
              <Tooltip v-if="(record.tags || []).length > 3" :title="record.tags.slice(3).join(', ')">
                <Tag :bordered="false" class="admin-tag-inline">+{{ record.tags.length - 3 }}</Tag>
              </Tooltip>
            </div>
          </template>
          <template #actions="{ record }">
            <div class="admin-action-group">
              <Tooltip title="编辑">
                <a-button class="admin-icon-btn" type="text" size="small" @click="admin.openDrawer(record as Content, 'edit')"><IconEdit /></a-button>
              </Tooltip>
              <Tooltip v-if="mode === 'pending'" title="通过">
                <a-button class="admin-icon-btn is-primary" type="text" size="small" @click="handleAudit(record.id, 'approved')"><IconCheck /></a-button>
              </Tooltip>
              <Tooltip v-if="mode === 'pending'" title="拒绝">
                <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleAudit(record.id, 'rejected')"><IconClose /></a-button>
              </Tooltip>
              <Tooltip title="删除">
                <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleDelete(record.id)"><IconDelete /></a-button>
              </Tooltip>
            </div>
          </template>
        </a-table>
      </template>

      <template v-else>
        <div class="admin-mobile-list">
          <div v-for="record in data.list" :key="record.id" class="admin-mobile-card" @click="admin.openDrawer(record, 'view')">
            <div class="flex gap-2.5 items-start">
              <div v-if="record.thumb" class="content-thumb content-thumb-lg">
                <MediaImage :src="record.thumb" :preview="false" alt="" />
              </div>
              <div class="flex-1 min-w-0">
                <div class="mobile-title admin-cell-title">{{ record.title || '无标题' }}</div>
                <div class="flex items-center gap-1.5 flex-wrap">
                  <span v-if="mode !== 'my'" class="admin-cell-3">{{ record.user?.username }}</span>
                </div>
              </div>
            </div>
            <div class="admin-mobile-actions" @click.stop>
              <a-button class="admin-icon-btn" type="text" size="small" @click="admin.openDrawer(record, 'edit')"><IconEdit /></a-button>
              <a-button v-if="mode === 'pending'" class="admin-icon-btn is-primary" type="text" size="small" @click="handleAudit(record.id, 'approved')"><IconCheck /></a-button>
              <a-button v-if="mode === 'pending'" class="admin-icon-btn is-danger" type="text" size="small" @click="handleAudit(record.id, 'rejected')"><IconClose /></a-button>
              <a-button class="admin-icon-btn is-danger" type="text" size="small" @click="handleDelete(record.id)"><IconDelete /></a-button>
            </div>
          </div>
          <a-empty v-if="data.list.length === 0" description="暂无内容" />
        </div>
        <div v-if="data.totalPages > 1" class="admin-mobile-pagination">
          <a-pagination :current="data.page" :total="data.total" :page-size="data.pageSize" size="small" @change="(current: number) => load(current)" />
        </div>
      </template>
    </a-spin>
  </AdminPanel>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.content-thumb {
  @include content-thumb(48px, 36px);
}

.content-thumb-lg {
  width: 56px;
  height: 42px;
}

.content-title {
  @include ellipsis;
  max-width: 240px;
  display: inline-block;
}

.mobile-title {
  @include ellipsis;
  margin-bottom: 4px;
}
</style>

<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import { adminApi } from '@/api'
import { getAvatarUrl, getImageUrl, renderMarkdown } from '@/utils'
import MediaImage from '@/components/MediaImage.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import { Tag, type FileItem } from '@arco-design/web-vue'
import { toast, useConfirm } from '@/composables/useToast'
import type { User } from '@/types'
import { IconUpload, IconSearch, IconUser } from '@arco-design/web-vue/es/icon'

const admin = useAdminStore()

const editFormModel: Record<string, unknown> = {}
const editTitle = ref('')
const editContent = ref('')
const editUrl = ref('')
const editTags = ref<string[]>([])
const editFile = ref<File | undefined>()
const editFileName = ref('')
const tabKey = ref('preview')

const authorKeyword = ref('')
const allUsers = ref<User[]>([])
const usersLoading = ref(false)

const renderedContent = computed(() => {
  if (!admin.drawerContent) return ''
  return renderMarkdown(admin.drawerContent.text || '')
})

// 保留 video/link 展示兜底（历史脏数据），用 string 比较避免收窄后的类型错误
const drawerTypeStr = computed(() => admin.drawerContent?.type as string)

const filteredUsers = computed(() => {
  if (!authorKeyword.value.trim()) return allUsers.value
  const kw = authorKeyword.value.toLowerCase()
  return allUsers.value.filter(u => u.username.toLowerCase().includes(kw))
})

watch(() => admin.drawerOpen, async (open) => {
  if (open && admin.drawerContent) {
    const detail = await admin.fetchContentDetail(admin.drawerContent.id)
    if (detail) admin.drawerContent = detail
    editTitle.value = admin.drawerContent.title
    editContent.value = admin.drawerContent.text || ''
    editUrl.value = admin.drawerContent.url || ''
    editTags.value = [...(admin.drawerContent.tags || [])]
    editFile.value = undefined
    editFileName.value = ''
    tabKey.value = 'preview'
    loadUsers()
  }
})

async function loadUsers() {
  usersLoading.value = true
  try {
    const r = await adminApi.getUsers({ page_size: 200 })
    if (r.code === 200) allUsers.value = r.data.list
  } catch { /* */ } finally { usersLoading.value = false }
}

async function handleSave() {
  if (!admin.drawerContent) return
  const ok = await admin.saveContent(admin.drawerContent.id, {
    title: editTitle.value, content: editContent.value, url: editUrl.value,
    tags: editTags.value, file: editFile.value,
  })
  if (ok) admin.closeDrawer()
}

async function handleChangeAuthor(userId: number, username: string) {
  if (!admin.drawerContent) return
  const { confirm } = useConfirm()
  const ok = await confirm(`确定将作者改为「${username}」吗？`)
  if (!ok) return
  const changed = await admin.changeAuthor(admin.drawerContent.id, userId)
  if (changed) admin.closeDrawer()
}

function setTagChecked(tag: string, checked: boolean) {
  const i = editTags.value.indexOf(tag)
  if (checked && i === -1) editTags.value.push(tag)
  if (!checked && i > -1) editTags.value.splice(i, 1)
}

function onFileChange(_fileList: FileItem[], fileItem: FileItem) {
  const raw = fileItem?.file as File | undefined
  if (!raw) return
  if (!raw.type.startsWith('image/') && !raw.type.startsWith('video/')) {
    toast.error('仅支持图片或视频文件')
    return
  }
  // 与后端保持一致：单文件最大 20MB
  if (raw.size > 20 * 1024 * 1024) {
    toast.error('文件大小不能超过 20MB')
    return
  }
  editFile.value = raw
  editFileName.value = raw.name
}
</script>

<template>
  <a-drawer
    class="content-drawer admin-theme"
    v-model:visible="admin.drawerOpen"
    :title="admin.drawerContent?.title || '内容详情'"
    :width="720"
    @close="admin.closeDrawer"
  >
    <template v-if="admin.drawerContent">
      <a-tabs v-model:active-key="tabKey">
        <a-tab-pane key="preview" title="预览">
          <div class="preview-meta">
            <Tag
              :color="drawerTypeStr === 'video' ? 'red' : drawerTypeStr === 'image' ? 'green' : drawerTypeStr === 'link' ? 'orange' : 'arcoblue'"
              :bordered="false"
              class="drawer-tag-inline"
            >
              {{ { video: '视频', image: '图片', link: '链接', text: '文字' }[drawerTypeStr] }}
            </Tag>
            <AdminStatus
              v-if="admin.drawerContent.audit_status"
              :type="admin.drawerContent.audit_status === 'approved' ? 'success' : admin.drawerContent.audit_status === 'pending' ? 'warning' : 'danger'"
              :label="{ approved: '已通过', pending: '审核中', rejected: '已拒绝' }[admin.drawerContent.audit_status]"
            />
            <span class="preview-meta-info">
              {{ admin.drawerContent.like_count || 0 }} 点赞 · {{ admin.drawerContent.user?.username }}
            </span>
          </div>
          <div class="preview-tags">
            <Tag v-for="tag in admin.drawerContent.tags" :key="tag">{{ tag }}</Tag>
          </div>
          <div v-if="admin.drawerContent.type === 'image'" class="preview-media-wrap">
            <MediaImage :src="admin.drawerContent.img" class="preview-media" :preview="false" alt="" />
          </div>

          <div v-else-if="drawerTypeStr === 'video'" class="preview-media-wrap">
            <video controls class="preview-media">
              <source :src="getImageUrl(admin.drawerContent.video)" />
              <track kind="captions" />
            </video>
          </div>

          <div v-else-if="drawerTypeStr === 'link'" class="preview-media-wrap">
            <a :href="admin.drawerContent.url" target="_blank" rel="noopener">
              <MediaImage
                v-if="admin.drawerContent.thumb"
                :src="admin.drawerContent.thumb"
                class="preview-media"
                :preview="false"
                alt=""
              />
              <div v-else class="link-box">{{ admin.drawerContent.url }}</div>
            </a>
          </div>

          <div v-if="admin.drawerContent.text" class="preview-text" v-html="renderedContent"></div>
        </a-tab-pane>

        <a-tab-pane key="edit" title="编辑">
          <a-form :model="editFormModel" layout="vertical" class="drawer-edit-form">
            <a-form-item label="标题" required>
              <a-input v-model="editTitle" :maxlength="200" show-word-limit placeholder="输入标题" />
            </a-form-item>

            <a-form-item v-if="drawerTypeStr === 'link'" label="链接">
              <a-input v-model="editUrl" placeholder="https://…" />
            </a-form-item>

            <a-form-item label="描述（Markdown，可留空为纯媒体内容）">
              <MarkdownEditor
                v-model="editContent"
                placeholder="支持 Markdown，可留空（纯媒体内容）"
                :height="360"
              />
            </a-form-item>

            <a-form-item label="媒体文件">
              <a-upload
                :auto-upload="false"
                :show-file-list="false"
                :limit="1"
                accept="image/*,video/*"
                @change="onFileChange"
              >
                <template #upload-button>
                  <a-button>
                    <IconUpload />
                    {{ editFileName || '选择图片 / 视频' }}
                  </a-button>
                </template>
              </a-upload>
              <a-typography-text
                v-if="editFileName"
                type="secondary"
                class="drawer-file-name"
              >
                {{ editFileName }}
              </a-typography-text>
              <a-typography-text
                v-else-if="drawerTypeStr === 'image' || drawerTypeStr === 'video'"
                type="secondary"
                class="drawer-file-name"
              >
                当前已有{{ drawerTypeStr === 'image' ? '图片' : '视频' }}，可在预览页查看；选择新文件可替换
              </a-typography-text>
            </a-form-item>

            <a-form-item label="标签">
              <div class="drawer-field-stack">
                <a-input-tag v-model="editTags" placeholder="输入标签后回车" allow-clear class="drawer-tags-input" />
                <div v-if="admin.tags.length" class="tag-pool">
                  <a-tag
                    v-for="tag in admin.tags"
                    :key="tag"
                    checkable
                    :checked="editTags.includes(tag)"
                    :bordered="false"
                    size="small"
                    class="tag-pool-item"
                    @check="setTagChecked(tag, $event)"
                  >
                    {{ tag }}
                  </a-tag>
                </div>
              </div>
            </a-form-item>
          </a-form>
        </a-tab-pane>

        <a-tab-pane key="author" title="修改作者">
          <div class="author-current">
            <span class="author-current-label">当前作者：</span>
            <Tag color="arcoblue" :bordered="false" class="drawer-tag-inline">{{ admin.drawerContent.user?.username }}</Tag>
          </div>
          <a-input v-model="authorKeyword" placeholder="搜索用户名..." allow-clear class="author-search">
            <template #prefix><IconSearch /></template>
          </a-input>
          <a-spin :loading="usersLoading">
            <a-list v-if="filteredUsers.length" :data="filteredUsers" :bordered="false" :split="false" class="user-list">
              <template #item="{ item }">
                <a-list-item class="user-item" @click="handleChangeAuthor(item.id, item.username)">
                  <a-avatar :size="24" :image-url="item.email ? getAvatarUrl(item.email) : ''">
                    <IconUser v-if="!item.email" />
                    <template #error><IconUser /></template>
                  </a-avatar>
                  <span class="admin-cell-title">{{ item.username }}</span>
                  <Tag v-if="item.is_admin" color="arcoblue" size="small" :bordered="false" class="drawer-tag-inline">管理员</Tag>
                </a-list-item>
              </template>
            </a-list>
            <a-empty v-else description="无匹配用户" />
          </a-spin>
        </a-tab-pane>
      </a-tabs>
    </template>

    <template #footer>
      <div class="drawer-footer">
        <a-button @click="admin.closeDrawer">关闭</a-button>
        <a-button
          v-if="tabKey === 'edit'"
          type="primary"
          :loading="admin.drawerSaving"
          @click="handleSave"
        >
          保存修改
        </a-button>
      </div>
    </template>
  </a-drawer>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.content-drawer :deep(.arco-drawer-body) { padding: 16px 24px; }
.preview-meta { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; flex-wrap: wrap; }
.preview-meta-info { font-size: 13px; color: $admin-text-3; font-variant-numeric: tabular-nums; }
.preview-tags { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 16px; }
.preview-media-wrap { display: flex; justify-content: center; margin-bottom: 16px; }
.preview-media { display: block; max-width: 100%; border-radius: 10px; }
.preview-media :deep(.arco-image-img) { max-width: 100%; height: auto; display: block; }
.link-box { padding: 16px; background: $admin-fill; border-radius: 10px; text-align: center; color: $admin-text-3; }
.preview-text { line-height: 1.8; }
.preview-text :deep(h1), .preview-text :deep(h2), .preview-text :deep(h3) { margin: 16px 0 8px; color: $admin-text; }
.preview-text :deep(h1) { font-size: 24px; }
.preview-text :deep(h2) { font-size: 20px; }
.preview-text :deep(h3) { font-size: 16px; }
.preview-text :deep(p) { margin-bottom: 12px; color: $admin-text-2; }
.preview-text :deep(ul), .preview-text :deep(ol) { margin-bottom: 12px; padding-left: 24px; }
.preview-text :deep(blockquote) { border-left: 3px solid $admin-primary; padding-left: 12px; margin: 12px 0; color: $admin-text-3; font-style: italic; }
.preview-text :deep(code) { background: $admin-fill; padding: 2px 6px; border-radius: 4px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
.preview-text :deep(pre) {
  background: #1a1a1a;
  color: #e0e0e0;
  padding: 12px;
  border-radius: 10px;
  overflow-x: auto;
  margin: 12px 0;
}

body[arco-theme='dark'] .preview-text :deep(pre) {
  background: var(--color-bg-3);
  color: var(--color-text-1);
  border: 1px solid var(--color-border);
}
.preview-text :deep(pre code) { background: none; padding: 0; color: inherit; }
.tag-pool {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
  padding: 10px 12px;
  border-radius: 8px;
  background: $admin-fill;
}

.drawer-file-name {
  display: block;
  margin-top: 8px;
}

.drawer-edit-form :deep(.arco-form-item-content) {
  display: block;
  width: 100%;
}

.drawer-field-stack {
  display: block;
  width: 100%;
}

.drawer-tags-input {
  width: 100%;
}

.tag-pool-item {
  margin: 0;
}

.drawer-tag-inline {
  margin: 0;
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.author-current {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 12px;
}

.author-current-label {
  font-size: 13px;
  color: $admin-text-3;
}

.author-search {
  margin-bottom: 12px;
}

.user-list {
  max-height: 320px;
  overflow-y: auto;
}

.user-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s ease;

  &:hover {
    background: $admin-fill;
  }
}
</style>

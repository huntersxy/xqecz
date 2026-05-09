<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { contentApi } from '@/api'
import type { Content, ListParams, User } from '@/types'

function getImageUrl(image: string | undefined, filePath: string | undefined, contentType?: string): string {
  if (image) {
    return image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
  }
  if (filePath) {
    if (contentType === 'video') {
      const thumbPath = filePath.replace(/\.[^.]+$/, '_thumb.webp')
      return `https://xqapi.xiey.work/thumbnails/${thumbPath}`
    }
    return `https://xqapi.xiey.work/uploads/${filePath}`
  }
  return ''
}

function getPreviewText(text: string, maxLength: number = 120): string {
  if (!text) return ''
  const plainText = text
    .replace(/#{1,6}\s/g, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/```[\s\S]*?```/g, '[代码块]')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[-*+]\s/g, '')
    .replace(/\n/g, ' ')
    .trim()
  return plainText.length > maxLength ? plainText.substring(0, maxLength) + '...' : plainText
}

const router = useRouter()
const contents = ref<Content[]>([])
const allTags = ref<string[]>([])
const selectedTags = ref<string[]>([])
const searchKeyword = ref('')
const selectedTypes = ref<string[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const totalPages = ref(1)

const sortedTags = computed(() => {
  return [...allTags.value].sort()
})

const contentTypes = ['video', 'image', 'text']
const contentTypeLabels: Record<string, string> = {
  video: '视频',
  image: '图片',
  text: '图文',
}

function normalizeContent(content: Content | Record<string, unknown>): Content {
  const getVal = (key: keyof Content, altKey: string): unknown => {
    return (content as Record<string, unknown>)[key] ?? (content as Record<string, unknown>)[altKey]
  }

  return {
    id: Number(getVal('id', 'ID')) || 0,
    title: String(getVal('title', 'Title')) || '',
    type: (String(getVal('type', 'Type')) as 'video' | 'image' | 'text') || 'text',
    content: String(getVal('content', 'Content')) || '',
    file_path: String(getVal('file_path', 'FilePath')) || '',
    file_size: Number(getVal('file_size', 'FileSize')) || 0,
    user_id: Number(getVal('user_id', 'UserID')) || 0,
    user: (getVal('user', 'User') as User) || { id: 0, username: '', is_admin: false, is_banned: false, created_at: '', updated_at: '' },
    tags: Array.isArray(getVal('tags', 'Tags')) ? getVal('tags', 'Tags') as string[] : [],
    audit_status: (String(getVal('audit_status', 'AuditStatus')) as 'pending' | 'approved' | 'rejected') || 'pending',
    created_at: String(getVal('created_at', 'CreatedAt')) || '',
    updated_at: String(getVal('updated_at', 'UpdatedAt')) || '',
    image: String(getVal('image', '')) || '',
    video: String(getVal('video', '')) || '',
  }
}

async function loadContents() {
  try {
    let res
    if (searchKeyword.value) {
      res = await contentApi.search(searchKeyword.value, { page: page.value, page_size: pageSize.value })
    } else {
      const params: ListParams = { page: page.value, page_size: pageSize.value }
      if (selectedTags.value.length > 0) params.tag = selectedTags.value.join(',')
      if (selectedTypes.value.length > 0) params.type = selectedTypes.value.join(',')
      res = await contentApi.list(params)
    }
    if (res.code === 200) {
      contents.value = res.data.list.map((item: Record<string, unknown>) => normalizeContent(item))
      total.value = res.data.total
      totalPages.value = res.data.total_page
    }
  } catch (error) {
    console.error('加载内容失败', error)
  }
}

async function loadTags() {
  try {
    const res = await contentApi.getTags()
    if (res.code === 200) {
      allTags.value = res.data
    }
  } catch (error) {
    console.error('加载标签失败', error)
  }
}

function selectTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
  page.value = 1
  loadContents()
}

function selectType(type: string) {
  const index = selectedTypes.value.indexOf(type)
  if (index > -1) {
    selectedTypes.value.splice(index, 1)
  } else {
    selectedTypes.value.push(type)
  }
  page.value = 1
  loadContents()
}

function handleSearch() {
  page.value = 1
  loadContents()
}

function goToDetail(content: Content) {
  if (content.id) {
    router.push(`/content/${content.id}`)
  }
}

function goToPage(p: number) {
  if (p >= 1 && p <= totalPages.value) {
    page.value = p
    loadContents()
  }
}

function goToEasterEgg() {
  router.push('/easter-egg')
}

onMounted(() => {
  loadContents()
  loadTags()
})
</script>

<template>
  <div class="home-container">
    <div class="mac-window main-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">小泉动漫二创站</div>
      </div>

      <div class="content-area">
        <div class="header-section">
          <img src="/F775F3831CE0AAF6B17116666DD812F8.png" alt="小泉动漫二创站" class="app-logo" />
          <a @click="goToEasterEgg" class="app-subtitle egg-link">
            🎉 发现精彩内容
          </a>
        </div>

        <div class="search-section">
          <div class="search-box">
            <input
              v-model="searchKeyword"
              type="text"
              placeholder="搜索内容..."
              @keyup.enter="handleSearch"
              class="mac-input search-input"
            />
            <button @click="handleSearch" class="mac-btn search-btn">搜索</button>
          </div>
        </div>

        <div class="filter-section">
          <div class="filter-group">
            <label class="filter-label">类型</label>
            <div class="type-cloud">
              <button
                v-for="type in contentTypes"
                :key="type"
                @click="selectType(type)"
                :class="['type-cloud-item', { active: selectedTypes.includes(type) }]"
              >
                {{ contentTypeLabels[type] }}
              </button>
            </div>
          </div>

          <div class="filter-group">
            <label class="filter-label">标签云</label>
            <div class="tag-cloud">
              <button
                v-for="tag in sortedTags"
                :key="tag"
                @click="selectTag(tag)"
                :class="['tag-cloud-item', { active: selectedTags.includes(tag) }]"
              >
                {{ tag }}
              </button>
            </div>
          </div>
        </div>

        <div class="content-section">
          <div class="section-header">
            <h2 class="section-title">内容列表</h2>
            <span class="section-count">共 {{ total }} 条</span>
          </div>

          <div class="content-grid">
            <div
              v-for="content in contents"
              :key="content.id || content.ID"
              @click="goToDetail(content)"
              class="content-card mac-card"
            >
              <div class="card-media">
                <template v-if="content.type === 'image'">
                  <img
                    :src="getImageUrl(content.image, content.file_path, content.type)"
                    alt="内容图片"
                    class="card-image"
                  />
                </template>
                <template v-else-if="content.type === 'video'">
                  <img
                    :src="getImageUrl(content.image, content.file_path, content.type)"
                    alt="视频封面"
                    class="card-image"
                  />
                  <div class="play-overlay">
                    <svg class="play-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8 5v14l11-7z"/>
                    </svg>
                  </div>
                </template>
                <template v-else>
                  <div class="text-preview">
                    <p class="preview-text">{{ getPreviewText(content.content || '暂无内容') }}</p>
                  </div>
                </template>
              </div>

              <div class="card-info">
                <h3 class="card-title">{{ content.title }}</h3>
                <div class="card-meta">
                  <span class="meta-item">
                    <svg class="meta-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                      <circle cx="12" cy="7" r="4"/>
                    </svg>
                    {{ content.user?.username }}
                  </span>
                  <div class="tags-wrapper">
                    <span v-for="tag in content.tags" :key="tag" class="meta-tag">
                      {{ tag }}
                    </span>
                  </div>
                </div>
                <div class="card-type">
                  <span :class="['type-badge', content.type]">
                    {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : '文字' }}
                  </span>
                  <span v-if="content.audit_status !== 'approved'" :class="['status-badge', content.audit_status]">
                    {{ content.audit_status === 'pending' ? '审核中' : '已拒绝' }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="contents.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <p>暂无内容</p>
          </div>
        </div>

        <div v-if="totalPages > 1" class="pagination-section">
          <button
            @click="goToPage(page - 1)"
            :disabled="page <= 1"
            class="mac-btn pagination-btn"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7"/>
            </svg>
            上一页
          </button>
          <span class="pagination-info">第 {{ page }} / {{ totalPages }} 页</span>
          <button
            @click="goToPage(page + 1)"
            :disabled="page >= totalPages"
            class="mac-btn pagination-btn"
          >
            下一页
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7"/>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.home-container {
  min-height: 100vh;
  padding: 20px;
  display: flex;
  justify-content: center;
}

.main-window {
  width: 100%;
  max-width: 1200px;
  min-height: 80vh;
  overflow: hidden;
}

.mac-title-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 10px 16px;
  background: linear-gradient(180deg, rgba(0,0,0,0.08) 0%, rgba(0,0,0,0.02) 100%);
  border-radius: 12px 12px 0 0;
}

.window-title {
  margin-left: 16px;
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.content-area {
  padding: 24px;
}

.header-section {
  text-align: center;
  margin-bottom: 32px;
}

.app-logo {
  max-width: 100%;
  height: auto;
  max-height: 120px;
  margin-bottom: 8px;
}

.app-subtitle {
  font-size: 14px;
  color: #888;
  margin: 0;
}

.egg-link {
  display: block;
  width: fit-content;
  margin: 0 auto;
  padding: 8px 16px;
  border-radius: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 16px;
  font-weight: 500;
  text-align: center;
}

.egg-link:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.search-section {
  margin-bottom: 24px;
}

.search-box {
  display: flex;
  gap: 12px;
  max-width: 600px;
  margin: 0 auto;
}

.search-input {
  flex: 1;
  font-size: 14px;
}

.search-btn {
  white-space: nowrap;
}

.filter-section {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 24px;
  padding: 16px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 10px;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-label {
  font-size: 12px;
  color: #666;
  font-weight: 500;
}

.filter-select {
  min-width: 140px;
  font-size: 14px;
}

.tag-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-cloud-item {
  padding: 4px 12px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 20px;
  font-size: 13px;
  color: #555;
  cursor: pointer;
  transition: all 0.2s ease;
}

@supports (backdrop-filter: blur(10px)) or (-webkit-backdrop-filter: blur(10px)) {
  .tag-cloud-item {
    background: rgba(255, 255, 255, 0.8);
    -webkit-backdrop-filter: blur(10px);
    backdrop-filter: blur(10px);
  }
}

.tag-cloud-item:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

.tag-cloud-item.active {
  background: rgba(59, 130, 246, 0.15);
  color: #3b82f6;
  font-weight: 500;
  border-color: rgba(59, 130, 246, 0.3);
}

.type-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.type-cloud-item {
  padding: 4px 12px;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 20px;
  font-size: 13px;
  color: #555;
  cursor: pointer;
  transition: all 0.2s ease;
}

@supports (backdrop-filter: blur(10px)) or (-webkit-backdrop-filter: blur(10px)) {
  .type-cloud-item {
    background: rgba(255, 255, 255, 0.8);
    -webkit-backdrop-filter: blur(10px);
    backdrop-filter: blur(10px);
  }
}

.type-cloud-item:hover {
  background: rgba(255, 255, 255, 0.95);
  color: #059669;
  border-color: rgba(5, 150, 105, 0.3);
}

.type-cloud-item.active {
  background: rgba(5, 150, 105, 0.15);
  color: #059669;
  font-weight: 500;
  border-color: rgba(5, 150, 105, 0.3);
}

.content-section {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.section-count {
  font-size: 13px;
  color: #888;
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

.content-card {
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.content-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.card-media {
  position: relative;
  width: 100%;
  padding-top: 62.5%;
  background: #f8f9fa;
  overflow: hidden;
}

.card-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.card-video {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.play-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 48px;
  height: 48px;
  background: rgba(0, 0, 0, 0.6);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.play-icon {
  width: 20px;
  height: 20px;
  color: white;
  margin-left: 3px;
}

.text-preview {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  padding: 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.preview-text {
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-info {
  padding: 16px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 10px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #888;
}

.meta-icon {
  width: 14px;
  height: 14px;
}

.tags-wrapper {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.meta-tag {
  padding: 2px 6px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  font-size: 11px;
  color: #666;
}

.card-type {
  display: flex;
  gap: 8px;
}

.type-badge,
.status-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

.type-badge.video {
  background: #fff0e6;
  color: #d97706;
}

.type-badge.image {
  background: #ecfdf5;
  color: #059669;
}

.type-badge.text {
  background: #f0f9ff;
  color: #0284c7;
}

.status-badge.approved {
  background: #ecfdf5;
  color: #059669;
}

.status-badge.pending {
  background: #fef3c7;
  color: #d97706;
}

.status-badge.rejected {
  background: #fef2f2;
  color: #dc2626;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #999;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}

.pagination-section {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.pagination-btn {
  display: flex;
  align-items: center;
  gap: 6px;
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination-btn svg {
  width: 14px;
  height: 14px;
}

.pagination-info {
  font-size: 14px;
  color: #666;
}

@media screen and (max-width: 768px) {
  .home-container {
    padding: 0;
    min-height: calc(100vh - 60px);
  }

  .main-window {
    border-radius: 0;
    box-shadow: none;
    min-height: calc(100vh - 60px);
    background: rgba(255, 255, 255, 0.9);
  }

  .mac-title-bar {
    display: none;
  }

  .content-area {
    padding: 16px;
  }

  .header-section {
    margin-bottom: 20px;
  }

  .app-logo {
    max-height: 80px;
  }

  .app-subtitle {
    font-size: 13px;
  }

  .search-box {
    flex-direction: column;
    gap: 10px;
  }

  .search-input {
    min-height: 48px;
    font-size: 15px;
  }

  .search-btn {
    min-height: 48px;
    font-size: 15px;
  }

  .filter-section {
    flex-direction: column;
    gap: 12px;
    padding: 14px;
  }

  .filter-group {
    width: 100%;
  }

  .filter-label {
    font-size: 13px;
  }

  .tag-cloud-item,
  .type-cloud-item {
    padding: 6px 14px;
    font-size: 14px;
  }

  .content-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .content-card {
    background: rgba(255, 255, 255, 0.95);
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  }

  .card-media {
    padding-top: 56.25%;
  }

  .play-overlay {
    width: 40px;
    height: 40px;
  }

  .play-icon {
    width: 16px;
    height: 16px;
  }

  .card-info {
    padding: 12px;
  }

  .card-title {
    font-size: 14px;
    margin-bottom: 6px;
  }

  .card-meta {
    gap: 6px;
    margin-bottom: 6px;
  }

  .meta-item {
    font-size: 11px;
  }

  .meta-icon {
    width: 12px;
    height: 12px;
  }

  .meta-tag {
    padding: 1px 5px;
    font-size: 10px;
  }

  .type-badge,
  .status-badge {
    padding: 2px 6px;
    font-size: 10px;
  }

  .section-header {
    margin-bottom: 12px;
  }

  .section-title {
    font-size: 16px;
  }

  .section-count {
    font-size: 12px;
  }

  .pagination-section {
    gap: 12px;
    padding-top: 12px;
  }

  .pagination-btn {
    min-width: 80px;
    padding: 10px 12px;
  }

  .pagination-info {
    font-size: 13px;
  }

  .empty-state {
    padding: 40px 16px;
  }

  .empty-icon {
    width: 48px;
    height: 48px;
  }
}

@media screen and (max-width: 480px) {
  .content-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .content-card {
    background: rgba(255, 255, 255, 0.98);
  }

  .card-info {
    padding: 10px;
  }

  .card-title {
    font-size: 13px;
  }
}
</style>

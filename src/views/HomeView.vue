<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { motion, AnimatePresence } from 'motion-v'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import type { Content, ListParams, User, RecommendContent } from '@/types'
import PollComponent from '@/components/PollComponent.vue'

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

function getImageUrl(
  image: string | undefined,
  filePath: string | undefined,
  contentType?: string,
): string {
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
const homeStore = useHomeStore()

const contents = ref<Content[]>([])
const recommendContents = ref<RecommendContent[]>([])
const allTags = ref<string[]>([])

// 从 store 初始化状态
const selectedTags = ref<string[]>(homeStore.selectedTags)
const searchKeyword = ref(homeStore.searchKeyword)
const selectedTypes = ref<string[]>(homeStore.selectedTypes)
const page = ref(homeStore.page)
const recommendPage = ref(homeStore.recommendPage)

const pageSize = ref(12)
const total = ref(0)
const totalPages = ref(1)
const recommendHint = ref('')
const recommendPerPage = 16
const recommendTotal = 100
const maxRecommendPages = Math.floor(recommendTotal / recommendPerPage)
const swapSections = ref(false)

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

  const rawUser = getVal('user', 'User') as Record<string, unknown> | undefined
  const normalizeUser = (u: Record<string, unknown> | undefined): User => {
    if (!u)
      return {
        id: 0,
        username: '',
        is_admin: false,
        is_banned: false,
        created_at: '',
        updated_at: '',
      }
    const getUserVal = (key: string, altKey: string): unknown => {
      return u[key] ?? u[altKey]
    }
    return {
      id: Number(getUserVal('id', 'ID')) || 0,
      username: String(getUserVal('username', 'Username')) || '',
      is_admin: Boolean(getUserVal('is_admin', 'IsAdmin')) || false,
      is_banned: Boolean(getUserVal('is_banned', 'IsBanned')) || false,
      created_at: String(getUserVal('created_at', 'CreatedAt')) || '',
      updated_at: String(getUserVal('updated_at', 'UpdatedAt')) || '',
    }
  }

  return {
    id: Number(getVal('id', 'ID')) || 0,
    title: String(getVal('title', 'Title')) || '',
    type: (String(getVal('type', 'Type')) as 'video' | 'image' | 'text') || 'text',
    content: String(getVal('content', 'Content')) || '',
    file_path: String(getVal('file_path', 'FilePath')) || '',
    file_size: Number(getVal('file_size', 'FileSize')) || 0,
    user_id: Number(getVal('user_id', 'UserID')) || 0,
    user: normalizeUser(rawUser),
    tags: Array.isArray(getVal('tags', 'Tags')) ? (getVal('tags', 'Tags') as string[]) : [],
    audit_status:
      (String(getVal('audit_status', 'AuditStatus')) as 'pending' | 'approved' | 'rejected') ||
      'pending',
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
      res = await contentApi.search(searchKeyword.value, {
        page: page.value,
        page_size: pageSize.value,
      })
    } else {
      const params: ListParams = {
        page: page.value,
        page_size: pageSize.value,
        sort_by: 'created_at',
        order: 'desc',
      }
      if (selectedTags.value.length > 0) params.tag = selectedTags.value.join(',')
      if (selectedTypes.value.length > 0) params.type = selectedTypes.value.join(',')
      res = await contentApi.list(params)
    }
    if (res.code === 200) {
      contents.value = res.data.list.map((item: Record<string, unknown>) => normalizeContent(item))
      total.value = res.data.total
      totalPages.value = res.data.total_page
      swapSections.value =
        selectedTags.value.length > 0 ||
        selectedTypes.value.length > 0 ||
        !!searchKeyword.value.trim()
    }
  } catch (error) {
    console.error('加载内容失败', error)
  }
}

async function loadTags() {
  try {
    const res = await contentApi.getTags()
    if (res.code === 200) {
      const tags = res.data as string[]
      const shuffled = [...tags].sort(() => Math.random() - 0.5)
      allTags.value = shuffled.slice(0, 15)
    }
  } catch (error) {
    console.error('加载标签失败', error)
  }
}

function selectTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value = []
  } else {
    selectedTags.value = [tag]
  }
  page.value = 1
  loadContents()
}

function selectType(type: string) {
  const index = selectedTypes.value.indexOf(type)
  if (index > -1) {
    selectedTypes.value = []
  } else {
    selectedTypes.value = [type]
  }
  page.value = 1
  loadContents()
}

function handleSearch() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  searchDebounceTimer = setTimeout(() => {
    page.value = 1
    loadContents()
  }, 300)
}

function goToDetail(content: Content) {
  if (content.id) {
    // 保存当前滚动位置
    homeStore.saveState({
      searchKeyword: searchKeyword.value,
      selectedTags: selectedTags.value,
      selectedTypes: selectedTypes.value,
      page: page.value,
      recommendPage: recommendPage.value,
      scrollPosition: window.scrollY,
    })
    router.push(`/content/${content.id}`)
  }
}

// 路由离开前保存状态
onBeforeRouteLeave((to, from, next) => {
  // 只有在跳转到详情页时才保存状态
  if (to.path.startsWith('/content/')) {
    homeStore.saveState({
      searchKeyword: searchKeyword.value,
      selectedTags: selectedTags.value,
      selectedTypes: selectedTypes.value,
      page: page.value,
      recommendPage: recommendPage.value,
      scrollPosition: window.scrollY,
    })
  } else if (to.path === '/') {
    // 如果是返回首页，不保存状态，让首页恢复时使用已有状态
  } else {
    // 跳转到其他页面（非详情页、非首页）时清除状态
    homeStore.clearState()
  }
  next()
})

function goToPage(p: number) {
  if (p >= 1 && p <= totalPages.value) {
    page.value = p
    loadContents()
  }
}

function goToEasterEgg() {
  router.push('/easter-egg')
}

function normalizeRecommendContent(content: RecommendContent): RecommendContent {
  return {
    id: Number(content.id) || Number(content.ID) || 0,
    title: content.title || content.Title || '',
    type: ((content.type || content.Type) as 'video' | 'image' | 'text') || 'image',
    file_path: content.file_path || content.FilePath || '',
    image: content.image || '',
    tags: Array.isArray(content.tags)
      ? content.tags
      : Array.isArray(content.Tags)
        ? content.Tags
        : [],
    view_count: content.view_count,
    user: {
      id: Number(content.user?.id) || Number(content.User?.ID) || 0,
      username: content.user?.username || content.User?.Username || '',
    },
    created_at: content.created_at || content.CreatedAt || '',
  }
}

async function loadRecommendContents() {
  try {
    const res = await contentApi.recommend(recommendPerPage, recommendPage.value)
    if (res.code === 200) {
      recommendContents.value = res.data.list.map((item: RecommendContent) =>
        normalizeRecommendContent(item),
      )
      recommendHint.value = ''
    }
  } catch (error) {
    console.error('加载推荐内容失败', error)
  }
}

function refreshRecommend() {
  recommendPage.value++
  if (recommendPage.value > maxRecommendPages) {
    recommendPage.value = 1
  }
  loadRecommendContents()
}

onMounted(() => {
  // 检测是否是页面刷新（通过 performance.getEntriesByType）
  const navigationEntries = performance.getEntriesByType(
    'navigation',
  ) as PerformanceNavigationTiming[]
  const isReload = navigationEntries.length > 0 && navigationEntries[0].type === 'reload'

  // 如果是页面刷新，清除状态
  if (isReload) {
    homeStore.clearState()
  }

  // 如果有保存的状态，恢复滚动位置
  if (homeStore.hasLoaded) {
    nextTick(() => {
      homeStore.restoreScroll()
    })
  }
  loadContents()
  loadTags()
  loadRecommendContents()
})
</script>

<template>
  <div class="home-container">
    <div class="main-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">小泉动漫二创站</div>
      </div>

      <div class="content-area">
        <div class="header-section">
          <img src="/F775F3831CE0AAF6B17116666DD812F8.png" alt="小泉动漫二创站" class="app-logo" />
          <a @click="goToEasterEgg" class="app-subtitle egg-link"> 🎉 发现精彩内容 </a>
        </div>

        <div class="search-section">
          <div class="search-box">
            <input
              v-model="searchKeyword"
              type="text"
              placeholder="搜索内容..."
              @keyup.enter="handleSearch"
              class="search-input"
            />
            <button @click="handleSearch" class="search-btn">搜索</button>
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

        <!-- 最新投票 -->
        <div class="poll-section">
          <div class="section-header">
            <h2 class="section-title">📊 最新投票</h2>
          </div>
          <PollComponent />
        </div>

        <motion.div
          class="sections-container"
          :animate="{
            height: 'auto',
            transition: { duration: 0.5, ease: [0.25, 0.1, 0.25, 1] },
          }"
          style="overflow: hidden; min-height: 100px"
        >
          <motion.div
            class="sections-wrapper"
            :animate="{
              transition: { duration: 0.5, ease: [0.25, 0.1, 0.25, 1] },
            }"
          >
            <motion.div
              id="recommend-section"
              class="recommend-section"
              :initial="{ opacity: 1, y: 0 }"
              :animate="{
                opacity: swapSections ? 0 : 1,
                y: swapSections ? -100 : 0,
                height: swapSections ? 0 : 'auto',
                transition: { duration: 0.5, ease: [0.25, 0.1, 0.25, 1] },
              }"
              style="overflow: hidden"
            >
              <div class="section-header">
                <h2 class="section-title">🔥 推荐内容</h2>
                <div class="recommend-actions">
                  <span class="section-count">精选推荐</span>
                  <button @click="refreshRecommend" class="refresh-btn">
                    <svg
                      class="refresh-icon"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <polyline points="23 4 23 10 17 10" />
                      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
                    </svg>
                    刷新
                  </button>
                </div>
                <span v-if="recommendHint" class="recommend-hint">{{ recommendHint }}</span>
              </div>

              <div class="recommend-grid">
                <AnimatePresence>
                  <motion.div
                    v-for="content in recommendContents"
                    :key="content.id"
                    @click="
                      goToDetail({
                        ...content,
                        type: content.type,
                        audit_status: 'approved',
                      } as unknown as Content)
                    "
                    class="recommend-card"
                    :initial="{ opacity: 0, y: 30, scale: 0.9 }"
                    :animate="{ opacity: 1, y: 0, scale: 1 }"
                    :exit="{ opacity: 0, scale: 0 }"
                    :transition="{
                      duration: 0.35,
                      ease: [0.55, 0.055, 0.675, 0.19],
                    }"
                  >
                    <div class="recommend-media">
                      <img
                        :src="
                          content.image.replace(
                            /http:\/\/localhost:8080/,
                            'https://xqapi.xiey.work',
                          )
                        "
                        :alt="content.title"
                        class="recommend-image"
                        loading="lazy"
                      />
                      <div v-if="content.type === 'video'" class="play-overlay">
                        <svg class="play-icon" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M8 5v14l11-7z" />
                        </svg>
                      </div>
                    </div>
                    <div class="recommend-info">
                      <h3 class="recommend-title">{{ content.title }}</h3>
                      <div class="recommend-meta">
                        <span class="recommend-author">{{ content.user.username }}</span>
                        <span class="recommend-tags">
                          <span
                            v-for="tag in content.tags.slice(0, 2)"
                            :key="tag"
                            class="recommend-tag"
                            >{{ tag }}</span
                          >
                        </span>
                      </div>
                    </div>
                  </motion.div>
                </AnimatePresence>
              </div>
            </motion.div>

            <motion.div
              class="content-section"
              :initial="{ opacity: 1, y: 0 }"
              :animate="{
                opacity: 1,
                y: swapSections ? -20 : 0,
                transition: { duration: 0.5, ease: [0.25, 0.1, 0.25, 1] },
              }"
            >
              <div class="section-header">
                <h2 class="section-title">📅 最近上传</h2>
                <span class="section-count">共 {{ total }} 条</span>
              </div>

              <div class="content-grid">
                <AnimatePresence>
                  <motion.div
                    v-for="content in contents"
                    :key="content.id || content.ID"
                    @click="goToDetail(content)"
                    class="content-card"
                    :initial="{ opacity: 0, y: 30, scale: 0.9 }"
                    :animate="{ opacity: 1, y: 0, scale: 1 }"
                    :exit="{ opacity: 0, scale: 0 }"
                    :transition="{
                      duration: 0.35,
                      ease: [0.55, 0.055, 0.675, 0.19],
                    }"
                  >
                    <div class="card-media">
                      <template v-if="content.type === 'image'">
                        <img
                          :src="getImageUrl(content.image, content.file_path, content.type)"
                          alt="内容图片"
                          class="card-image"
                          loading="lazy"
                        />
                      </template>
                      <template v-else-if="content.type === 'video'">
                        <img
                          :src="getImageUrl(content.image, content.file_path, content.type)"
                          alt="视频封面"
                          class="card-image"
                          loading="lazy"
                        />
                        <div class="play-overlay">
                          <svg class="play-icon" viewBox="0 0 24 24" fill="currentColor">
                            <path d="M8 5v14l11-7z" />
                          </svg>
                        </div>
                      </template>
                      <template v-else>
                        <div class="text-preview">
                          <p class="preview-text">
                            {{ getPreviewText(content.content || '暂无内容') }}
                          </p>
                        </div>
                      </template>
                    </div>
                    <div class="card-info">
                      <h3 class="card-title">{{ content.title }}</h3>
                      <div class="card-meta">
                        <span class="meta-item">
                          <svg
                            class="meta-icon"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                          >
                            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                            <circle cx="12" cy="7" r="4" />
                          </svg>
                          {{ content.user?.username }}
                        </span>
                        <div class="tags-wrapper">
                          <span v-for="tag in content.tags" :key="tag" class="meta-tag">{{
                            tag
                          }}</span>
                        </div>
                      </div>
                      <div class="card-type">
                        <span :class="['type-badge', content.type]">
                          {{
                            content.type === 'video'
                              ? '视频'
                              : content.type === 'image'
                                ? '图片'
                                : '文字'
                          }}
                        </span>
                        <span
                          v-if="content.audit_status !== 'approved'"
                          :class="['status-badge', content.audit_status]"
                        >
                          {{ content.audit_status === 'pending' ? '审核中' : '已拒绝' }}
                        </span>
                      </div>
                    </div>
                  </motion.div>
                </AnimatePresence>
              </div>
              <div v-if="contents.length === 0" class="empty-state">
                <svg
                  class="empty-icon"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.5"
                >
                  <circle cx="12" cy="12" r="10" />
                  <polyline points="12 6 12 12 16 14" />
                </svg>
                <p>暂无内容</p>
              </div>
            </motion.div>
          </motion.div>
        </motion.div>

        <div v-if="totalPages > 1" class="pagination-section">
          <button @click="goToPage(page - 1)" :disabled="page <= 1" class="pagination-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7" />
            </svg>
            上一页
          </button>
          <span class="pagination-info">第 {{ page }} / {{ totalPages }} 页</span>
          <button @click="goToPage(page + 1)" :disabled="page >= totalPages" class="pagination-btn">
            下一页
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7" />
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
  min-height: 100vh;
  overflow: hidden;
  transition: min-height 0.5s ease;
  background: rgba(255, 255, 255, 0.75);
  border-radius: 12px;
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.08),
    0 2px 8px rgba(0, 0, 0, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.4);
}

.mac-title-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 10px 16px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.08) 0%, rgba(0, 0, 0, 0.02) 100%);
  border-radius: 12px 12px 0 0;
}

.mac-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.mac-dot-red {
  background: #ff5f57;
}

.mac-dot-yellow {
  background: #febc2e;
}

.mac-dot-green {
  background: #28c840;
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
  padding: 10px 16px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  background: white;
  outline: none;
  transition:
    border-color 0.2s ease,
    box-shadow 0.2s ease;
}

.search-input:focus {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-btn {
  padding: 10px 20px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.search-btn:hover {
  background: #2563eb;
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

.sections-container {
  position: relative;
}

.poll-section {
  margin-bottom: 24px;
}

.recommend-section {
  margin-bottom: 24px;
}

.recommend-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.recommend-card {
  overflow: hidden;
  cursor: pointer;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
}

.recommend-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

.recommend-media {
  position: relative;
  width: 100%;
  padding-top: 56.25%;
  background: #f8f9fa;
  overflow: hidden;
}

.recommend-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.recommend-info {
  padding: 12px;
}

.recommend-title {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 8px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recommend-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.recommend-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.refresh-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 13px;
  color: #555;
  cursor: pointer;
  transition: all 0.2s ease;
}

.refresh-btn:hover {
  background: #f0f0f0;
  border-color: #3b82f6;
  color: #3b82f6;
}

.refresh-icon {
  width: 14px;
  height: 14px;
}

.recommend-hint {
  position: absolute;
  right: 24px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: #ef4444;
  animation: fadeInOut 2s ease;
}

@keyframes fadeInOut {
  0%,
  100% {
    opacity: 0;
  }
  10%,
  90% {
    opacity: 1;
  }
}

.recommend-author {
  font-size: 12px;
  color: #888;
}

.recommend-tags {
  display: flex;
  gap: 4px;
}

.recommend-tag {
  padding: 1px 6px;
  background: rgba(251, 146, 60, 0.15);
  border-radius: 4px;
  font-size: 11px;
  color: #fb923c;
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
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease;
  background: rgba(255, 255, 255, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
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
  padding: 8px 16px;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 14px;
  color: #333;
  cursor: pointer;
  transition: all 0.2s ease;
}

.pagination-btn:hover:not(:disabled) {
  background: #f8f9fa;
  border-color: #3b82f6;
  color: #3b82f6;
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

  .recommend-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .recommend-card {
    background: rgba(255, 255, 255, 0.95);
    border-radius: 12px;
  }

  .recommend-info {
    padding: 10px;
  }

  .recommend-title {
    font-size: 13px;
    margin-bottom: 6px;
  }

  .recommend-author {
    font-size: 11px;
  }

  .recommend-tag {
    padding: 1px 5px;
    font-size: 10px;
  }

  .refresh-btn {
    padding: 4px 10px;
    font-size: 12px;
  }

  .refresh-icon {
    width: 12px;
    height: 12px;
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
  .recommend-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .recommend-card {
    background: rgba(255, 255, 255, 0.98);
  }

  .recommend-info {
    padding: 8px;
  }

  .recommend-title {
    font-size: 12px;
  }

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

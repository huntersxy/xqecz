<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { contentApi } from '@/api'
import { useRouter } from 'vue-router'
import { getImageUrl } from '@/utils'
import logoImg from '@/assets/logo.webp'
import type { Content, RecommendContent } from '@/types'

const router = useRouter()
const contents = ref<Content[]>([])
const recommendContents = ref<RecommendContent[]>([])
const allTags = ref<string[]>([])
const activeTag = ref<string>('')
const loading = ref(true)
const loadMoreLoading = ref(false)
const page = ref(1)
const total = ref(0)
const totalPages = ref(1)
const allLoaded = computed(() => page.value >= totalPages.value)
const showMore = ref(false)
const maxVisibleTags = 18
const windowWidth = ref(window.innerWidth)
const observerTarget = ref<HTMLDivElement | null>(null)
let observer: IntersectionObserver | null = null

const displayedTags = computed(() => {
  if (showMore.value) {
    return allTags.value
  }
  return allTags.value.slice(0, maxVisibleTags)
})

const hasMoreTags = computed(() => allTags.value.length > maxVisibleTags)

function toggleMore() {
  showMore.value = !showMore.value
}

async function loadData() {
  try {
    loading.value = true
    const [contentsRes, recommendRes, tagsRes] = await Promise.all([
      contentApi.list({ page: 1, page_size: 12, sort_by: 'created_at', order: 'desc' }),
      contentApi.recommend(20, 1),
      contentApi.getTags()
    ])
    const listData = contentsRes.data
    contents.value = listData?.list || []
    total.value = listData?.total || 0
    totalPages.value = listData?.total_page || 1
    page.value = 1
    recommendContents.value = recommendRes.data?.list || []
    allTags.value = tagsRes.data || []
  } catch (error) {
    console.error('加载失败:', error)
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadMoreLoading.value || allLoaded.value) return
  try {
    loadMoreLoading.value = true
    const nextPage = page.value + 1
    const res = await contentApi.list({ page: nextPage, page_size: 12, sort_by: 'created_at', order: 'desc' })
    const list = res.data?.list || []
    contents.value.push(...list)
    page.value = nextPage
    totalPages.value = res.data?.total_page || totalPages.value
    total.value = res.data?.total || total.value
  } catch (error) {
    console.error('加载更多失败:', error)
  } finally {
    loadMoreLoading.value = false
  }
}

function setupObserver() {
  if (!observerTarget.value) return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0].isIntersecting && !loadMoreLoading.value && !allLoaded.value) {
        loadMore()
      }
    },
    { rootMargin: '200px' }
  )
  observer.observe(observerTarget.value)
}

function selectTag(tag: string) {
  activeTag.value = activeTag.value === tag ? '' : tag
}

function formatNumber(num: number): string {
  if (num >= 10000) {
    return (num / 10000).toFixed(1) + '万'
  }
  return num.toString()
}

function formatDate(timestamp: number): string {
  const d = new Date(timestamp * 1000)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  
  if (days === 0) return '今天'
  if (days === 1) return '昨天'
  if (days < 7) return `${days}天前`
  if (days < 30) return `${Math.floor(days / 7)}周前`
  return `${Math.floor(days / 30)}个月前`
}

function goToContent(id: number) {
  router.push(`/content/${id}`)
}

onMounted(() => {
  loadData().then(() => {
    setupObserver()
  })
  window.addEventListener('resize', () => {
    windowWidth.value = window.innerWidth
  })
})

onUnmounted(() => {
  if (observer) observer.disconnect()
})
</script>

<template>
  <div class="bilibili-theme min-h-screen">
    <!-- 顶部横幅 -->
    <div class="h-32 flex items-center justify-center">
      <img :src="logoImg" alt="小泉动漫二创站" class="max-w-full h-auto max-h-[80px]" />
    </div>

    <!-- 分类标签 -->
    <div class="bg-white/90 backdrop-blur-md max-w-[1400px] mx-auto w-full rounded-lg mb-4 border border-gray-200/50">
      <div class="px-4 py-3">
        <div class="flex flex-wrap gap-2">
          <button
            v-for="tag in displayedTags"
            :key="tag"
            @click="selectTag(tag)"
            class="px-3 py-1.5 bg-gray-100 border border-gray-200 rounded-md text-sm text-gray-600 transition-all cursor-pointer hover:text-pink-500 hover:border-pink-500/30"
            :class="activeTag === tag ? 'bg-pink-500/10 text-pink-500 font-medium border-pink-500/30' : ''">
            {{ tag }}
          </button>
          <button
            v-if="hasMoreTags"
            @click="toggleMore"
            class="px-3 py-1.5 bg-gray-100 border border-gray-200 rounded-md text-sm text-gray-600 transition-all cursor-pointer hover:text-pink-500 hover:border-pink-500/30 flex items-center gap-1">
            {{ showMore ? '收起' : '更多' }}
            <svg class="w-3 h-3 transition-transform" :class="{ 'rotate-180': showMore }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M6 9l6 6 6-6"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- 内容区 -->
    <div class="max-w-[1400px] mx-auto px-4 py-6">
      <div v-if="loading" class="flex justify-center py-20">
        <div class="w-10 h-10 border-3 border-pink-200 border-t-pink-500 rounded-full animate-spin"></div>
      </div>

      <template v-else>
        <!-- 推荐区：大图+小卡 -->
        <div v-if="recommendContents.length > 0" class="flex gap-4 mb-6">
          <div
            @click="goToContent(recommendContents[0].id)"
            class="w-1/2 min-[1367px]:w-2/5 relative rounded-xl overflow-hidden cursor-pointer group">
            <img :src="getImageUrl(recommendContents[0].thumb)" class="absolute inset-0 w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" />
            <div class="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent"></div>
            <div class="absolute bottom-0 left-0 right-0 p-6">
              <h3 class="text-white text-xl font-bold mb-2 line-clamp-2">{{ recommendContents[0].title }}</h3>
              <div class="flex items-center gap-3 text-white/70 text-xs">
                <span>{{ recommendContents[0].user?.username }}</span>
                <span>{{ formatNumber(recommendContents[0].view_count || 0) }} 次观看</span>
              </div>
            </div>
          </div>
          <div class="w-1/2 min-[1367px]:w-3/5 grid grid-cols-2 min-[1367px]:grid-cols-3 gap-4">
            <div
              v-for="item in recommendContents.slice(1, windowWidth >= 1367 ? 7 : 5)"
              :key="item.id"
              @click="goToContent(item.id)"
              class="bg-white rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer group">
              <div class="relative aspect-video overflow-hidden">
                <img :src="getImageUrl(item.thumb)" class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" />
                <div class="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-0.5 rounded">
                  {{ item.type === 'video' ? '视频' : item.type === 'image' ? '图片' : item.type === 'link' ? '链接' : '文字' }}
                </div>
              </div>
              <div class="p-3">
                <h4 class="text-sm font-medium text-gray-800 line-clamp-2 mb-2 group-hover:text-pink-500 transition-colors">{{ item.title }}</h4>
                <div class="flex items-center justify-between text-xs text-gray-500">
                  <span class="flex items-center gap-1">
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
                    {{ formatNumber(item.view_count || 0) }}
                  </span>
                  <span>{{ formatDate(item.created_at) }}</span>
                </div>
                <div class="mt-2 flex items-center justify-between text-xs text-gray-400">
                  <span>{{ item.user?.username }}</span>
                  <div class="flex gap-1">
                    <span v-for="tag in item.tags?.slice(0, 2)" :key="tag" class="px-1 py-0.5 bg-gray-100 rounded text-xs">{{ tag }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 推荐区：两行网格 -->
        <div v-if="recommendContents.length > (windowWidth >= 1367 ? 7 : 5)" class="grid grid-cols-4 min-[1367px]:grid-cols-5 gap-4 mb-6">
          <div
            v-for="item in recommendContents.slice(windowWidth >= 1367 ? 7 : 5, windowWidth >= 1367 ? 17 : 13)"
            :key="item.id"
            @click="goToContent(item.id)"
            class="bg-white rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer group">
            <div class="relative aspect-video overflow-hidden">
              <img :src="getImageUrl(item.thumb)" class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" />
              <div class="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-0.5 rounded">
                {{ item.type === 'video' ? '视频' : item.type === 'image' ? '图片' : item.type === 'link' ? '链接' : '文字' }}
              </div>
            </div>
            <div class="p-3">
              <h4 class="text-sm font-medium text-gray-800 line-clamp-2 mb-2 group-hover:text-pink-500 transition-colors">{{ item.title }}</h4>
              <div class="flex items-center justify-between text-xs text-gray-500">
                <span class="flex items-center gap-1">
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
                  {{ formatNumber(item.view_count || 0) }}
                </span>
                <span>{{ formatDate(item.created_at) }}</span>
              </div>
              <div class="mt-2 flex items-center justify-between text-xs text-gray-400">
                <span>{{ item.user?.username }}</span>
                <div class="flex gap-1">
                  <span v-for="tag in item.tags?.slice(0, 2)" :key="tag" class="px-1 py-0.5 bg-gray-100 rounded text-xs">{{ tag }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 间隔 -->
        <div class="flex items-center justify-between mb-3 mt-8">
          <h2 class="text-base font-semibold text-gray-800">最近上传</h2>
          <span class="text-xs text-gray-400">共 {{ total }} 条</span>
        </div>

        <!-- 网格内容 -->
        <div class="grid grid-cols-4 min-[1367px]:grid-cols-5 gap-4">
          <div
            v-for="item in contents"
            :key="item.id"
            @click="goToContent(item.id)"
            class="bg-white rounded-lg overflow-hidden shadow-sm hover:shadow-md transition-shadow cursor-pointer group">
            <div class="relative aspect-video overflow-hidden">
              <img :src="getImageUrl(item.thumb)" class="w-full h-full object-cover transition-transform duration-300 group-hover:scale-105" loading="lazy" />
              <div class="absolute bottom-2 right-2 bg-black/70 text-white text-xs px-2 py-0.5 rounded">
                {{ item.type === 'video' ? '视频' : item.type === 'image' ? '图片' : item.type === 'link' ? '链接' : '文字' }}
              </div>
            </div>
            <div class="p-3">
              <h4 class="text-sm font-medium text-gray-800 line-clamp-2 mb-2 group-hover:text-pink-500 transition-colors">{{ item.title }}</h4>
              <div class="flex items-center justify-between text-xs text-gray-500">
                <span class="flex items-center gap-1">
                  <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><path d="M12 4.5C7 4.5 2.73 7.61 1 12c1.73 4.39 6 7.5 11 7.5s9.27-3.11 11-7.5c-1.73-4.39-6-7.5-11-7.5zM12 17c-2.76 0-5-2.24-5-5s2.24-5 5-5 5 2.24 5 5-2.24 5-5 5zm0-8c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z"/></svg>
                  {{ formatNumber(item.view_count) }}
                </span>
                <span>{{ formatDate(item.created_at) }}</span>
              </div>
              <div class="mt-2 text-xs text-gray-400">{{ item.user?.username }}</div>
            </div>
          </div>
        </div>

        <!-- 无限滚动触发区 -->
        <div ref="observerTarget" class="flex justify-center py-8">
          <div v-if="loadMoreLoading" class="w-8 h-8 border-3 border-pink-200 border-t-pink-500 rounded-full animate-spin"></div>
          <span v-else-if="allLoaded" class="text-sm text-gray-400">— 已全部加载 —</span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.bilibili-theme {
  background: transparent !important;
}

.bilibili-theme :deep(.theme-card) {
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
}

.bilibili-theme :deep(.theme-glass-layer) {
  display: none !important;
}

.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>

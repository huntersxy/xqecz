<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRouter, onBeforeRouteLeave } from 'vue-router'
import { motion } from 'motion-v'
import logoImg from '@/assets/logo.webp'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'
import type { Content, ListParams, User, RecommendContent } from '@/types'
import PollComponent from '@/components/PollComponent.vue'
import ThemeSelector from '@/components/ThemeSelector.vue'

let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

function getImageUrl(image?: string): string {
  if (image) {
    let url = image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
    if (url.startsWith('/')) {
      url = `https://xqapi.xiey.work${url}`
    }
    return url
  }
  return ''
}

function onImageLoad(e: Event) {
  const img = e.target as HTMLImageElement
  if (img.naturalHeight > img.naturalWidth * 1.2) {
    img.style.objectPosition = '50% 8%'
  }
}

function formatTime(ts: number): string {
  if (!ts) return ''
  return new Date(ts * 1000).toLocaleDateString('zh-CN')
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

const selectedTags = ref<string[]>(homeStore.selectedTags)
const searchKeyword = ref(homeStore.searchKeyword)
const selectedTypes = ref<string[]>(homeStore.selectedTypes)
const page = ref(homeStore.page)
const recommendPage = ref(homeStore.recommendPage)

const isLoading = ref(false)
const isRecommendLoading = ref(false)
const contentKey = ref(0)
const recommendKey = ref(0)

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

const contentTypes = ['video', 'image', 'text', 'link']
const contentTypeLabels: Record<string, string> = {
  video: '视频',
  image: '图片',
  text: '图文',
  link: '链接',
}

function normalizeContent(content: Content | Record<string, unknown>): Content {
  const rawUser = (content as Record<string, unknown>).user as Record<string, unknown> | undefined

  return {
    id: Number(content.id) || 0,
    title: String(content.title || ''),
    type: (String(content.type) as 'video' | 'image' | 'text' | 'link') || 'text',
    text: String(content.text || ''),
    url: String(content.url || ''),
    thumb: String(content.thumb || ''),
    video: String(content.video || ''),
    img: String((content as Record<string, unknown>).img || ''),
    file_size: Number(content.file_size) || 0,
    user: rawUser
      ? {
          id: Number(rawUser.id) || 0,
          username: String(rawUser.username || ''),
          is_admin: Boolean(rawUser.is_admin),
          is_banned: Boolean(rawUser.is_banned),
          created_at: Number(rawUser.created_at) || 0,
          updated_at: Number(rawUser.updated_at) || 0,
        }
      : {
          id: 0,
          username: '',
          is_admin: false,
          is_banned: false,
          created_at: 0,
          updated_at: 0,
        },
    tags: Array.isArray(content.tags) ? (content.tags as string[]) : [],
    created_at: Number(content.created_at) || 0,
    updated_at: Number(content.updated_at) || 0,
    view_count: Number(content.view_count) || 0,
  }
}

async function loadContents() {
  if (isLoading.value) return
  
  isLoading.value = true
  
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
      contentKey.value++
    }
  } catch (error) {
    console.error('加载内容失败', error)
  } finally {
    isLoading.value = false
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
  loadContents().then(() => {
    document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function selectType(type: string) {
  const index = selectedTypes.value.indexOf(type)
  if (index > -1) {
    selectedTypes.value = []
  } else {
    selectedTypes.value = [type]
  }
  page.value = 1
  loadContents().then(() => {
    document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function handleSearch() {
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer)
  }
  searchDebounceTimer = setTimeout(() => {
    page.value = 1
    loadContents().then(() => {
      document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }, 300)
}

function goToDetail(content: Content) {
  const linkUrl = content.type === 'link' && content.url
  if (linkUrl) {
    window.open(linkUrl, '_blank')
    return
  }
  if (content.id) {
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

onBeforeRouteLeave((to, from, next) => {
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
  } else {
    homeStore.clearState()
  }
  next()
})

function goToPage(p: number) {
  if (p >= 1 && p <= totalPages.value && !isLoading.value) {
    page.value = p
    loadContents().then(() => {
      document.getElementById('content-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }
}

function goToEasterEgg() {
  router.push('/easter-egg')
}

function normalizeRecommendContent(content: RecommendContent): RecommendContent {
  return {
    id: Number(content.id) || 0,
    title: content.title || '',
    type: (content.type as 'video' | 'image' | 'text' | 'link') || 'image',
    url: content.url || '',
    thumb: content.thumb || '',
    tags: Array.isArray(content.tags) ? content.tags : [],
    view_count: content.view_count || 0,
    user: {
      id: Number(content.user?.id) || 0,
      username: content.user?.username || '',
    },
    created_at: content.created_at || 0,
  }
}

async function loadRecommendContents() {
  if (isRecommendLoading.value) return
  
  isRecommendLoading.value = true
  
  try {
    const res = await contentApi.recommend(recommendPerPage, recommendPage.value)
    if (res.code === 200) {
      recommendContents.value = res.data.list.map((item: RecommendContent) =>
        normalizeRecommendContent(item),
      )
      recommendHint.value = ''
      recommendKey.value++
    }
  } catch (error) {
    console.error('加载推荐内容失败', error)
  } finally {
    isRecommendLoading.value = false
  }
}

function refreshRecommend() {
  if (isRecommendLoading.value) return
  
  recommendPage.value++
  if (recommendPage.value > maxRecommendPages) {
    recommendPage.value = 1
  }
  loadRecommendContents().then(() => {
    document.getElementById('recommend-section')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

onMounted(() => {
  const navigationEntries = performance.getEntriesByType(
    'navigation',
  ) as PerformanceNavigationTiming[]
  const isReload = navigationEntries.length > 0 && navigationEntries[0].type === 'reload'

  if (isReload) {
    homeStore.clearState()
  }

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
  <div class="min-h-screen p-2 sm:p-5 flex justify-center">
    <div class="w-full max-w-[1200px] min-h-screen overflow-hidden theme-card rounded-xl sm:rounded-xl shadow-lg shadow-black/5 border transition-all duration-500">
      <div class="flex items-center justify-between px-4 py-2.5 bg-gradient-to-b from-black/8 to-black/2">
        <div class="flex items-center">
          <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
          <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
          <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
          <div class="text-sm theme-text-secondary font-medium">小泉动漫二创站</div>
        </div>
        <ThemeSelector />
      </div>

      <div class="p-3 sm:p-6">
        <div class="text-center mb-5 sm:mb-8">
          <img :src="logoImg" alt="小泉动漫二创站" class="max-w-full h-auto max-h-[80px] sm:max-h-[120px] mx-auto mb-2" />
          <a @click="goToEasterEgg" class="block w-fit mx-auto px-3 py-1.5 sm:px-4 sm:py-2 rounded-full bg-gradient-to-r from-indigo-500 to-purple-600 text-white no-underline cursor-pointer transition-all duration-300 text-sm sm:text-base font-medium text-center hover:scale-105 hover:shadow-lg hover:shadow-indigo-500/40">🎉 发现精彩内容</a>
        </div>

        <div class="mb-4 sm:mb-6">
          <div class="flex flex-col sm:flex-row gap-2 sm:gap-3 max-w-[600px] mx-auto">
            <input
              v-model="searchKeyword"
              type="text"
              placeholder="搜索内容..."
              @keyup.enter="handleSearch"
              class="flex-1 text-sm px-4 py-3 sm:py-2.5 min-h-[44px] sm:min-h-[auto] border theme-border rounded-lg theme-surface outline-none transition-all focus:border-blue-500 focus:ring-3 focus:ring-blue-500/10"
            />
            <button @click="handleSearch" class="px-5 py-3 sm:py-2.5 min-h-[44px] sm:min-h-[auto] bg-blue-500 text-white border-none rounded-lg text-sm font-medium cursor-pointer transition-all hover:bg-blue-600 whitespace-nowrap">搜索</button>
          </div>
        </div>

        <div class="mb-4 sm:mb-6">
          <PollComponent />
        </div>

        <div class="flex flex-col sm:flex-row flex-wrap gap-3 sm:gap-5 mb-4 sm:mb-6 p-3 sm:p-4 theme-surface rounded-xl">
          <div class="flex flex-col gap-1.5">
            <span class="text-xs sm:text-sm theme-text-secondary font-medium">类型</span>
            <div class="flex flex-wrap gap-1.5 sm:gap-2">
              <button
                v-for="type in contentTypes"
                :key="type"
                @click="selectType(type)"
                class="px-2 py-1 sm:px-3 sm:py-1 theme-surface border theme-border rounded-full text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:text-emerald-600 hover:border-emerald-600/30"
                :class="{ 'bg-emerald-100/50 text-emerald-600 font-medium border-emerald-600/30': selectedTypes.includes(type) }"
              >
                {{ contentTypeLabels[type] }}
              </button>
            </div>
          </div>

          <div class="flex flex-col gap-1.5">
            <span class="text-xs sm:text-sm theme-text-secondary font-medium">标签云</span>
            <div class="flex flex-wrap gap-1.5 sm:gap-2">
              <button
                v-for="tag in sortedTags"
                :key="tag"
                @click="selectTag(tag)"
                class="px-2 py-1 sm:px-3 sm:py-1 theme-surface border theme-border rounded-full text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:text-blue-500 hover:border-blue-500/30"
                :class="{ 'bg-blue-100/50 text-blue-500 font-medium border-blue-500/30': selectedTags.includes(tag) }"
              >
                {{ tag }}
              </button>
            </div>
          </div>
        </div>

        <motion.div
          class="relative overflow-hidden"
          style="min-height: 100px"
          :animate="{ height: 'auto', transition: { duration: 0.3, ease: 'easeOut' } }"
        >
          <motion.div
            :animate="{ transition: { duration: 0.3, ease: 'easeOut' } }"
          >
            <motion.div
              id="recommend-section"
              class="mb-4 sm:mb-6"
              :style="{ scrollMarginTop: '120px' }"
              :initial="{ opacity: 1, y: 0 }"
              :animate="{
                opacity: swapSections ? 0 : 1,
                y: swapSections ? -100 : 0,
                height: swapSections ? 0 : 'auto',
                transition: { duration: 0.3, ease: 'easeOut' },
              }"
              style="overflow: hidden"
            >
              <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2 mb-3 sm:mb-4">
                <h2 class="text-base sm:text-lg font-semibold theme-text m-0">🔥 推荐内容</h2>
                <div class="flex items-center gap-2 sm:gap-3">
                  <span class="text-xs sm:text-sm theme-text-secondary">精选推荐</span>
                  <button @click="refreshRecommend" class="flex items-center gap-1 px-2 py-1 sm:px-3 sm:py-1.5 theme-surface border theme-border rounded-md text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-gray-100 hover:border-blue-500 hover:text-blue-500" :disabled="isRecommendLoading">
                    <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" :class="{ 'animate-spin': isRecommendLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="23 4 23 10 17 10" />
                      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
                    </svg>
                    {{ isRecommendLoading ? '加载中...' : '刷新' }}
                  </button>
                </div>
                <span v-if="recommendHint" class="text-xs sm:text-sm text-red-500 animate-pulse">{{ recommendHint }}</span>
              </div>

              <div class="relative">
                <div class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-4" :class="{ 'opacity-30 pointer-events-none transition-opacity': isRecommendLoading }">
                  <div
                    v-for="content in recommendContents"
                    :key="content.id"
                    @click="goToDetail({ ...content, type: content.type, audit_status: 'approved' } as unknown as Content)"
                    class="overflow-hidden cursor-pointer theme-card rounded-xl shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-lg"
                  >
                    <div class="relative w-full pt-[75%] bg-gray-100 overflow-hidden">
                      <img
                        :src="getImageUrl(content.thumb)"
                        :alt="content.title"
                        class="absolute top-0 left-0 w-full h-full object-cover"
                        loading="lazy"
                        @load="onImageLoad"
                      />
                      <div v-if="content.type === 'video'" class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 bg-black/60 rounded-full flex items-center justify-center">
                        <svg class="w-5 h-5 text-white ml-0.5" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M8 5v14l11-7z" />
                        </svg>
                      </div>
                    </div>
                    <div class="p-2 sm:p-3">
                      <h3 class="text-xs sm:text-sm font-semibold theme-text mb-1 sm:mb-2 overflow-hidden text-ellipsis whitespace-nowrap">{{ content.title }}</h3>
                      <div class="flex items-center gap-2 flex-wrap">
                        <span class="text-xs theme-text-secondary">{{ content.user.username }}</span>
                        <div class="flex gap-1">
                          <span v-for="tag in content.tags.slice(0, 2)" :key="tag" class="px-1.5 py-0.5 bg-orange-100/50 rounded text-xs text-orange-500">{{ tag }}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <div v-if="isRecommendLoading" class="absolute inset-0 flex flex-col items-center justify-center bg-white/50 z-10">
                  <div class="w-10 h-10 border-3 border-blue-500/20 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
                  <p class="text-sm theme-text-secondary">加载中...</p>
                </div>
              </div>
            </motion.div>

            <div id="content-section" class="mb-4 sm:mb-6" style="scroll-margin-top: 120px">
              <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-2 mb-3 sm:mb-4">
                <h2 class="text-base sm:text-lg font-semibold theme-text m-0">📅 最近上传</h2>
                <span class="text-xs sm:text-sm theme-text-secondary">共 {{ total }} 条</span>
              </div>

              <div class="relative">
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-5" :class="{ 'opacity-30 pointer-events-none transition-opacity': isLoading }">
                  <div
                    v-for="content in contents"
                    :key="content.id"
                    @click="goToDetail(content)"
                    class="overflow-hidden cursor-pointer theme-card rounded-xl shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-lg"
                  >
                    <div class="relative w-full pt-[75%] bg-gray-100 overflow-hidden">
                      <template v-if="content.type === 'image' || content.type === 'video' || content.type === 'link'">
                        <img
                          :src="getImageUrl(content.thumb)"
                          :alt="content.type === 'link' ? '链接' : (content.type === 'video' ? '视频封面' : '内容图片')"
                          class="absolute top-0 left-0 w-full h-full object-cover"
                          loading="lazy"
                          @load="onImageLoad"
                        />
                      </template>
                      <template v-if="content.type === 'video'">
                        <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-12 h-12 bg-black/60 rounded-full flex items-center justify-center">
                          <svg class="w-5 h-5 text-white ml-0.5" viewBox="0 0 24 24" fill="currentColor">
                            <path d="M8 5v14l11-7z" />
                          </svg>
                        </div>
                      </template>
                      <template v-if="content.type === 'link'">
                        <div class="absolute top-2 right-2 w-8 h-8 bg-violet-500/85 rounded-full flex items-center justify-center">
                          <svg class="w-4 h-4 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                          </svg>
                        </div>
                      </template>
                      <template v-if="content.type === 'text'">
                        <div class="absolute top-0 left-0 w-full h-full p-4 bg-gradient-to-br from-slate-50 to-slate-100 flex items-center justify-center text-center">
                          <p class="text-sm text-slate-500 m-0 line-clamp-4">{{ getPreviewText(content.text || '暂无内容') }}</p>
                        </div>
                      </template>
                    </div>
                    <div class="p-3 sm:p-4">
                      <h3 class="text-sm sm:text-base font-semibold theme-text mb-2 sm:mb-2.5 overflow-hidden text-ellipsis whitespace-nowrap">{{ content.title }}</h3>
                      <div class="flex items-center gap-1.5 sm:gap-2 mb-2 sm:mb-2.5 flex-wrap">
                        <span class="flex items-center gap-1 text-xs theme-text-secondary">
                          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                            <circle cx="12" cy="7" r="4" />
                          </svg>
                          {{ content.user?.username }}
                        </span>
                        <div class="flex gap-1 flex-wrap">
                          <span v-for="tag in content.tags" :key="tag" class="px-1.5 py-0.5 theme-surface rounded text-xs theme-text-secondary">{{ tag }}</span>
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        <span :class="['px-2 py-0.5 rounded-full text-xs font-medium', content.type === 'video' ? 'bg-orange-100 text-orange-600' : content.type === 'image' ? 'bg-emerald-100 text-emerald-600' : content.type === 'link' ? 'bg-violet-100 text-violet-600' : 'bg-sky-100 text-sky-600']">
                          {{ content.type === 'video' ? '视频' : content.type === 'image' ? '图片' : content.type === 'link' ? '链接' : '文字' }}
                        </span>
                        <span class="flex items-center gap-1 text-xs theme-text-secondary">
                          <svg class="w-3.5 h-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                            <circle cx="12" cy="12" r="3"/>
                          </svg>
                          {{ content.view_count }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
                
                <div v-if="isLoading" class="absolute inset-0 flex flex-col items-center justify-center bg-white/50 z-10">
                  <div class="w-10 h-10 border-3 border-blue-500/20 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
                  <p class="text-sm theme-text-secondary">加载中...</p>
                </div>
              </div>
              
              <div v-if="!isLoading && contents.length === 0" class="text-center py-16 theme-text-secondary">
                <svg class="w-16 h-16 mx-auto mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <circle cx="12" cy="12" r="10" />
                  <polyline points="12 6 12 12 16 14" />
                </svg>
                <p class="text-sm">暂无内容</p>
              </div>
            </div>
          </motion.div>
        </motion.div>

        <div v-if="totalPages > 1" class="flex flex-wrap justify-center items-center gap-2 sm:gap-4 pt-4 border-t theme-border px-2">
          <button @click="goToPage(1)" :disabled="page <= 1 || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-gray-100 hover:border-blue-500 hover:text-blue-500 disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 19l-7-7 7-7" />
            </svg>
            <span class="hidden sm:inline">首页</span>
          </button>
          <button @click="goToPage(page - 1)" :disabled="page <= 1 || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-gray-100 hover:border-blue-500 hover:text-blue-500 disabled:opacity-50 disabled:cursor-not-allowed">
            <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7" />
            </svg>
            <span class="hidden sm:inline">上一页</span>
          </button>
          <span class="px-2 py-2 text-xs sm:text-sm theme-text-secondary">第 {{ page }} / {{ totalPages }} 页</span>
          <button @click="goToPage(page + 1)" :disabled="page >= totalPages || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-gray-100 hover:border-blue-500 hover:text-blue-500 disabled:opacity-50 disabled:cursor-not-allowed">
            <span class="hidden sm:inline">下一页</span>
            <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7" />
            </svg>
          </button>
          <button @click="goToPage(totalPages)" :disabled="page >= totalPages || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-gray-100 hover:border-blue-500 hover:text-blue-500 disabled:opacity-50 disabled:cursor-not-allowed">
            <span class="hidden sm:inline">尾页</span>
            <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M5 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

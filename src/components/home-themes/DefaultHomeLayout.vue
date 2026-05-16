<script setup lang="ts">
import { onMounted } from 'vue'
import { motion } from 'motion-v'
import logoImg from '@/assets/logo.webp'
import PollComponent from '@/components/PollComponent.vue'
import { useHomeLogic } from '@/composables/useHomeLogic'
import type { Content } from '@/types'

const {
  contents,
  recommendContents,
  sortedTags,
  searchKeyword,
  selectedTags,
  selectedTypes,
  page,
  isLoading,
  isRecommendLoading,
  total,
  totalPages,
  recommendHint,
  swapSections,
  contentTypes,
  contentTypeLabels,
  loadContents,
  selectTag,
  selectType,
  handleSearch,
  goToDetail,
  goToPage,
  goToEasterEgg,
  refreshRecommend,
  onMountedHome,
  getImageUrl,
  onImageLoad,
  getPreviewText,
} = useHomeLogic()

onMounted(() => {
  onMountedHome()
})
</script>

<template>
  <div class="min-h-screen p-2 sm:p-5 flex justify-center">
    <div class="w-full max-w-[1200px] min-h-screen overflow-hidden theme-card rounded-xl sm:rounded-xl shadow-lg shadow-black/5 border transition-all duration-500 relative">
      <div class="theme-glass-layer rounded-xl sm:rounded-xl"></div>
      <div class="relative z-1">
        <div class="flex items-center justify-between px-4 py-2.5 theme-header-bg">
          <div class="flex items-center">
            <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
            <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
            <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
            <div class="text-sm theme-text-secondary font-medium">小泉动漫二创站</div>
          </div>
        </div>

        <div class="p-3 sm:p-6">
          <div class="text-center mb-5 sm:mb-8">
            <img :src="logoImg" alt="小泉动漫二创站" class="max-w-full h-auto max-h-[80px] sm:max-h-[120px] mx-auto mb-2" />
            <a @click="goToEasterEgg" class="block w-fit mx-auto px-3 py-1.5 sm:px-4 sm:py-2 rounded-full theme-accent-gradient text-(--theme-on-primary) no-underline cursor-pointer transition-all duration-300 text-sm sm:text-base font-medium text-center hover:scale-105 hover:shadow-lg">🎉 发现精彩内容</a>
          </div>

          <div class="mb-4 sm:mb-6">
            <div class="flex flex-col sm:flex-row gap-2 sm:gap-3 max-w-[600px] mx-auto">
              <input
                v-model="searchKeyword"
                type="text"
                placeholder="搜索内容..."
                @keyup.enter="handleSearch"
                class="flex-1 text-sm px-4 py-3 sm:py-2.5 min-h-[44px] sm:min-h-[auto] border theme-border rounded-lg theme-surface outline-none transition-all focus:border-(--theme-primary) focus:ring-3 focus:ring-(--theme-primary)/10"
              />
              <button @click="handleSearch" class="px-5 py-3 sm:py-2.5 min-h-[44px] sm:min-h-[auto] bg-(--theme-primary) text-(--theme-on-primary) border-none rounded-lg text-sm font-medium cursor-pointer transition-all hover:brightness-90 whitespace-nowrap">搜索</button>
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
                  class="px-2 py-1 sm:px-3 sm:py-1 theme-surface border theme-border rounded-full text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:text-(--theme-success) hover:border-(--theme-success)/30"
                  :class="{ 'bg-(--theme-success)/10 text-(--theme-success) font-medium border-(--theme-success)/30': selectedTypes.includes(type) }"
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
                  class="px-2 py-1 sm:px-3 sm:py-1 theme-surface border theme-border rounded-full text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:text-(--theme-primary) hover:border-(--theme-primary)/30"
                  :class="{ 'bg-(--theme-primary)/10 text-(--theme-primary) font-medium border-(--theme-primary)/30': selectedTags.includes(tag) }"
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
                    <button @click="refreshRecommend" class="flex items-center gap-1 px-2 py-1 sm:px-3 sm:py-1.5 theme-surface border theme-border rounded-md text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-(--theme-hover-bg) hover:border-(--theme-primary) hover:text-(--theme-primary)" :disabled="isRecommendLoading">
                      <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" :class="{ 'animate-spin': isRecommendLoading }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="23 4 23 10 17 10" />
                        <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
                      </svg>
                      {{ isRecommendLoading ? '加载中...' : '刷新' }}
                    </button>
                  </div>
                  <span v-if="recommendHint" class="text-xs sm:text-sm text-(--theme-danger) animate-pulse">{{ recommendHint }}</span>
                </div>

                <div class="relative">
                  <div class="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-4 gap-2 sm:gap-4" :class="{ 'opacity-30 pointer-events-none transition-opacity': isRecommendLoading }">
                    <div
                      v-for="content in recommendContents"
                      :key="content.id"
                      @click="goToDetail({ ...content, type: content.type, audit_status: 'approved' } as unknown as Content)"
                      class="overflow-hidden cursor-pointer theme-card rounded-xl shadow-md transition-all duration-200 hover:-translate-y-1 hover:shadow-lg"
                    >
                      <div class="relative w-full pt-[75%] theme-placeholder-bg overflow-hidden">
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
                            <span v-for="tag in content.tags.slice(0, 2)" :key="tag" class="px-1.5 py-0.5 bg-(--theme-warning)/10 rounded text-xs text-(--theme-warning)">{{ tag }}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div v-if="isRecommendLoading" class="absolute inset-0 flex flex-col items-center justify-center theme-overlay-bg z-10">
                    <div class="w-10 h-10 border-3 border-(--theme-primary)/20 border-t-(--theme-primary) rounded-full animate-spin mx-auto mb-4"></div>
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
                      <div class="relative w-full pt-[75%] theme-placeholder-bg overflow-hidden">
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
                          <div class="absolute top-2 right-2 w-8 h-8 bg-(--theme-secondary) rounded-full flex items-center justify-center">
                            <svg class="w-4 h-4 text-(--theme-on-primary)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
                              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
                            </svg>
                          </div>
                        </template>
                        <template v-if="content.type === 'text'">
                          <div class="absolute top-0 left-0 w-full h-full p-4 theme-surface flex items-center justify-center text-center">
                            <p class="text-sm theme-text-secondary m-0 line-clamp-4">{{ getPreviewText(content.text || '暂无内容') }}</p>
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
                          <span :class="['px-2 py-0.5 rounded-full text-xs font-medium', content.type === 'video' ? 'bg-(--theme-warning)/10 text-(--theme-warning)' : content.type === 'image' ? 'bg-(--theme-success)/10 text-(--theme-success)' : content.type === 'link' ? 'bg-(--theme-secondary)/10 text-(--theme-secondary)' : 'bg-(--theme-primary)/10 text-(--theme-primary)']">
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
                  
                  <div v-if="isLoading" class="absolute inset-0 flex flex-col items-center justify-center theme-overlay-bg z-10">
                    <div class="w-10 h-10 border-3 border-(--theme-primary)/20 border-t-(--theme-primary) rounded-full animate-spin mx-auto mb-4"></div>
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
            <button @click="goToPage(1)" :disabled="page <= 1 || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-(--theme-hover-bg) hover:border-(--theme-primary) hover:text-(--theme-primary) disabled:opacity-50 disabled:cursor-not-allowed">
              <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 19l-7-7 7-7" />
              </svg>
              <span class="hidden sm:inline">首页</span>
            </button>
            <button @click="goToPage(page - 1)" :disabled="page <= 1 || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-(--theme-hover-bg) hover:border-(--theme-primary) hover:text-(--theme-primary) disabled:opacity-50 disabled:cursor-not-allowed">
              <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 19l-7-7 7-7" />
              </svg>
              <span class="hidden sm:inline">上一页</span>
            </button>
            <span class="px-2 py-2 text-xs sm:text-sm theme-text-secondary">第 {{ page }} / {{ totalPages }} 页</span>
            <button @click="goToPage(page + 1)" :disabled="page >= totalPages || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-(--theme-hover-bg) hover:border-(--theme-primary) hover:text-(--theme-primary) disabled:opacity-50 disabled:cursor-not-allowed">
              <span class="hidden sm:inline">下一页</span>
              <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 5l7 7-7 7" />
              </svg>
            </button>
            <button @click="goToPage(totalPages)" :disabled="page >= totalPages || isLoading" class="flex items-center gap-1 px-2 sm:px-3 py-2 theme-surface border theme-border rounded-lg text-xs sm:text-sm theme-text-secondary cursor-pointer transition-all hover:bg-(--theme-hover-bg) hover:border-(--theme-primary) hover:text-(--theme-primary) disabled:opacity-50 disabled:cursor-not-allowed">
              <span class="hidden sm:inline">尾页</span>
              <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M5 5l7 7-7 7" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

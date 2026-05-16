<script setup lang="ts">
import { useThemeStore, THEME_LIST, type ThemeType } from '@/stores/theme'
import { useRouter } from 'vue-router'

const themeStore = useThemeStore()
const router = useRouter()

function selectTheme(key: ThemeType) {
  themeStore.setTheme(key)
}

function goBack() {
  router.push('/')
}
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
            <div class="text-sm theme-text-secondary font-medium">主题设置</div>
          </div>
          <button @click="goBack" class="flex items-center gap-2 px-3 py-2 rounded-lg theme-surface theme-border text-sm transition-all hover:bg-[var(--theme-hover-bg)] theme-text-secondary">
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5M12 19l-7-7 7-7" />
            </svg>
            <span>返回首页</span>
          </button>
        </div>

        <div class="p-3 sm:p-6">
          <div class="text-center mb-5 sm:mb-8">
            <h1 class="text-xl sm:text-2xl font-bold theme-text mb-2">选择你的主题</h1>
            <p class="text-sm theme-text-secondary">每个主题都有独特的布局和视觉体验</p>
          </div>

          <div class="max-w-[1000px] mx-auto mb-4 sm:mb-6">
            <h2 class="text-lg sm:text-xl font-bold theme-text">MAC风</h2>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4 sm:gap-6 max-w-[1000px] mx-auto">
            <div
              v-for="theme in THEME_LIST"
              :key="theme.key"
              class="theme-option-card theme-card rounded-xl overflow-hidden border-2 transition-all duration-300 cursor-pointer hover:-translate-y-1 hover:shadow-lg group relative"
              :class="{ 'border-[var(--theme-primary)] shadow-md': themeStore.currentTheme === theme.key, 'theme-border': themeStore.currentTheme !== theme.key }"
              @click="selectTheme(theme.key)"
            >
              <div class="theme-preview h-40 sm:h-48 relative overflow-hidden" :style="{ background: `linear-gradient(135deg, ${theme.previewColor}, ${theme.previewColor}88)` }">
                <div class="preview-mockup absolute inset-4 sm:inset-6 theme-surface rounded-lg shadow-lg overflow-hidden" :class="`preview-${theme.key}`">
                  <div class="flex items-center gap-1.5 px-3 py-2 theme-header-bg">
                    <div class="w-2 h-2 rounded-full bg-[#ff5f57]"></div>
                    <div class="w-2 h-2 rounded-full bg-[#febc2e]"></div>
                    <div class="w-2 h-2 rounded-full bg-[#28c840]"></div>
                  </div>
                  <div class="p-2 sm:p-3">
                    <div class="h-1.5 rounded-full theme-placeholder-bg w-3/5 mb-2"></div>
                    <div class="h-1.5 rounded-full theme-placeholder-bg w-full mb-3"></div>
                    <div class="grid grid-cols-3 gap-1.5">
                      <div class="h-8 rounded theme-placeholder-bg"></div>
                      <div class="h-8 rounded theme-placeholder-bg"></div>
                      <div class="h-8 rounded theme-placeholder-bg"></div>
                    </div>
                  </div>
                </div>
                <div v-if="themeStore.currentTheme === theme.key" class="absolute top-2 right-2 sm:top-3 sm:right-3 w-7 h-7 sm:w-8 sm:h-8 bg-white rounded-full flex items-center justify-center shadow-md">
                  <svg class="w-4 h-4 sm:w-5 sm:h-5" :style="{ color: theme.previewColor }" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12" />
                  </svg>
                </div>
              </div>
              <div class="p-3 sm:p-4">
                <div class="flex items-center justify-between">
                  <h3 class="text-sm sm:text-base font-semibold theme-text">{{ theme.name }}</h3>
                  <div class="flex items-center gap-2">
                    <span v-if="theme.performanceLabel" class="px-2 py-0.5 rounded-full text-xs font-medium bg-orange-500/10 text-orange-500">{{ theme.performanceLabel }}</span>
                    <span v-if="themeStore.currentTheme === theme.key" class="px-2 py-0.5 rounded-full text-xs font-medium bg-[var(--theme-primary)]/10 text-[var(--theme-primary)]">当前使用</span>
                  </div>
                </div>
                <p class="text-xs sm:text-sm theme-text-secondary mt-1">{{ theme.description }}</p>
              </div>
            </div>
          </div>

          <div v-if="themeStore.currentTheme === 'liquidGlass'" class="max-w-[800px] mx-auto mt-6 sm:mt-8 p-4 sm:p-6 theme-card rounded-xl theme-border">
            <h3 class="text-base sm:text-lg font-semibold theme-text mb-4">液态玻璃设置</h3>
            <div class="space-y-4">
              <div>
                <div class="flex items-center justify-between mb-2">
                  <label class="text-sm theme-text-secondary">模糊程度</label>
                  <span class="text-sm font-medium theme-text">{{ themeStore.glassBlur }}px</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="30"
                  step="1"
                  v-model.number="themeStore.glassBlur"
                  class="w-full h-2 rounded-lg appearance-none cursor-pointer bg-[var(--theme-placeholder-bg)] accent-[var(--theme-primary)]"
                />
                <div class="flex justify-between mt-1 text-xs theme-text-secondary">
                  <span>无模糊</span>
                  <span>强模糊</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.theme-option-card {
  border-color: var(--theme-card-border);
}

.preview-default {
  --theme-surface: rgba(255, 255, 255, 0.92);
  --theme-header-bg: linear-gradient(to bottom, rgba(0,0,0,0.08), rgba(0,0,0,0.02));
  --theme-placeholder-bg: #f3f4f6;
  --theme-text: #1f2937;
  --theme-text-secondary: #6b7280;
}

.preview-dark {
  --theme-surface: rgba(40, 40, 70, 0.9);
  --theme-header-bg: linear-gradient(to bottom, rgba(255,255,255,0.06), rgba(255,255,255,0.02));
  --theme-placeholder-bg: #374151;
  --theme-text: #f3f4f6;
  --theme-text-secondary: #9ca3af;
}

.preview-liquidGlass {
  --theme-surface: rgba(255, 255, 255, 0.75);
  --theme-header-bg: linear-gradient(to bottom, rgba(255,255,255,0.15), rgba(255,255,255,0.05));
  --theme-placeholder-bg: rgba(255, 255, 255, 0.15);
  --theme-text: #000000;
  --theme-text-secondary: #333333;
}
</style>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useThemeStore } from '@/stores/theme'
import type { ThemeType } from '@/stores/theme'

const themeStore = useThemeStore()

const showDropdown = ref(false)

const themeOptions = computed(() => {
  return Object.entries(themeStore.themes).map(([key, config]) => ({
    key: key as ThemeType,
    name: config.name,
    color: config.primary
  }))
})

function handleClickOutside(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (!target.closest('.theme-selector')) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div class="theme-selector relative">
    <button
      class="flex items-center gap-2 px-3 py-2 rounded-lg theme-surface backdrop-blur-sm theme-border text-sm transition-all hover:bg-[var(--theme-hover-bg)]"
      @click="showDropdown = !showDropdown"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"></circle>
        <path d="M12 1v6m0 6v6m4.28-13.34a4 4 0 0 1 0 5.66m-4.28 5.66a4 4 0 0 0 0 5.66M21 12H3"></path>
      </svg>
      <span>{{ themeStore.themes[themeStore.currentTheme]?.name }}</span>
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </button>
    
    <div 
      v-if="showDropdown"
      class="absolute right-0 top-full mt-2 w-36 theme-surface backdrop-blur-sm rounded-lg shadow-lg theme-border overflow-hidden z-50"
    >
      <button
        v-for="option in themeOptions"
        :key="option.key"
        @click="themeStore.setTheme(option.key); showDropdown = false"
        class="w-full flex items-center gap-3 px-4 py-2.5 text-sm theme-text hover:bg-[var(--theme-hover-bg)] transition-colors"
        :class="{ 'bg-[var(--theme-primary)]/10 text-[var(--theme-primary)]': themeStore.currentTheme === option.key }"
      >
        <span 
          class="w-3 h-3 rounded-full"
          :style="{ backgroundColor: option.color }"
        ></span>
        <span>{{ option.name }}</span>
      </button>
    </div>
  </div>
</template>
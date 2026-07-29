import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { applyThemeColors } from '@/composables/themeColors'

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<'light' | 'dark'>('light')

  function setMode(newMode: 'light' | 'dark') {
    mode.value = newMode
    try {
      localStorage.setItem('theme_mode', newMode)
    } catch {}
    applyThemeColors(mode.value)
  }

  function toggleMode() {
    setMode(mode.value === 'light' ? 'dark' : 'light')
  }

  try {
    const savedMode = localStorage.getItem('theme_mode') as 'light' | 'dark' | null
    if (savedMode === 'light' || savedMode === 'dark') {
      mode.value = savedMode
    }
  } catch {}

  // 首次注入 class、后续 watch mode 变化自动切换
  applyThemeColors(mode.value)
  watch(mode, (v) => applyThemeColors(v))

  return { mode, setMode, toggleMode }
})

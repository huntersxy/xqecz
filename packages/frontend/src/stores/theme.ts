import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 切换日间/暗色：仅操作 document.documentElement 的 dark class。
 * CSS 变量值由 main.css 中的 `:root` / `html.dark` 控制。
 */
function applyMode(mode: 'light' | 'dark') {
  const root = document.documentElement
  root.classList.toggle('dark', mode === 'dark')
  root.classList.toggle('light', mode === 'light')
}

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<'light' | 'dark'>('light')

  function setMode(newMode: 'light' | 'dark') {
    mode.value = newMode
    try {
      localStorage.setItem('theme_mode', newMode)
    } catch {}
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
  applyMode(mode.value)
  watch(mode, (v) => applyMode(v))

  return { mode, setMode, toggleMode }
})

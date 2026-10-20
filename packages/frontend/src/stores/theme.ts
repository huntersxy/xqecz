import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

function applyMode(mode: 'light' | 'dark') {
  if (typeof document !== 'undefined' && document.body) {
    if (mode === 'dark') document.body.setAttribute('arco-theme', 'dark')
    else document.body.removeAttribute('arco-theme')
    document.documentElement.classList.toggle('dark', mode === 'dark')
  }
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

  applyMode(mode.value)
  watch(mode, (v) => applyMode(v))

  return { mode, setMode, toggleMode }
})

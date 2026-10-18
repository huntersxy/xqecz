import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 切换日间/暗色：body 上的 arco-theme 属性供 Arco 和公开调色板使用，
 * html.dark 同步给 Tailwind/首屏脚本使用，两个状态必须一起增删，避免视觉已变浅色
 * 但根节点仍残留 dark class。
 * 工作台与管理端通过 admin-theme 复用这组语义色，状态色仍走 Arco token。
 */
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

  // 启动时按本地偏好立即应用一次，之后由 watch 跟随 mode 变化切换
  applyMode(mode.value)
  watch(mode, (v) => applyMode(v))

  return { mode, setMode, toggleMode }
})

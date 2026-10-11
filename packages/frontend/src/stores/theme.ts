import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 切换日间/暗色：只在 <body> 上增删 arco-theme="dark" 属性，Arco 按自身
 * body[arco-theme='dark'] 选择器翻日/夜；公开创作界面的 --creative-* 调色板
 * （assets/main.css）同样挂在 body[arco-theme='dark'] 下，因此随之切换。
 * 工作台与管理端通过 admin-theme 复用这组语义色，状态色仍走 Arco token。
 */
function applyMode(mode: 'light' | 'dark') {
  if (typeof document !== 'undefined' && document.body) {
    if (mode === 'dark') document.body.setAttribute('arco-theme', 'dark')
    else document.body.removeAttribute('arco-theme')
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

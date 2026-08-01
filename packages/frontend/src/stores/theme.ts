import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

/**
 * 切换日间/暗色：仅通过 <body arco-theme="dark"> 触发 Arco 原生暗色。
 * 项目原有调色板已移除，自定义 UI 颜色全部走 Arco token（arco.css 的
 * body[arco-theme='dark'] 选择器），随其自动翻日/夜。antd 组件走自身 ConfigProvider。
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

  // 首次注入 class、后续 watch mode 变化自动切换
  applyMode(mode.value)
  watch(mode, (v) => applyMode(v))

  return { mode, setMode, toggleMode }
})

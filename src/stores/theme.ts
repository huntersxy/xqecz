import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeType = 'default' | 'dark' | 'liquidGlass' | 'immersiveMusic'

export interface ThemeInfo {
  key: ThemeType
  name: string
  description: string
  previewColor: string
  performanceLabel?: string
  category?: 'default' | 'special'
}

export const THEME_LIST: ThemeInfo[] = [
  {
    key: 'default',
    name: '默认主题',
    description: '经典浅色风格，清爽明亮',
    previewColor: '#3b82f6',
    category: 'default'
  },
  {
    key: 'dark',
    name: '深色主题',
    description: '沉浸深色风格，护眼舒适',
    previewColor: '#60a5fa',
    category: 'default'
  },
  {
    key: 'liquidGlass',
    name: '液态玻璃',
    description: '玻璃拟态风格，通透质感（仅主页生效）',
    previewColor: '#22d3ee',
    performanceLabel: '高性能消耗',
    category: 'default'
  },
  {
    key: 'immersiveMusic',
    name: '沉浸音乐',
    description: '特殊主题，隐藏首页内容，保留壁纸与音频动态效果',
    previewColor: '#a855f7',
    performanceLabel: '中等性能消耗',
    category: 'special'
  }
]

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref<ThemeType>('default')
  const glassBlur = ref(5)

  function setTheme(theme: ThemeType) {
    currentTheme.value = theme
  }

  function setGlassBlur(blur: number) {
    glassBlur.value = blur
  }

  function applyThemeClass() {
    document.documentElement.classList.remove('theme-default', 'theme-dark', 'theme-liquidGlass', 'theme-immersiveMusic')
    document.documentElement.classList.add(`theme-${currentTheme.value}`)
  }

  function applyGlassBlur() {
    document.documentElement.style.setProperty('--theme-blur', `${glassBlur.value}px`)
  }

  const savedTheme = localStorage.getItem('theme') as ThemeType | null
  if (savedTheme && THEME_LIST.find(t => t.key === savedTheme)) {
    currentTheme.value = savedTheme
  }

  const savedBlur = localStorage.getItem('glassBlur')
  if (savedBlur) {
    glassBlur.value = parseInt(savedBlur, 10)
  }

  applyThemeClass()
  applyGlassBlur()

  watch(currentTheme, () => {
    localStorage.setItem('theme', currentTheme.value)
    applyThemeClass()
  }, { immediate: true })

  watch(glassBlur, () => {
    localStorage.setItem('glassBlur', String(glassBlur.value))
    applyGlassBlur()
  })

  return {
    currentTheme,
    glassBlur,
    setTheme,
    setGlassBlur
  }
})

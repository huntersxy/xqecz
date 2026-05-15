import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeType = 'default' | 'dark' | 'light' | 'custom'

export interface ThemeConfig {
  name: string
  bgImage: string
  bgColor: string
  primary: string
  secondary: string
  accent: string
  text: string
  cardBg: string
  cardBorder: string
  success: string
  warning: string
  danger: string
}

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref<ThemeType>('default')
  
  const themes: Record<ThemeType, ThemeConfig> = {
    default: {
      name: '默认主题',
      bgImage: 'url("./src/assets/bg.webp")',
      bgColor: '#000000',
      primary: '#3b82f6',
      secondary: '#6366f1',
      accent: '#ec4899',
      text: '#1f2937',
      cardBg: 'rgba(255, 255, 255, 0.75)',
      cardBorder: 'rgba(255, 255, 255, 0.4)',
      success: '#22c55e',
      warning: '#f59e0b',
      danger: '#ef4444'
    },
    dark: {
      name: '深色主题',
      bgImage: 'linear-gradient(135deg, #1a1a2e 0%, #16213e 100%)',
      bgColor: '#1a1a2e',
      primary: '#60a5fa',
      secondary: '#818cf8',
      accent: '#f472b6',
      text: '#f3f4f6',
      cardBg: 'rgba(30, 30, 50, 0.85)',
      cardBorder: 'rgba(255, 255, 255, 0.1)',
      success: '#4ade80',
      warning: '#fbbf24',
      danger: '#f87171'
    },
    light: {
      name: '浅色主题',
      bgImage: 'linear-gradient(135deg, #fef3c7 0%, #fce7f3 100%)',
      bgColor: '#fef3c7',
      primary: '#2563eb',
      secondary: '#4f46e5',
      accent: '#db2777',
      text: '#111827',
      cardBg: 'rgba(255, 255, 255, 0.9)',
      cardBorder: 'rgba(0, 0, 0, 0.08)',
      success: '#16a34a',
      warning: '#d97706',
      danger: '#dc2626'
    },
    custom: {
      name: '紫色主题',
      bgImage: 'linear-gradient(135deg, #a855f7 0%, #ec4899 100%)',
      bgColor: '#a855f7',
      primary: '#c084fc',
      secondary: '#f472b6',
      accent: '#fbbf24',
      text: '#ffffff',
      cardBg: 'rgba(30, 30, 50, 0.75)',
      cardBorder: 'rgba(255, 255, 255, 0.15)',
      success: '#4ade80',
      warning: '#fbbf24',
      danger: '#f87171'
    }
  }

  function setTheme(theme: ThemeType) {
    currentTheme.value = theme
  }

  function applyTheme() {
    const theme = themes[currentTheme.value]
    const root = document.documentElement
    
    Object.entries(theme).forEach(([key, value]) => {
      root.style.setProperty(`--theme-${key}`, value)
    })
  }

  const savedTheme = localStorage.getItem('theme') as ThemeType | null
  if (savedTheme && themes[savedTheme]) {
    currentTheme.value = savedTheme
  }

  watch(currentTheme, () => {
    applyTheme()
    localStorage.setItem('theme', currentTheme.value)
  }, { immediate: true })

  return {
    currentTheme,
    themes,
    setTheme,
    applyTheme
  }
})
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeType = 'default' | 'dark' | 'light' | 'custom' | 'liquidGlass'

export interface ThemeConfig {
  name: string
  bgImage: string
  bgColor: string
  primary: string
  secondary: string
  accent: string
  text: string
  textSecondary: string
  surface: string
  headerBg: string
  accentGradient: string
  hoverBg: string
  placeholderBg: string
  onPrimary: string
  overlayBg: string
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
      textSecondary: '#6b7280',
      surface: 'rgba(255, 255, 255, 0.92)',
      headerBg: 'linear-gradient(to bottom, rgba(0,0,0,0.08), rgba(0,0,0,0.02))',
      accentGradient: 'linear-gradient(to right, #6366f1, #a855f7)',
      hoverBg: '#f3f4f6',
      placeholderBg: '#f3f4f6',
      onPrimary: '#ffffff',
      overlayBg: 'rgba(255, 255, 255, 0.5)',
      cardBg: 'rgba(255, 255, 255, 0.75)',
      cardBorder: 'rgba(255, 255, 255, 0.4)',
      success: '#22c55e',
      warning: '#f59e0b',
      danger: '#ef4444'
    },
    dark: {
      name: '深色主题',
      bgImage: 'url("./src/assets/bg.webp")',
      bgColor: '#1a1a2e',
      primary: '#60a5fa',
      secondary: '#818cf8',
      accent: '#f472b6',
      text: '#f3f4f6',
      textSecondary: '#9ca3af',
      surface: 'rgba(40, 40, 70, 0.9)',
      headerBg: 'linear-gradient(to bottom, rgba(255,255,255,0.06), rgba(255,255,255,0.02))',
      accentGradient: 'linear-gradient(to right, #818cf8, #c084fc)',
      hoverBg: 'rgba(255, 255, 255, 0.08)',
      placeholderBg: '#374151',
      onPrimary: '#ffffff',
      overlayBg: 'rgba(0, 0, 0, 0.5)',
      cardBg: 'rgba(30, 30, 50, 0.85)',
      cardBorder: 'rgba(255, 255, 255, 0.1)',
      success: '#4ade80',
      warning: '#fbbf24',
      danger: '#f87171'
    },
    light: {
      name: '浅色主题',
      bgImage: 'url("./src/assets/bg.webp")',
      bgColor: '#fef3c7',
      primary: '#2563eb',
      secondary: '#4f46e5',
      accent: '#db2777',
      text: '#111827',
      textSecondary: '#6b7280',
      surface: 'rgba(255, 255, 255, 0.95)',
      headerBg: 'linear-gradient(to bottom, rgba(0,0,0,0.06), rgba(0,0,0,0.02))',
      accentGradient: 'linear-gradient(to right, #6366f1, #a855f7)',
      hoverBg: '#f3f4f6',
      placeholderBg: '#f3f4f6',
      onPrimary: '#ffffff',
      overlayBg: 'rgba(255, 255, 255, 0.5)',
      cardBg: 'rgba(255, 255, 255, 0.9)',
      cardBorder: 'rgba(0, 0, 0, 0.08)',
      success: '#16a34a',
      warning: '#d97706',
      danger: '#dc2626'
    },
    custom: {
      name: '紫色主题',
      bgImage: 'url("./src/assets/bg.webp")',
      bgColor: '#a855f7',
      primary: '#c084fc',
      secondary: '#f472b6',
      accent: '#fbbf24',
      text: '#ffffff',
      textSecondary: '#e5e7eb',
      surface: 'rgba(40, 40, 70, 0.8)',
      headerBg: 'linear-gradient(to bottom, rgba(255,255,255,0.08), rgba(255,255,255,0.03))',
      accentGradient: 'linear-gradient(to right, #c084fc, #f472b6)',
      hoverBg: 'rgba(255, 255, 255, 0.08)',
      placeholderBg: '#374151',
      onPrimary: '#ffffff',
      overlayBg: 'rgba(0, 0, 0, 0.5)',
      cardBg: 'rgba(30, 30, 50, 0.75)',
      cardBorder: 'rgba(255, 255, 255, 0.15)',
      success: '#4ade80',
      warning: '#fbbf24',
      danger: '#f87171'
    },
    liquidGlass: {
      name: '液态玻璃',
      bgImage: 'url("./src/assets/bg.webp")',
      bgColor: '#0ea5e9',
      primary: '#0ea5e9',
      secondary: '#8b5cf6',
      accent: '#f472b6',
      text: '#1e293b',
      textSecondary: '#64748b',
      surface: 'rgba(255, 255, 255, 0.78)',
      headerBg: 'linear-gradient(to bottom, rgba(255,255,255,0.15), rgba(255,255,255,0.05))',
      accentGradient: 'linear-gradient(to right, #0ea5e9, #8b5cf6)',
      hoverBg: 'rgba(14, 165, 233, 0.1)',
      placeholderBg: 'rgba(14, 165, 233, 0.08)',
      onPrimary: '#ffffff',
      overlayBg: 'rgba(255, 255, 255, 0.55)',
      cardBg: 'rgba(255, 255, 255, 0.65)',
      cardBorder: 'rgba(255, 255, 255, 0.55)',
      success: '#10b981',
      warning: '#f59e0b',
      danger: '#ef4444'
    }
  }

  function setTheme(theme: ThemeType) {
    currentTheme.value = theme
  }

  function applyTheme() {
    const theme = themes[currentTheme.value]
    const root = document.documentElement
    
    Object.entries(theme).forEach(([key, value]) => {
      const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase()
      root.style.setProperty(`--theme-${cssKey}`, value)
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
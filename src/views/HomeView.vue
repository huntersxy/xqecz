<script setup lang="ts">
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'
import DefaultHomeLayout from '@/components/home-themes/DefaultHomeLayout.vue'
import DarkHomeLayout from '@/components/home-themes/DarkHomeLayout.vue'
import LiquidGlassHomeLayout from '@/components/home-themes/LiquidGlassHomeLayout.vue'
import ImmersiveMusicHomeLayout from '@/components/home-themes/ImmersiveMusicHomeLayout.vue'
import BilibiliStyleHomeLayout from '@/components/home-themes/BilibiliStyleHomeLayout.vue'

const themeStore = useThemeStore()

const themeComponent = computed(() => {
  const themeMap: Record<string, any> = {
    default: DefaultHomeLayout,
    dark: DarkHomeLayout,
    liquidGlass: LiquidGlassHomeLayout,
    immersiveMusic: ImmersiveMusicHomeLayout,
    bilibiliStyle: BilibiliStyleHomeLayout,
  }
  return themeMap[themeStore.currentTheme] || DefaultHomeLayout
})
</script>

<template>
  <component :is="themeComponent" />
</template>

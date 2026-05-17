<script setup lang="ts">
import ImmersiveMusicHomeLayout from '@/components/home-themes/ImmersiveMusicHomeLayout.vue'
</script>

<template>
  <div class="music-page">
    <ImmersiveMusicHomeLayout />
  </div>
</template>

<style scoped>
.music-page {
  --theme-text: #f3f4f6;
  --theme-text-secondary: #9ca3af;
  --theme-surface: rgba(40, 40, 70, 0.6);
  --theme-header-bg: linear-gradient(to bottom, rgba(255,255,255,0.06), rgba(255,255,255,0.02));
  --theme-hover-bg: rgba(255, 255, 255, 0.08);
  --theme-placeholder-bg: #374151;
  --theme-overlay-bg: rgba(0, 0, 0, 0.5);
  --theme-card-bg: rgba(30, 30, 50, 0.6);
  --theme-card-border: rgba(255, 255, 255, 0.1);
  --theme-primary: #a855f7;
  --theme-secondary: #818cf8;
  --theme-accent: #f472b6;
  --theme-success: #4ade80;
  --theme-warning: #fbbf24;
  --theme-danger: #f87171;
  
  color: #f3f4f6;
  min-height: 100vh;
}

.music-page :deep(input::placeholder),
.music-page :deep(textarea::placeholder) {
  color: rgba(255, 255, 255, 0.3) !important;
}
</style>

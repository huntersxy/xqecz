<script setup lang="ts">
export type SiteIconName =
  | 'menu' | 'close' | 'sun' | 'moon' | 'plus' | 'bulb' | 'users' | 'chart'
  | 'upload' | 'user' | 'refresh' | 'image' | 'image-broken' | 'arrow-right'
  | 'check-circle' | 'search'

const PATHS: Record<SiteIconName, string> = {
  menu: 'M4 7h16M4 12h16M4 17h16',
  close: 'M6 6l12 12M18 6L6 18',
  sun: 'M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M5.3 18.7l1.4-1.4M17.3 6.7l1.4-1.4',
  moon: 'M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5z',
  plus: 'M12 5v14M5 12h14',
  bulb: 'M9 18h6M10 21h4M12 3a6 6 0 0 0-3.5 10.9c.6.5 1 1.2 1 2V16h5v-.1c0-.8.4-1.5 1-2A6 6 0 0 0 12 3z',
  users: 'M9 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7zM2.5 20a6.5 6.5 0 0 1 13 0M16 4.3a3.5 3.5 0 0 1 0 6.4M18 14a6.5 6.5 0 0 1 3.5 6',
  chart: 'M5 20V11M12 20V4M19 20v-6',
  upload: 'M12 16V4M7 9l5-5 5 5M5 20h14',
  user: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4.5 20.5a7.5 7.5 0 0 1 15 0',
  refresh: 'M20 11a8 8 0 0 0-14.3-4.5L4 8M4 4v4h4M4 13a8 8 0 0 0 14.3 4.5L20 16M20 20v-4h-4',
  image: 'M4 5h16v14H4zM4 16l5-5 4 4 3-3 4 4M15.5 9.5h.01',
  'image-broken': 'M4 5h16v14H4zM4 15l4-4 3 3 3-4 6 6M9 9.5h.01',
  'arrow-right': 'M5 12h14M13 6l6 6-6 6',
  'check-circle': 'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM8 12.5l3 3 5-6',
  search: 'M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14zM20 20l-4-4',
}

defineProps<{ name: SiteIconName; spin?: boolean }>()
</script>

<template>
  <svg
    class="site-icon"
    :class="{ 'is-spin': spin }"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="PATHS[name]" />
  </svg>
</template>

<style scoped>
.site-icon { width: 1em; height: 1em; flex-shrink: 0; }
.site-icon.is-spin { animation: site-icon-spin 1s linear infinite; }
@keyframes site-icon-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) {
  .site-icon.is-spin { animation-duration: 2.4s; }
}
</style>

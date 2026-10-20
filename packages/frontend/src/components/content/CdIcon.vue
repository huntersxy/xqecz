<script setup lang="ts">
/**
 * 详情页专用的内联图标（24×24 描边）。
 * 不依赖任何组件库；`filled` 用于已点亮的喜欢/收藏。
 */
export type CdIconName =
  | 'arrow-left' | 'chevron-left' | 'chevron-right' | 'heart' | 'star' | 'share'
  | 'download' | 'calendar' | 'eye' | 'copy' | 'reply' | 'trash' | 'flag'
  | 'send' | 'image-broken' | 'refresh' | 'close'

const PATHS: Record<CdIconName, string> = {
  'arrow-left': 'M19 12H5M11 6l-6 6 6 6',
  'chevron-left': 'M15 5l-7 7 7 7',
  'chevron-right': 'M9 5l7 7-7 7',
  heart: 'M12 20.5s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7.6a4.3 4.3 0 0 1 7.5 2.7c0 5.6-7.5 10.2-7.5 10.2z',
  star: 'M12 3.5l2.6 5.4 5.9.8-4.3 4.1 1 5.9L12 16.9 6.8 19.7l1-5.9L3.5 9.7l5.9-.8z',
  share: 'M12 4v12M7 9l5-5 5 5M5 14v5a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-5',
  download: 'M12 4v12M7 11l5 5 5-5M5 20h14',
  calendar: 'M4 7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2zM4 10h16M8 3v4M16 3v4',
  eye: 'M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12zM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6z',
  copy: 'M9 9h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V10a1 1 0 0 1 1-1zM16 9V5a1 1 0 0 0-1-1H5a1 1 0 0 0-1 1v10a1 1 0 0 0 1 1h3',
  reply: 'M10 8L4 13l6 5M4 13h9a7 7 0 0 1 7 7',
  trash: 'M4 7h16M10 11v6M14 11v6M6 7l1 12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1l1-12M9 7V4h6v3',
  flag: 'M5 21V4M5 5h12l-2 4 2 4H5',
  send: 'M21 3L10 14M21 3l-7 18-4-7-7-4z',
  'image-broken': 'M4 5h16v14H4zM4 15l4-4 3 3 3-4 6 6M9 9.5h.01',
  refresh: 'M20 11a8 8 0 0 0-14.3-4.5L4 8M4 4v4h4M4 13a8 8 0 0 0 14.3 4.5L20 16M20 20v-4h-4',
  close: 'M6 6l12 12M18 6L6 18',
}

defineProps<{ name: CdIconName; filled?: boolean }>()
</script>

<template>
  <svg
    class="cd-icon"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
  >
    <path :d="PATHS[name]" :fill="filled ? 'currentColor' : 'none'" />
  </svg>
</template>

<style scoped>
.cd-icon { width: 1em; height: 1em; flex-shrink: 0; }
</style>

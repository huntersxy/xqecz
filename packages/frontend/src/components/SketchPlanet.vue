<script setup lang="ts">
defineProps<{ loading?: boolean }>()
</script>

<template>
  <svg class="sketch-planet" :class="{ 'is-loading': loading }" viewBox="0 0 32 32" fill="none" aria-hidden="true" focusable="false">
    <path class="sketch-loading-core" d="M24.2 15.7c.3 4.5-3.2 8.2-7.8 8.4-4.4.2-8.2-3.1-8.4-7.7-.2-4.6 3.2-8.1 7.7-8.4 4.5-.3 8.2 3.2 8.5 7.7Z" />
    <path class="sketch-loading-orbit" d="M9.2 11.7c-4.3.5-7 2-6.6 3.9.5 2.7 7 4.8 14.3 4.7 7.4-.1 12.5-2.3 12-4.8-.3-1.5-2.4-2.8-5.5-3.6" transform="rotate(-24 16 16)" />
    <path d="m25.7 3.2.4 1.6 1.7.6-1.7.5-.4 1.6-.5-1.6-1.6-.5 1.6-.6.5-1.6Z" />
    <path d="M11.7 13.5c.5-1.4 1.4-2.2 2.7-2.5" opacity=".55" />
  </svg>
</template>

<style scoped>
.sketch-planet { display: inline-block; width: 1.5em; height: 1.5em; flex-shrink: 0; vertical-align: middle; stroke: currentColor; stroke-width: 1.5; stroke-linecap: round; stroke-linejoin: round; }
.is-loading .sketch-loading-core { fill: var(--creative-soft); }
.is-loading .sketch-loading-orbit { stroke-dasharray: 4 3; animation: sketch-orbit-travel 1.4s linear infinite; }
@keyframes sketch-orbit-travel { to { stroke-dashoffset: -28; } }
</style>

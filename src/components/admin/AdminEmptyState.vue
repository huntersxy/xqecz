<script setup lang="ts">
interface Props {
  text?: string;
  type?: 'content' | 'pending' | 'user';
}

const props = withDefaults(defineProps<Props>(), {
  text: '暂无内容',
  type: 'content',
});
</script>

<template>
  <div class="text-center py-12 px-5 theme-text-secondary">
    <svg v-if="type === 'content' || type === 'pending'" class="w-16 h-16 mx-auto mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <circle cx="12" cy="12" r="10"/>
      <polyline points="12 6 12 12 16 14"/>
    </svg>
    <svg v-else-if="type === 'user'" class="w-16 h-16 mx-auto mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
      <circle cx="9" cy="7" r="4"/>
      <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
    </svg>
    <p class="text-sm">{{ text }}</p>
  </div>
</template>

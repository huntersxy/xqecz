<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getAvatarUrl } from '@/utils'
import defaultAvatar from '@/assets/avatar-default.svg'

const props = defineProps<{ src?: string; email?: string; username?: string; size?: number }>()
const source = computed(() => props.src?.trim() || getAvatarUrl(props.email || ''))
const failedSource = ref('')
const loadedSource = ref('')
watch(source, () => { failedSource.value = ''; loadedSource.value = '' }, { flush: 'sync' })
const label = computed(() => props.username ? props.username + '的头像' : '默认头像')

function onLoad(event: Event) {
  const src = (event.currentTarget as HTMLImageElement).getAttribute('src') || ''
  if (src === source.value) loadedSource.value = src
}
function onError(event: Event) {
  const src = (event.currentTarget as HTMLImageElement).getAttribute('src') || ''
  if (src === source.value) failedSource.value = src
}
</script>

<template>
  <span class="site-avatar" :style="size ? { '--avatar-size': size + 'px' } : undefined" role="img" :aria-label="label">
    <img class="site-avatar-default" :src="defaultAvatar" alt="" aria-hidden="true" />
    <img
      v-if="source && source !== failedSource"
      :key="source"
      class="site-avatar-image"
      :class="{ 'is-loaded': loadedSource === source }"
      :src="source"
      alt=""
      aria-hidden="true"
      loading="lazy"
      decoding="async"
      referrerpolicy="no-referrer"
      @load="onLoad"
      @error="onError"
    />
  </span>
</template>

<style scoped>
.site-avatar { position: relative; display: inline-block; flex-shrink: 0; width: var(--avatar-size, 32px); height: var(--avatar-size, 32px); overflow: hidden; border-radius: 50%; vertical-align: middle; }
.site-avatar img { display: block; width: 100%; height: 100%; object-fit: cover; }
.site-avatar-image { position: absolute; inset: 0; opacity: 0; }
.site-avatar-image.is-loaded { opacity: 1; }
</style>

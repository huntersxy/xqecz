<script setup lang="ts">
import { ref, useAttrs, computed, watch } from 'vue'

defineOptions({ inheritAttrs: false })

const attrs = useAttrs()
const errored = ref(false)

const imgAttrs = computed(() => {
  const { class: _c, style: _s, ...rest } = attrs
  return rest
})

watch(() => attrs.src, () => { errored.value = false })
</script>

<template>
  <div class="safe-image-root" :class="attrs.class" :style="attrs.style">
    <img
      v-bind="imgAttrs"
      :class="{ 'safe-image-hidden': errored }"
      @error="errored = true"
      @load="errored = false"
    />
    <div v-show="errored" class="safe-image-fallback">
      <svg viewBox="0 0 80 80" fill="none" xmlns="http://www.w3.org/2000/svg">
        <rect x="8" y="12" width="64" height="56" rx="6" fill="currentColor" opacity="0.06" />
        <rect
          x="8" y="12" width="64" height="56" rx="6"
          stroke="currentColor" stroke-width="1.2" opacity="0.18"
        />
        <circle cx="28" cy="28" r="5.5" fill="currentColor" opacity="0.12" />
        <path d="M8 56 L26 36 L40 48 L52 36 L72 56 Z" fill="currentColor" opacity="0.08" />
        <path
          d="M8 56 L26 36 L40 48 L52 36 L72 56"
          stroke="currentColor" stroke-width="1.2" stroke-linejoin="round" opacity="0.18"
        />
      </svg>
    </div>
  </div>
</template>

<style scoped>
.safe-image-root {
  position: relative;
}
.safe-image-hidden {
  visibility: hidden;
}
.safe-image-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  background: var(--color-placeholder);
  border-radius: inherit;
}
.safe-image-fallback svg {
  width: 36%;
  min-width: 24px;
  max-width: 64px;
  height: auto;
}
</style>

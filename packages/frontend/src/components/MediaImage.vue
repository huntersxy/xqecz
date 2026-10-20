<script setup lang="ts">
import { toRef } from 'vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import { useImageFallback } from '@/composables/useImageFallback'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{ src?: string; alt?: string; fallbackSrc?: string | string[] }>(),
  { src: '', alt: '', fallbackSrc: '' },
)

const emit = defineEmits<{ error: []; fallback: [failedUrl: string] }>()

const { currentSrc, finalFailed, onLoadError } = useImageFallback({
  src: toRef(props, 'src'),
  fallbackSrc: toRef(props, 'fallbackSrc'),
  onFallback: (failed) => emit('fallback', failed),
  onFinalError: () => emit('error'),
})
</script>

<template>
  <a-image v-bind="$attrs" :src="currentSrc" :alt="alt">
    <template #extra>
      <slot name="extra" />
    </template>
    <template v-if="$slots.loader" #loader>
      <slot name="loader" />
    </template>
    <template #error>
      <span :ref="(el) => el && onLoadError()" />
      <slot v-if="finalFailed" name="error">
        <div class="arco-image-error">
          <div class="arco-image-error-icon">
            <IconImageClose />
          </div>
          <div v-if="alt" class="arco-image-error-alt">{{ alt }}</div>
        </div>
      </slot>
      <slot v-else name="loader" />
    </template>
  </a-image>
</template>

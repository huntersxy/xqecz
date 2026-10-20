<script setup lang="ts">
import { toRef } from 'vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import { useImageFallback } from '@/composables/useImageFallback'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{ src?: string; alt?: string; fallbackSrc?: string | string[] }>(),
  { src: '', alt: '', fallbackSrc: '' },
)

/**
 * `fallback` 会带上**失败的那个地址**，调用方据此判断是哪一级不可达
 *（OpenList 还是 R2），从而只修正那一级的结论。
 */
const emit = defineEmits<{ error: []; fallback: [failedUrl: string] }>()

// 兜底链与逐级回退规则在 useImageFallback 里，SiteImage 共用同一套。
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

<script setup lang="ts">
import { ref, watch } from 'vue'
import { IconImageClose } from '@arco-design/web-vue/es/icon'
import { getImageUrl, getRemoteFallbackUrl } from '@/utils'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{ src?: string; alt?: string }>(),
  { src: '', alt: '' }
)

const emit = defineEmits<{ error: [] }>()

const currentSrc = ref(getImageUrl(props.src))
const finalFailed = ref(false)
let triedRemote = false

watch(
  () => props.src,
  (value) => {
    currentSrc.value = getImageUrl(value)
    triedRemote = false
    finalFailed.value = false
  }
)

/**
 * 首次加载失败：换成本站媒体同路径的生产服务器地址重试；
 * 生产地址也失败（或本就没有可兜底地址）后，才对外 emit error 并展示坏图。
 */
function onLoadError() {
  if (finalFailed.value) return
  if (!triedRemote) {
    const remote = getRemoteFallbackUrl(currentSrc.value)
    if (remote && remote !== currentSrc.value) {
      triedRemote = true
      currentSrc.value = remote
      return
    }
  }
  finalFailed.value = true
  emit('error')
}
</script>

<template>
  <a-image v-bind="$attrs" :src="currentSrc" :alt="alt">
    <template #extra>
      <slot name="extra" />
    </template>
    <template #error>
      <span :ref="(el) => el && onLoadError()" />
      <div v-if="finalFailed" class="arco-image-error">
        <div class="arco-image-error-icon">
          <IconImageClose />
        </div>
        <div v-if="alt" class="arco-image-error-alt">{{ alt }}</div>
      </div>
    </template>
  </a-image>
</template>

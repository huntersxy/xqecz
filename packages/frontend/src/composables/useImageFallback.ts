import { computed, ref, watch, type Ref } from 'vue'
import { getImageUrl, getRemoteFallbackUrl } from '@/utils'

export interface ImageFallbackOptions {
  src: Ref<string | undefined>
  fallbackSrc: Ref<string | string[] | undefined>
  onFallback?: (failedUrl: string) => void
  onFinalError?: () => void
}

export function useImageFallback(options: ImageFallbackOptions) {
  const fallbacks = computed(() => {
    const raw = options.fallbackSrc.value
    const list = Array.isArray(raw) ? raw : [raw]
    return list.filter((u): u is string => typeof u === 'string' && u !== '')
  })

  const currentSrc = ref(getImageUrl(options.src.value ?? ''))
  const finalFailed = ref(false)
  let attempted: string[] = []
  let triedRemote = false

  watch(options.src, (value) => {
    currentSrc.value = getImageUrl(value ?? '')
    attempted = []
    triedRemote = false
    finalFailed.value = false
  })

  function onLoadError() {
    if (finalFailed.value) return
    const failed = currentSrc.value
    attempted.push(failed)

    for (const fb of fallbacks.value) {
      if (!attempted.includes(fb)) {
        currentSrc.value = fb
        options.onFallback?.(failed)
        return
      }
    }
    if (!triedRemote) {
      const remote = getRemoteFallbackUrl(failed)
      if (remote && !attempted.includes(remote)) {
        triedRemote = true
        currentSrc.value = remote
        return
      }
    }
    finalFailed.value = true
    options.onFinalError?.()
  }

  return { currentSrc, finalFailed, onLoadError }
}

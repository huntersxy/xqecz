import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { contentApi } from '@/api'
import type { MetadataOptions } from '@/types'
import { hasAIGCTag, mergeAIGCTag } from '@/utils/contentTags'

// One owner for all upload entrances: switching off restores the untouched manual draft.
export function useAutoMetadata(file: () => File | undefined) {
  const autoMetadata = ref(false)
  const isAiGenerated = ref(false)
  const metadataOptions = ref<MetadataOptions>({ available: false, ready: false, placeholder_title: '', placeholder_tag: '' })
  const canGenerate = computed(() => !!file()?.type.startsWith('image/') && metadataOptions.value.available)
  let controller: AbortController | undefined
  let disposed = false
  async function refreshMetadataOptions() {
    if (disposed) return
    controller?.abort()
    const request = new AbortController()
    controller = request
    try {
      const res = await contentApi.metadataOptions(request.signal)
      if (!request.signal.aborted && res.code === 200) metadataOptions.value = res.data
    } catch (error) {
      if (!request.signal.aborted) console.warn('自动标题与标签配置暂不可用', error)
    }
  }
  onMounted(refreshMetadataOptions)
  onBeforeUnmount(() => { disposed = true; controller?.abort() })
  watch(canGenerate, available => { if (!available) autoMetadata.value = false })
  const metadataTitle = (manual: string) => autoMetadata.value ? metadataOptions.value.placeholder_title : manual
  const metadataTags = (manual: string[]) => mergeAIGCTag(
    autoMetadata.value ? [metadataOptions.value.placeholder_tag] : manual,
    isAiGenerated.value || hasAIGCTag(manual),
  )
  return { autoMetadata, isAiGenerated, metadataOptions, canGenerate, metadataTitle, metadataTags, refreshMetadataOptions }
}

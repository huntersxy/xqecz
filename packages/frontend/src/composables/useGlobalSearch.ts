import { computed, watch } from 'vue'
import { useHomeStore } from '@/stores/home'

export function useGlobalSearch() {
  const homeStore = useHomeStore()

  const searchKeyword = computed({
    get: () => homeStore.searchKeyword,
    set: (v: string) => { homeStore.searchKeyword = v },
  })

  function triggerSearch() {
    homeStore.triggerSearch()
  }

  return { searchKeyword, triggerSearch }
}

export function watchGlobalSearch(handler: () => void) {
  const homeStore = useHomeStore()
  watch(() => homeStore.searchTrigger, handler)
}

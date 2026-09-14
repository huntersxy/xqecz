import { ref, computed } from 'vue'
import { useStorage, useDebounceFn } from '@vueuse/core'
import { contentApi } from '@/api'
import { useHomeStore } from '@/stores/home'

function handleSearch(onSearch: () => void) {
  const debouncedSearch = useDebounceFn(onSearch, 300)
  debouncedSearch()
}

export function useSearchFilter() {
  const homeStore = useHomeStore()

  const allTags = ref<string[]>([])
  const selectedTags = ref<string[]>(homeStore.selectedTags)
  // 搜索关键字直接用 store 的 ref（单一真相源）
  const searchKeyword = computed({
    get: () => homeStore.searchKeyword,
    set: (v: string) => { homeStore.searchKeyword = v },
  })

  const cachedTags = useStorage<{ tags: string[]; date: string } | null>('home_tags_cache', null)

  const sortedTags = computed(() => {
    return [...allTags.value].sort((a, b) => a.localeCompare(b, 'zh-CN'))
  })

  async function loadTags() {
    try {
      const today = new Date().toDateString()
      if (cachedTags.value?.date === today && Array.isArray(cachedTags.value?.tags)) {
        allTags.value = cachedTags.value.tags
        return
      }

      const res = await contentApi.getTags()
      if (res.code === 200) {
        allTags.value = res.data
        cachedTags.value = { tags: res.data, date: today }
      }
    } catch (error) {
      console.error('加载标签失败', error)
    }
  }

  function selectTag(tag: string, onFilterChange: () => void) {
    const index = selectedTags.value.indexOf(tag)
    if (index > -1) {
      selectedTags.value = []
    } else {
      selectedTags.value = [tag]
    }
    onFilterChange()
  }

  return {
    allTags,
    selectedTags,
    searchKeyword,
    sortedTags,
    loadTags,
    selectTag,
    handleSearch,
  }
}

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

  // 展示顺序独立于每日缓存；只在加载或主动漫游时洗牌，筛选与重渲染不改顺序。
  const shuffledTags = ref<string[]>([])
  function reshuffleTags() {
    const tags = [...allTags.value]
    for (let i = tags.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1))
      const current = tags[i]!
      tags[i] = tags[j]!
      tags[j] = current
    }
    shuffledTags.value = tags
  }

  async function loadTags() {
    try {
      const today = new Date().toDateString()
      if (cachedTags.value?.date === today && Array.isArray(cachedTags.value?.tags)) {
        allTags.value = cachedTags.value.tags
        reshuffleTags()
        return
      }

      const res = await contentApi.getTags()
      if (res.code === 200) {
        allTags.value = res.data
        reshuffleTags()
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
    shuffledTags,
    reshuffleTags,
    loadTags,
    selectTag,
    handleSearch,
  }
}

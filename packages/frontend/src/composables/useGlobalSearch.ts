import { computed, watch } from 'vue'
import { useHomeStore } from '@/stores/home'

// 全局搜索：单一真相源在 home store。
// App.vue 通过 searchKeyword 写入，HomeView 通过 watchGlobalSearch 监听触发查询。
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

// 监听搜索触发器：HomeView 订阅，触发时重新查询
export function watchGlobalSearch(handler: () => void) {
  const homeStore = useHomeStore()
  watch(() => homeStore.searchTrigger, handler)
}

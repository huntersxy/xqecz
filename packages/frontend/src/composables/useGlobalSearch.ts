import { ref, watch } from 'vue'

// 模块级单例：搜索关键字。App.vue 的搜索框写入，WaterfallTheme 监听变化触发请求。
const searchKeyword = ref('')
const searchTrigger = ref(0)

export function useGlobalSearch() {
  function triggerSearch() {
    searchTrigger.value++
  }
  return { searchKeyword, searchTrigger, triggerSearch }
}

// 直接 watch 模块级 ref：App.vue 触发时，订阅者会拿到新值
export function watchGlobalSearch(handler: () => void) {
  watch(searchTrigger, handler)
}

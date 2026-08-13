import type { Content } from '@/types'

const CACHE_KEY = 'home-list-cache'
const CACHE_VERSION = 1
const CACHE_TTL = 7 * 24 * 60 * 60 * 1000 // 7天

interface CacheData {
  version: number
  timestamp: number
  data: {
    list: Content[]
    total: number
    totalPages: number
  }
}

interface DiffResult {
  merged: Content[]
  added: Set<string | number>
  removed: Set<string | number>
  updated: Set<string | number>
}

export function diffLists(cached: Content[], fresh: Content[]): DiffResult {
  const cachedMap = new Map(cached.map((item) => [item.id, item]))
  const freshMap = new Map(fresh.map((item) => [item.id, item]))

  const added = new Set<string | number>()
  const removed = new Set<string | number>()
  const updated = new Set<string | number>()

  // removed 判定前提：fresh 必须覆盖 cached 全集（fresh 是"全量快照"）。
  // 若 fresh 只是部分拉取（数量不足），把 cached 中未覆盖的条目判为已删除是误判，
  // 会导致已加载列表被截断——此时 removed 恒空，只做新增合并，保证安全。
  const isFullSnapshot = fresh.length >= cached.length

  for (const id of freshMap.keys()) {
    if (!cachedMap.has(id)) added.add(id)
  }

  for (const id of cachedMap.keys()) {
    if (!freshMap.has(id)) {
      if (isFullSnapshot) removed.add(id)
    } else {
      updated.add(id)
    }
  }

  const merged: Content[] = fresh.map((item) => {
    const cachedItem = cachedMap.get(item.id)
    return cachedItem ? { ...cachedItem, ...item } : item
  })

  return { merged, added, removed, updated }
}

export function useListCache() {
  function save(list: Content[], total: number, totalPages: number) {
    try {
      const cache: CacheData = {
        version: CACHE_VERSION,
        timestamp: Date.now(),
        data: { list, total, totalPages },
      }
      localStorage.setItem(CACHE_KEY, JSON.stringify(cache))
    } catch (e) {
      console.warn('缓存写入失败:', e)
    }
  }

  function load(): { list: Content[]; total: number; totalPages: number } | null {
    try {
      const raw = localStorage.getItem(CACHE_KEY)
      if (!raw) return null

      const cache: CacheData = JSON.parse(raw)
      if (cache.version !== CACHE_VERSION) return null
      if (Date.now() - cache.timestamp > CACHE_TTL) return null

      return cache.data
    } catch (e) {
      console.warn('缓存读取失败:', e)
      return null
    }
  }

  function clear() {
    localStorage.removeItem(CACHE_KEY)
  }

  return { save, load, clear }
}

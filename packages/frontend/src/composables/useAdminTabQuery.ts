import { computed, onMounted, toValue, watch } from 'vue'
import type { ComputedRef, MaybeRefOrGetter, Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export const COMMON_TAB_KEYS = ['my', 'upload', 'api-keys', 'api-docs'] as const

export const ADMIN_TAB_KEYS = [
  'dashboard',
  'my',
  'upload',
  'review',
  'all',
  'users',
  'polls',
  'reports',
  'vision',
  'ai',
  'api-keys',
  'api-docs',
  'logs',
] as const

export const TAB_ALIASES: Record<string, string> = {
  pending: 'review',
  claims: 'review',
}

export const ADMIN_DEFAULT_TAB = 'dashboard'
export const USER_DEFAULT_TAB = 'my'

export function adminTabsFor(isAdmin: boolean): readonly string[] {
  return isAdmin ? ADMIN_TAB_KEYS : COMMON_TAB_KEYS
}

export function defaultAdminTab(isAdmin: boolean): string {
  return isAdmin ? ADMIN_DEFAULT_TAB : USER_DEFAULT_TAB
}

export interface UseAdminTabQueryOptions {
  isAdmin: MaybeRefOrGetter<boolean>
  activeTab: Ref<string>
}

export interface UseAdminTabQuery {
  allowedTabs: ComputedRef<readonly string[]>
  defaultTab: ComputedRef<string>
  isAllowed: (key: unknown) => key is string
  selectTab: (key: string) => void
}

export function useAdminTabQuery(options: UseAdminTabQueryOptions): UseAdminTabQuery {
  const route = useRoute()
  const router = useRouter()

  const allowedTabs = computed(() => adminTabsFor(toValue(options.isAdmin)))
  const defaultTab = computed(() => defaultAdminTab(toValue(options.isAdmin)))
  const allowedSet = computed(() => new Set(allowedTabs.value))

  function isAllowed(key: unknown): key is string {
    return typeof key === 'string' && allowedSet.value.has(key)
  }

  function routeTab(): string | undefined {
    const raw = route.query.tab
    return typeof raw === 'string' ? raw : undefined
  }

  function queryWith(tab: string) {
    const next: Record<string, string | string[]> = {}
    for (const [key, value] of Object.entries(route.query)) {
      if (key === 'sub' || value === null || value === undefined) continue
      next[key] = Array.isArray(value) ? value.filter((item): item is string => item !== null) : value
    }
    next.tab = tab
    return next
  }

  function normalize() {
    const raw = routeTab()
    if (raw && isAllowed(raw)) {
      options.activeTab.value = raw
      if (route.query.sub) void router.replace({ query: queryWith(raw) })
      return
    }
    const merged = raw ? TAB_ALIASES[raw] : undefined
    if (merged && isAllowed(merged)) {
      options.activeTab.value = merged
      void router.replace({ query: queryWith(merged) })
      return
    }
    const fallback = defaultTab.value
    options.activeTab.value = fallback
    if (raw !== fallback) void router.replace({ query: queryWith(fallback) })
  }

  const initial = routeTab()
  const initialKey = initial ? (isAllowed(initial) ? initial : TAB_ALIASES[initial]) : undefined
  options.activeTab.value = initialKey && isAllowed(initialKey) ? initialKey : defaultTab.value

  onMounted(normalize)
  watch(() => route.query.tab, normalize)
  watch(() => toValue(options.isAdmin), normalize)

  function selectTab(key: string) {
    const target = isAllowed(key) ? key : TAB_ALIASES[key]
    if (!target || !isAllowed(target)) return
    options.activeTab.value = target
    if (routeTab() === target && !route.query.sub) return
    void router.push({ query: queryWith(target) })
  }

  return { allowedTabs, defaultTab, isAllowed, selectTab }
}

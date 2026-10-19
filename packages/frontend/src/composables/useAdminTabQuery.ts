import { computed, onMounted, toValue, watch } from 'vue'
import type { ComputedRef, MaybeRefOrGetter, Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

/** 任意登录用户都能访问的页签（与 AdminView 的「内容 / 系统」分组一致）。 */
export const COMMON_TAB_KEYS = ['my', 'upload', 'api-keys', 'api-docs'] as const

/** 13 个页签按 AdminView 导航顺序排列；角色过滤交给 adminTabsFor。 */
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

/**
 * 旧页签别名：审核内容与认领管理已合并为「审核与认领」。
 * 老书签 / 旧链接里的 ?tab=pending、?tab=claims 规范化到 review，不丢原意图。
 */
export const TAB_ALIASES: Record<string, string> = {
  pending: 'review',
  claims: 'review',
}

export const ADMIN_DEFAULT_TAB = 'dashboard'
export const USER_DEFAULT_TAB = 'my'

/** 角色 → 合法页签集合。AdminView 可直接用它过滤导航，避免两处名单漂移。 */
export function adminTabsFor(isAdmin: boolean): readonly string[] {
  return isAdmin ? ADMIN_TAB_KEYS : COMMON_TAB_KEYS
}

/** §7.4 / D2：管理员默认总览，普通用户默认我的内容。 */
export function defaultAdminTab(isAdmin: boolean): string {
  return isAdmin ? ADMIN_DEFAULT_TAB : USER_DEFAULT_TAB
}

export interface UseAdminTabQueryOptions {
  /**
   * 管理员判定用 getter。用户资料可能异步到达；用 getter 而非布尔值，
   * 角色到位后会重新校验当前页签，把越权页签替换回默认页。
   */
  isAdmin: MaybeRefOrGetter<boolean>
  /**
   * Pinia 的 `admin.activeTab`，必须是可写 ref：
   * `const { activeTab } = storeToRefs(useAdminStore())`。
   * 直接传 `admin.activeTab` 只会拿到当时的字符串快照，不会跟随变化。
   */
  activeTab: Ref<string>
}

export interface UseAdminTabQuery {
  /** 当前角色合法页签（导航顺序）。 */
  allowedTabs: ComputedRef<readonly string[]>
  /** 当前角色的默认页签。 */
  defaultTab: ComputedRef<string>
  /** key 是否属于当前角色的合法页签；数组、null（?tab 无值）等一律非法。 */
  isAllowed: (key: unknown) => key is string
  /** 导航点击入口：合法才切换并 push；非法/越权直接忽略，不制造 403 跳转。 */
  selectTab: (key: string) => void
}

/**
 * 把 URL 的 `?tab=` 与 `admin.activeTab` 绑成唯一事实源（§7.4）。
 *
 * 数据流是单向的：route.query.tab → activeTab → 组件；导航只走 selectTab。
 * 不做 activeTab → router 的反向 watch，避免双向回环。
 *
 * 首帧在 setup 里同步定下页签，深链直接渲染目标页，不闪默认页；
 * 需要 replace 的规范化（缺失/非法/越权）推迟到 onMounted——App.vue 以
 * `route.fullPath` 作整壳 key，setup 阶段导航会立刻把组件连同状态重挂载。
 */
export function useAdminTabQuery(options: UseAdminTabQueryOptions): UseAdminTabQuery {
  const route = useRoute()
  const router = useRouter()

  const allowedTabs = computed(() => adminTabsFor(toValue(options.isAdmin)))
  const defaultTab = computed(() => defaultAdminTab(toValue(options.isAdmin)))
  const allowedSet = computed(() => new Set(allowedTabs.value))

  function isAllowed(key: unknown): key is string {
    return typeof key === 'string' && allowedSet.value.has(key)
  }

  // 数组（?tab=a&tab=b）与 null（?tab 无值）都不是合法单值，按非法处理。
  function routeTab(): string | undefined {
    const raw = route.query.tab
    return typeof raw === 'string' ? raw : undefined
  }

  // 保留其余 query 参数，只覆盖 tab；合并前遗留的 sub 参数一并清掉
  function queryWith(tab: string) {
    const next: Record<string, string | string[]> = {}
    for (const [key, value] of Object.entries(route.query)) {
      if (key === 'sub' || value === null || value === undefined) continue
      next[key] = Array.isArray(value) ? value.filter((item): item is string => item !== null) : value
    }
    next.tab = tab
    return next
  }

  /** 把 URL 与 activeTab 拉回合法状态；值没变就不导航，避免 watch 回环与多余历史。 */
  function normalize() {
    const raw = routeTab()
    if (raw && isAllowed(raw)) {
      options.activeTab.value = raw
      if (route.query.sub) void router.replace({ query: queryWith(raw) })
      return
    }
    // 旧别名先翻译，再退回默认页：老链接打开后应落在合并后的页签上
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
  // 浏览器前进/后退、手改地址栏都会走这里
  watch(() => route.query.tab, normalize)
  // 登录态/角色变化后，原先合法的页签可能变成越权页签
  watch(() => toValue(options.isAdmin), normalize)

  function selectTab(key: string) {
    // 导航与仪表盘待处理带仍可能传来旧 key，这里统一收口到合并后的页签
    const target = isAllowed(key) ? key : TAB_ALIASES[key]
    if (!target || !isAllowed(target)) return
    options.activeTab.value = target
    // tab 已在地址栏时不再 push，避免同一页签堆历史记录
    if (routeTab() === target && !route.query.sub) return
    void router.push({ query: queryWith(target) })
  }

  return { allowedTabs, defaultTab, isAllowed, selectTab }
}

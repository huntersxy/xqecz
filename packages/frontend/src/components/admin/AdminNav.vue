<script lang="ts">
import type { Component } from 'vue'

interface AdminNavItem {
  key: string
  title: string
  icon: Component
  badge?: number
}

export interface AdminNavGroup {
  label: string
  items: AdminNavItem[]
}
</script>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { IconDown } from '@arco-design/web-vue/es/icon'
import { closeShellSidebar } from '@/utils/shellSidebar'

interface Props {
  groups: AdminNavGroup[]
  active: string
  isAdmin?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  select: [key: string]
}>()

const COLLAPSE_KEY = 'admin_nav_collapsed_groups'

function readCollapsed(): string[] {
  try {
    const raw = localStorage.getItem(COLLAPSE_KEY)
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((label): label is string => typeof label === 'string')
  } catch {
    return []
  }
}

const collapsed = ref<string[]>(readCollapsed())

function isCollapsed(label: string) {
  return collapsed.value.includes(label)
}

function toggleGroup(label: string) {
  const next = isCollapsed(label)
    ? collapsed.value.filter((item) => item !== label)
    : [...collapsed.value, label]
  collapsed.value = next
  try {
    localStorage.setItem(COLLAPSE_KEY, JSON.stringify(next))
  } catch {
  }
}

const activeGroupLabel = computed(
  () => props.groups.find((group) => group.items.some((item) => item.key === props.active))?.label,
)

function revealActiveGroup() {
  const label = activeGroupLabel.value
  if (label && isCollapsed(label)) collapsed.value = collapsed.value.filter((item) => item !== label)
}

const listEl = ref<HTMLElement | null>(null)
const itemEls = new Map<string, HTMLElement>()
const indicatorY = ref(0)
const indicatorHeight = ref(0)
const indicatorVisible = ref(false)
const reduceMotion = ref(false)

function setItemRef(key: string, el: unknown) {
  if (el instanceof HTMLElement) itemEls.set(key, el)
  else itemEls.delete(key)
}

function measureIndicator() {
  const list = listEl.value
  const activeEl = itemEls.get(props.active)
  const label = activeGroupLabel.value
  if (!list || !activeEl || (label && isCollapsed(label))) {
    indicatorVisible.value = false
    return
  }
  indicatorY.value = activeEl.offsetTop
  indicatorHeight.value = activeEl.offsetHeight
  indicatorVisible.value = true
}

const indicatorStyle = computed(() => ({
  transform: `translateY(${indicatorY.value}px)`,
  height: `${indicatorHeight.value}px`,
}))

let observer: ResizeObserver | undefined

function scheduleMeasure() {
  void nextTick(measureIndicator)
}

onMounted(() => {
  revealActiveGroup()
  reduceMotion.value = globalThis.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
  scheduleMeasure()
  if (typeof ResizeObserver !== 'undefined' && listEl.value) {
    observer = new ResizeObserver(scheduleMeasure)
    observer.observe(listEl.value)
  }
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = undefined
})

watch(
  () => props.active,
  () => {
    revealActiveGroup()
    scheduleMeasure()
  },
)

watch(
  () => props.groups,
  () => scheduleMeasure(),
  { deep: true },
)

function select(key: string) {
  emit('select', key)
  closeShellSidebar()
}
</script>

<template>
  <Teleport defer to="#public-shell-workspace">
    <nav class="admin-nav" aria-label="后台导航">
      <div ref="listEl" class="admin-nav-list">
        <span
          v-if="indicatorVisible"
          class="admin-nav-indicator"
          :class="{ 'is-static': reduceMotion }"
          aria-hidden="true"
          :style="indicatorStyle"
        />

        <section
          v-for="(group, groupIndex) in props.groups"
          :key="group.label"
          class="admin-nav-group"
          :aria-label="group.label"
        >
          <h2 class="admin-nav-group-head">
            <button
              type="button"
              class="admin-nav-group-toggle"
              :class="{ 'is-collapsed': isCollapsed(group.label) }"
              :aria-expanded="!isCollapsed(group.label)"
              :aria-controls="`admin-nav-group-${groupIndex}`"
              @click="toggleGroup(group.label)"
            >
              <span>{{ group.label }}</span>
              <IconDown class="admin-nav-chevron" aria-hidden="true" />
            </button>
          </h2>

          <div
            :id="`admin-nav-group-${groupIndex}`"
            class="admin-nav-group-body"
            :class="{ 'is-collapsed': isCollapsed(group.label) }"
          >
            <div class="admin-nav-group-inner">
              <template v-for="item in group.items" :key="item.key">
                <button
                  :ref="(el) => setItemRef(item.key, el)"
                  type="button"
                  class="admin-nav-item"
                  :class="{ 'is-active': props.active === item.key }"
                  :aria-current="props.active === item.key ? 'page' : undefined"
                  :aria-label="item.badge ? `${item.title}，${item.badge} 项待处理` : undefined"
                  @click="select(item.key)"
                >
                  <component :is="item.icon" class="admin-nav-icon" aria-hidden="true" />
                  <span class="admin-nav-item-label">{{ item.title }}</span>
                  <span v-if="item.badge" class="admin-nav-badge">{{ item.badge > 99 ? '99+' : item.badge }}</span>
                </button>
              </template>
            </div>
          </div>
        </section>
      </div>
    </nav>
  </Teleport>
</template>

<style lang="scss" scoped>
.admin-nav {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 0 6px;
  color: var(--adm-muted, var(--admin-text-2, #6b6360));
}

@media (hover: hover) and (pointer: fine) {
  .admin-nav { --adm-nav-item-h: 32px; }
}

.admin-nav-list {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.admin-nav-indicator {
  position: absolute;
  top: 0;
  left: 0;
  width: 3px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-accent, var(--site-accent));
  pointer-events: none;
  transition:
    transform var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1)),
    height var(--adm-dur-base, 220ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));
}

.admin-nav-indicator.is-static {
  transition: none;
}

.admin-nav-group {
  min-width: 0;
}

.admin-nav-group-head {
  margin: 0;
  font: inherit;
}

.admin-nav-group-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-height: 22px;
  padding: 1px 10px;
  border: 0;
  border-radius: var(--adm-r-control, 10px);
  background: transparent;
  color: var(--adm-muted, var(--admin-text-3, #8a8078));
  font: inherit;
  font-size: var(--adm-font-xs, 12px);
  letter-spacing: 0.1em;
  cursor: pointer;
}

.admin-nav-group-toggle:hover {
  color: var(--adm-ink, var(--admin-text, #221c1e));
}

.admin-nav-chevron {
  flex: 0 0 auto;
  font-size: 10px;
  transition: transform var(--adm-dur-fast, 120ms) var(--adm-ease-out, cubic-bezier(0.22, 1, 0.36, 1));
}

.admin-nav-group-toggle.is-collapsed .admin-nav-chevron {
  transform: rotate(-90deg);
}

.admin-nav-group-body {
  display: grid;
  grid-template-rows: 1fr;
  transition: grid-template-rows var(--adm-dur-base, 220ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));
}

.admin-nav-group-body.is-collapsed {
  grid-template-rows: 0fr;
}

.admin-nav-group-inner {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-height: 0;
  padding-bottom: 2px;
  overflow: hidden;
  opacity: 1;
  transition: opacity var(--adm-dur-fast, 120ms) linear;
}

.admin-nav-group-body.is-collapsed .admin-nav-group-inner {
  opacity: 0;
}

.admin-nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: var(--adm-nav-item-h, var(--adm-touch, 36px));
  padding: 5px 10px;
  border: 0;
  border-radius: var(--adm-r-control, 10px);
  background: transparent;
  color: var(--adm-ink, var(--admin-text, #221c1e));
  font: inherit;
  font-size: var(--adm-font-sm, 13px);
  text-align: left;
  cursor: pointer;
  transition: background-color var(--adm-dur-fast, 120ms) linear, color var(--adm-dur-fast, 120ms) linear;
}

.admin-nav-item:hover {
  background: var(--adm-soft, var(--admin-fill, #f4efec));
}

.admin-nav-item.is-active {
  background: var(--adm-accent-soft, var(--admin-primary-soft, #f2e7e3));
  color: var(--adm-accent, var(--site-accent));
  font-weight: 600;
}

.admin-nav-icon {
  flex: 0 0 auto;
  font-size: 15px;
  color: inherit;
}

.admin-nav-item-label {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-nav-badge {
  flex: 0 0 auto;
  min-width: 18px;
  padding: 0 5px;
  border-radius: var(--adm-r-pill, 999px);
  background: var(--adm-danger, var(--admin-danger, #c0392b));
  color: #fff;
  font-size: var(--adm-font-xs, 12px);
  font-variant-numeric: var(--adm-numeric, tabular-nums);
  line-height: 18px;
  text-align: center;
}

.admin-nav-item:focus-visible,
.admin-nav-group-toggle:focus-visible {
  outline: 2px solid var(--adm-accent, var(--site-accent));
  outline-offset: 2px;
}

@media (prefers-reduced-motion: reduce) {
  .admin-nav-indicator,
  .admin-nav-chevron,
  .admin-nav-group-body,
  .admin-nav-group-inner,
  .admin-nav-item {
    transition: none !important;
  }
}
</style>

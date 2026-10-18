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
interface Props {
  groups: AdminNavGroup[]
  active: string
  isAdmin?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  select: [key: string]
}>()
</script>

<template>
  <nav class="admin-nav" aria-label="工作区功能">
    <div class="admin-nav-heading">
      <strong>{{ isAdmin ? '管理工作台' : '创作者空间' }}</strong>
      <span>在这里切换功能</span>
    </div>

    <div class="admin-nav-groups">
      <section
        v-for="group in groups"
        :key="group.label"
        class="admin-nav-group"
        :class="{ 'is-wide': group.label === '管理' }"
        :aria-label="group.label"
      >
        <h2 class="admin-nav-group-label">{{ group.label }}</h2>
        <div class="admin-nav-items">
          <button
            v-for="item in group.items"
            :key="item.key"
            type="button"
            class="admin-nav-item"
            :class="{ 'is-active': active === item.key }"
            :aria-current="active === item.key ? 'page' : undefined"
            :aria-label="item.badge ? `${item.title}，${item.badge} 项待处理` : undefined"
            @click="emit('select', item.key)"
          >
            <component :is="item.icon" class="admin-nav-icon" aria-hidden="true" />
            <span class="admin-nav-item-label">{{ item.title }}</span>
            <span v-if="item.badge" class="admin-nav-badge">{{ item.badge > 99 ? '99+' : item.badge }}</span>
          </button>
        </div>
      </section>
    </div>
  </nav>
</template>

<style lang="scss" scoped>
.admin-nav {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  align-items: start;
  gap: 24px;
  padding: 8px 0 20px;
  border-bottom: 1px solid var(--site-line);
}

.admin-nav-heading {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding-top: 2px;

  strong {
    color: var(--site-ink);
    font: 600 17px/1.25 var(--site-title-font);
    letter-spacing: -.02em;
  }

  span {
    color: var(--site-muted);
    font-size: 10px;
  }
}

.admin-nav-groups {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px 24px;
  min-width: 0;
}

.admin-nav-group {
  min-width: 0;
}

.admin-nav-group.is-wide {
  flex: 1 1 420px;
}

.admin-nav-group-label {
  margin: 0 0 7px 8px;
  color: var(--site-muted);
  font-size: 10px;
  font-weight: 500;
  letter-spacing: .08em;
}

.admin-nav-items {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 3px;
}

.admin-nav-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
  padding: 0 9px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--site-muted);
  font: inherit;
  font-size: 11px;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color .16s ease, color .16s ease, border-color .16s ease;

  &:hover {
    background: var(--site-soft);
    color: var(--site-ink);
  }

  &.is-active {
    border-color: color-mix(in srgb, var(--site-accent) 18%, transparent);
    background: var(--site-accent-soft);
    color: var(--site-accent);
    font-weight: 600;
  }

  &:focus-visible {
    outline: 2px solid var(--site-accent);
    outline-offset: 2px;
  }
}

.admin-nav-icon {
  flex: 0 0 auto;
  font-size: 14px;
}

.admin-nav-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 17px;
  height: 17px;
  padding: 0 4px;
  border-radius: 9px;
  background: var(--admin-danger-soft);
  color: var(--admin-danger);
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

@media (max-width: 900px) {
  .admin-nav {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
    padding: 4px 0 16px;
  }

  .admin-nav-heading {
    flex-direction: row;
    align-items: baseline;
    justify-content: space-between;
  }

  .admin-nav-groups {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 14px;
  }

  .admin-nav-group.is-wide {
    grid-column: 1 / -1;
  }

  .admin-nav-item {
    min-height: 32px;
    padding-inline: 8px;
  }
}

@media (max-width: 520px) {
  .admin-nav-groups {
    grid-template-columns: minmax(0, 1fr);
  }

  .admin-nav-group.is-wide {
    grid-column: auto;
  }

  .admin-nav-items {
    gap: 2px;
  }

  .admin-nav-item {
    font-size: 10px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .admin-nav-item { transition: none; }
}
</style>

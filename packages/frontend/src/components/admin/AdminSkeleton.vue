<script setup lang="ts">
// 加载骨架（M11）：用静态占位 + 一道扫光，替代会让整块内容空转的 spin。
// §8.3 只允许 M10/M11 无限循环，所以这里显式提供暂停能力：页面不可见时停、回来再继续；
// 命中 prefers-reduced-motion 时直接不起动画（样式里处理，JS 不参与）。
interface Props {
  /** rows：表格 / 列表的行骨架；cards：卡片（含手机卡片列表）骨架 */
  variant?: 'rows' | 'cards'
  /** 占位行 / 卡片数量 */
  rows?: number
  /** rows 模式每行的占位单元格数量 */
  columns?: number
  /** 读屏播报文案 */
  label?: string
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'rows',
  rows: 5,
  columns: 4,
  label: '正在加载',
})

// 骨架只是占位，行数与列数收个上限，避免调用方传大数字把页面撑爆
const rowCount = computed(() => Math.min(20, Math.max(1, props.rows)))
const colCount = computed(() => Math.min(8, Math.max(1, props.columns)))

const paused = ref(false)

function syncPaused() {
  paused.value = document.visibilityState === 'hidden'
}

onMounted(() => {
  syncPaused()
  document.addEventListener('visibilitychange', syncPaused)
})

onBeforeUnmount(() => document.removeEventListener('visibilitychange', syncPaused))

// 宽度按固定序列错开，避免每行等长产生「栅栏感」；用下标取模保证每次渲染完全一致
const widths = ['46%', '72%', '58%', '34%', '64%', '42%', '56%', '38%']

function cellWidth(row: number, col: number) {
  return widths[(row * 3 + col) % widths.length] ?? '50%'
}
</script>

<template>
  <div
    class="admin-skeleton"
    :class="[`is-${variant}`, { 'is-paused': paused }]"
    role="status"
    aria-busy="true"
  >
    <span class="admin-skeleton-sr">{{ label }}</span>
    <template v-if="variant === 'rows'">
      <div v-for="row in rowCount" :key="`row-${row}`" class="admin-skeleton-row">
        <span
          v-for="col in colCount"
          :key="`cell-${col}`"
          class="admin-skeleton-cell"
          :style="{ width: cellWidth(row, col) }"
        />
      </div>
    </template>
    <template v-else>
      <div v-for="card in rowCount" :key="`card-${card}`" class="admin-skeleton-card">
        <span class="admin-skeleton-line is-title" />
        <span class="admin-skeleton-line is-wide" />
        <span class="admin-skeleton-line is-short" />
      </div>
    </template>
  </div>
</template>

<style lang="scss" scoped>
.admin-skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-md, 12px);
  padding: var(--adm-gap-lg, 16px);

  &.is-rows {
    gap: 0;
    padding: 0;
  }

  &.is-cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    align-items: stretch;
  }
}

.admin-skeleton-sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

.admin-skeleton-row {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-lg, 16px);
  padding: 16px var(--adm-gap-xl, 24px);
  border-bottom: 1px solid var(--admin-border-soft);

  &:last-child {
    border-bottom: 0;
  }
}

.admin-skeleton-cell {
  height: 12px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--adm-line, var(--admin-border)) 45%, var(--adm-soft, var(--admin-fill)));
}

.admin-skeleton-card {
  display: flex;
  flex-direction: column;
  gap: var(--adm-gap-sm, 8px);
  padding: var(--adm-gap-lg, 16px);
  border: 1px solid var(--admin-border-soft);
  border-radius: var(--adm-r-card, var(--admin-radius));
}

.admin-skeleton-line {
  height: 10px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--adm-line, var(--admin-border)) 45%, var(--adm-soft, var(--admin-fill)));

  &.is-title {
    height: 14px;
    width: 52%;
  }

  &.is-wide {
    width: 68%;
  }

  &.is-short {
    width: 44%;
  }
}

// 扫光：一条高光带用 transform 扫过整行，只动 transform（§8.1 第 2 条），
// 不用 background-position 动画，避免每帧重绘。
.admin-skeleton-row,
.admin-skeleton-card {
  position: relative;
  overflow: hidden;

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    pointer-events: none;
    background: linear-gradient(
      100deg,
      transparent 30%,
      color-mix(in srgb, var(--adm-surface, var(--admin-surface)) 72%, transparent) 50%,
      transparent 70%
    );
    transform: translateX(-100%);
    animation: adm-skeleton-sweep 1.5s var(--adm-ease-std, cubic-bezier(.4, 0, .2, 1)) infinite;
  }
}

.is-paused .admin-skeleton-row::after,
.is-paused .admin-skeleton-card::after {
  animation-play-state: paused;
}

@keyframes adm-skeleton-sweep {
  to {
    transform: translateX(100%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .admin-skeleton-row::after,
  .admin-skeleton-card::after {
    animation: none;
  }
}
</style>

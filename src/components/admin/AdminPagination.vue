<script setup lang="ts">
interface Props {
  currentPage: number
  totalPages: number
}

const props = defineProps<Props>()

const emit = defineEmits<{
  change: [page: number]
}>()

const handlePrev = () => {
  if (props.currentPage > 1) emit('change', props.currentPage - 1)
}

const handleNext = () => {
  if (props.currentPage < props.totalPages) emit('change', props.currentPage + 1)
}
</script>

<template>
  <div class="pagination-section">
    <button @click="handlePrev" :disabled="currentPage <= 1" class="mac-btn pagination-btn">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 19l-7-7 7-7"/>
      </svg>
      上一页
    </button>
    <span class="pagination-info">第 {{ currentPage }} / {{ totalPages }} 页</span>
    <button @click="handleNext" :disabled="currentPage >= totalPages" class="mac-btn pagination-btn">
      下一页
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 5l7 7-7 7"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
interface Props {
  currentPage: number;
  totalPages: number;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  change: [page: number];
}>();

const handlePrev = () => {
  if (props.currentPage > 1) emit('change', props.currentPage - 1);
};

const handleNext = () => {
  if (props.currentPage < props.totalPages) emit('change', props.currentPage + 1);
};
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

<style scoped>
.pagination-section {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.pagination-btn {
  display: flex;
  align-items: center;
  gap: 6px;
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination-btn svg {
  width: 14px;
  height: 14px;
}

.pagination-info {
  font-size: 14px;
  color: #666;
}

@media screen and (max-width: 768px) {
  .pagination-section {
    gap: 10px;
  }

  .pagination-btn {
    min-width: 80px;
    padding: 10px;
    font-size: 14px;
  }

  .pagination-info {
    font-size: 13px;
  }
}
</style>
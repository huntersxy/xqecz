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
  <div class="flex justify-center items-center gap-4 pt-4 border-t border-gray-200/50">
    <button
      @click="handlePrev"
      :disabled="currentPage <= 1"
      class="flex items-center gap-1.5 px-3 py-2 text-[14px] theme-text bg-[var(--theme-surface)] border border-gray-200 rounded-md hover:text-[var(--theme-primary)] hover:border-blue-300 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:theme-text disabled:hover:border-gray-200 transition-colors"
    >
      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M15 19l-7-7 7-7"/>
      </svg>
      上一页
    </button>
    <span class="text-[14px] theme-text-secondary">第 {{ currentPage }} / {{ totalPages }} 页</span>
    <button
      @click="handleNext"
      :disabled="currentPage >= totalPages"
      class="flex items-center gap-1.5 px-3 py-2 text-[14px] theme-text bg-[var(--theme-surface)] border border-gray-200 rounded-md hover:text-[var(--theme-primary)] hover:border-blue-300 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:theme-text disabled:hover:border-gray-200 transition-colors"
    >
      下一页
      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 5l7 7-7 7"/>
      </svg>
    </button>
  </div>
</template>

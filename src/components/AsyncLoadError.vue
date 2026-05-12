<script setup lang="ts">
const props = defineProps<{
  error?: Error
  retry?: () => void
}>()
</script>

<template>
  <div class="async-load-error">
    <svg class="error-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10"/>
      <line x1="12" y1="8" x2="12" y2="12"/>
      <line x1="12" y1="16" x2="12.01" y2="16"/>
    </svg>
    <h3>组件加载失败</h3>
    <p>{{ error?.message || '网络连接可能有问题' }}</p>
    <button v-if="retry" @click="retry" class="retry-btn">
      重试
    </button>
  </div>
</template>

<style scoped>
.async-load-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.error-icon {
  width: 64px;
  height: 64px;
  color: #ff6b6b;
  margin-bottom: 16px;
}

.async-load-error h3 {
  font-size: 18px;
  color: #333;
  margin-bottom: 8px;
}

.async-load-error p {
  font-size: 14px;
  color: #666;
  margin-bottom: 24px;
}

.retry-btn {
  padding: 10px 24px;
  background: #007aff;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.retry-btn:hover {
  background: #0056b3;
}
</style>

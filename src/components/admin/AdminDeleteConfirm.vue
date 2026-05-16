<script setup lang="ts">
interface Props {
  visible: boolean;
}

const props = defineProps<Props>();

const emit = defineEmits<{
  close: [];
  confirm: [];
}>();
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-content">
      <div class="modal-header">
        <h2>确认删除</h2>
        <span class="modal-close" @click="$emit('close')">×</span>
      </div>
      <div class="modal-body">
        <p>确定要删除这条内容吗？此操作不可撤销。</p>
      </div>
      <div class="modal-footer">
        <button @click="$emit('close')" class="mac-btn">取消</button>
        <button @click="$emit('confirm')" class="mac-btn danger-btn">删除</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
}

.modal-content {
  width: 90%;
  max-width: 600px;
  background: rgba(255, 255, 255, 0.98);
  border-radius: 12px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  overflow: hidden;
  position: relative;
  z-index: 10000;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.05) 0%, transparent 100%);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.modal-header h2 {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.modal-close {
  font-size: 24px;
  color: #999;
  cursor: pointer;
  line-height: 1;
}

.modal-close:hover {
  color: #3b82f6;
}

.modal-body {
  padding: 20px;
}

.modal-body p {
  margin: 0;
  color: #333;
  line-height: 1.6;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(0, 0, 0, 0.03);
}

@media screen and (max-width: 768px) {
  .modal-content {
    width: 95%;
    max-width: none;
    margin: 12px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.98);
  }

  .modal-body {
    padding: 16px;
  }

  .modal-footer {
    padding: 14px 16px;
    flex-wrap: wrap;
  }
}
</style>
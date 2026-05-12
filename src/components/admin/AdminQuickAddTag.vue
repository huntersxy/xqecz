<script setup lang="ts">
import { ref } from 'vue'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  add: [tagName: string]
}>()

const tagName = ref('')

const handleAdd = () => {
  if (tagName.value.trim()) {
    emit('add', tagName.value.trim())
    tagName.value = ''
  }
}

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter') {
    handleAdd()
  }
}
</script>

<template>
  <div v-if="visible" class="tag-modal-overlay" @click.self="$emit('close')">
    <div class="tag-modal">
      <div class="tag-modal-header">
        <h3>添加标签</h3>
        <button class="tag-modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="tag-modal-body">
        <input
          v-model="tagName"
          type="text"
          class="mac-input"
          placeholder="请输入标签名称"
          @keyup.enter="handleAdd"
          autofocus
        />
      </div>
      <div class="tag-modal-footer">
        <button @click="$emit('close')" class="mac-btn">取消</button>
        <button @click="handleAdd" class="mac-btn primary-btn">确定</button>
      </div>
    </div>
  </div>
</template>

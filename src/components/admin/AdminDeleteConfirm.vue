<script setup lang="ts">
interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()
</script>

<template>
  <div v-if="visible" class="tag-modal-overlay" @click.self="$emit('close')">
    <div class="tag-modal">
      <div class="tag-modal-header">
        <h3>确认删除</h3>
        <button class="tag-modal-close" @click="$emit('close')">×</button>
      </div>
      <div class="tag-modal-body">
        <p>确定要删除这条内容吗？此操作不可撤销。</p>
      </div>
      <div class="tag-modal-footer">
        <button @click="$emit('close')" class="mac-btn">取消</button>
        <button @click="$emit('confirm')" class="mac-btn delete-btn">确定删除</button>
      </div>
    </div>
  </div>
</template>

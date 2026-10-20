<script setup lang="ts">
import { ref } from 'vue'
import { contentApi } from '@/api'
import { toast } from '@/composables/useToast'
import CdDialog from './CdDialog.vue'

const props = defineProps<{ open: boolean; contentId: number }>()
const emit = defineEmits<{ close: [] }>()

const reason = ref('')
const busy = ref(false)

async function submit() {
  if (busy.value) return
  const text = reason.value.trim()
  if (!text) {
    toast.warning('请输入认领理由')
    return
  }
  busy.value = true
  try {
    const res = await contentApi.submitClaim(props.contentId, text)
    if (res.code === 200) {
      toast.success('认领申请已提交，请等待管理员审核')
      reason.value = ''
      emit('close')
    } else {
      toast.error(res.message || '认领申请提交失败')
    }
  } catch {
    toast.error('认领申请提交失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <CdDialog :open="open" title="认领此内容" confirm-text="提交申请" :busy="busy" @close="emit('close')" @confirm="submit">
    <div class="cl-stack">
      <p class="cl-desc">请提供认领理由，管理员将在审核后决定是否将此内容转移给您。</p>
      <label class="cl-field">
        <span class="cl-label">认领理由 <span class="cl-required" aria-hidden="true">*</span></span>
        <textarea v-model="reason" class="cl-textarea" rows="4" required placeholder="请详细说明您认为此内容应归属于您的原因..."></textarea>
      </label>
    </div>
  </CdDialog>
</template>

<style scoped>
.cl-stack { display: flex; flex-direction: column; gap: 12px; }
.cl-desc { margin: 0; font-size: 13px; line-height: 1.5; color: var(--creative-muted); }
.cl-field { display: flex; flex-direction: column; gap: 4px; }
.cl-label { font-size: 13px; color: var(--creative-muted); }
.cl-required { color: #c0392b; }
.cl-textarea { min-height: 6.5em; max-height: 14em; padding: 10px 12px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); color: var(--creative-ink); font: inherit; font-size: 13px; line-height: 1.7; outline: 0; resize: vertical; field-sizing: content; }
.cl-textarea:focus { border-color: var(--creative-accent); }
</style>

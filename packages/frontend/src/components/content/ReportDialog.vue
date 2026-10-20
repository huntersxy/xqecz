<script setup lang="ts">
import { ref, watch } from 'vue'
import { commentApi } from '@/api'
import { toast } from '@/composables/useToast'
import CdDialog from './CdDialog.vue'
import type { Comment } from '@/types'

const props = defineProps<{ target: Comment | null }>()
const emit = defineEmits<{ close: [] }>()

const reason = ref('')
const busy = ref(false)

watch(() => props.target, (v) => { if (v) reason.value = '' })

async function submit() {
  if (!props.target || busy.value) return
  busy.value = true
  try {
    const res = await commentApi.report(props.target.id, reason.value.trim() || undefined)
    if (res.code === 200) {
      toast.success('举报成功，管理员将尽快处理')
      emit('close')
    } else {
      toast.error(res.message || '举报失败')
    }
  } catch {
    toast.error('举报失败')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <CdDialog :open="!!target" title="举报评论" confirm-text="确认举报" :busy="busy" @close="emit('close')" @confirm="submit">
    <div class="rd-stack">
      <div>
        <div class="rd-label">您正在举报以下评论：</div>
        <div class="rd-quote">{{ target?.text }}</div>
      </div>
      <label class="rd-field">
        <span class="rd-label">举报原因（可选）</span>
        <input v-model="reason" class="rd-input" type="text" maxlength="200" placeholder="请输入举报原因" @keydown.enter.prevent="submit" />
      </label>
    </div>
  </CdDialog>
</template>

<style scoped>
.rd-stack { display: flex; flex-direction: column; gap: 14px; }
.rd-field { display: flex; flex-direction: column; gap: 4px; }
.rd-label { margin-bottom: 4px; font-size: 13px; color: var(--creative-muted); }
.rd-field .rd-label { margin-bottom: 0; }
.rd-quote { padding: 8px 10px; border-left: 2px solid var(--creative-accent); border-radius: 12px; background: var(--creative-soft); font-size: 13px; font-style: italic; color: var(--creative-muted); overflow-wrap: anywhere; }
.rd-input { height: 36px; padding: 0 12px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); color: var(--creative-ink); font: inherit; font-size: 13px; outline: 0; }
.rd-input:focus { border-color: var(--creative-accent); }
</style>

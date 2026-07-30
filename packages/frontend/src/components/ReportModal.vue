<script setup lang="ts">
import { ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { commentApi } from '@/api'
import type { Comment } from '@/types'

const props = defineProps<{ target: Comment | null }>()
const emit = defineEmits<{ close: [] }>()

const reportReason = ref('')
const loading = ref(false)

watch(() => props.target, () => { reportReason.value = '' })

async function submitReport() {
  if (!props.target) return
  loading.value = true
  try {
    const res = await commentApi.report(props.target.id, reportReason.value || undefined)
    if (res.code === 200) {
      message.success('举报成功，管理员将尽快处理')
      emit('close')
    } else {
      message.error(res.message || '举报失败')
    }
  } catch {
    message.error('举报失败')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <a-modal
    :open="!!target"
    title="举报评论"
    @cancel="emit('close')"
    :confirm-loading="loading"
    @ok="submitReport"
    ok-text="确认举报"
    cancel-text="取消"
    :z-index="10000"
  >
    <div style="display: flex; flex-direction: column; gap: 12px;">
      <div>
        <div style="font-size: 13px; color: var(--theme-text-secondary); margin-bottom: 4px;">您正在举报以下评论：</div>
        <div style="padding: 8px 10px; background: var(--theme-hover-bg); border-radius: 6px; font-size: 13px; color: var(--theme-text-secondary); font-style: italic;">
          {{ target?.text }}
        </div>
      </div>
      <div>
        <div style="font-size: 13px; color: var(--theme-text-secondary); margin-bottom: 4px;">举报原因（可选）</div>
        <a-input v-model:value="reportReason" placeholder="请输入举报原因" />
      </div>
    </div>
  </a-modal>
</template>

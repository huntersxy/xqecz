<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { contentApi } from '@/api'

const props = defineProps<{ open: boolean; contentId: number }>()
const emit = defineEmits<{ close: [] }>()

const claimReason = ref('')
const isSubmitting = ref(false)

async function submitClaim() {
  if (!claimReason.value.trim()) {
    message.warning('请输入认领理由')
    return
  }
  try {
    isSubmitting.value = true
    const res = await contentApi.submitClaim(props.contentId, claimReason.value.trim())
    if (res.code === 200) {
      message.success('认领申请已提交，请等待管理员审核')
      claimReason.value = ''
      emit('close')
    } else {
      message.error(res.message || '认领申请提交失败')
    }
  } catch {
    message.error('认领申请提交失败')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <a-modal
    :open="open"
    title="认领此内容"
    @cancel="emit('close')"
    :confirm-loading="isSubmitting"
    @ok="submitClaim"
    ok-text="提交申请"
    cancel-text="取消"
    :z-index="10000"
  >
    <div style="display: flex; flex-direction: column; gap: 12px;">
      <p style="font-size: 13px; color: var(--theme-text-secondary); line-height: 1.5; margin: 0;">
        请提供认领理由，管理员将在审核后决定是否将此内容转移给您。
      </p>
      <div>
        <div style="font-size: 13px; color: var(--theme-text-secondary); margin-bottom: 4px;">
          认领理由 <span style="color: var(--theme-danger);">*</span>
        </div>
        <a-textarea
          v-model:value="claimReason"
          placeholder="请详细说明您认为此内容应归属于您的原因..."
          :rows="4"
          :disabled="isSubmitting"
        />
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { contentApi } from '@/api'

const props = defineProps<{ open: boolean; contentId: number }>()
const emit = defineEmits<{ close: [] }>()

const claimReason = ref('')
const visible = ref(props.open)
watch(
  () => props.open,
  v => {
    if (v !== visible.value) visible.value = v
  },
)
watch(visible, v => {
  if (!v) emit('close')
})

async function handleOk(): Promise<boolean> {
  if (!claimReason.value.trim()) {
    Message.warning('请输入认领理由')
    return false
  }
  try {
    const res = await contentApi.submitClaim(props.contentId, claimReason.value.trim())
    if (res.code === 200) {
      Message.success('认领申请已提交，请等待管理员审核')
      claimReason.value = ''
      return true
    }
    Message.error(res.message || '认领申请提交失败')
    return false
  } catch {
    Message.error('认领申请提交失败')
    return false
  }
}
</script>

<template>
  <a-modal
    v-model:visible="visible"
    title="认领此内容"
    :on-before-ok="handleOk"
    ok-text="提交申请"
    cancel-text="取消"
    :z-index="10000"
  >
    <div class="cm-stack">
      <p class="cm-desc">
        请提供认领理由，管理员将在审核后决定是否将此内容转移给您。
      </p>
      <div>
        <div class="cm-label">
          认领理由 <span class="cm-required">*</span>
        </div>
        <a-textarea
          v-model="claimReason"
          placeholder="请详细说明您认为此内容应归属于您的原因..."
          :auto-size="{ minRows: 4, maxRows: 8 }"
        />
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.cm-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cm-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  margin: 0;
}

.cm-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}

.cm-required {
  color: var(--color-danger);
}
</style>

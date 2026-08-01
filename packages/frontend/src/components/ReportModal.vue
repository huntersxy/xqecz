<script setup lang="ts">
import { ref, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { commentApi } from '@/api'
import type { Comment } from '@/types'

const props = defineProps<{ target: Comment | null }>()
const emit = defineEmits<{ close: [] }>()

const reportReason = ref('')
const visible = ref(!!props.target)
watch(
  () => props.target,
  v => {
    const nv = !!v
    if (nv !== visible.value) visible.value = nv
    if (v) reportReason.value = ''
  },
)
watch(visible, v => {
  if (!v) emit('close')
})

async function handleOk(): Promise<boolean> {
  if (!props.target) return false
  try {
    const res = await commentApi.report(props.target.id, reportReason.value || undefined)
    if (res.code === 200) {
      Message.success('举报成功，管理员将尽快处理')
      return true
    }
    Message.error(res.message || '举报失败')
    return false
  } catch {
    Message.error('举报失败')
    return false
  }
}
</script>

<template>
  <a-modal
    v-model:visible="visible"
    title="举报评论"
    :on-before-ok="handleOk"
    ok-text="确认举报"
    cancel-text="取消"
    :z-index="10000"
  >
    <div class="rm-stack">
      <div>
        <div class="rm-label">您正在举报以下评论：</div>
        <div class="rm-quote">
          {{ target?.text }}
        </div>
      </div>
      <div>
        <div class="rm-label">举报原因（可选）</div>
        <a-input v-model="reportReason" placeholder="请输入举报原因" />
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.rm-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.rm-label {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}

.rm-quote {
  padding: 8px 10px;
  background: var(--color-hover);
  border-radius: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
  font-style: italic;
}
</style>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useConfirm } from '@/composables/useToast'

const { pendingConfirm, respond } = useConfirm()
// 注意：pendingConfirm 是 Ref 对象，必须用 .value，否则 !!Ref 永远为 true 导致弹窗默认打开
const visible = ref(!!pendingConfirm.value)
watch(pendingConfirm, v => {
  visible.value = !!v
}, { immediate: true })
</script>

<template>
  <a-modal
    v-model:visible="visible"
    title="确认操作"
    @cancel="respond(false)"
    @ok="respond(true)"
    ok-text="确认"
    cancel-text="取消"
    :z-index="10001"
  >
    <p style="font-size: 14px; color: var(--theme-text); margin: 0;">
      {{ pendingConfirm?.message }}
    </p>
  </a-modal>
</template>

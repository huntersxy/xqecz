<script setup lang="ts">
import { useConfirm } from '@/composables/useToast'

const { pendingConfirm, respond } = useConfirm()
</script>

<template>
  <a-modal
    :open="!!pendingConfirm"
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

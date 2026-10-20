<script setup lang="ts">
import { computed } from 'vue'
import { useConfirm } from '@/composables/useToast'
import SiteOverlay from '@/components/SiteOverlay.vue'
import SiteButton from '@/components/SiteButton.vue'
import SiteIcon from '@/components/SiteIcon.vue'

const { pendingConfirm, respond } = useConfirm()
const open = computed({
  get: () => !!pendingConfirm.value,
  set: (value) => { if (!value) respond(false) },
})
</script>

<template>
  <SiteOverlay v-model="open" class="creative-theme site-confirm" labelledby="site-confirm-title" :z-index="10001">
    <header class="site-confirm-head">
      <h2 id="site-confirm-title"><span aria-hidden="true">✧</span>确认操作</h2>
      <button class="site-confirm-close" type="button" aria-label="关闭" @click="respond(false)"><SiteIcon name="close" /></button>
    </header>
    <p class="site-confirm-message">{{ pendingConfirm?.message }}</p>
    <footer class="site-confirm-foot">
      <SiteButton @click="respond(false)">取消</SiteButton>
      <SiteButton variant="primary" @click="respond(true)">确认</SiteButton>
    </footer>
  </SiteOverlay>
</template>

<style scoped>
.site-confirm-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 18px 24px; border-bottom: 1px solid var(--creative-line); }
.site-confirm-head h2 { display: flex; align-items: center; gap: 9px; margin: 0; font-size: 15px; font-weight: 600; }
.site-confirm-head h2 span { color: var(--creative-accent); }
.site-confirm-close { display: grid; place-items: center; width: 28px; height: 28px; padding: 0; border: 0; border-radius: 50%; background: transparent; color: var(--creative-muted); cursor: pointer; transition: background-color .18s ease, color .18s ease; }
.site-confirm-close:hover { background: var(--creative-soft); color: var(--creative-accent); }
.site-confirm-message { margin: 0; padding: 24px; color: var(--creative-ink); font-size: 14px; line-height: 1.8; overflow-wrap: anywhere; }
.site-confirm-foot { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 24px 18px; border-top: 1px solid var(--creative-line); }
</style>

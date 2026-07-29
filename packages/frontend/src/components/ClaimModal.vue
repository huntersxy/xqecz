<script setup lang="ts">
import { ref } from 'vue'
import { contentApi } from '@/api'

const props = defineProps<{ open: boolean; contentId: number }>()
const emit = defineEmits<{
  close: []
  success: [msg: string]
}>()

const claimReason = ref('')
const isSubmitting = ref(false)

async function submitClaim() {
  if (!claimReason.value.trim()) {
    emit('success', '请输入认领理由')
    return
  }
  try {
    isSubmitting.value = true
    const res = await contentApi.submitClaim(props.contentId, claimReason.value.trim())
    if (res.code === 200) {
      emit('success', '认领申请已提交，请等待管理员审核')
      claimReason.value = ''
      emit('close')
    } else {
      emit('success', res.message || '认领申请提交失败')
    }
  } catch {
    emit('success', '认领申请提交失败')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="cd-modal-overlay" @click.self="emit('close')">
      <div class="cd-modal">
        <div class="cd-modal-head">
          <h3>认领此内容</h3>
          <button type="button" @click="emit('close')">×</button>
        </div>
        <div class="cd-modal-body">
          <p class="cd-modal-desc">请提供认领理由，管理员将在审核后决定是否将此内容转移给您。</p>
          <label class="cd-modal-label" for="claim-reason-textarea">认领理由 <span class="cd-required">*</span></label>
          <textarea id="claim-reason-textarea" v-model="claimReason" class="cd-modal-textarea" placeholder="请详细说明您认为此内容应归属于您的原因..." :disabled="isSubmitting"></textarea>
        </div>
        <div class="cd-modal-foot">
          <button type="button" :disabled="isSubmitting" @click="emit('close')">取消</button>
          <button type="button" class="cd-modal-primary" :disabled="isSubmitting" @click="submitClaim">
            {{ isSubmitting ? '提交中...' : '提交申请' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.cd-modal-overlay {
  position: fixed; inset: 0; z-index: 10000;
  display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, 0.5);
}
.cd-modal {
  width: 90%; max-width: 450px; background: var(--theme-surface);
  border-radius: 12px; box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden; border: 1px solid var(--theme-card-border);
}
.cd-modal-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 0.875rem 1rem; border-bottom: 1px solid var(--theme-card-border);
}
.cd-modal-head h3 { font-size: 0.9375rem; font-weight: 600; color: var(--theme-text); margin: 0; }
.cd-modal-head button {
  background: none; border: none; font-size: 1.25rem; line-height: 1;
  color: var(--theme-text-secondary); cursor: pointer;
}
.cd-modal-head button:hover { color: var(--theme-primary); }
.cd-modal-body { padding: 1rem; display: flex; flex-direction: column; gap: 0.5rem; }
.cd-modal-label { font-size: 0.8125rem; color: var(--theme-text-secondary); }
.cd-modal-desc { font-size: 0.8125rem; color: var(--theme-text-secondary); line-height: 1.5; }
.cd-modal-textarea {
  width: 100%; padding: 0.5rem 0.625rem;
  border: 1px solid var(--theme-card-border); border-radius: 8px;
  background: var(--theme-bg-color); color: var(--theme-text);
  font-size: 0.8125rem; font-family: inherit; min-height: 100px; resize: vertical;
}
.cd-modal-textarea:focus { outline: none; border-color: var(--theme-primary); }
.cd-required { color: var(--theme-danger); }
.cd-modal-foot {
  display: flex; justify-content: flex-end; gap: 0.625rem;
  padding: 0.875rem 1rem; border-top: 1px solid var(--theme-card-border);
  background: var(--theme-hover-bg);
}
.cd-modal-foot button {
  padding: 0.375rem 0.875rem; font-size: 0.8125rem;
  border-radius: 6px; border: none; cursor: pointer;
}
.cd-modal-foot > button:first-child {
  background: transparent; color: var(--theme-text);
}
.cd-modal-foot > button:first-child:hover { color: var(--theme-primary); }
.cd-modal-primary {
  background: var(--theme-primary) !important; color: var(--theme-on-primary) !important;
  font-weight: 600;
}
.cd-modal-primary:hover { filter: brightness(0.92); }
.cd-modal-primary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>

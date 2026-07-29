<script setup lang="ts">
import { ref, watch } from 'vue'
import { commentApi } from '@/api'
import type { Comment } from '@/types'

const props = defineProps<{ target: Comment | null }>()
const emit = defineEmits<{
  close: []
  success: [msg: string]
}>()

const reportReason = ref('')

watch(() => props.target, () => { reportReason.value = '' })

async function submitReport() {
  if (!props.target) return
  try {
    const res = await commentApi.report(props.target.id, reportReason.value || undefined)
    if (res.code === 200) {
      emit('success', '举报成功，管理员将尽快处理')
      emit('close')
    } else {
      emit('success', res.message || '举报失败')
    }
  } catch {
    emit('success', '举报失败')
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="target" class="cd-modal-overlay" @click.self="emit('close')">
      <div class="cd-modal">
        <div class="cd-modal-head">
          <h3>举报评论</h3>
          <button type="button" @click="emit('close')">×</button>
        </div>
        <div class="cd-modal-body">
          <p class="cd-modal-label">您正在举报以下评论：</p>
          <p class="cd-modal-quote">{{ target.text }}</p>
          <label class="cd-modal-label" for="report-reason-input">举报原因（可选）</label>
          <input id="report-reason-input" v-model="reportReason" type="text" class="cd-modal-input" placeholder="请输入举报原因" />
        </div>
        <div class="cd-modal-foot">
          <button type="button" @click="emit('close')">取消</button>
          <button type="button" class="cd-modal-primary" @click="submitReport">确认举报</button>
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
.cd-modal-quote {
  padding: 0.5rem 0.625rem; background: var(--theme-hover-bg); border-radius: 6px;
  font-size: 0.8125rem; color: var(--theme-text-secondary); font-style: italic;
}
.cd-modal-input {
  width: 100%; padding: 0.5rem 0.625rem;
  border: 1px solid var(--theme-card-border); border-radius: 8px;
  background: var(--theme-bg-color); color: var(--theme-text);
  font-size: 0.8125rem; font-family: inherit;
}
.cd-modal-input:focus { outline: none; border-color: var(--theme-primary); }
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
</style>

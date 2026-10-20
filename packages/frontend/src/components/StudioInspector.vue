<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconCopy, IconImage } from '@arco-design/web-vue/es/icon'
import CreativeImage from '@/components/CreativeImage.vue'
import { toast } from '@/composables/useToast'
import type { AIAsset, AITaskView } from '@/types/schemas'
import { formatImageSize } from '@/utils/imageSize'
import { TASK_LABELS, clockOf, dayOf, statusClass } from '@/utils/studioTask'

const props = defineProps<{
  task: AITaskView | null
  asset: AIAsset | null
  liked: boolean
}>()
const emit = defineEmits<{
  preview: [id: number]
  cancel: [row: AITaskView]
}>()

const expanded = ref(false)
const references = computed(() => props.task?.references ?? [])
const promptText = computed(() => props.task?.task.prompt ?? '')
const collapsible = computed(() => promptText.value.length > 140 || promptText.value.includes('\n'))
watch(
  () => props.task?.task.id,
  () => (expanded.value = false),
)

async function copyPrompt() {
  if (!promptText.value) return
  try {
    await navigator.clipboard.writeText(promptText.value)
    toast.success('提示词已复制')
  } catch {
    toast.error('复制失败，请手动选中提示词复制')
  }
}
</script>

<template>
  <aside class="ws-detail" aria-label="任务详情">
    <Transition name="ws-detail-swap" mode="out-in">
      <div v-if="task" :key="task.task.id" class="ws-detail-body">
        <header class="ws-detail-head">
          <span class="ws-status" :class="statusClass(task.task.status)">
            <i aria-hidden="true" />{{ TASK_LABELS[task.task.status] }}
          </span>
          <time class="ws-detail-time" :datetime="task.task.created_at">
            {{ dayOf(task.task.created_at) }} · {{ clockOf(task.task.created_at) }}
          </time>
        </header>

        <dl class="ws-stats">
          <div>
            <dt>模型</dt>
            <dd :title="task.task.model">{{ task.task.model }}</dd>
          </div>
          <div>
            <dt>规格</dt>
            <dd>{{ formatImageSize(task.task.size) }}</dd>
          </div>
          <div>
            <dt>数量</dt>
            <dd>{{ task.task.count }} 张</dd>
          </div>
          <div>
            <dt>结果</dt>
            <dd>{{ task.assets.length }} 张</dd>
          </div>
        </dl>

        <section class="ws-prompt-card" aria-label="提示词">
          <header>
            <h3>提示词</h3>
            <button type="button" class="ws-copy" aria-label="复制提示词" title="复制提示词" @click="copyPrompt">
              <IconCopy />
            </button>
          </header>
          <p :class="{ 'is-clamped': collapsible && !expanded }">{{ task.task.prompt }}</p>
          <button
            v-if="collapsible"
            type="button"
            class="ws-prompt-toggle"
            :aria-expanded="expanded"
            @click="expanded = !expanded"
          >
            {{ expanded ? '收起' : '展开全文' }}
          </button>
        </section>

        <section v-if="references.length" class="ws-refs" aria-label="参考图">
          <h3>参考图 <small>{{ references.length }}</small></h3>
          <div class="ws-refs-row">
            <button
              v-for="(reference, index) in references"
              :key="reference.id"
              type="button"
              :aria-label="`预览参考图 ${index + 1}`"
              @click="emit('preview', reference.id)"
            >
              <CreativeImage :id="reference.id" />
              <span aria-hidden="true">@{{ index + 1 }}</span>
            </button>
          </div>
        </section>

        <p v-if="task.task.error" class="ws-detail-error" role="note">{{ task.task.error }}</p>

        <footer v-if="task.task.status === 'queued' || task.task.idea_id" class="ws-detail-actions">
          <button v-if="task.task.status === 'queued'" type="button" @click="emit('cancel', task)">
            取消排队
          </button>
          <RouterLink v-if="task.task.idea_id" :to="`/ideas/${task.task.idea_id}`">赞助创意</RouterLink>
        </footer>
      </div>

      <div v-else-if="asset" :key="`asset-${asset.id}`" class="ws-detail-body">
        <header class="ws-detail-head">
          <span class="ws-status" :class="liked ? 'is-accent' : 'is-muted'">
            <i aria-hidden="true" />{{ liked ? '已喜欢' : '私人结果' }}
          </span>
          <time class="ws-detail-time" :datetime="asset.created_at">
            {{ dayOf(asset.created_at) }} · {{ clockOf(asset.created_at) }}
          </time>
        </header>
        <dl class="ws-stats">
          <div>
            <dt>尺寸</dt>
            <dd>{{ asset.width }} × {{ asset.height }}</dd>
          </div>
          <div>
            <dt>编号</dt>
            <dd>#{{ asset.id }}</dd>
          </div>
        </dl>
        <section class="ws-prompt-card" aria-label="图片名称">
          <header><h3>名称</h3></header>
          <p>{{ asset.title || `图片 ${asset.id}` }}</p>
        </section>
      </div>

      <div v-else key="empty" class="ws-detail-empty">
        <span><IconImage /></span>
        <p>选择一个任务，查看提示词、参考图与生成参数。</p>
      </div>
    </Transition>
  </aside>
</template>

<style scoped>
.ws-detail {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow-y: auto;
  padding: 18px;
  border: 1px solid color-mix(in srgb, var(--site-line) 80%, transparent);
  border-radius: 22px;
  background: color-mix(in srgb, var(--site-paper) 86%, transparent);
  box-shadow: 0 1px 2px color-mix(in srgb, var(--site-ink) 5%, transparent),
    0 18px 44px color-mix(in srgb, var(--site-ink) 9%, transparent);
  backdrop-filter: blur(16px) saturate(1.2);
  scrollbar-width: thin;
  scrollbar-color: var(--site-line) transparent;
  animation: ws-detail-in 0.46s cubic-bezier(0.22, 1, 0.36, 1) both;
}
@keyframes ws-detail-in {
  from {
    opacity: 0;
    transform: translateX(14px);
  }
}
.ws-detail-swap-enter-active,
.ws-detail-swap-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.ws-detail-swap-enter-from {
  opacity: 0;
  transform: translateY(6px);
}
.ws-detail-swap-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.ws-detail-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ws-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.ws-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 11px 4px 9px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--tone) 12%, transparent);
  color: var(--tone);
  font-size: 11.5px;
  font-weight: 600;
  --tone: var(--site-muted);
}
.ws-status i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}
.ws-status.is-success {
  --tone: rgb(var(--success-6));
}
.ws-status.is-danger {
  --tone: rgb(var(--danger-6));
}
.ws-status.is-accent {
  --tone: var(--site-accent);
}
.ws-detail-time {
  color: var(--site-muted);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
}

.ws-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
}
.ws-stats > div {
  min-width: 0;
  padding: 10px 12px;
  border-radius: 13px;
  background: var(--site-soft);
}
.ws-stats dt {
  color: var(--site-muted);
  font-size: 10.5px;
  letter-spacing: 0.04em;
}
.ws-stats dd {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--site-ink);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.ws-prompt-card {
  padding: 13px 14px 12px;
  border-radius: 15px;
  background: color-mix(in srgb, var(--site-accent-soft) 55%, var(--site-paper));
}
.ws-prompt-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}
.ws-prompt-card h3,
.ws-refs h3 {
  margin: 0;
  color: var(--site-muted);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.06em;
}
.ws-prompt-card p {
  margin: 0;
  color: var(--site-ink);
  font-size: 12.5px;
  line-height: 1.75;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
.ws-prompt-card p.is-clamped {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 5;
}
.ws-copy {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  margin: -4px -6px -4px 0;
  padding: 0;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--site-muted);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}
.ws-copy:hover {
  background: color-mix(in srgb, var(--site-accent) 12%, transparent);
  color: var(--site-accent);
}
.ws-copy svg {
  width: 14px;
  height: 14px;
}
.ws-prompt-toggle {
  margin-top: 6px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--site-accent);
  font-size: 11.5px;
  cursor: pointer;
}

.ws-refs h3 small {
  margin-left: 4px;
  color: var(--site-muted);
  font-weight: 400;
  opacity: 0.7;
}
.ws-refs-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.ws-refs-row button {
  position: relative;
  width: 56px;
  height: 56px;
  padding: 0;
  overflow: hidden;
  border: 1px solid var(--site-line);
  border-radius: 12px;
  background: var(--site-soft);
  cursor: zoom-in;
  transition: transform 0.2s cubic-bezier(0.22, 1, 0.36, 1), box-shadow 0.2s ease;
}
.ws-refs-row button:hover {
  box-shadow: 0 6px 16px color-mix(in srgb, var(--site-ink) 16%, transparent);
  transform: translateY(-2px);
}
.ws-refs-row .creative-image {
  min-height: 0;
}
.ws-refs-row :deep(.creative-image img) {
  object-fit: cover;
}
.ws-refs-row button > span {
  position: absolute;
  bottom: 3px;
  left: 3px;
  padding: 0 5px;
  border-radius: 999px;
  background: rgba(24, 18, 20, 0.55);
  color: #fff;
  font-size: 9px;
  line-height: 16px;
}

.ws-detail-error {
  margin: 0;
  padding: 10px 12px;
  border-radius: 12px;
  background: color-mix(in srgb, rgb(var(--danger-6)) 8%, transparent);
  color: rgb(var(--danger-6));
  font-size: 11.5px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.ws-detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ws-detail-actions button,
.ws-detail-actions a {
  display: inline-flex;
  align-items: center;
  min-height: 32px;
  padding: 0 14px;
  border: 1px solid var(--site-line);
  border-radius: 999px;
  background: var(--site-paper);
  color: var(--site-ink);
  font-size: 11.5px;
  text-decoration: none;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}
.ws-detail-actions button:hover,
.ws-detail-actions a:hover {
  border-color: var(--site-accent);
  background: var(--site-accent-soft);
}

.ws-detail-empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 28px 12px;
  color: var(--site-muted);
  text-align: center;
}
.ws-detail-empty > span {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border: 1px dashed var(--site-line);
  border-radius: 16px;
}
.ws-detail-empty svg {
  width: 22px;
  height: 22px;
}
.ws-detail-empty p {
  max-width: 22ch;
  margin: 0;
  font-size: 12px;
  line-height: 1.7;
}
.ws-detail button:focus-visible,
.ws-detail a:focus-visible {
  outline: 2px solid var(--site-accent);
  outline-offset: 2px;
}
@media (prefers-reduced-motion: reduce) {
  .ws-detail {
    animation: none;
  }
}
</style>

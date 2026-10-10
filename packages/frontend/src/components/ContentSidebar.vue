<script setup lang="ts">
import { computed, ref } from 'vue'
import { IconCopy } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'
import { renderMarkdown } from '@/utils'
import { useUserStore } from '@/stores/user'
import CommentSections from '@/components/CommentSections.vue'
import UserAvatar from '@/components/UserAvatar.vue'
import type { Content, Comment } from '@/types'

interface Props {
  content: Content
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'open-claim': []
  'report-comment': [comment: Comment]
}>()

const router = useRouter()
const userStore = useUserStore()
const commentRef = ref<InstanceType<typeof CommentSections> | null>(null)

const renderedText = computed(() => {
  const t = props.content.text
  return t ? renderMarkdown(t) : ''
})

const refImages = computed(() => {
  const t = props.content.text
  if (!t) return [] as { alt: string; url: string }[]
  const re = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g
  const out: { alt: string; url: string }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(t)) !== null) out.push({ alt: m[1] || '参考图', url: m[2] })
  return out
})

const genParams = computed(() => {
  const list: { label: string; value: string }[] = []
  for (const tag of props.content.tags || []) {
    const m = /^([a-zA-Z_]+):(.+)$/.exec(tag)
    if (m) list.push({ label: m[1].toUpperCase(), value: m[2] })
  }
  return list
})

async function copyPrompt() {
  if (!props.content.text) return
  try { await navigator.clipboard?.writeText(props.content.text) } catch { /* */ }
}

defineExpose({ commentRef })
</script>

<template>
  <aside class="cd-side">
    <div v-if="content.user" class="cd-author">
      <div class="cd-author-left">
        <UserAvatar :src="content.avatar_url" :email="content.user.email" :username="content.user.username" :size="38" class="cd-avatar" />
        <div class="cd-author-info">
          <span class="cd-author-name">{{ content.user.username }}</span>
          <span class="cd-author-id">造物主 · ID #{{ content.user.id }}</span>
        </div>
      </div>
      <div class="cd-author-actions">
        <button class="cd-claim-btn" type="button" @click="userStore.isLoggedIn ? $emit('open-claim') : router.push('/login')">认领</button>
      </div>
    </div>

    <div v-if="(content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t)).length > 0" class="cd-section">
      <div class="cd-section-head"><span class="cd-section-title">标签</span></div>
      <div class="cd-tag-list">
        <span v-for="tag in (content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t))" :key="tag" class="cd-tag">{{ tag }}</span>
      </div>
    </div>

    <div v-if="content.text" class="cd-section">
      <div class="cd-section-head">
        <span class="cd-section-title">关于这颗星球</span>
        <button class="cd-copy-btn" type="button" @click="copyPrompt"><IconCopy /> <span>复制正文</span></button>
      </div>
      <div class="cd-prompt" v-html="renderedText"></div>
    </div>

    <div v-if="refImages.length > 0" class="cd-section">
      <div class="cd-section-head"><span class="cd-section-title">参考图片</span></div>
      <div class="cd-ref-grid">
        <a v-for="(img, i) in refImages" :key="i" :href="img.url" target="_blank" rel="noopener" class="cd-ref-thumb">
          <a-image :src="img.url" :alt="img.alt" :preview="false" />
        </a>
      </div>
    </div>

    <div v-if="genParams.length > 0" class="cd-section">
      <div class="cd-section-head"><span class="cd-section-title">生成参数</span></div>
      <div class="cd-gen-params">
        <div v-for="p in genParams" :key="p.label" class="cd-gen-chip">
          <span class="cd-gen-label">{{ p.label }}</span>
          <span class="cd-gen-value">{{ p.value }}</span>
        </div>
      </div>
    </div>

    <CommentSections
      ref="commentRef"
      :content-id="Number(content?.id) || 0"
      :is-logged-in="userStore.isLoggedIn"
      @report-comment="emit('report-comment', $event)"
    />
  </aside>
</template>

<style scoped>
.cd-side {
  flex: 0 0 clamp(360px, 27vw, 420px); min-width: 0; display: flex; flex-direction: column; gap: 24px;
  padding: 28px; background: var(--creative-paper);
  border-left: 1px solid var(--creative-line); overflow-y: auto; scrollbar-gutter: stable;
  scrollbar-width: thin; scrollbar-color: var(--creative-line) transparent;
}

.cd-author {
  display: flex; flex-shrink: 0; align-items: center; justify-content: space-between; gap: 16px;
  padding: 0 0 24px; border-bottom: 1px solid var(--creative-line);
}
.cd-author-left { display: flex; align-items: center; gap: 0.625rem; min-width: 0; }
.cd-avatar { border: 1px solid var(--creative-line); }
.cd-author-info { display: flex; flex-direction: column; gap: 5px; min-width: 0; }
.cd-author-name { font-size: 0.875rem; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-author-id { font-size: 0.6875rem; color: var(--color-text-secondary); }
.cd-author-actions { display: flex; gap: 0.375rem; flex-shrink: 0; }
.cd-claim-btn {
  position: relative; padding: 7px 12px; font: inherit; font-size: 11px; color: var(--creative-accent);
  background: var(--creative-paper); border: 1px solid color-mix(in srgb, var(--creative-accent) 25%, var(--creative-line)); border-radius: 999px; cursor: pointer;
  transition: background .18s, color .18s, box-shadow .18s;
}
.cd-claim-btn:hover { color: #fff; background: var(--creative-accent); box-shadow: 0 5px 14px color-mix(in srgb, var(--creative-accent) 18%, transparent); }
.cd-copy-btn:hover { color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 35%, var(--creative-line)); background: var(--creative-soft); }

.cd-section {
  display: flex; flex-direction: column; flex-shrink: 0; gap: 12px; padding: 0 0 22px;
  background: transparent; border: 0; border-bottom: 1px solid var(--creative-line); border-radius: 0;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title { display: inline-flex; align-items: center; gap: 7px; font-size: 0.75rem; font-weight: 600; color: var(--creative-ink); letter-spacing: .04em; }
.cd-section-title::before { content: ''; width: 5px; height: 5px; border-radius: 50%; background: var(--creative-accent); box-shadow: 0 0 0 4px var(--creative-soft); }
.cd-copy-btn {
  display: inline-flex; align-items: center; gap: 5px; background: var(--creative-canvas); border: 1px solid var(--creative-line);
  color: var(--creative-muted); font: inherit; font-size: 11px; padding: 5px 9px;
  border-radius: 999px; cursor: pointer; transition: color .18s, border-color .18s, background .18s;
}
.cd-copy-btn svg { width: 13px; height: 13px; }
.cd-tag-list,
.cd-gen-params { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-tag {
  display: inline-block; padding: 0.1875rem 0.625rem; font-size: 0.75rem; color: var(--color-primary);
  background: var(--creative-soft);
  border: 1px solid transparent; border-radius: 999px;
}

.cd-prompt {
  font-size: 12px;
  line-height: 1.9;
  color: var(--color-text);
  max-height: 260px;
  overflow-y: auto;
  word-break: break-word;
}
.cd-prompt :deep(p) { margin: 0 0 0.5em; }
.cd-prompt :deep(p:last-child) { margin-bottom: 0; }
.cd-prompt :deep(h1),
.cd-prompt :deep(h2),
.cd-prompt :deep(h3),
.cd-prompt :deep(h4) {
  margin: 0.9em 0 0.45em;
  font-weight: 700;
  line-height: 1.4;
  color: var(--color-text);
}
.cd-prompt :deep(h1) { font-size: 1.05rem; }
.cd-prompt :deep(h2) { font-size: 0.975rem; }
.cd-prompt :deep(h3) { font-size: 0.9rem; }
.cd-prompt :deep(h4) { font-size: 0.85rem; }
.cd-prompt :deep(h1:first-child),
.cd-prompt :deep(h2:first-child) { margin-top: 0; }
.cd-prompt :deep(pre) { background: var(--color-hover); padding: 0.5rem 0.625rem; border-radius: 6px; overflow-x: auto; font-size: 0.75rem; }
.cd-prompt :deep(code) { background: var(--color-hover); padding: 1px 4px; border-radius: 3px; font-size: 0.75em; }
.cd-prompt :deep(pre code) { background: none; padding: 0; }
.cd-prompt :deep(ul), .cd-prompt :deep(ol) { padding-left: 1.25em; margin: 0.25em 0 0.5em; }
.cd-prompt :deep(li) { margin: 0.2em 0; }
.cd-prompt :deep(blockquote) {
  margin: 0.5em 0;
  padding: 0.4em 0.75em;
  border-left: 3px solid rgb(var(--primary-6));
  background: var(--color-fill-1);
  border-radius: 0 6px 6px 0;
  color: var(--color-text-2);
}
.cd-prompt :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 0.5em auto;
  border-radius: 6px;
}
.cd-prompt :deep(table) { border-collapse: collapse; margin: 0.5em 0; width: 100%; font-size: 0.75rem; }
.cd-prompt :deep(th),
.cd-prompt :deep(td) { border: 1px solid var(--color-border-2); padding: 0.35em 0.5em; text-align: left; }
.cd-prompt :deep(th) { background: var(--color-fill-2); font-weight: 600; }
.cd-prompt :deep(a) { color: var(--color-primary); }
.cd-prompt :deep(hr) { border: none; border-top: 1px solid var(--color-border-2); margin: 1em 0; }

.cd-ref-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 0.375rem; }
.cd-ref-thumb { aspect-ratio: 1; border-radius: 6px; overflow: hidden; background: var(--color-placeholder); display: block; }
.cd-ref-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; }
.cd-ref-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.2s; }
.cd-ref-thumb:hover :deep(.arco-image-img) { transform: scale(1.06); }

.cd-gen-chip {
  display: inline-flex; align-items: center; gap: 4px; padding: 0.1875rem 0.5rem;
  font-size: 0.6875rem; background: var(--color-hover);
  border: 1px solid var(--color-border); border-radius: 4px;
}
.cd-gen-label { color: var(--color-text-secondary); font-weight: 600; letter-spacing: 0.04em; }
.cd-gen-value { color: var(--color-text); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
@media (min-width: 769px) and (max-width: 1024px) { .cd-side { flex-basis: 340px; padding: 22px; } }
@media (max-width: 768px) { .cd-side { padding: 24px 20px; gap: 22px; } }
</style>

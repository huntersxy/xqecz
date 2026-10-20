<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { IconCalendar, IconCopy, IconEye } from '@arco-design/web-vue/es/icon'
import { useRouter } from 'vue-router'
import { formatTime, renderMarkdown } from '@/utils'
import { useUserStore } from '@/stores/user'
import CommentSections from '@/components/CommentSections.vue'
import ContentActions from '@/components/ContentActions.vue'
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

const hasMedia = computed(() => !!(props.content.img || props.content.video))

const renderedText = computed(() => {
  const t = props.content.text
  return t ? renderMarkdown(t) : ''
})

const LONG_TEXT = 280
const isLongText = computed(() => (props.content.text || '').length > LONG_TEXT)
const textExpanded = ref(false)
watch(() => props.content.id, () => { textExpanded.value = false })

const refImages = computed(() => {
  const t = props.content.text
  if (!t) return [] as { alt: string; url: string }[]
  const re = /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g
  const out: { alt: string; url: string }[] = []
  let m: RegExpExecArray | null
  while ((m = re.exec(t)) !== null) out.push({ alt: m[1] || '参考图', url: m[2] })
  return out
})

const plainTags = computed(() => (props.content.tags || []).filter(t => !/^[a-zA-Z_]+:.+$/.test(t)))

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
  try { await navigator.clipboard?.writeText(props.content.text) } catch { }
}

defineExpose({ commentRef })
</script>

<template>
  <aside class="cd-side">
    <header class="cd-head">
      <span class="cd-eyebrow">作品详情 <span aria-hidden="true">/</span> WORK</span>
      <h2 class="cd-title">{{ content.title }}</h2>
      <div class="cd-meta">
        <span class="cd-meta-item"><IconCalendar />{{ formatTime(content.created_at) }}</span>
        <span v-if="content.view_count > 0" class="cd-meta-item"><IconEye />{{ content.view_count }} 浏览</span>
        <span v-if="content.ai_enhanced" class="cd-meta-ai" title="发布时启用了提示词增强">
          <span class="cd-meta-ai-dot" aria-hidden="true"></span>
          提示词增强
        </span>
      </div>
    </header>

    <div v-if="content.user" class="cd-author">
      <div class="cd-author-left">
        <UserAvatar :src="content.avatar_url" :email="content.user.email" :username="content.user.username" :size="40" class="cd-avatar" />
        <div class="cd-author-info">
          <span class="cd-author-name">{{ content.user.username }}</span>
          <span class="cd-author-id">造物主 · ID #{{ content.user.id }}</span>
        </div>
      </div>
      <button class="cd-claim-btn" type="button" @click="userStore.isLoggedIn ? $emit('open-claim') : router.push('/login')">认领</button>
    </div>

    <ContentActions :content="content" />

    <div v-if="plainTags.length > 0" class="cd-section">
      <div class="cd-section-head"><span class="cd-section-title">标签</span></div>
      <div class="cd-tag-list">
        <span v-for="tag in plainTags" :key="tag" class="cd-tag">{{ tag }}</span>
      </div>
    </div>

    <div v-if="content.text && hasMedia" class="cd-section">
      <div class="cd-section-head">
        <span class="cd-section-title">关于这颗星球</span>
        <button class="cd-copy-btn" type="button" @click="copyPrompt"><IconCopy /> <span>复制正文</span></button>
      </div>
      <div class="cd-prompt" :class="{ 'is-collapsed': isLongText && !textExpanded }" v-html="renderedText"></div>
      <button
        v-if="isLongText"
        class="cd-more-btn"
        type="button"
        :aria-expanded="textExpanded"
        @click="textExpanded = !textExpanded"
      >{{ textExpanded ? '收起' : '展开全文' }}</button>
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
  flex: 0 0 clamp(380px, 30vw, 460px); min-width: 0; display: flex; flex-direction: column; gap: 22px;
  padding: 64px 28px 32px; background: var(--creative-paper);
  border-left: 1px solid var(--creative-line); overflow-y: auto; scrollbar-gutter: stable;
  scrollbar-width: thin; scrollbar-color: var(--creative-line) transparent;
}

.cd-head { display: flex; flex-direction: column; flex-shrink: 0; gap: 10px; }
.cd-eyebrow { color: var(--creative-muted); font-size: 9px; letter-spacing: .14em; text-transform: uppercase; }
.cd-eyebrow > span { margin-left: 5px; color: var(--creative-accent); }
.cd-title { margin: 0; color: var(--creative-ink); font: 600 clamp(22px, 2vw, 28px)/1.25 var(--creative-title-font); letter-spacing: -.025em; overflow-wrap: anywhere; }
.cd-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 14px; color: var(--creative-muted); font-size: 11px; }
.cd-meta-item { display: inline-flex; align-items: center; gap: 5px; }
.cd-meta-item svg { width: 13px; height: 13px; flex-shrink: 0; }
.cd-meta-ai { display: inline-flex; align-items: center; gap: 5px; padding: 1px 8px; border: 1px solid var(--creative-line); border-radius: 999px; color: var(--creative-accent); }
.cd-meta-ai-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }

.cd-author {
  display: flex; flex-shrink: 0; align-items: center; justify-content: space-between; gap: 16px;
  padding: 14px 0; border-block: 1px solid var(--creative-line);
}
.cd-author-left { display: flex; align-items: center; gap: 0.75rem; min-width: 0; }
.cd-avatar { border: 1px solid var(--creative-line); }
.cd-author-info { display: flex; flex-direction: column; gap: 5px; min-width: 0; }
.cd-author-name { font-size: 0.875rem; font-weight: 600; color: var(--creative-ink); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-author-id { font-size: 0.6875rem; color: var(--creative-muted); }
.cd-claim-btn {
  flex-shrink: 0; padding: 7px 14px; font: inherit; font-size: 11px; color: var(--creative-accent);
  background: var(--creative-paper); border: 1px solid color-mix(in srgb, var(--creative-accent) 25%, var(--creative-line)); border-radius: 999px; cursor: pointer;
  transition: background .18s, color .18s, box-shadow .18s;
}
.cd-claim-btn:hover { color: #fff; background: var(--creative-accent); box-shadow: 0 5px 14px color-mix(in srgb, var(--creative-accent) 18%, transparent); }

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
.cd-copy-btn:hover { color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 35%, var(--creative-line)); background: var(--creative-soft); }
.cd-tag-list,
.cd-gen-params { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-tag {
  display: inline-block; padding: 0.1875rem 0.625rem; font-size: 0.75rem; color: var(--creative-accent);
  background: var(--creative-soft);
  border: 1px solid transparent; border-radius: 999px;
}

.cd-prompt {
  position: relative;
  font-size: 13px;
  line-height: 1.9;
  color: var(--creative-ink);
  word-break: break-word;
}
.cd-prompt.is-collapsed { max-height: 11.5em; overflow: hidden; }
.cd-prompt.is-collapsed::after {
  content: ''; position: absolute; inset: auto 0 0; height: 3.5em; pointer-events: none;
  background: linear-gradient(to bottom, transparent, var(--creative-paper));
}
.cd-more-btn {
  align-self: flex-start; padding: 0; border: 0; background: transparent;
  color: var(--creative-accent); font: inherit; font-size: 12px; cursor: pointer;
}
.cd-more-btn:hover { text-decoration: underline; text-underline-offset: 3px; }
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
.cd-prompt :deep(pre) { background: var(--creative-canvas); padding: 0.5rem 0.625rem; border-radius: 6px; overflow-x: auto; font-size: 0.75rem; }
.cd-prompt :deep(code) { background: var(--creative-canvas); padding: 1px 4px; border-radius: 3px; font-size: 0.75em; }
.cd-prompt :deep(pre code) { background: none; padding: 0; }
.cd-prompt :deep(ul), .cd-prompt :deep(ol) { padding-left: 1.25em; margin: 0.25em 0 0.5em; }
.cd-prompt :deep(li) { margin: 0.2em 0; }
.cd-prompt :deep(blockquote) {
  margin: 0.5em 0;
  padding: 0.4em 0.75em;
  border-left: 3px solid var(--creative-accent);
  background: var(--creative-soft);
  border-radius: 0 6px 6px 0;
  color: var(--creative-muted);
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
.cd-prompt :deep(td) { border: 1px solid var(--creative-line); padding: 0.35em 0.5em; text-align: left; }
.cd-prompt :deep(th) { background: var(--creative-soft); font-weight: 600; }
.cd-prompt :deep(a) { color: var(--creative-accent); }
.cd-prompt :deep(hr) { border: none; border-top: 1px solid var(--creative-line); margin: 1em 0; }

.cd-ref-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 0.375rem; }
.cd-ref-thumb { aspect-ratio: 1; border-radius: 6px; overflow: hidden; background: var(--creative-canvas); display: block; }
.cd-ref-thumb :deep(.arco-image) { display: block; width: 100%; height: 100%; }
.cd-ref-thumb :deep(.arco-image-img) { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.2s; }
.cd-ref-thumb:hover :deep(.arco-image-img) { transform: scale(1.06); }

.cd-gen-chip {
  display: inline-flex; align-items: center; gap: 4px; padding: 0.1875rem 0.5rem;
  font-size: 0.6875rem; background: var(--creative-canvas);
  border: 1px solid var(--creative-line); border-radius: var(--radius-sm);
}
.cd-gen-label { color: var(--creative-muted); font-weight: 600; letter-spacing: 0.04em; }
.cd-gen-value { color: var(--creative-ink); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
@media (min-width: 769px) and (max-width: 1024px) { .cd-side { flex-basis: 340px; padding: 64px 22px 28px; } }
@media (max-width: 768px) {
  .cd-side { flex: 0 0 auto; width: 100%; max-width: 100%; padding: 22px 20px 32px; gap: 20px; border-left: 0; border-top: 1px solid var(--creative-line); overflow-y: visible; scrollbar-gutter: auto; }
  .cd-title { font-size: 22px; }
}

</style>

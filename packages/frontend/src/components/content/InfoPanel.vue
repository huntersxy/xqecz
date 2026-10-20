<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import './prose.css'
import { formatTime, getImageUrl, renderMarkdown } from '@/utils'
import { retryOnRemote, rewriteMediaInHtml } from '@/utils/contentMedia'
import { useUserStore } from '@/stores/user'
import { toast } from '@/composables/useToast'
import UserAvatar from '@/components/UserAvatar.vue'
import ActionBar from './ActionBar.vue'
import CommentThread from './CommentThread.vue'
import SiteIcon from '@/components/SiteIcon.vue'
import type { Content, Comment } from '@/types'

const props = defineProps<{ content: Content }>()
const emit = defineEmits<{
  'open-claim': []
  'report-comment': [comment: Comment]
}>()

const router = useRouter()
const userStore = useUserStore()
const commentRef = ref<InstanceType<typeof CommentThread> | null>(null)

const hasMedia = computed(() => !!(props.content.img || props.content.video))

const renderedText = computed(() => {
  const t = props.content.text
  return t ? rewriteMediaInHtml(renderMarkdown(t)) : ''
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
  while ((m = re.exec(t)) !== null) out.push({ alt: m[1] || '参考图', url: getImageUrl(m[2]) })
  return out
})

const KEYED = /^[a-zA-Z_]+:.+$/
const plainTags = computed(() => (props.content.tags || []).filter(t => !KEYED.test(t)))
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
  try {
    await navigator.clipboard?.writeText(props.content.text)
    toast.success('已复制正文')
  } catch { toast.error('复制失败，请手动选择文本') }
}

function onRefError(e: Event) {
  const img = e.currentTarget as HTMLImageElement
  if (!retryOnRemote(img)) img.style.visibility = 'hidden'
}

defineExpose({ commentRef })
</script>

<template>
  <aside class="cd-side">
    <header class="cd-head">
      <span class="cd-eyebrow">作品详情 <span aria-hidden="true">/</span> WORK</span>
      <h2 class="cd-title">{{ content.title }}</h2>
      <div class="cd-meta">
        <span class="cd-meta-item"><SiteIcon name="calendar" />{{ formatTime(content.created_at) }}</span>
        <span v-if="content.view_count > 0" class="cd-meta-item"><SiteIcon name="eye" />{{ content.view_count }} 浏览</span>
        <span v-if="content.ai_enhanced" class="cd-meta-ai" title="发布时启用了提示词增强">
          <span class="cd-meta-ai-dot" aria-hidden="true"></span>提示词增强
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
      <button class="cd-claim-btn" type="button" @click="userStore.isLoggedIn ? emit('open-claim') : router.push('/login')">认领</button>
    </div>

    <ActionBar :content="content" />

    <section v-if="plainTags.length > 0" class="cd-section">
      <h3 class="cd-section-title">标签</h3>
      <div class="cd-chips">
        <span v-for="tag in plainTags" :key="tag" class="cd-tag">{{ tag }}</span>
      </div>
    </section>

    <section v-if="content.text && hasMedia" class="cd-section">
      <div class="cd-section-head">
        <h3 class="cd-section-title">关于这颗星球</h3>
        <button class="cd-copy-btn" type="button" @click="copyPrompt"><SiteIcon name="copy" /><span>复制正文</span></button>
      </div>
      <div class="cd-prose cd-prompt" :class="{ 'is-collapsed': isLongText && !textExpanded }" v-html="renderedText"></div>
      <button v-if="isLongText" class="cd-more-btn" type="button" :aria-expanded="textExpanded" @click="textExpanded = !textExpanded">
        {{ textExpanded ? '收起' : '展开全文' }}
      </button>
    </section>

    <section v-if="refImages.length > 0" class="cd-section">
      <h3 class="cd-section-title">参考图片</h3>
      <div class="cd-ref-grid">
        <a v-for="(img, i) in refImages" :key="i" :href="img.url" target="_blank" rel="noopener" class="cd-ref-thumb">
          <img :src="img.url" :alt="img.alt" loading="lazy" decoding="async" @error="onRefError" />
        </a>
      </div>
    </section>

    <section v-if="genParams.length > 0" class="cd-section">
      <h3 class="cd-section-title">生成参数</h3>
      <div class="cd-chips">
        <div v-for="p in genParams" :key="p.label" class="cd-gen-chip">
          <span class="cd-gen-label">{{ p.label }}</span>
          <span class="cd-gen-value">{{ p.value }}</span>
        </div>
      </div>
    </section>

    <CommentThread
      ref="commentRef"
      :content-id="Number(content?.id) || 0"
      :is-logged-in="userStore.isLoggedIn"
      @report-comment="emit('report-comment', $event)"
    />
  </aside>
</template>

<style scoped>
.cd-side {
  flex: 0 0 clamp(380px, 30vw, 460px); min-width: 0; display: flex; flex-direction: column; gap: 24px;
  padding: 68px 30px 36px; background: var(--creative-paper); border-left: 1px solid var(--creative-line);
  overflow-y: auto; scrollbar-gutter: stable; scrollbar-width: thin; scrollbar-color: var(--creative-line) transparent;
}
.cd-head { display: flex; flex-direction: column; flex-shrink: 0; gap: 12px; }
.cd-eyebrow { display: inline-flex; align-items: center; gap: 8px; color: var(--creative-muted); font-size: 10px; letter-spacing: .16em; text-transform: uppercase; }
.cd-eyebrow::before { content: ''; width: 18px; height: 1px; background: var(--creative-accent); }
.cd-eyebrow > span { color: var(--creative-accent); }
.cd-title { margin: 0; color: var(--creative-ink); font: 600 clamp(24px, 2.1vw, 30px)/1.25 var(--creative-title-font); letter-spacing: -.025em; overflow-wrap: anywhere; }
.cd-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; color: var(--creative-muted); font-size: 11px; }
.cd-meta-item { display: inline-flex; align-items: center; gap: 5px; padding: 3px 10px; border-radius: 999px; background: var(--creative-soft); }
.cd-meta-item .site-icon { font-size: 13px; }
.cd-meta-ai { display: inline-flex; align-items: center; gap: 6px; padding: 3px 10px; border-radius: 999px; background: color-mix(in srgb, var(--creative-accent) 10%, var(--creative-paper)); color: var(--creative-accent); }
.cd-meta-ai-dot { width: 5px; height: 5px; border-radius: 50%; background: currentColor; }

.cd-author { display: flex; flex-shrink: 0; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 16px; border: 1px solid var(--creative-line); border-radius: 16px; background: var(--creative-canvas); }
.cd-author-left { display: flex; align-items: center; gap: .75rem; min-width: 0; }
.cd-avatar { border: 2px solid var(--creative-paper); box-shadow: 0 0 0 1px var(--creative-line); }
.cd-author-info { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.cd-author-name { overflow: hidden; font-size: .9375rem; font-weight: 600; color: var(--creative-ink); text-overflow: ellipsis; white-space: nowrap; }
.cd-author-id { font-size: .6875rem; color: var(--creative-muted); }
.cd-claim-btn { flex-shrink: 0; padding: 7px 16px; border: 1px solid color-mix(in srgb, var(--creative-accent) 30%, var(--creative-line)); border-radius: 999px; background: var(--creative-paper); color: var(--creative-accent); font: inherit; font-size: 12px; cursor: pointer; transition: background .18s, color .18s, box-shadow .18s; }
.cd-claim-btn:hover { color: #fff; background: var(--creative-accent); box-shadow: 0 5px 14px color-mix(in srgb, var(--creative-accent) 18%, transparent); }

.cd-section { display: flex; flex-direction: column; flex-shrink: 0; gap: 14px; }
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: .5rem; }
.cd-section-title { display: inline-flex; align-items: center; gap: 8px; margin: 0; font-size: .8125rem; font-weight: 600; letter-spacing: .04em; color: var(--creative-ink); }
.cd-section-title::before { content: ''; width: 3px; height: 13px; border-radius: 2px; background: var(--creative-accent); }
.cd-copy-btn { display: inline-flex; align-items: center; gap: 5px; padding: 5px 10px; border: 1px solid var(--creative-line); border-radius: 999px; background: var(--creative-paper); color: var(--creative-muted); font: inherit; font-size: 11px; cursor: pointer; transition: color .18s, border-color .18s, background .18s; }
.cd-copy-btn:hover { color: var(--creative-accent); border-color: color-mix(in srgb, var(--creative-accent) 35%, var(--creative-line)); background: var(--creative-soft); }
.cd-chips { display: flex; flex-wrap: wrap; gap: 8px; }
.cd-tag { display: inline-block; padding: 4px 12px; border-radius: 999px; background: var(--creative-soft); color: var(--creative-accent); font-size: .75rem; transition: background .18s; }
.cd-tag:hover { background: color-mix(in srgb, var(--creative-accent) 14%, var(--creative-paper)); }

.cd-prompt { position: relative; font-size: 13px; line-height: 1.9; }
.cd-prompt.is-collapsed { max-height: 11.5em; overflow: hidden; }
.cd-prompt.is-collapsed::after { content: ''; position: absolute; inset: auto 0 0; height: 3.5em; pointer-events: none; background: linear-gradient(to bottom, transparent, var(--creative-canvas)); }
.cd-section:has(.cd-prompt) { padding: 16px 18px; border: 1px solid var(--creative-line); border-radius: 16px; background: var(--creative-canvas); }
.cd-section:has(.cd-prompt) .cd-copy-btn { background: var(--creative-paper); }
.cd-more-btn { align-self: flex-start; padding: 0; border: 0; background: transparent; color: var(--creative-accent); font: inherit; font-size: 12px; cursor: pointer; }
.cd-more-btn:hover { text-decoration: underline; text-underline-offset: 3px; }

.cd-ref-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(84px, 1fr)); gap: 8px; }
.cd-ref-thumb { display: block; aspect-ratio: 1; overflow: hidden; border: 1px solid var(--creative-line); border-radius: 12px; background: var(--creative-canvas); }
.cd-ref-thumb img { display: block; width: 100%; height: 100%; object-fit: cover; transition: transform .25s; }
.cd-ref-thumb:hover img { transform: scale(1.06); }

.cd-gen-chip { display: inline-flex; align-items: stretch; overflow: hidden; border: 1px solid var(--creative-line); border-radius: 8px; background: var(--creative-paper); font-size: .6875rem; }
.cd-gen-label { padding: 4px 8px; background: var(--creative-soft); color: var(--creative-muted); font-weight: 600; letter-spacing: .06em; }
.cd-gen-value { padding: 4px 9px; color: var(--creative-ink); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }

@media (min-width: 769px) and (max-width: 1024px) { .cd-side { flex-basis: 340px; padding: 68px 22px 28px; } }
@media (max-width: 768px) {
  .cd-side { flex: 0 0 auto; width: 100%; padding: 22px 20px 32px; gap: 20px; border-left: 0; border-top: 1px solid var(--creative-line); overflow-y: visible; scrollbar-gutter: auto; }
  .cd-title { font-size: 22px; }
}
</style>

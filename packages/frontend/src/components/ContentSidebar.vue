<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { renderMarkdown } from '@/utils'
import { useUserStore } from '@/stores/user'
import CommentSections from '@/components/CommentSections.vue'
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
        <img v-if="content.avatar_url" :src="content.avatar_url" class="cd-avatar-img" :alt="content.user.username" />
        <div v-else class="cd-avatar">{{ (content.user.username || '?').slice(0, 1).toUpperCase() }}</div>
        <div class="cd-author-info">
          <span class="cd-author-name">{{ content.user.username }}</span>
          <span class="cd-author-id">ID #{{ content.user.id }}</span>
        </div>
      </div>
      <div class="cd-author-actions">
        <button class="cd-claim-btn" type="button" @click="userStore.isLoggedIn ? $emit('open-claim') : router.push('/login')">认领</button>
        <button class="cd-follow-btn" type="button">+ 关注</button>
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
        <span class="cd-section-title">简介</span>
        <button class="cd-copy-btn" type="button" @click="copyPrompt">复制</button>
      </div>
      <div class="cd-prompt" v-html="renderedText"></div>
    </div>

    <div v-if="refImages.length > 0" class="cd-section">
      <div class="cd-section-head"><span class="cd-section-title">参考图片</span></div>
      <div class="cd-ref-grid">
        <a v-for="(img, i) in refImages" :key="i" :href="img.url" target="_blank" rel="noopener" class="cd-ref-thumb">
          <img :src="img.url" :alt="img.alt" loading="lazy" />
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
      :content-id="content?.id || 0"
      :is-logged-in="userStore.isLoggedIn"
      @report-comment="emit('report-comment', $event)"
    />
  </aside>
</template>

<style scoped>
.cd-side {
  flex: 0 0 40%; max-width: 460px; display: flex; flex-direction: column; gap: 0.75rem;
  padding: 1rem; background: var(--theme-header-bg);
  border-left: 1px solid var(--theme-card-border); overflow-y: scroll; scrollbar-gutter: stable;
}

.cd-author {
  display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
  padding: 0.75rem; background: var(--theme-surface);
  border: 1px solid var(--theme-card-border); border-radius: 10px;
}
.cd-author-left { display: flex; align-items: center; gap: 0.625rem; min-width: 0; }
.cd-avatar {
  width: 38px; height: 38px; border-radius: 50%;
  background: linear-gradient(135deg, var(--theme-primary), var(--theme-secondary));
  color: var(--theme-on-primary); display: flex; align-items: center; justify-content: center;
  font-size: 1rem; font-weight: 700; flex-shrink: 0;
}
.cd-avatar-img { width: 38px; height: 38px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.cd-author-info { display: flex; flex-direction: column; min-width: 0; }
.cd-author-name { font-size: 0.875rem; font-weight: 600; color: var(--theme-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.cd-author-id { font-size: 0.6875rem; color: var(--theme-text-secondary); }
.cd-author-actions { display: flex; gap: 0.375rem; flex-shrink: 0; }
.cd-claim-btn {
  padding: 0.375rem 0.625rem; font-size: 0.6875rem; color: var(--theme-text-secondary);
  background: var(--theme-hover-bg); border: 1px solid var(--theme-card-border); border-radius: 999px; cursor: pointer;
}
.cd-claim-btn:hover { color: var(--theme-primary); border-color: var(--theme-primary); }
.cd-follow-btn {
  padding: 0.375rem 0.875rem; font-size: 0.75rem; font-weight: 600;
  color: var(--theme-on-primary); background: var(--theme-primary);
  border: none; border-radius: 999px; cursor: pointer;
}
.cd-follow-btn:hover { filter: brightness(0.92); }

.cd-section {
  display: flex; flex-direction: column; gap: 0.5rem; padding: 0.75rem;
  background: var(--theme-surface); border: 1px solid var(--theme-card-border); border-radius: 10px;
}
.cd-section-head { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.cd-section-title { font-size: 0.75rem; font-weight: 600; color: var(--theme-text-secondary); letter-spacing: 0.05em; text-transform: uppercase; }
.cd-copy-btn {
  background: transparent; border: 1px solid var(--theme-card-border);
  color: var(--theme-text-secondary); font-size: 0.6875rem; padding: 2px 8px;
  border-radius: 4px; cursor: pointer; transition: all 0.15s;
}
.cd-copy-btn:hover { border-color: var(--theme-primary); color: var(--theme-primary); }

.cd-tag-list { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-tag {
  display: inline-block; padding: 0.1875rem 0.625rem; font-size: 0.75rem; color: var(--theme-primary);
  background: color-mix(in srgb, var(--theme-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--theme-primary) 25%, transparent); border-radius: 999px;
}

.cd-prompt { font-size: 0.8125rem; line-height: 1.65; color: var(--theme-text); max-height: 220px; overflow-y: auto; word-break: break-word; }
.cd-prompt :deep(p) { margin: 0 0 0.5em; }
.cd-prompt :deep(p:last-child) { margin-bottom: 0; }
.cd-prompt :deep(pre) { background: var(--theme-hover-bg); padding: 0.5rem 0.625rem; border-radius: 6px; overflow-x: auto; font-size: 0.75rem; }
.cd-prompt :deep(code) { background: var(--theme-hover-bg); padding: 1px 4px; border-radius: 3px; font-size: 0.75em; }
.cd-prompt :deep(ul), .cd-prompt :deep(ol) { padding-left: 1.25em; margin: 0.25em 0; }
.cd-prompt :deep(a) { color: var(--theme-primary); }

.cd-ref-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr)); gap: 0.375rem; }
.cd-ref-thumb { aspect-ratio: 1; border-radius: 6px; overflow: hidden; background: var(--theme-placeholder-bg); display: block; }
.cd-ref-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.2s; }
.cd-ref-thumb:hover img { transform: scale(1.06); }

.cd-gen-params { display: flex; flex-wrap: wrap; gap: 0.375rem; }
.cd-gen-chip {
  display: inline-flex; align-items: center; gap: 4px; padding: 0.1875rem 0.5rem;
  font-size: 0.6875rem; background: var(--theme-hover-bg);
  border: 1px solid var(--theme-card-border); border-radius: 4px;
}
.cd-gen-label { color: var(--theme-text-secondary); font-weight: 600; letter-spacing: 0.04em; }
.cd-gen-value { color: var(--theme-text); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
</style>

<script setup lang="ts">
import { ref, onMounted, onActivated, onDeactivated, onBeforeUnmount } from 'vue'
import { aiApi } from '@/api/creative'
import type { AIIdeaView } from '@/types/schemas'
import CreativeImage from '@/components/CreativeImage.vue'
import { ideaPromptPreview } from '@/utils/creativeIdeas'
import { IconArrowRight } from '@arco-design/web-vue/es/icon'
const rows = ref<AIIdeaView[]>([])
let controller: AbortController | undefined
let loading = false
async function load() {
  // Mounted and activated can both run on the first KeepAlive entry.
  if (loading) return
  loading = true
  controller = new AbortController()
  const signal = controller.signal
  try {
    const next = await aiApi.ideas({}, signal)
    if (!signal.aborted)
      rows.value = next.filter((r) => ['waiting', 'realized'].includes(r.idea.status)).slice(0, 4)
  } catch {
    /* Optional discovery section has no effect on the content feed. */
  } finally {
    loading = false
  }
}
onMounted(load)
onActivated(load)
onDeactivated(() => controller?.abort())
onBeforeUnmount(() => controller?.abort())
</script>
<template>
  <section v-if="rows.length" class="home-ideas" aria-labelledby="home-ideas-title">
    <header>
      <h2 id="home-ideas-title">一起，让创意发生</h2>
      <RouterLink to="/ideas">创意广场 <IconArrowRight aria-hidden="true" /></RouterLink>
    </header>
    <div class="home-idea-list">
      <RouterLink
        v-for="row in rows"
        :key="row.idea.id"
        :to="`/ideas/${row.idea.id}`"
        class="home-idea-preview"
      >
        <div class="home-idea-copy">
          <div class="home-idea-heading">
            <strong>{{ row.idea.title }}</strong
            ><small>{{ row.idea.status === 'realized' ? '已实现' : '等待赞助' }}</small>
          </div>
          <p class="home-idea-prompt">{{ ideaPromptPreview(row.idea, 120) }}</p>
          <span class="home-idea-meta"
            >{{ row.author.username
            }}<span v-if="row.sponsorable"> · {{ row.current_cost }} 额度</span></span
          >
        </div>
        <div v-if="row.references.length" class="home-idea-references" aria-label="参考图预览">
          <div
            v-for="(asset, index) in row.references.slice(0, 3)"
            :key="asset.id"
            class="home-idea-reference"
          >
            <CreativeImage :id="asset.id" :idea-id="row.idea.id" />
            <span v-if="index === 2 && row.references.length > 3" class="home-idea-more"
              >+{{ row.references.length - 3 }}</span
            >
          </div>
        </div>
      </RouterLink>
    </div>
  </section>
</template>
<style scoped>
.home-ideas {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  align-items: start;
  gap: 24px;
  margin-block: 4px 28px;
  padding-block: 20px;
  border-block: 1px solid var(--creative-line);
  color: var(--creative-ink);
}
.home-ideas header {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding-top: 8px;
}
.home-ideas h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  line-height: 1.6;
}
.home-ideas small {
  color: var(--creative-accent);
  font-size: 11px;
  white-space: nowrap;
}
.home-ideas header a {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  color: var(--creative-accent);
  text-decoration: none;
  font-size: 12px;
}
.home-idea-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 260px), 1fr));
  gap: 16px;
  min-width: 0;
}
.home-idea-preview {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  text-decoration: none;
  color: var(--creative-ink);
  padding: 16px;
  min-width: 0;
  border-radius: 12px;
  border: 1px solid var(--creative-line);
  background: var(--creative-paper);
  transition:
    background 0.2s,
    border-color 0.2s;
}
.home-idea-preview:hover {
  background: var(--creative-soft);
  border-color: var(--creative-accent);
}
.home-ideas a:focus-visible {
  outline: 2px solid var(--creative-accent);
  outline-offset: 4px;
}
.home-idea-copy {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.home-idea-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px 12px;
}
.home-idea-heading strong {
  font-size: 14px;
  line-height: 1.6;
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.home-idea-prompt {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  overflow-wrap: anywhere;
  margin: 0;
  color: var(--creative-muted);
  font-size: 13px;
  line-height: 1.7;
}
.home-idea-meta {
  font-size: 12px;
  color: var(--creative-muted);
  overflow-wrap: anywhere;
  margin-top: auto;
}
.home-idea-references {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 44px;
}
.home-idea-reference {
  position: relative;
  width: 44px;
  height: 44px;
  border-radius: 6px;
  overflow: hidden;
  background: var(--creative-canvas);
}
.home-idea-reference :deep(.creative-image) {
  min-height: 0;
  font-size: 10px;
}
.home-idea-more {
  position: absolute;
  right: 0;
  bottom: 0;
  padding: 1px 4px;
  border-radius: 4px 0 0;
  background: var(--creative-ink);
  color: var(--creative-paper);
  font-size: 10px;
}
@media (max-width: 1100px) {
  .home-ideas {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }
  .home-ideas header {
    flex-direction: row;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 0;
  }
}
@media (max-width: 540px) {
  .home-ideas {
    margin-bottom: 20px;
    padding-block: 16px;
  }
  .home-ideas h2 {
    font-size: 16px;
  }
  .home-idea-preview {
    padding: 12px;
    gap: 12px;
  }
}
</style>

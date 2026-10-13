<script setup lang="ts">
import { ref, onMounted, onActivated, onDeactivated, onBeforeUnmount } from 'vue'
import { aiApi } from '@/api/creative'
import type { AIIdeaView } from '@/types/schemas'
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
  <section v-if="rows.length" class="home-ideas">
    <header>
      <div>
        <span>✧ 灵感接力</span>
        <h2>一起，让创意发生</h2>
      </div>
      <RouterLink to="/ideas">创意广场 →</RouterLink>
    </header>
    <div class="home-idea-list">
      <RouterLink v-for="row in rows" :key="row.idea.id" :to="`/ideas/${row.idea.id}`"
        ><small>{{ row.idea.status === 'realized' ? '已实现' : '等待赞助' }}</small
        ><strong>{{ row.idea.title }}</strong
        ><span>{{ row.author.username }} · {{ row.current_cost }} 额度</span></RouterLink
      >
    </div>
  </section>
</template>
<style scoped>
.home-ideas {
  margin: 24px 0 34px;
  padding: 24px 28px;
  border-block: 1px solid var(--creative-line);
  color: var(--creative-ink);
}
.home-ideas header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
  margin-bottom: 20px;
}
.home-ideas h2 {
  margin: 6px 0 0;
  font-size: 20px;
}
.home-ideas header span,
.home-ideas small {
  color: var(--creative-accent);
  font-size: 11px;
}
.home-idea-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 24px;
}
.home-idea-list a {
  display: grid;
  gap: 10px;
  text-decoration: none;
  color: var(--creative-ink);
}
.home-idea-list span {
  font-size: 12px;
  color: var(--creative-muted);
}
@media (max-width: 800px) {
  .home-idea-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .home-ideas {
    padding: 20px 12px;
  }
}
</style>

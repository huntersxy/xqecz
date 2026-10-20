<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { aiApi, creativeApi } from '@/api/creative'
import type { GenerationInput } from '@/api/creative'
import type { AIIdeaView, CreativeResource } from '@/types/schemas'
import { useUserStore } from '@/stores/user'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
import { ideaPromptPreview } from '@/utils/creativeIdeas'
import '@/assets/creative-pages.css'
const route = useRoute(),
  router = useRouter(),
  user = useUserStore(),
  { confirm } = useConfirm()
const rows = ref<AIIdeaView[]>([]),
  detail = ref<AIIdeaView | null>(null),
  characters = ref<CreativeResource[]>([]),
  search = ref(''),
  status = ref(''),
  character = ref(''),
  error = ref(''),
  loading = ref(false),
  busy = ref(false)
let controller: AbortController | undefined, timer: ReturnType<typeof setTimeout> | undefined
const labels: Record<string, string> = {
  waiting: '等待赞助',
  generating: '正在实现',
  realized: '已实现',
  unknown: '结果待核查',
  unavailable: '暂不可实现',
  hidden: '已下架',
  withdrawn: '已撤回',
}
const input = computed<Partial<GenerationInput>>(() => {
  try {
    return JSON.parse(detail.value?.idea.snapshot ?? '{}')
  } catch {
    return {}
  }
})
async function load() {
  if (timer) clearTimeout(timer)
  timer = undefined
  controller?.abort()
  controller = new AbortController()
  const signal = controller.signal
  error.value = ''
  loading.value = true
  try {
    if (route.params.id) {
      const next = await aiApi.idea(Number(route.params.id), signal)
      if (signal.aborted) return
      detail.value = next
    } else {
      detail.value = null
      const next = await aiApi.ideas(
        { search: search.value, status: status.value, character: character.value },
        signal,
      )
      if (signal.aborted) return
      rows.value = next
    }
    if (signal.aborted) return
    if (detail.value?.idea.status === 'generating') timer = setTimeout(load, 4000)
  } catch (e) {
    if (!signal.aborted) error.value = (e as Error).message
  } finally {
    if (!signal.aborted) loading.value = false
  }
}
watch(
  [() => route.params.id, status, character],
  () => {
    if (detail.value && String(detail.value.idea.id) !== String(route.params.id))
      detail.value = null
    if (timer) clearTimeout(timer)
    void load()
  },
  { immediate: true },
)
const characterController = new AbortController()
void creativeApi
  .resources('character', undefined, characterController.signal)
  .then((r) => {
    if (!characterController.signal.aborted) characters.value = r
  })
  .catch((e) => {
    if (!characterController.signal.aborted) error.value = (e as Error).message
  })
async function sponsor() {
  if (!detail.value) return
  if (!user.isLoggedIn) {
    await router.push({ path: '/login', query: { redirect: route.fullPath } })
    return
  }
  const row = detail.value
  if (
    !(await confirm(
      `赞助「${row.idea.title}」${user.user?.is_admin ? '不消耗管理员的站点额度' : `将预留 ${row.current_cost} 额度`}。提示词、参考图和成功结果已由投稿者同意公开；赞助不代表取得著作权。确认？`,
    ))
  )
    return
  busy.value = true
  try {
    const storageKey = `xqecz:ai-sponsor:${user.user?.id}:${row.idea.id}`
    let key = localStorage.getItem(storageKey)
    if (!key) {
      key = crypto.randomUUID()
      localStorage.setItem(storageKey, key)
    }
    await aiApi.sponsor(row.idea.id, row.current_cost, key)
    localStorage.removeItem(storageKey)
    await load()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
async function withdraw() {
  if (
    !detail.value ||
    !(await confirm('撤回后此创意和成果将不再公开。已开始的生成仍会执行与结算。确认撤回？'))
  )
    return
  try {
    await aiApi.withdraw(detail.value.idea.id)
    await router.push('/ideas')
  } catch (e) {
    error.value = (e as Error).message
  }
}
onBeforeUnmount(() => {
  characterController.abort()
  controller?.abort()
  if (timer) clearTimeout(timer)
})
</script>
<template>
  <div class="creative-page ideas-gallery-page creative-theme">
    <header class="creative-page-head" :class="{ 'discovery-heading': !detail }">
      <div>
        <h1>{{ detail?.idea.title || '创意广场' }}</h1>
        <p v-if="detail?.idea.description">{{ detail.idea.description }}</p>
      </div>
      <div class="creative-actions">
        <RouterLink to="/studio"><a-button type="primary">提出创意</a-button></RouterLink
        ><RouterLink to="/studio?history=all">我的生成与赞助</RouterLink
        ><RouterLink v-if="detail" to="/ideas">全部创意</RouterLink>
      </div>
    </header>
    <div v-if="error" class="creative-error" role="alert">{{ error }}</div>
    <section v-else-if="loading && route.params.id && !detail" class="creative-box is-skeleton" role="status" aria-label="正在加载创意">
      <span class="page-skeleton-line is-title"></span>
      <span class="page-skeleton-line"></span>
      <span class="page-skeleton-line"></span>
      <span class="page-skeleton-line is-short"></span>
    </section>
    <template v-else-if="detail"
      ><div class="idea-authors">
        <span>创意：{{ detail.author.username }}</span
        ><span v-if="detail.sponsor">赞助：{{ detail.sponsor.username }}</span
        ><a-tag>{{ labels[detail.idea.status] }}</a-tag>
      </div>
      <div v-if="detail.results.length" class="creative-grid idea-results">
        <div v-for="asset in detail.results" :key="asset.id" class="creative-box">
          <CreativeImage :id="asset.id" :idea-id="detail.idea.id" /><RouterLink
            v-if="asset.content_id"
            :to="`/content/${asset.content_id}`"
            >查看站内作品与评论</RouterLink
          >
        </div>
      </div>
      <section class="creative-box">
        <h2>完整创作方案</h2>
        <pre class="idea-prompt">{{ input.prompt }}</pre>
        <p>
          {{ detail.idea.model }} · {{ input.size }} · {{ input.count }} 张 ·
          {{ input.quality || '默认画质' }}
        </p>
        <div class="creative-grid">
          <div v-for="asset in detail.references" :key="asset.id" class="idea-reference">
            <CreativeImage :id="asset.id" :idea-id="detail.idea.id" />
            <p>{{ input.references?.find((r) => r.asset_id === asset.id)?.note || asset.title }}</p>
          </div>
        </div>
        <div class="creative-actions">
          <p v-if="detail.idea.status === 'waiting' && !detail.sponsorable" class="creative-note">
            当前供应商或参考图暂不可用，等待管理员配置后可赞助。
          </p>
          <a-button v-if="detail.sponsorable" type="primary" :loading="busy" @click="sponsor">{{
            user.user?.is_admin
              ? '管理员赞助生成（不消耗额度）'
              : `赞助生成 · ${detail.current_cost} 额度`
          }}</a-button
          ><a-button v-if="user.user?.id === detail.idea.user_id" status="danger" @click="withdraw"
            >撤回创意</a-button
          >
        </div>
        <p class="creative-note">
          同一创意最多一次成功实现。结果不明确时保留预留额度，由管理员核查。赞助不代表取得作品著作权。
        </p>
      </section></template
    >
    <template v-else
      ><div class="creative-actions idea-filters">
        <a-input v-model="search" placeholder="搜索创意" @press-enter="load" /><a-select
          v-model="status"
          placeholder="全部状态"
          allow-clear
          ><a-option value="waiting">等待赞助</a-option
          ><a-option value="realized">已实现</a-option></a-select
        ><a-select v-model="character" placeholder="全部角色" allow-clear
          ><a-option v-for="row in characters" :key="row.id" :value="String(row.id)">{{
            row.title
          }}</a-option></a-select
        ><a-button @click="load">搜索</a-button>
      </div>
      <div class="creative-grid idea-list">
        <template v-if="loading && !rows.length">
          <div
            v-for="n in 4"
            :key="`idea-skeleton-${n}`"
            class="creative-box creative-card idea-card is-skeleton"
            aria-hidden="true"
          >
            <div class="idea-card-main">
              <span class="page-skeleton-line is-title"></span>
              <span class="page-skeleton-line"></span>
              <span class="page-skeleton-line"></span>
              <span class="page-skeleton-line is-short"></span>
            </div>
            <div class="idea-card-references">
              <span class="page-skeleton-thumb"></span>
            </div>
          </div>
        </template>
        <RouterLink
          v-for="row in rows"
          :key="row.idea.id"
          :to="`/ideas/${row.idea.id}`"
          class="creative-box creative-card idea-card"
          :class="{ 'idea-card-text-only': !row.references.length }"
          :data-status="row.idea.status"
        >
          <div class="idea-card-main">
            <div class="idea-card-heading">
              <h2>{{ row.idea.title }}</h2>
              <a-tag>{{ labels[row.idea.status] }}</a-tag>
            </div>
            <p class="idea-card-prompt">{{ ideaPromptPreview(row.idea) }}</p>
            <div class="creative-note">{{ row.author.username }} · {{ row.idea.model }}</div>
            <p v-if="row.sponsorable" class="idea-cost">{{ row.current_cost }} 额度</p>
          </div>
          <div v-if="row.references.length" class="idea-card-references" aria-label="参考图预览">
            <div
              v-for="(asset, index) in row.references.slice(0, 3)"
              :key="asset.id"
              class="idea-card-reference"
            >
              <CreativeImage :id="asset.id" :idea-id="row.idea.id" />
              <span v-if="index === 2 && row.references.length > 3" class="idea-reference-more"
                >+{{ row.references.length - 3 }}</span
              >
            </div>
          </div>
        </RouterLink>
      </div>
      <p v-if="!loading && !rows.length && !error" class="creative-empty">
        还没有创意投稿，从你的工作台开始吧。
      </p></template
    >
  </div>
</template>
<style scoped>
.idea-authors {
  display: flex;
  gap: 22px;
  flex-wrap: wrap;
  margin-bottom: 24px;
}
.idea-prompt {
  white-space: pre-wrap;
  line-height: 1.8;
  color: var(--creative-ink);
  margin: 20px 0;
  padding: 24px;
  border-radius: 14px;
  background: var(--creative-canvas);
  font: inherit;
  overflow-wrap: anywhere;
}
.idea-results {
  margin-bottom: 24px;
}
.idea-results .creative-image {
  height: 420px;
}
.idea-reference .creative-image {
  height: 220px;
}
.idea-filters {
  margin-bottom: 24px;
  padding: 16px;
  border: 1px solid var(--creative-line);
  border-radius: 16px;
  background: var(--creative-paper);
}
.idea-filters .arco-input-wrapper {
  max-width: 320px;
}
.idea-filters :deep(.arco-select-view-single) {
  width: 180px;
}
.idea-cost {
  color: var(--creative-accent) !important;
  padding-top: 14px;
  border-top: 1px solid var(--creative-line);
  font-size: 12px;
  margin-bottom: 0 !important;
}
@media (max-width: 540px) {
  .idea-filters .arco-input-wrapper {
    max-width: none;
    width: 100%;
  }
  .idea-filters :deep(.arco-select-view-single) {
    flex: 1;
    min-width: 120px;
    width: auto;
  }
}

.ideas-gallery-page {
  max-width: 1440px;
  margin-top: clamp(24px, 4vw, 48px);
  padding-inline: clamp(20px, 4vw, 56px);
}
.ideas-gallery-page .creative-page-head {
  padding-top: 26px;
  padding-bottom: 34px;
  margin-bottom: 28px;
}
.ideas-gallery-page .creative-page-head h1 {
  max-width: 760px;
  font-family: var(--creative-title-font);
  font-size: clamp(34px, 5vw, 58px);
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1.22;
}
.ideas-gallery-page .creative-page-head p {
  max-width: 560px;
  font-size: 14px;
}
.ideas-gallery-page .creative-page-head .creative-actions {
  gap: 8px;
}
.ideas-gallery-page .creative-page-head .creative-actions > a {
  border-radius: 8px;
  padding-inline: 14px;
  background: transparent;
}
.ideas-gallery-page .idea-filters {
  display: grid;
  grid-template-columns: minmax(140px, 1fr) minmax(110px, 150px) minmax(110px, 150px) auto;
  align-items: center;
  gap: 8px;
  padding: 12px 0 20px;
  margin-bottom: 24px;
  border-width: 0 0 1px;
  border-radius: 0;
  background: transparent;
}
.ideas-gallery-page .idea-filters .arco-input-wrapper,
.ideas-gallery-page .idea-filters :deep(.arco-select-view-single) {
  border-radius: 8px;
  background: var(--creative-paper);
}
.ideas-gallery-page .idea-filters .arco-input-wrapper,
.ideas-gallery-page .idea-filters :deep(.arco-select-view-single) {
  width: 100%;
  max-width: none;
}
.ideas-gallery-page .idea-filters .arco-input-wrapper,
.ideas-gallery-page .idea-filters :deep(.arco-select-view-single),
.ideas-gallery-page .idea-filters .arco-btn {
  height: 38px;
  min-height: 38px;
}
.ideas-gallery-page .idea-filters .arco-btn {
  padding-inline: 16px;
  border-radius: 8px;
}
.ideas-gallery-page .idea-list.creative-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 20px;
  align-items: start;
}
.ideas-gallery-page .idea-card.creative-card {
  grid-template-columns: minmax(0, 1fr) 96px;
  gap: 20px;
  min-height: 260px;
  padding: 24px;
  border: 1px solid var(--creative-line);
  border-radius: 12px;
  background: var(--creative-paper);
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease;
}
.ideas-gallery-page .idea-card.creative-card:hover {
  border-color: color-mix(in srgb, var(--creative-accent) 55%, var(--creative-line));
  box-shadow: 0 8px 24px color-mix(in srgb, var(--creative-ink) 5%, transparent);
  transform: none;
}
.ideas-gallery-page .idea-card.idea-card-text-only {
  grid-template-columns: minmax(0, 1fr);
}
.ideas-gallery-page .idea-card-heading {
  gap: 8px 12px;
}
.ideas-gallery-page .idea-card-heading h2 {
  flex-basis: 140px;
  font-family: inherit;
  font-size: 19px;
  font-weight: 600;
  line-height: 1.5;
}
.ideas-gallery-page .idea-card-heading .arco-tag {
  flex-shrink: 0;
  margin-top: 2px;
  border-radius: 6px;
  background: var(--creative-soft);
  font-size: 11px;
}
.ideas-gallery-page .idea-card[data-status='realized'] .arco-tag {
  color: var(--creative-ink);
  background: var(--creative-canvas);
}
.ideas-gallery-page .idea-card-prompt {
  margin: 18px 0 20px;
  padding-left: 14px;
  border-left: 2px solid var(--creative-line);
  -webkit-line-clamp: 5;
  font-size: 13px;
  line-height: 1.85;
}
.ideas-gallery-page .idea-card-main > .creative-note {
  margin-top: auto;
  overflow-wrap: anywhere;
  font-size: 11px;
}
.ideas-gallery-page .idea-card-references {
  gap: 8px;
}
.ideas-gallery-page .idea-card-reference {
  border-radius: 8px;
  padding: 4px;
  background: var(--creative-canvas);
}
.ideas-gallery-page .idea-cost {
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px solid var(--creative-line);
}
.ideas-gallery-page .creative-page-head.discovery-heading {
  align-items: center;
  padding: 0;
  margin-bottom: 24px;
  border: 0;
  background: transparent;
}
.ideas-gallery-page .discovery-heading::before {
  display: none;
}
.ideas-gallery-page .creative-empty {
  padding: 48px 20px;
  border: 1px solid var(--creative-line);
  border-radius: 12px;
  background: var(--creative-paper);
}
.ideas-gallery-page .discovery-heading h1 {
  margin: 0;
  color: var(--creative-muted);
  font-family: inherit;
  font-size: 13px;
  font-weight: 500;
}
.ideas-gallery-page .discovery-heading .creative-actions > a {
  padding: 0;
  font-size: 12px;
}
.ideas-gallery-page .discovery-heading .arco-btn {
  border-radius: 8px;
}
.ideas-gallery-page .idea-authors {
  align-items: center;
  padding: 10px 0 20px;
  border-bottom: 1px solid var(--creative-line);
  color: var(--creative-muted);
  font-size: 12px;
}
.ideas-gallery-page .idea-authors .arco-tag {
  border-radius: 2px;
}
.ideas-gallery-page .creative-box {
  border-radius: 4px;
  box-shadow: none;
}
.ideas-gallery-page .idea-prompt {
  padding: 22px;
  border-radius: 2px;
  border-left: 2px solid var(--creative-accent);
  background: color-mix(in srgb, var(--creative-soft) 35%, var(--creative-paper));
}
.ideas-gallery-page .idea-results {
  gap: 12px;
}
.ideas-gallery-page .idea-results .creative-box {
  padding: 10px;
}
@media (max-width: 760px) {
  .ideas-gallery-page .idea-filters {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    align-items: stretch;
  }
  .ideas-gallery-page .idea-filters .arco-input-wrapper {
    grid-column: 1 / -1;
    max-width: none;
  }
  .ideas-gallery-page .idea-list.creative-grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .ideas-gallery-page .idea-card.creative-card {
    padding: 20px;
    grid-template-columns: minmax(0, 1fr) 72px;
    gap: 14px;
  }
  .ideas-gallery-page .idea-card.idea-card-text-only {
    grid-template-columns: minmax(0, 1fr);
  }
  .ideas-gallery-page .creative-page-head.discovery-heading {
    align-items: stretch;
    gap: 16px;
  }
  .ideas-gallery-page .idea-filters :deep(.arco-select-view-single) {
    width: auto;
    flex: 1 1 120px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .ideas-gallery-page .idea-card.creative-card {
    transition: none;
  }
}
</style>

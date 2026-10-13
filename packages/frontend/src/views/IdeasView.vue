<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { aiApi, creativeApi } from '@/api/creative'
import type { GenerationInput } from '@/api/creative'
import type { AIIdeaView, CreativeResource } from '@/types/schemas'
import { useUserStore } from '@/stores/user'
import { useConfirm } from '@/composables/useToast'
import CreativeImage from '@/components/CreativeImage.vue'
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
  <div class="creative-page creative-theme">
    <header class="creative-page-head">
      <div>
        <span class="creative-eyebrow">✧ 灵感接力 / IDEAS</span>
        <h1>{{ detail?.idea.title || '一起，让创意发生' }}</h1>
        <p>
          {{ detail?.idea.description || '提出一个完整的创作方案，或用你的额度支持别人的灵感。' }}
        </p>
      </div>
      <div class="creative-actions">
        <RouterLink to="/studio"><a-button type="primary">提出创意</a-button></RouterLink
        ><RouterLink to="/studio?history=all">我的生成与赞助</RouterLink
        ><RouterLink v-if="detail" to="/ideas">全部创意</RouterLink>
      </div>
    </header>
    <div v-if="error" class="creative-error" role="alert">{{ error }}</div>
    <template v-if="detail"
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
      <div class="creative-grid">
        <RouterLink
          v-for="row in rows"
          :key="row.idea.id"
          :to="`/ideas/${row.idea.id}`"
          class="creative-box creative-card idea-card"
          ><div v-if="row.results[0] || row.references[0]" class="creative-image-tile">
            <CreativeImage :id="(row.results[0] || row.references[0])!.id" :idea-id="row.idea.id" />
          </div>
          <div v-else class="creative-image-tile creative-card-placeholder">
            <span aria-hidden="true">✧</span>一段灵感，等待被点亮
          </div>
          <a-tag>{{ labels[row.idea.status] }}</a-tag>
          <h2>{{ row.idea.title }}</h2>
          <p>{{ row.idea.description }}</p>
          <div class="creative-note">{{ row.author.username }} · {{ row.idea.model }}</div>
          <p v-if="row.sponsorable" class="idea-cost">
            {{ row.current_cost }} 额度 · 等待你的支持
          </p></RouterLink
        >
      </div>
      <p v-if="!rows.length && !error" class="creative-empty">
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
.idea-card {
  text-decoration: none;
  color: var(--creative-ink);
}
.idea-card h2 {
  margin: 18px 0 8px;
}
.idea-card .arco-tag {
  margin-top: 16px;
  align-self: flex-start;
  border-radius: 999px;
  color: var(--creative-accent);
  background: var(--creative-soft);
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
</style>

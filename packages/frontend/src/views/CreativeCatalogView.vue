<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { creativeApi, aiApi } from '@/api/creative'
import type { CreativeResource, AIAsset } from '@/types/schemas'
import CreativeImage from '@/components/CreativeImage.vue'
import '@/assets/creative-pages.css'
import { resolveCreativeResource } from '@/utils/creativeResources'
const route = useRoute(),
  rows = ref<CreativeResource[]>([]),
  assets = ref<AIAsset[]>([]),
  detail = ref<CreativeResource | null>(null),
  search = ref(''),
  purpose = ref(''),
  error = ref(''),
  preview = ref<AIAsset | null>(null)
let controller: AbortController | undefined
const originalTitle = document.title
const descriptionMeta = document.createElement('meta')
descriptionMeta.name = 'description'
document.head.append(descriptionMeta)
const filtered = computed(() =>
  rows.value.filter((r) => `${r.title} ${r.summary}`.includes(search.value)),
)
const resolved = computed(() =>
  detail.value
    ? resolveCreativeResource(detail.value, rows.value)
    : { assets: [], characters: [], text: '' },
)
const allImages = computed(() =>
  resolved.value.assets
    .map((id) => assets.value.find((a) => a.id === id))
    .filter((a): a is AIAsset => !!a),
)
const images = computed(() =>
  allImages.value.filter((a) => !purpose.value || a.purpose === purpose.value),
)
const purposes = computed(() => [...new Set(allImages.value.map((a) => a.purpose).filter(Boolean))])
watch(
  () => route.params.id,
  async (id) => {
    controller?.abort()
    controller = new AbortController()
    const signal = controller.signal
    error.value = ''
    detail.value = null
    purpose.value = ''
    preview.value = null
    try {
      const nextRows = await creativeApi.resources(undefined, undefined, signal)
      if (signal.aborted) return
      rows.value = nextRows.filter((row) => row.kind !== 'template')
      if (id) {
        const nextDetail = /^\d+$/.test(String(id))
          ? await creativeApi.resource(Number(id), signal)
          : rows.value.find((row) => row.slug === String(id))
        if (!nextDetail || nextDetail.kind === 'template') throw new Error('角色资源不存在')
        const nextAssets = await aiApi.publicAssets(
          signal,
          resolveCreativeResource(nextDetail, rows.value).assets,
        )
        if (signal.aborted) return
        detail.value = nextDetail
        assets.value = nextAssets
      }
      if (signal.aborted) return
      document.title = detail.value
        ? `${detail.value.title} · 角色设定 · 小泉二创站`
        : '角色设定库 · 小泉二创站'
      descriptionMeta.content = detail.value?.summary || '浏览官方角色设定、外观特征与授权参考图。'
    } catch (e) {
      if (!signal.aborted) error.value = (e as Error).message
    }
  },
  { immediate: true },
)
async function download(asset: AIAsset) {
  try {
    const blob = await aiApi.publicBlob(asset.id)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${asset.title || 'reference'}.${asset.mime.split('/')[1]}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    error.value = (e as Error).message
  }
}
onBeforeUnmount(() => {
  controller?.abort()
  document.title = originalTitle
  descriptionMeta.remove()
})
</script>
<template>
  <div class="creative-page characters-gallery-page creative-theme">
    <header v-if="detail" class="creative-page-head">
      <div>
        <h1>{{ detail.title }}</h1>
        <p v-if="detail.summary">{{ detail.summary }}</p>
      </div>
      <div class="creative-actions">
        <RouterLink to="/studio">进入 AI 工作台</RouterLink
        ><RouterLink v-if="detail" to="/characters">全部角色</RouterLink>
      </div>
    </header>
    <header v-else class="character-toolbar">
      <h1>角色设定库</h1>
      <a-input
        v-model="search"
        allow-clear
        placeholder="搜索角色名称与简介"
        aria-label="搜索角色名称与简介"
        class="catalog-search"
      >
        <template #prefix>
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            aria-hidden="true"
          >
            <circle cx="10.5" cy="10.5" r="6.5" />
            <path d="m16 16 4.5 4.5" />
          </svg>
        </template>
      </a-input>
    </header>
    <div v-if="error" class="creative-error">{{ error }}</div>
    <template v-if="detail"
      ><section class="creative-box character-story">
        <p>{{ detail.payload.description }}</p>
        <h2 v-if="detail.payload.appearance">外观特征</h2>
        <p class="character-text">{{ detail.payload.appearance }}</p>
        <RouterLink
          :to="{
            path: '/studio',
            query: {
              assets: resolved.assets.join(','),
              characters: resolved.characters.join(','),
              prompt: resolved.text,
            },
          }"
          ><a-button type="primary">带角色设定图进入工作台</a-button></RouterLink
        >
      </section>
      <label class="creative-field"
        >设定图用途<select v-model="purpose">
          <option value="">全部用途</option>
          <option v-for="p in purposes" :key="p">{{ p }}</option>
        </select></label
      >
      <div class="creative-grid">
        <article v-for="asset in images" :key="asset.id" class="creative-box">
          <button class="character-preview" @click="preview = asset">
            <CreativeImage :id="asset.id" public-asset />
          </button>
          <h3>{{ asset.title || '设定图' }}</h3>
          <p>{{ asset.description }}</p>
          <p class="creative-note">{{ asset.purpose }} · {{ asset.width }} × {{ asset.height }}</p>
          <p class="creative-note">授权：{{ asset.license }}</p>
          <a-button v-if="asset.download_allowed" size="small" @click="download(asset)"
            >下载设定图</a-button
          >
        </article>
      </div></template
    >
    <template v-else>
      <div class="character-index">
        <RouterLink
          v-for="row in filtered"
          :key="row.id"
          :to="`/characters/${row.slug}`"
          class="character-card"
          ><div class="creative-image-tile">
            <CreativeImage
              v-if="row.payload.asset_ids[0]"
              :id="row.payload.asset_ids[0]"
              public-asset
            /><span v-else class="character-mark" aria-hidden="true">{{
              row.title.slice(0, 1)
            }}</span>
          </div>
          <h2>{{ row.title }}</h2>
          <p v-if="row.summary">{{ row.summary }}</p></RouterLink
        >
      </div>
      <div v-if="!filtered.length && !error" class="creative-empty">
        角色设定正在整理中
      </div></template
    >
    <a-modal
      :visible="!!preview"
      class="creative-theme creative-dialog"
      :title="preview?.title || '设定图预览'"
      :footer="false"
      @cancel="preview = null"
      ><div class="character-large">
        <CreativeImage v-if="preview" :id="preview.id" public-asset full /></div
    ></a-modal>
  </div>
</template>
<style scoped>
.characters-gallery-page {
  max-width: 1440px;
  margin: clamp(24px, 4vw, 48px) auto 64px;
  padding-inline: clamp(20px, 4vw, 56px);
}
.character-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 32px;
}
.character-toolbar h1 {
  flex-shrink: 0;
  margin: 0;
  color: var(--creative-muted);
  font-size: 13px;
  line-height: 1.6;
  font-weight: 500;
}
.catalog-search {
  width: min(100%, 300px);
  min-height: 38px;
  border-radius: 8px;
  background: var(--creative-paper);
}
.character-index {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 40px 28px;
}
.character-card {
  min-width: 0;
  color: var(--creative-ink);
  text-decoration: none;
  border-radius: 8px;
}
.character-card:focus-visible {
  outline: 2px solid var(--creative-accent);
  outline-offset: 6px;
}
.character-card .creative-image-tile {
  height: auto;
  aspect-ratio: 4 / 5;
  padding: clamp(12px, 2vw, 24px);
  border-radius: 8px;
  background: var(--creative-paper);
  transition: background-color 160ms ease;
}
.character-card:hover .creative-image-tile {
  background: var(--creative-soft);
}
.character-card h2 {
  margin: 16px 0 6px;
  font-size: 18px;
  line-height: 1.5;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.character-card p {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  margin: 0;
  color: var(--creative-muted);
  font-size: 13px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}
.character-mark {
  display: grid;
  place-items: center;
  height: 100%;
  color: var(--creative-muted);
  font-family: var(--creative-title-font);
  font-size: 64px;
}
.character-story {
  max-width: 940px;
  margin: 0 auto 26px;
  padding: clamp(24px, 5vw, 52px);
  box-shadow: none;
}
.character-story p,
.character-text {
  white-space: pre-wrap;
}
.character-story h2 {
  margin-top: 30px;
  font-size: 25px;
  font-weight: 500;
}
.character-story .arco-btn {
  margin-top: 20px;
}
.characters-gallery-page .creative-field {
  display: flex;
  align-items: center;
  gap: 10px;
  width: min(100%, 380px);
  margin: 0 auto 24px;
  color: var(--creative-muted);
  font-size: 12px;
}
.characters-gallery-page .creative-field select {
  flex: 1;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid var(--creative-line);
  border-radius: 6px;
  background: var(--creative-paper);
  color: var(--creative-ink);
}
.character-preview {
  display: block;
  width: 100%;
  height: 280px;
  padding: 12px;
  border: 0;
  border-radius: 8px;
  background: var(--creative-paper);
  cursor: zoom-in;
}
.character-large {
  height: min(68vh, 720px);
}
@media (max-width: 1100px) {
  .character-index {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 540px) {
  .character-toolbar {
    flex-wrap: wrap;
    gap: 16px;
    margin-bottom: 24px;
  }
  .catalog-search {
    width: 100%;
  }
  .character-index {
    gap: 28px 14px;
  }
  .character-card .creative-image-tile {
    padding: 8px;
  }
  .character-card h2 {
    margin-top: 12px;
    font-size: 16px;
  }
  .character-card p {
    font-size: 12px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .character-card .creative-image-tile {
    transition: none;
  }
}
</style>

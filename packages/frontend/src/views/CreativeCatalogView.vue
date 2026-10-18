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
    <header class="creative-page-head">
      <div>
        <span class="creative-eyebrow">✧ 角色手册 / CHARACTERS</span>
        <h1>{{ detail?.title || '角色设定库' }}</h1>
        <p>{{ detail?.summary || '从角色的轮廓、表情与细节出发，延续你的创作。' }}</p>
      </div>
      <div class="creative-actions">
        <RouterLink to="/studio">进入 AI 工作台</RouterLink
        ><RouterLink v-if="detail" to="/characters">全部角色</RouterLink>
      </div>
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
    <template v-else
      ><a-input v-model="search" placeholder="搜索角色名称与简介" class="catalog-search" />
      <div class="creative-grid">
        <RouterLink
          v-for="row in filtered"
          :key="row.id"
          :to="`/characters/${row.slug}`"
          class="creative-box creative-card character-card"
          ><div class="creative-image-tile">
            <CreativeImage
              v-if="row.payload.asset_ids[0]"
              :id="row.payload.asset_ids[0]"
              public-asset
            /><span v-else class="character-mark" aria-hidden="true">✧</span>
          </div>
          <h2>{{ row.title }}</h2>
          <p>{{ row.summary }}</p>
          <span class="character-card-footer"
            >查看角色设定 <span aria-hidden="true">↗</span></span
          ></RouterLink
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
.catalog-search {
  max-width: 420px;
  margin-bottom: 24px;
}
.character-card {
  text-decoration: none;
  color: var(--creative-ink);
}
.character-card h2 {
  margin: 20px 0 8px;
}
.character-card-footer {
  display: flex;
  justify-content: space-between;
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid var(--creative-line);
  color: var(--creative-accent);
  font-size: 12px;
}
.character-story {
  margin-bottom: 24px;
}
.character-story p,
.character-text {
  white-space: pre-wrap;
}
.character-preview {
  height: 240px;
  display: block;
  width: 100%;
  border: 0;
  border-radius: 8px;
  background: var(--creative-canvas);
  cursor: zoom-in;
}
.character-mark {
  display: grid;
  place-items: center;
  height: 100%;
  font-size: 70px;
  color: var(--creative-accent);
}
.character-large {
  height: 60vh;
}

/* The character index keeps the gallery language: images lead, metadata is
   quiet, and the detail view reads as a single reference sheet. */
.characters-gallery-page {
  max-width: 1440px;
  margin-top: clamp(24px, 5vw, 60px);
  padding-inline: clamp(20px, 5vw, 64px);
}
.characters-gallery-page .creative-page-head {
  padding-top: 26px;
  padding-bottom: 34px;
  margin-bottom: 26px;
}
.characters-gallery-page .creative-page-head h1 {
  font-family: var(--creative-title-font);
  font-size: clamp(34px, 5vw, 58px);
  font-weight: 500;
  letter-spacing: -.02em;
  line-height: 1.22;
}
.characters-gallery-page .creative-page-head p {
  max-width: 560px;
}
.characters-gallery-page .catalog-search {
  width: min(100%, 420px);
  margin-bottom: 26px;
}
.characters-gallery-page .catalog-search :deep(.arco-input-wrapper) {
  min-height: 42px;
  border-radius: 3px;
  background: var(--creative-paper);
}
.characters-gallery-page .creative-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 24px 16px;
}
.characters-gallery-page .character-card.creative-card {
  display: block;
  padding: 0 0 16px;
  border-width: 0 0 1px;
  border-radius: 0;
  background: transparent;
  transition: transform .18s ease, border-color .18s ease;
}
.characters-gallery-page .character-card.creative-card:hover {
  border-color: var(--creative-accent);
  transform: translateY(-3px);
}
.characters-gallery-page .creative-image-tile {
  height: 280px;
  overflow: hidden;
  border-radius: 2px;
  background: var(--creative-paper);
}
.characters-gallery-page .creative-image-tile .creative-image {
  height: 100%;
  transition: transform .35s ease;
}
.characters-gallery-page .character-card:hover .creative-image {
  transform: scale(1.025);
}
.characters-gallery-page .character-card h2 {
  margin: 16px 0 6px;
  font-family: var(--creative-title-font);
  font-size: 22px;
  font-weight: 500;
}
.characters-gallery-page .character-card p {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  min-height: 42px;
  margin: 0;
  font-size: 12px;
  line-height: 1.8;
}
.characters-gallery-page .character-card-footer {
  margin-top: 12px;
  padding-top: 0;
  border-top: 0;
  font-size: 11px;
  letter-spacing: .04em;
}
.characters-gallery-page .character-story {
  max-width: 940px;
  margin: 0 auto 26px;
  padding: clamp(24px, 5vw, 52px);
  border-radius: 4px;
  box-shadow: none;
}
.characters-gallery-page .character-story h2 {
  margin-top: 30px;
  font-family: var(--creative-title-font);
  font-size: 25px;
  font-weight: 500;
}
.characters-gallery-page .character-story .arco-btn {
  margin-top: 20px;
  border-radius: 3px;
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
  border-radius: 3px;
  background: var(--creative-paper);
  color: var(--creative-ink);
}
.characters-gallery-page .creative-grid > .creative-box {
  border-radius: 4px;
  box-shadow: none;
}
.characters-gallery-page .character-preview {
  border-radius: 2px;
}
.characters-gallery-page .character-large {
  height: min(68vh, 720px);
}
@media (max-width: 980px) {
  .characters-gallery-page .creative-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (max-width: 640px) {
  .characters-gallery-page .creative-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 20px 12px;
  }
  .characters-gallery-page .creative-image-tile {
    height: 220px;
  }
  .characters-gallery-page .character-card h2 {
    font-size: 19px;
  }
}
</style>

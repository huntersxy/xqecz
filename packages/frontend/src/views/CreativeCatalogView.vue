<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { creativeApi, aiApi } from '@/api/creative'
import type { CreativeResource, AIAsset } from '@/types/schemas'
import CreativeImage from '@/components/CreativeImage.vue'
import '@/assets/creative-pages.css'
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
const allImages = computed(() =>
  (detail.value?.payload.asset_ids ?? [])
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
      const nextRows = await creativeApi.resources('character', undefined, signal)
      if (signal.aborted) return
      rows.value = nextRows
      if (id) {
        const nextDetail = /^\d+$/.test(String(id))
          ? await creativeApi.resource(Number(id), signal)
          : await creativeApi.character(String(id), signal)
        if (nextDetail.kind !== 'character') throw new Error('角色不存在')
        const nextAssets = await aiApi.publicAssets(signal, nextDetail.payload.asset_ids)
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
  <div class="creative-page creative-theme">
    <header class="creative-page-head">
      <div>
        <span class="creative-eyebrow">XQECZ / CHARACTERS</span>
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
        <RouterLink :to="{ path: '/studio', query: { assets: detail.payload.asset_ids.join(',') } }"
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
          class="creative-box character-card"
          ><div class="creative-image-tile">
            <CreativeImage
              v-if="row.payload.asset_ids[0]"
              :id="row.payload.asset_ids[0]"
              public-asset
            /><span v-else class="character-mark">✧</span>
          </div>
          <h2>{{ row.title }}</h2>
          <p>{{ row.summary }}</p></RouterLink
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
</style>

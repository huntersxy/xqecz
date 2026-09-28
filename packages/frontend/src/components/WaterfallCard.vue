<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import MediaImage from '@/components/MediaImage.vue'
import { getPreviewText } from '@/utils'
import type { Content } from '@/types'

interface Props {
  item: Content
}

const props = defineProps<Props>()
const emit = defineEmits<{ click: [content: Content]; imageLoaded: [id: string | number] }>()

// 纯文本卡：从正文提炼一段摘要（去 Markdown），无正文时回退到标题
const previewText = computed(() => {
  const source = props.item.text || props.item.title || ''
  return getPreviewText(source, 96)
})

// 瀑布流依赖卡片高度变化触发重排：用 ResizeObserver 监听整卡高度，
// 只在高度发生有意义变化时 emit（初始观察回调不重复触发），图片加载/失败、
// 字体变化等都会反映到高度上。
const cardRef = ref<HTMLElement | null>(null)
let ro: ResizeObserver | null = null
let lastHeight = 0

function onCardResize(entry: ResizeObserverEntry) {
  const h = entry.contentRect.height
  if (h > 0 && Math.abs(h - lastHeight) > 0.5) {
    lastHeight = h
    emit('imageLoaded', props.item.id)
  }
}

onMounted(() => {
  if (cardRef.value) {
    ro = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) onCardResize(entry)
    })
    ro.observe(cardRef.value)
  }
})
onBeforeUnmount(() => ro?.disconnect())
</script>

<template>
  <div ref="cardRef" class="wf-card" @click="emit('click', props.item)" @keydown.enter="emit('click', props.item)" tabindex="0">
    <template v-if="props.item.thumb">
      <div class="wf-card-media">
        <MediaImage :src="props.item.thumb" :alt="props.item.title" :preview="false" loading="lazy" decoding="async" />
        <div v-if="props.item.tags?.some(t => /ai/i.test(t))" class="wf-badge-ai">AI</div>
      </div>
    </template>
    <template v-else>
      <div class="wf-card-text-body">
        <span class="wf-card-text-mark" aria-hidden="true">&ldquo;</span>
        <p class="wf-card-text-excerpt">{{ previewText }}</p>
        <span class="wf-card-text-more">阅读全文</span>
      </div>
    </template>
    <div class="wf-card-info">
      <span class="wf-card-title">{{ props.item.title }}</span>
      <div class="wf-card-meta">
        <span class="wf-card-user">{{ props.item.user?.username }}</span>
        <span class="wf-card-views">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>
          {{ props.item.like_count || 0 }}
        </span>
      </div>
      <div v-if="props.item.tags?.length" class="wf-card-tags">
        <span v-for="tag in props.item.tags.slice(0, 3)" :key="tag" class="wf-mini-tag">{{ tag }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.wf-card {
  margin-bottom: 10px;
  border-radius: 0.625rem;
  overflow: hidden;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
  /* 卡片宽度由瀑布流按列数算出（JS 写死内联 width），与视口宽度并不等价：
     同样 375px 的窄屏，2 列时每列 ~170px、单列时整屏宽。故这里声明为容器，
     内部排版按「这张卡片实际多宽」自适应，而不是按媒体查询猜视口。
     inline-size 只隔离行内方向，块高仍由内容决定 —— 瀑布流靠 ResizeObserver
     量卡片高度重排，这一点不能被 contain 掉。
      另：inline-size 包含只让「固有宽度」不再由内容撑出，而卡片宽度自始至终是 JS
      写死的内联 px（未布局那一帧为 0px，与加容器前一致），故不改变既有几何；
      卡片本就是 position:absolute，成为包含块/层叠上下文也不是新事。 */
  container-type: inline-size;
  container-name: wfc;
}
.wf-card:hover { transform: translateY(-2px); box-shadow: 0 8px 24px rgba(0,0,0,0.1); }
.wf-card:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }

.wf-card-media {
  position: relative; width: 100%; overflow: hidden; line-height: 0;
  min-height: 80px; background: var(--color-placeholder);
}
/* Arco <Image> 包裹层默认 inline-block，需改为块级填满卡片宽度；破图时 wrapper 不能塌成 0，否则 .arco-image-error 绝对定位 overlay 无高度可显示 */
.wf-card-media :deep(.arco-image) { display: block; width: 100%; min-height: 80px; border-radius: 0; }
.wf-card-media :deep(.arco-image-img) { width: 100%; height: auto; display: block; vertical-align: top; }

/* 暗色下给卡片图片叠一层灰色半透明遮罩，与壁纸压暗保持一致 */
body[arco-theme='dark'] .wf-card-media::after {
  content: '';
  position: absolute;
  inset: 0;
  z-index: 1;
  background: rgba(0, 0, 0, 0.35);
  pointer-events: none;
}

.wf-badge-ai {
  position: absolute; top: 0.375rem; left: 0.375rem;
  padding: 0.0625rem 0.375rem; font-size: 0.5625rem; font-weight: 700;
  letter-spacing: 0.04em; color: #fff;
  background: rgba(var(--purple-6), 0.85); border-radius: 0.25rem;
  z-index: 2; /* 保持在遮罩之上 */
}

.wf-card-text-body {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 150px;
  padding: 1rem 0.875rem 0.75rem;
  background: linear-gradient(165deg, var(--color-fill-2), var(--color-bg-2));
}

.wf-card-text-mark {
  position: absolute;
  top: 0.25rem;
  left: 0.625rem;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 2.25rem;
  line-height: 1;
  color: rgb(var(--primary-3));
  opacity: 0.85;
  pointer-events: none;
}

.wf-card-text-excerpt {
  margin: 1.375rem 0 0.5rem;
  font-size: 0.8125rem;
  line-height: 1.65;
  color: var(--color-text-2);
  display: -webkit-box; -webkit-line-clamp: 5; -webkit-box-orient: vertical; overflow: hidden;
  word-break: break-word;
}

.wf-card-text-more {
  margin-top: auto;
  align-self: flex-end;
  font-size: 0.6875rem;
  font-weight: 600;
  color: rgb(var(--primary-6));
  opacity: 0.75;
  transition: opacity 0.2s, transform 0.2s;
}

.wf-card:hover .wf-card-text-more {
  opacity: 1;
  transform: translateX(2px);
}

.wf-card-info { padding: 0.5rem 0.625rem 0.625rem; }
.wf-card-title {
  display: block; font-size: 0.8125rem; font-weight: 600; color: var(--color-text);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-bottom: 0.25rem;
}
.wf-card-meta { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.wf-card-user { font-size: 0.6875rem; color: var(--color-text-secondary); }
.wf-card-views {
  display: flex; align-items: center; gap: 0.25rem;
  font-size: 0.625rem; color: var(--color-text-secondary);
}
.wf-card-views svg { width: 0.75rem; height: 0.75rem; }
.wf-card-tags { display: flex; gap: 0.25rem; margin-top: 0.375rem; flex-wrap: wrap; }
.wf-mini-tag {
  font-size: 0.5625rem; padding: 0.0625rem 0.375rem; border-radius: 1rem;
  background: var(--color-hover); color: var(--color-text-secondary);
}

/* ── 按卡片自身宽度（而非视口）自适应排版 ──
   窄列（多列大屏、2 列手机）下省字号省留白，避免标签把标题挤成纯省略号；
   宽列（1-2 列）下放开字号与摘要行数，否则整屏宽的单列卡片只有 13px 正文，
   读起来像缩略图。阈值取当前列宽分布的两端，不是随手数字。 */
@container wfc (max-width: 190px) {
  .wf-card-info { padding: 0.375rem 0.5rem 0.5rem; }
  .wf-card-title { font-size: 0.75rem; }
  .wf-card-user { font-size: 0.625rem; }
  .wf-card-views { font-size: 0.5625rem; }
  .wf-mini-tag { font-size: 0.5rem; padding: 0.0625rem 0.25rem; }
  .wf-card-text-body { min-height: 120px; padding: 0.75rem 0.625rem 0.5rem; }
  .wf-card-text-excerpt { -webkit-line-clamp: 3; font-size: 0.75rem; }
}

@container wfc (min-width: 260px) {
  .wf-card-info { padding: 0.625rem 0.875rem 0.75rem; }
  .wf-card-title { font-size: 0.9375rem; }
  .wf-card-text-body { min-height: 170px; padding: 1.25rem 1.125rem 0.875rem; }
  .wf-card-text-excerpt { -webkit-line-clamp: 7; font-size: 0.875rem; }
}
</style>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  AUTO_IMAGE_SIZE,
  calculateImageSize,
  formatImageSize,
  IMAGE_SIZE_RATIOS,
  IMAGE_SIZE_TIERS,
  normalizeImageSize,
  parseImageRatio,
  parseImageSize,
  sameImageSize,
  type ImageSizeTier,
} from '@/utils/imageSize'

const props = defineProps<{
  visible: boolean
  modelValue: string
}>()
const emit = defineEmits<{
  'update:visible': [visible: boolean]
  select: [size: string]
}>()

type PickerMode = 'auto' | 'ratio' | 'custom'
const mode = ref<PickerMode>('auto')
const tier = ref<ImageSizeTier>('1K')
const ratio = ref('1:1')
const customRatio = ref('5:4')
const customWidth = ref('1024')
const customHeight = ref('1024')

function findPreset(value: string) {
  const normalized = normalizeImageSize(value)
  for (const nextTier of IMAGE_SIZE_TIERS) {
    for (const item of IMAGE_SIZE_RATIOS) {
      const nextSize = calculateImageSize(nextTier, item.value)
      if (nextSize && sameImageSize(nextSize, normalized)) return { tier: nextTier, ratio: item.value }
    }
  }
  return null
}

function reset() {
  const current = props.modelValue.trim()
  const parsed = parseImageSize(current)
  const preset = current && current.toLowerCase() !== AUTO_IMAGE_SIZE ? findPreset(current) : null
  if (!current || current.toLowerCase() === AUTO_IMAGE_SIZE) {
    mode.value = 'auto'
  } else if (preset) {
    mode.value = 'ratio'
    tier.value = preset.tier
    ratio.value = preset.ratio
  } else if (parsed) {
    mode.value = 'custom'
    customWidth.value = String(parsed.width)
    customHeight.value = String(parsed.height)
  } else {
    mode.value = 'auto'
  }
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) reset()
  },
)

const activeRatio = computed(() => (ratio.value === 'custom' ? customRatio.value : ratio.value))
const ratioValue = computed(() => parseImageRatio(activeRatio.value))
const ratioSize = computed(() =>
  ratioValue.value ? calculateImageSize(tier.value, activeRatio.value) || '' : '',
)
const customValue = computed(() => {
  const width = Number(customWidth.value)
  const height = Number(customHeight.value)
  return Number.isInteger(width) && Number.isInteger(height) && width > 0 && height > 0
    ? `${width}x${height}`
    : ''
})
const customSize = computed(() => (customValue.value ? normalizeImageSize(customValue.value) : ''))
const customClamped = computed(() => !!customValue.value && customValue.value !== customSize.value)
const previewSize = computed(() => {
  if (mode.value === 'auto') return AUTO_IMAGE_SIZE
  if (mode.value === 'ratio') return ratioSize.value
  return customSize.value
})
const canConfirm = computed(() => !!previewSize.value)

function selectRatio(value: string) {
  ratio.value = value
}
function selectTier(value: ImageSizeTier) {
  tier.value = value
}
function confirm() {
  if (!canConfirm.value) return
  emit('select', previewSize.value)
  emit('update:visible', false)
}
</script>

<template>
  <a-modal
    :visible="visible"
    class="creative-theme creative-dialog ai-dialog studio-size-dialog"
    title="设置图像尺寸"
    :width="640"
    :footer="false"
    @cancel="emit('update:visible', false)"
  >
    <p class="studio-size-current">当前：{{ formatImageSize(modelValue || AUTO_IMAGE_SIZE) }}</p>
    <div class="studio-size-tabs" role="tablist" aria-label="尺寸选择方式">
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'auto'"
        :class="{ 'is-active': mode === 'auto' }"
        @click="mode = 'auto'"
      >
        自动
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'ratio'"
        :class="{ 'is-active': mode === 'ratio' }"
        @click="mode = 'ratio'"
      >
        按比例
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'custom'"
        :class="{ 'is-active': mode === 'custom' }"
        @click="mode = 'custom'"
      >
        自定义宽高
      </button>
    </div>

    <div v-if="mode === 'auto'" class="studio-size-auto">
      <div class="studio-size-auto-icon" aria-hidden="true">↯</div>
      <h3>自动尺寸</h3>
      <p>不向模型传递具体的分辨率参数<br />由提示词和模型自己决定生成尺寸</p>
    </div>

    <div v-else-if="mode === 'ratio'" class="studio-size-ratio-panel">
      <section>
        <div class="studio-size-section-label">基准分辨率</div>
        <div class="studio-size-tiers" role="listbox" aria-label="基准分辨率">
          <button
            v-for="item in IMAGE_SIZE_TIERS"
            :key="item"
            type="button"
            role="option"
            :aria-selected="tier === item"
            :class="{ 'is-selected': tier === item }"
            @click="selectTier(item)"
          >
            <strong>{{ item }}</strong>
            <small>{{ item === '1K' ? '约 100 万像素' : item === '2K' ? '约 400 万像素' : '约 830 万像素' }}</small>
          </button>
        </div>
      </section>

      <section>
        <div class="studio-size-section-label">图像比例</div>
        <div class="studio-size-ratios" role="listbox" aria-label="图像比例">
          <button
            v-for="item in IMAGE_SIZE_RATIOS"
            :key="item.value"
            type="button"
            role="option"
            :aria-selected="ratio === item.value"
            :class="{ 'is-selected': ratio === item.value }"
            @click="selectRatio(item.value)"
          >
            <span
              class="studio-size-ratio-swatch"
              :style="{ aspectRatio: item.value.replace(':', ' / ') }"
              aria-hidden="true"
            />
            <span>{{ item.label }}</span>
          </button>
          <button
            type="button"
            role="option"
            aria-label="自定义比例"
            :aria-selected="ratio === 'custom'"
            :class="{ 'is-selected': ratio === 'custom' }"
            @click="selectRatio('custom')"
          >
            <span class="studio-size-ratio-swatch is-custom" aria-hidden="true">＋</span>
            <span>自定义</span>
          </button>
        </div>
        <label v-if="ratio === 'custom'" class="studio-size-ratio-input">
          <span>输入比例</span>
          <input
            v-model="customRatio"
            type="text"
            inputmode="decimal"
            aria-label="自定义图像比例"
            placeholder="例如 5:4 / 2.39:1"
          />
        </label>
        <p v-if="ratio === 'custom' && !ratioValue" class="studio-size-warning" role="alert">
          请输入有效比例，例如 5:4 或 2.39:1。
        </p>
      </section>
    </div>

    <div v-else class="studio-size-custom">
      <p class="creative-note">
        输入具体像素值。系统会按模型安全范围规整：宽高为 16 的倍数、最大边 3840px、比例不超过 3:1、总像素约 65 万至 829 万。
      </p>
      <div class="studio-size-fields">
        <label>
          <span>宽度</span>
          <input v-model="customWidth" type="number" min="16" step="16" inputmode="numeric" aria-label="图像宽度" />
        </label>
        <span class="studio-size-times" aria-hidden="true">×</span>
        <label>
          <span>高度</span>
          <input v-model="customHeight" type="number" min="16" step="16" inputmode="numeric" aria-label="图像高度" />
        </label>
      </div>
      <p v-if="customClamped" class="studio-size-warning" role="status">
        将自动规整为 {{ formatImageSize(customSize) }}。
      </p>
      <p v-else-if="customSize" class="studio-size-confirmed">将使用 {{ formatImageSize(customSize) }}</p>
    </div>

    <div class="studio-size-footer">
      <span class="creative-note">将使用 {{ formatImageSize(previewSize) }}</span>
      <div>
        <a-button @click="emit('update:visible', false)">取消</a-button>
        <a-button type="primary" :disabled="!canConfirm" @click="confirm">确定</a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { formatImageRatio, formatImageSize, parseImageSize, sameImageSize } from '@/utils/imageSize'

const props = defineProps<{
  visible: boolean
  modelValue: string
  sizes: string[]
}>()
const emit = defineEmits<{
  'update:visible': [visible: boolean]
  select: [size: string]
}>()

type PickerMode = 'ratio' | 'custom'
const mode = ref<PickerMode>('ratio')
const draft = ref(props.modelValue)
const customWidth = ref('')
const customHeight = ref('')

const options = computed(() =>
  props.sizes.map((value) => ({ value, parsed: parseImageSize(value) })),
)
const customValue = computed(() => {
  const width = Number(customWidth.value)
  const height = Number(customHeight.value)
  return width > 0 && height > 0 ? `${width}x${height}` : ''
})
const customMatch = computed(() =>
  options.value.find((option) => option.parsed && sameImageSize(option.value, customValue.value)),
)
const canConfirm = computed(() => (mode.value === 'ratio' ? !!draft.value : !!customMatch.value))

function reset() {
  draft.value = props.modelValue
  mode.value = 'ratio'
  const parsed = parseImageSize(props.modelValue)
  customWidth.value = parsed ? String(parsed.width) : ''
  customHeight.value = parsed ? String(parsed.height) : ''
}
watch(
  () => props.visible,
  (visible) => {
    if (visible) reset()
  },
)
function selectPreset(value: string) {
  draft.value = value
  const parsed = parseImageSize(value)
  if (parsed) {
    customWidth.value = String(parsed.width)
    customHeight.value = String(parsed.height)
  }
}
function isSelected(value: string) {
  return value === draft.value || sameImageSize(value, draft.value)
}
function chooseCustom() {
  if (customMatch.value) draft.value = customMatch.value.value
}
function confirm() {
  const value = mode.value === 'ratio' ? draft.value : customMatch.value?.value
  if (!value) return
  emit('select', value)
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
    <p class="studio-size-current">当前：{{ formatImageSize(modelValue) }}</p>
    <div class="studio-size-tabs" role="tablist" aria-label="尺寸选择方式">
      <button
        type="button"
        role="tab"
        :aria-selected="mode === 'ratio'"
        :class="{ 'is-active': mode === 'ratio' }"
        @click="mode = 'ratio'"
      >
        按比例选择
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

    <div
      v-if="mode === 'ratio'"
      class="studio-size-options"
      role="listbox"
      aria-label="支持的图像尺寸"
    >
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        class="studio-size-option"
        :class="{ 'is-selected': isSelected(option.value) }"
        role="option"
        :aria-selected="isSelected(option.value)"
        @click="selectPreset(option.value)"
      >
        <span
          v-if="option.parsed"
          class="studio-size-swatch"
          :style="{ aspectRatio: `${option.parsed.width} / ${option.parsed.height}` }"
          aria-hidden="true"
        />
        <span v-else class="studio-size-swatch is-raw" aria-hidden="true">预设</span>
        <span class="studio-size-option-copy">
          <strong>{{ option.parsed ? formatImageRatio(option.parsed) : '供应商预设' }}</strong>
          <small>{{ option.parsed ? option.value.replace('x', '×') : option.value }}</small>
        </span>
      </button>
      <p v-if="!options.length" class="creative-note">当前供应商没有尺寸预设，请先在管理端配置。</p>
    </div>

    <div v-else class="studio-size-custom">
      <p class="creative-note">
        只能提交当前供应商允许的尺寸。先在这里输入宽高，再由下方提示确认是否已配置。
      </p>
      <div class="studio-size-fields">
        <label>
          <span>宽度</span>
          <input
            v-model="customWidth"
            type="number"
            min="1"
            inputmode="numeric"
            aria-label="图像宽度"
            @input="chooseCustom"
          />
        </label>
        <span class="studio-size-times" aria-hidden="true">×</span>
        <label>
          <span>高度</span>
          <input
            v-model="customHeight"
            type="number"
            min="1"
            inputmode="numeric"
            aria-label="图像高度"
            @input="chooseCustom"
          />
        </label>
      </div>
      <p v-if="customValue && !customMatch" class="studio-size-warning" role="alert">
        {{ customValue.replace('x', '×') }} 未在当前供应商的允许列表中，请先在管理端添加。
      </p>
      <p v-else-if="customMatch" class="studio-size-confirmed">
        将使用 {{ formatImageSize(customMatch.value) }}
      </p>
    </div>

    <div class="studio-size-footer">
      <span class="creative-note"
        >将使用 {{ formatImageSize((mode === 'custom' ? customMatch?.value : draft) || '') }}</span
      >
      <div>
        <a-button @click="emit('update:visible', false)">取消</a-button>
        <a-button type="primary" :disabled="!canConfirm" @click="confirm">确定</a-button>
      </div>
    </div>
  </a-modal>
</template>

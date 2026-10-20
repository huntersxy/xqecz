<script setup lang="ts">
import type { MetadataOptions } from '@/types'
import SiteCheckbox from '@/components/SiteCheckbox.vue'
defineProps<{ options: MetadataOptions; canGenerate: boolean; disabled?: boolean }>()
const enabled = defineModel<boolean>({ required: true })
</script>

<template>
  <div class="metadata-option">
    <SiteCheckbox v-model="enabled" :disabled="disabled || !canGenerate">自动生成图片标题与标签</SiteCheckbox>
    <p v-if="!options.available">自动生成暂不可用，可手动填写。</p>
    <p v-else-if="!canGenerate">选择图片后可开启，视频和纯文字作品请手动填写。</p>
    <p v-else-if="enabled">先以占位信息上传，服务器会将图片预览发送给视觉服务生成标题与标签；失败可由管理员重试。{{ options.ready ? '' : '服务尚未启用，任务将等待管理员配置。' }}</p>
    <p v-else>开启后标题与标签会临时占位，关闭可恢复手动填写的内容。</p>
  </div>
</template>

<style scoped>
.metadata-option { padding: 12px 14px; border: 1px solid var(--creative-line, var(--color-border-2)); border-radius: 10px; background: var(--creative-soft, var(--color-fill-1)); }
.metadata-option p { margin: 7px 0 0; color: var(--creative-muted, var(--color-text-3)); font-size: 11px; line-height: 1.7; }
</style>

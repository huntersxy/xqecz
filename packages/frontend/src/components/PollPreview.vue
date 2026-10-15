<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { IconArrowRight } from '@arco-design/web-vue/es/icon'
import type { Poll } from '@/types/schemas'
defineProps<{ poll: Poll }>()
</script>

<template>
  <RouterLink :to="`/polls/${poll.id}`" class="poll-preview">
    <h3>{{ poll.title }}</h3>
    <p v-if="poll.description" class="poll-preview-description">{{ poll.description }}</p>
    <div class="poll-preview-options" aria-label="选项预览">
      <span v-for="(option, index) in poll.options.slice(0, 3)" :key="index">{{ option }}</span>
      <small v-if="poll.options.length > 3">另有 {{ poll.options.length - 3 }} 项</small>
    </div>
    <div class="poll-preview-footer">
      <span>{{ poll.vote_count }} 票 · {{ poll.options.length }} 个选项</span>
      <span class="poll-preview-action">参与投票 <IconArrowRight aria-hidden="true" /></span>
    </div>
  </RouterLink>
</template>

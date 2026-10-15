<script setup lang="ts">
import { onActivated, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { IconArrowRight } from '@arco-design/web-vue/es/icon'
import { usePollList } from '@/composables/usePollList'
import PollPreview from '@/components/PollPreview.vue'
import '@/assets/creative-pages.css'

const { rows, loading, error, load } = usePollList(2)
onMounted(() => load())
onActivated(() => load())
</script>

<template>
  <section class="home-polls" aria-labelledby="home-polls-title" :aria-busy="loading">
    <header>
      <h2 id="home-polls-title">大家来投一票</h2>
      <RouterLink to="/polls">全部投票 <IconArrowRight aria-hidden="true" /></RouterLink>
    </header>
    <div v-if="error" class="poll-section-state" role="alert">
      <span>{{ error }}</span
      ><button type="button" @click="load()">重新加载</button>
    </div>
    <div v-else-if="rows.length" class="poll-preview-grid">
      <PollPreview v-for="poll in rows" :key="poll.id" :poll="poll" />
    </div>
    <p v-else class="poll-section-state" role="status">
      {{ loading ? '正在加载投票…' : '暂时没有投票，稍后来看看。' }}
    </p>
  </section>
</template>

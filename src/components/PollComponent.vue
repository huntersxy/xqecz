<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { pollApi } from '@/api'
import type { PollDetail } from '@/types'

const pollDetail = ref<PollDetail | null>(null)
const loading = ref(false)
const voting = ref(false)
const voted = ref(false)
const errorMessage = ref('')

const colors = [
  '#3b82f6',
  '#ef4444',
  '#22c55e',
  '#f59e0b',
  '#8b5cf6',
  '#06b6d4',
]

async function loadLatestPoll() {
  try {
    loading.value = true
    errorMessage.value = ''
    const res = await pollApi.list()
    if (res.code === 200 && res.data.list.length > 0) {
      const latestPoll = res.data.list.reduce((prev, current) => {
        return new Date(prev.created_at) > new Date(current.created_at) ? prev : current
      })
      const detailRes = await pollApi.detail(latestPoll.id)
      if (detailRes.code === 200) {
        pollDetail.value = detailRes.data
        voted.value = detailRes.data.my_vote !== null
      }
    }
  } catch (error) {
    console.error('加载投票失败', error)
    errorMessage.value = '加载投票失败，请刷新重试'
  } finally {
    loading.value = false
  }
}

async function handleVote(optionIndex: number) {
  if (!pollDetail.value || voting.value || voted.value) return

  try {
    voting.value = true
    errorMessage.value = ''
    const res = await pollApi.vote(pollDetail.value.poll.id, optionIndex)
    if (res.code === 200) {
      pollDetail.value.my_vote = optionIndex
      pollDetail.value.vote_counts[optionIndex] =
        (pollDetail.value.vote_counts[optionIndex] || 0) + 1
      pollDetail.value.total_votes += 1
      pollDetail.value.poll.vote_count += 1
      voted.value = true
    } else {
      errorMessage.value = res.message || '投票失败'
    }
  } catch (error: any) {
    console.error('投票失败:', error)
    errorMessage.value =
      error.response?.data?.message || error.message || '投票失败，请重试'
  } finally {
    voting.value = false
  }
}

const getPercentage = (count: number) => {
  if (!pollDetail.value || pollDetail.value.total_votes === 0) return 0
  return Math.round((count / pollDetail.value.total_votes) * 100)
}

const getColor = (index: number) => colors[index % colors.length]

const getGradientColor = (index: number) => {
  const color = getColor(index)
  const lighterColor = getLighterColor(color)
  return `linear-gradient(to right, ${color}, ${lighterColor})`
}

const getLighterColor = (color: string) => {
  const lightnessMap: Record<string, string> = {
    '#3b82f6': '#60a5fa',
    '#ef4444': '#f87171',
    '#22c55e': '#4ade80',
    '#f59e0b': '#fbbf24',
    '#8b5cf6': '#a78bfa',
    '#06b6d4': '#22d3ee',
  }
  return lightnessMap[color] || color
}

onMounted(() => {
  loadLatestPoll()
})
</script>

<template>
  <div class="poll-component">
    <div v-if="loading" class="loading-state">
      <svg
        class="loading-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="M21 12a9 9 0 1 1-6.219-8.56" />
      </svg>
      <p>加载投票中...</p>
    </div>

    <div v-else-if="pollDetail" class="poll-card">
      <div v-if="pollDetail.poll.title" class="poll-header">
        <h2 class="poll-title">{{ pollDetail.poll.title }}</h2>
      </div>

      <div class="pk-container">
        <template v-for="(option, index) in pollDetail.poll.options" :key="index">
          <div v-if="index > 0" class="pk-vs">VS</div>
          <div class="pk-option" :class="{ 'pk-voted': pollDetail.my_vote === index }">
            <button
              @click="handleVote(index)"
              :disabled="voted || voting"
              class="pk-button"
              :style="{
                '--option-color': getColor(index),
                '--border-color': getColor(index),
              }"
            >
              <span class="pk-text">{{ option }}</span>
              <div class="pk-stats">
                <span class="pk-count">{{ pollDetail.vote_counts[index] || 0 }} 票</span>
                <span v-if="pollDetail.total_votes > 0" class="pk-percentage">
                  {{ getPercentage(pollDetail.vote_counts[index] || 0) }}%
                </span>
              </div>
            </button>
          </div>
        </template>
      </div>

      <div class="pk-progress-container">
        <div
          v-for="(option, index) in pollDetail.poll.options"
          :key="index"
          class="pk-progress-bar"
          :style="{
            width: getPercentage(pollDetail.vote_counts[index] || 0) + '%',
            background: getGradientColor(index),
          }"
        ></div>
      </div>

      <div v-if="errorMessage" class="error-message">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ errorMessage }}</span>
      </div>
    </div>

    <div v-else class="empty-state">
      <svg
        class="empty-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
      >
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
      <p>暂无投票</p>
    </div>
  </div>
</template>

<style scoped>
.poll-component {
  width: 100%;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 24px 16px;
  color: #999;
}

.loading-icon,
.empty-icon {
  width: 32px;
  height: 32px;
  margin-bottom: 12px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.poll-card {
  width: 100%;
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 10px;
  padding: 12px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
  box-sizing: border-box;
}

.poll-header {
  margin-bottom: 10px;
}

.poll-title {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.pk-container {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
}

.pk-option {
  flex: 1;
}

.pk-button {
  width: 100%;
  padding: 8px 10px;
  background: white;
  border: 2px solid var(--border-color);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  color: var(--option-color);
}

.pk-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  background: color-mix(in srgb, var(--option-color) 8%, white);
}

.pk-button:disabled {
  cursor: not-allowed;
}

.pk-option.pk-voted .pk-button {
  border-width: 4px;
  font-weight: 600;
  background: color-mix(in srgb, var(--option-color) 12%, white);
}

.pk-text {
  font-size: 13px;
  font-weight: 500;
  word-break: break-word;
}

.pk-stats {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
}

.pk-count {
  font-size: 13px;
  font-weight: 600;
}

.pk-percentage {
  font-size: 11px;
  opacity: 0.7;
}

.pk-vs {
  font-size: 12px;
  font-weight: 700;
  color: #888;
  padding: 0 4px;
}

.pk-progress-container {
  position: relative;
  height: 16px;
  border-radius: 8px;
  overflow: hidden;
  background: #f0f0f0;
  display: flex;
  margin-bottom: 8px;
}

.pk-progress-bar {
  height: 100%;
  transition: width 0.5s ease;
}

.pk-progress-bar:first-child {
  border-radius: 8px 0 0 8px;
}

.pk-progress-bar:last-child {
  border-radius: 0 8px 8px 0;
}

.error-message {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
  padding: 12px;
  background: rgba(239, 68, 68, 0.1);
  border-radius: 8px;
  font-size: 14px;
  color: #dc2626;
}

.error-message svg {
  width: 16px;
  height: 16px;
}

@media screen and (max-width: 768px) {
  .poll-card {
    padding: 20px 16px;
  }

  .poll-title {
    font-size: 18px;
  }

  .pk-button {
    padding: 16px 12px;
    min-height: 80px;
  }

  .pk-text {
    font-size: 14px;
  }

  .pk-count {
    font-size: 18px;
  }

  .pk-vs {
    font-size: 14px;
  }
}
</style>

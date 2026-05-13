<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { pollApi } from '@/api'
import type { PollDetail, VoteData } from '@/types'

const pollDetail = ref<PollDetail | null>(null)
const loading = ref(false)
const voting = ref(false)
const voted = ref(false)
const errorMessage = ref('')

const colors = [
  '#3b82f6', // blue
  '#ef4444', // red
  '#22c55e', // green
  '#f59e0b', // orange
  '#8b5cf6', // purple
  '#06b6d4', // cyan
]

const isDualOption = computed(() => pollDetail.value?.poll.options.length === 2)

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
    const voteData: VoteData = { option_index: optionIndex }
    const res = await pollApi.vote(pollDetail.value.poll.id, voteData)
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
      <div class="poll-header">
        <h2 class="poll-title">📊 {{ pollDetail.poll.title }}</h2>
        <p v-if="pollDetail.poll.description" class="poll-description">
          {{ pollDetail.poll.description }}
        </p>
        <div class="poll-meta">
          <span>投票人数: {{ pollDetail.total_votes }}</span>
          <span v-if="pollDetail.poll.user" class="poll-author">
            发起者: {{ pollDetail.poll.user.username }}
          </span>
        </div>
      </div>

      <!-- 双选项 PK 模式 -->
      <div v-if="isDualOption" class="pk-container">
        <div class="pk-option pk-left" :class="{ 'pk-voted': pollDetail.my_vote === 0 }">
          <button @click="handleVote(0)" :disabled="voted || voting" class="pk-button">
            <span class="pk-text">{{ pollDetail.poll.options[0] }}</span>
            <div class="pk-stats">
              <span class="pk-count">{{ pollDetail.vote_counts[0] || 0 }} 票</span>
              <span v-if="pollDetail.total_votes > 0" class="pk-percentage">
                {{ getPercentage(pollDetail.vote_counts[0] || 0) }}%
              </span>
            </div>
          </button>
        </div>
        <div class="pk-vs">VS</div>
        <div class="pk-option pk-right" :class="{ 'pk-voted': pollDetail.my_vote === 1 }">
          <button @click="handleVote(1)" :disabled="voted || voting" class="pk-button">
            <span class="pk-text">{{ pollDetail.poll.options[1] }}</span>
            <div class="pk-stats">
              <span class="pk-count">{{ pollDetail.vote_counts[1] || 0 }} 票</span>
              <span v-if="pollDetail.total_votes > 0" class="pk-percentage">
                {{ getPercentage(pollDetail.vote_counts[1] || 0) }}%
              </span>
            </div>
          </button>
        </div>
      </div>

      <!-- 单进度条 PK -->
      <div v-if="isDualOption" class="pk-progress-container">
        <div
          class="pk-progress-bar pk-progress-left"
          :style="{ width: getPercentage(pollDetail.vote_counts[0] || 0) + '%' }"
        ></div>
        <div
          class="pk-progress-bar pk-progress-right"
          :style="{ width: getPercentage(pollDetail.vote_counts[1] || 0) + '%' }"
        ></div>
      </div>

      <!-- 多选项模式 -->
      <div v-else class="options-container">
        <div
          v-for="(option, index) in pollDetail.poll.options"
          :key="index"
          class="option-wrapper"
          :class="{ 'option-voted': pollDetail.my_vote === index }"
        >
          <button
            @click="handleVote(index)"
            :disabled="voted || voting"
            class="option-button"
            :style="{
              '--option-color': getColor(index),
            }"
          >
            <div class="option-inner">
              <span class="option-text">{{ option }}</span>
              <div class="vote-stats">
                <span class="vote-count">{{ pollDetail.vote_counts[index] || 0 }} 票</span>
                <span v-if="pollDetail.total_votes > 0" class="vote-percentage">
                  {{ getPercentage(pollDetail.vote_counts[index] || 0) }}%
                </span>
              </div>
            </div>
            <div
              class="progress-bar"
              :style="{ width: getPercentage(pollDetail.vote_counts[index] || 0) + '%' }"
            ></div>
          </button>
        </div>
      </div>

      <div v-if="voted" class="voted-message">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12" />
        </svg>
        <span>您已投票</span>
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
  max-width: 600px;
  margin: 0 auto;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: #999;
}

.loading-icon,
.empty-icon {
  width: 48px;
  height: 48px;
  margin-bottom: 16px;
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
  background: rgba(255, 255, 255, 0.95);
  border: 1px solid rgba(255, 255, 255, 0.36);
  border-radius: 12px;
  padding: 24px;
  box-shadow:
    0 4px 16px rgba(0, 0, 0, 0.08),
    0 1px 4px rgba(0, 0, 0, 0.04);
}

.poll-header {
  margin-bottom: 24px;
}

.poll-title {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 8px 0;
}

.poll-description {
  font-size: 14px;
  color: #666;
  margin: 0 0 12px 0;
  line-height: 1.5;
}

.poll-meta {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #888;
  flex-wrap: wrap;
}

.poll-author {
  color: #007aff;
}

/* ========== 双选项 PK 模式 ========== */
.pk-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.pk-option {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.pk-button {
  width: 100%;
  padding: 20px 16px;
  background: white;
  border: 3px solid;
  border-radius: 12px;
  font-size: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: center;
  min-height: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.pk-left .pk-button {
  border-color: #3b82f6;
  color: #3b82f6;
}

.pk-right .pk-button {
  border-color: #ef4444;
  color: #ef4444;
}

.pk-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.pk-left .pk-button:hover:not(:disabled) {
  background: rgba(59, 130, 246, 0.08);
}

.pk-right .pk-button:hover:not(:disabled) {
  background: rgba(239, 68, 68, 0.08);
}

.pk-button:disabled {
  cursor: not-allowed;
}

.pk-option.pk-voted .pk-button {
  border-width: 4px;
  font-weight: 600;
}

.pk-left.pk-voted .pk-button {
  background: rgba(59, 130, 246, 0.12);
}

.pk-right.pk-voted .pk-button {
  background: rgba(239, 68, 68, 0.12);
}

.pk-text {
  font-size: 16px;
  font-weight: 500;
  word-break: break-word;
}

.pk-stats {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.pk-count {
  font-size: 20px;
  font-weight: 700;
}

.pk-percentage {
  font-size: 13px;
  opacity: 0.8;
}

.pk-vs {
  font-size: 18px;
  font-weight: 700;
  color: #888;
  padding: 0 8px;
}

.pk-progress-container {
  position: relative;
  height: 24px;
  border-radius: 12px;
  overflow: hidden;
  background: #f0f0f0;
  display: flex;
  margin-bottom: 8px;
}

.pk-progress-bar {
  height: 100%;
  transition: width 0.5s ease;
}

.pk-progress-left {
  background: linear-gradient(to right, #3b82f6, #60a5fa);
  border-radius: 12px 0 0 12px;
}

.pk-progress-right {
  background: linear-gradient(to left, #ef4444, #f87171);
  border-radius: 0 12px 12px 0;
  margin-left: auto;
}

/* ========== 多选项模式 ========== */
.options-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.option-wrapper {
  position: relative;
}

.option-button {
  position: relative;
  width: 100%;
  padding: 16px 20px;
  background: white;
  border: 2px solid var(--option-color);
  border-radius: 12px;
  font-size: 16px;
  color: var(--option-color);
  cursor: pointer;
  transition: all 0.3s ease;
  text-align: left;
  overflow: hidden;
  z-index: 1;
}

.option-button:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.9);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.option-button:disabled {
  cursor: not-allowed;
}

.option-wrapper.option-voted .option-button {
  background: color-mix(in srgb, var(--option-color) 15%, white);
  border-width: 3px;
}

.option-inner {
  position: relative;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.option-text {
  font-size: 16px;
  font-weight: 500;
  word-break: break-word;
}

.vote-stats {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  min-width: 80px;
  text-align: right;
}

.vote-count {
  font-size: 18px;
  font-weight: 700;
}

.vote-percentage {
  font-size: 13px;
  opacity: 0.8;
}

.progress-bar {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 100%;
  background: color-mix(in srgb, var(--option-color) 10%, white);
  transition: width 0.5s ease;
  z-index: 0;
}

.voted-message {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 20px;
  padding: 12px;
  background: rgba(52, 211, 153, 0.1);
  border-radius: 8px;
  font-size: 14px;
  color: #059669;
}

.voted-message svg {
  width: 16px;
  height: 16px;
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

  .option-button {
    padding: 14px 16px;
  }

  .option-text {
    font-size: 15px;
  }

  .vote-count {
    font-size: 16px;
  }
}
</style>

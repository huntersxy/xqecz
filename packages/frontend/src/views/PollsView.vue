<script setup lang="ts">
import { computed, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { IconArrowLeft, IconCheck } from '@arco-design/web-vue/es/icon'
import { usePollBallot } from '@/composables/usePollBallot'
import { usePollList } from '@/composables/usePollList'
import PollPreview from '@/components/PollPreview.vue'
import '@/assets/creative-pages.css'

const route = useRoute()
const router = useRouter()
const {
  rows,
  total,
  totalPages,
  loading: listLoading,
  error: listError,
  load: loadList,
  stop: stopList,
} = usePollList(12)
const isDetail = computed(() => route.params.id !== undefined)
const page = computed(() => {
  const value = Number(route.query.page)
  return Number.isSafeInteger(value) && value > 0 ? value : 1
})
const ballotId = computed(() => (isDetail.value ? Number(route.params.id) || -1 : 0))
const {
  detail,
  loading,
  error,
  voteError,
  selected,
  submitting,
  resultsVisible,
  canVote,
  vote,
  percentage,
  loadDetail,
} = usePollBallot(ballotId)

watch(
  () => [route.params.id, route.query.page],
  () => {
    if (isDetail.value) {
      stopList()
    } else void loadList(page.value)
    window.scrollTo({ top: 0 })
  },
  { immediate: true },
)

function goPage(value: number) {
  void router.push({ path: '/polls', query: value > 1 ? { page: value } : {} })
}
</script>

<template>
  <div class="creative-page creative-theme poll-page">
    <header class="creative-page-head">
      <div>
        <h1>投票广场</h1>
        <p>选一份偏爱，让大家看见你的选择。</p>
      </div>
      <RouterLink to="/" class="poll-back"
        ><IconArrowLeft aria-hidden="true" /> 返回首页</RouterLink
      >
    </header>

    <template v-if="isDetail">
      <RouterLink to="/polls" class="poll-back poll-list-back"
        ><IconArrowLeft aria-hidden="true" /> 全部投票</RouterLink
      >
      <p v-if="loading" class="poll-page-state" role="status">正在加载投票…</p>
      <div v-else-if="error" class="poll-page-state" role="alert">
        <p>{{ error }}</p>
        <button class="poll-secondary" type="button" @click="loadDetail">重新加载</button>
      </div>
      <article v-else-if="detail" class="poll-ballot">
        <header class="poll-ballot-heading">
          <h2>{{ detail.poll.title }}</h2>
          <p class="poll-byline">
            {{ detail.poll.user?.username || '已注销用户' }} 发起 · {{ detail.total_votes }} 票
          </p>
          <p v-if="detail.poll.description" class="poll-description">
            {{ detail.poll.description }}
          </p>
        </header>
        <form @submit.prevent="vote">
          <fieldset :disabled="submitting">
            <legend>选择一个选项</legend>
            <label
              v-for="(option, index) in detail.poll.options"
              :key="index"
              class="poll-option"
              :class="{ 'is-selected': selected === index, 'is-voted': detail.my_vote === index }"
            >
              <span
                v-if="resultsVisible"
                class="poll-option-fill"
                :style="{ width: percentage(index) + '%' }"
                aria-hidden="true"
              ></span>
              <input
                v-model="selected"
                type="radio"
                :name="`poll-${detail.poll.id}`"
                :value="index"
              />
              <span class="poll-option-name"
                >{{ option }}
                <small v-if="detail.my_vote === index"
                  ><IconCheck aria-hidden="true" /> 已投</small
                ></span
              >
              <span v-if="resultsVisible" class="poll-option-result"
                >{{ detail.vote_counts[String(index)] || 0 }} 票
                <b>{{ percentage(index) }}%</b></span
              >
            </label>
          </fieldset>
          <div v-if="voteError" class="poll-vote-error" role="alert">{{ voteError }}</div>
          <div class="poll-ballot-actions">
            <button class="poll-submit" type="submit" :disabled="!canVote" :aria-busy="submitting">
              {{
                submitting
                  ? '正在提交…'
                  : detail.my_vote === null
                    ? '确认投票'
                    : selected === detail.my_vote
                      ? '已投票'
                      : '确认改票'
              }}
            </button>
            <button
              v-if="detail.my_vote === null"
              class="poll-secondary"
              type="button"
              @click="resultsVisible = !resultsVisible"
            >
              {{ resultsVisible ? '收起结果' : '查看结果' }}
            </button>
            <span v-else class="poll-vote-status" role="status"
              >投票已记录，可重新选择后改票。</span
            >
          </div>
          <p class="poll-participation-note">
            单选投票，游客也可参与。改票会更新你的选择，不增加总票数。
          </p>
        </form>
      </article>
    </template>

    <template v-else>
      <div class="poll-list-heading">
        <h2>全部投票</h2>
        <span>共 {{ total }} 项</span>
      </div>
      <p v-if="listLoading" class="poll-page-state" role="status">正在加载投票…</p>
      <div v-else-if="listError" class="poll-page-state" role="alert">
        <p>{{ listError }}</p>
        <button class="poll-secondary" type="button" @click="loadList(page)">重新加载</button>
      </div>
      <p v-else-if="!rows.length" class="poll-page-state" role="status">
        暂时没有投票，稍后来看看。
      </p>
      <div v-else class="poll-preview-grid">
        <PollPreview v-for="poll in rows" :key="poll.id" :poll="poll" />
      </div>
      <nav v-if="totalPages > 1" class="poll-pagination" aria-label="投票分页">
        <button
          class="poll-secondary"
          type="button"
          :disabled="page <= 1 || listLoading"
          @click="goPage(page - 1)"
        >
          上一页
        </button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button
          class="poll-secondary"
          type="button"
          :disabled="page >= totalPages || listLoading"
          @click="goPage(page + 1)"
        >
          下一页
        </button>
      </nav>
    </template>
  </div>
</template>

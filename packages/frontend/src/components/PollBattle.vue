<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, useId } from 'vue'
import { IconCheck, IconPause, IconPlayArrow } from '@arco-design/web-vue/es/icon'
import { usePollBallot } from '@/composables/usePollBallot'
import type { Poll } from '@/types/schemas'

const props = defineProps<{ poll: Poll }>()
const uid = useId()
const {
  detail,
  loading,
  error,
  voteError,
  selected,
  submitting,
  canVote,
  vote,
  percentage,
  loadDetail,
} = usePollBallot(computed(() => props.poll.id))
const options = computed(() => detail.value?.poll.options ?? props.poll.options)
const duel = computed(() => options.value.length === 2)
const total = computed(() => detail.value?.total_votes ?? props.poll.vote_count)
const mode = computed(() =>
  duel.value ? '红蓝 PK' : options.value.length === 3 ? '三方对抗' : '多方争霸',
)
const collision = computed(() =>
  detail.value?.total_votes
    ? ((detail.value.vote_counts['0'] ?? 0) / detail.value.total_votes) * 600
    : 300,
)
const palette = [
  '#c34560',
  '#3478ca',
  '#998020',
  '#338877',
  '#9163b8',
  '#bd663b',
  '#a64b8d',
  '#54743c',
]
const color = (index: number) => palette[index] ?? `hsl(${(index * 137.508) % 360} 48% 44%)`
const factions = computed(() =>
  options.value.map((_, index) => {
    const angle = (index / options.value.length) * Math.PI * 2 - Math.PI / 2
    return { x: 300 + Math.cos(angle) * 260, y: 80 + Math.sin(angle) * 64 }
  }),
)
const status = computed(() => {
  if (!total.value) return '尚未开局，第一票由你决定'
  if (!detail.value) return '正在读取战况'
  const counts = options.value.map((_, index) => detail.value!.vote_counts[String(index)] ?? 0)
  const max = Math.max(...counts)
  if (counts.filter((count) => count === max).length > 1) return '势均力敌，等你打破僵局'
  const leader = counts.indexOf(max)
  return `${options.value[leader]} 暂时领先`
})

const root = ref<HTMLElement>()
const visible = ref(false)
const pageVisible = ref(true)
const active = ref(true)
const paused = ref(false)
const running = computed(() => visible.value && pageVisible.value && active.value && !paused.value)
let observer: IntersectionObserver | undefined
function visibilityChanged() {
  pageVisible.value = !document.hidden
}
onMounted(() => {
  visibilityChanged()
  document.addEventListener('visibilitychange', visibilityChanged)
  // Only an on-screen, active homepage owns a running energy loop.
  if (typeof IntersectionObserver !== 'undefined') {
    observer = new IntersectionObserver((entries) => {
      visible.value = entries[0]?.isIntersecting ?? false
    })
    if (root.value) observer.observe(root.value)
  }
})
onActivated(() => {
  active.value = true
})
onDeactivated(() => {
  active.value = false
})
onBeforeUnmount(() => {
  observer?.disconnect()
  document.removeEventListener('visibilitychange', visibilityChanged)
})
</script>

<template>
  <article
    ref="root"
    class="poll-battle"
    :class="{ 'is-running': running, 'is-duel': duel }"
    :aria-labelledby="`${uid}-title`"
    :aria-busy="loading || submitting"
  >
    <header class="poll-battle-heading">
      <h3 :id="`${uid}-title`">{{ detail?.poll.title ?? poll.title }}</h3>
      <div class="poll-battle-tools">
        <span class="poll-battle-mode">{{ mode }}</span>
        <button
          type="button"
          class="battle-motion-toggle"
          :aria-pressed="paused"
          :aria-label="paused ? '播放对抗动画' : '暂停对抗动画'"
          @click="paused = !paused"
        >
          <IconPlayArrow v-if="paused" aria-hidden="true" /><IconPause v-else aria-hidden="true" />
        </button>
      </div>
    </header>
    <div class="poll-battle-arena" aria-hidden="true">
      <svg
        viewBox="0 0 600 160"
        :preserveAspectRatio="duel ? 'none' : 'xMidYMid meet'"
        focusable="false"
      >
        <template v-if="duel">
          <path
            :d="`M 0 30 Q ${collision * 0.65} 40 ${collision} 80 Q ${collision * 0.65} 120 0 130 Z`"
            :fill="color(0)"
            opacity="0.14"
          />
          <path
            :d="`M 600 30 Q ${600 - (600 - collision) * 0.65} 40 ${collision} 80 Q ${600 - (600 - collision) * 0.65} 120 600 130 Z`"
            :fill="color(1)"
            opacity="0.14"
          />
          <path
            class="battle-beam"
            :d="`M 0 80 C ${collision * 0.25} 58 ${collision * 0.55} 102 ${collision} 80`"
            :stroke="color(0)"
            stroke-width="30"
          />
          <path
            class="battle-beam"
            :d="`M 600 80 C ${600 - (600 - collision) * 0.25} 102 ${600 - (600 - collision) * 0.55} 58 ${collision} 80`"
            :stroke="color(1)"
            stroke-width="30"
          />
          <path
            class="battle-stream"
            :d="`M 0 80 C ${collision * 0.25} 58 ${collision * 0.55} 102 ${collision} 80`"
            stroke="white"
          />
          <path
            class="battle-stream"
            :d="`M 600 80 C ${600 - (600 - collision) * 0.25} 102 ${600 - (600 - collision) * 0.55} 58 ${collision} 80`"
            stroke="white"
          />
          <g :transform="`translate(${collision} 80)`">
            <ellipse class="battle-impact" rx="9" ry="38" />
            <path class="battle-spark" d="M-12 -50 L7 -25 L-4 -10 L13 10 L-6 28 L12 50" />
          </g>
        </template>
        <template v-else>
          <g v-for="(faction, index) in factions" :key="index">
            <path
              class="battle-beam"
              :d="`M ${faction.x} ${faction.y} L 300 80`"
              :stroke="color(index)"
              :stroke-width="10 + percentage(index) * 0.35"
            />
            <path
              class="battle-stream"
              :d="`M ${faction.x} ${faction.y} L 300 80`"
              stroke="white"
              :style="{ animationDelay: `${index * -0.3}s` }"
            />
            <circle :cx="faction.x" :cy="faction.y" r="5" :fill="color(index)" />
          </g>
          <circle class="battle-core" cx="300" cy="80" r="26" />
          <circle class="battle-orbit" cx="300" cy="80" r="33" />
        </template>
        <text v-if="!duel" class="battle-pk" x="300" y="86" text-anchor="middle">PK</text>
      </svg>
    </div>
    <div v-if="error" class="poll-section-state" role="alert">
      <span>{{ error }}</span
      ><button type="button" @click="loadDetail">重新加载</button>
    </div>
    <p v-else-if="loading" class="poll-section-state" role="status">正在加载选项…</p>
    <form v-else-if="detail" @submit.prevent="vote">
      <fieldset class="battle-options" :disabled="submitting">
        <legend class="battle-legend">选择你支持的一方</legend>
        <label
          v-for="(option, index) in options"
          :key="index"
          class="battle-option"
          :class="{ 'is-selected': selected === index }"
          :style="{ '--faction-color': color(index) }"
        >
          <input v-model="selected" type="radio" :name="`${uid}-option`" :value="index" />
          <span class="battle-option-copy"
            ><span>{{ option }}</span
            ><small v-if="detail.my_vote === index"
              ><IconCheck aria-hidden="true" /> 已投</small
            ></span
          >
          <span class="battle-option-stat"
            ><b>{{ percentage(index) }}%</b
            ><small>{{ detail.vote_counts[String(index)] ?? 0 }} 票</small></span
          >
          <span class="battle-option-meter" aria-hidden="true"
            ><i :style="{ transform: `scaleX(${percentage(index) / 100})` }"></i
          ></span>
        </label>
      </fieldset>
      <p v-if="voteError" class="poll-vote-error" role="alert">{{ voteError }}</p>
      <footer class="poll-battle-footer">
        <div class="poll-battle-summary">
          <span class="poll-battle-status">{{ status }}</span>
          <span class="poll-battle-total" role="status"
            >{{ total }} 票<span v-if="detail.my_vote !== null"> · 已记录，可改票</span></span
          >
        </div>
        <button class="poll-submit" type="submit" :disabled="!canVote" :aria-busy="submitting">
          {{
            submitting
              ? '正在提交…'
              : detail.my_vote === null
                ? '为 TA 助力'
                : selected === detail.my_vote
                  ? '已助力'
                  : '确认改票'
          }}
        </button>
      </footer>
    </form>
  </article>
</template>

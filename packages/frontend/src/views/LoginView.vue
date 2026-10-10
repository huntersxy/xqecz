<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const isLoginMode = ref(true)
const username = ref('')
const email = ref('')
const password = ref('')
const isLoading = ref(false)

// 缺邮箱弹窗
const showEmailModal = ref(false)
const newEmail = ref('')
const emailModalError = ref('')
const emailModalLoading = ref(false)
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

async function handleSubmit() {
  if (isLoading.value) return
  isLoading.value = true

  try {
    if (isLoginMode.value) {
      const success = await userStore.login(username.value, password.value)
      if (success) {
        if (userStore.needsEmail) {
          showEmailModal.value = true
        } else {
          router.push('/')
        }
      } else {
        Message.error('用户名或密码错误')
      }
    } else {
      if (!email.value.trim() || !EMAIL_RE.test(email.value.trim())) {
        Message.error('请输入有效的邮箱地址')
        isLoading.value = false
        return
      }
      const res = await authApi.register(username.value, email.value.trim(), password.value)
      if (res.code === 200) {
        Message.success('注册成功，请登录')
        isLoginMode.value = true
        username.value = ''
        email.value = ''
        password.value = ''
      } else {
        Message.error(res.message || '注册失败')
      }
    }
  } catch {
    Message.error('网络错误，请稍后重试')
  } finally {
    isLoading.value = false
  }
}

async function submitEmail(): Promise<boolean> {
  if (emailModalLoading.value) return false
  if (!EMAIL_RE.test(newEmail.value.trim())) {
    emailModalError.value = '请输入有效的邮箱地址'
    return false
  }
  emailModalLoading.value = true
  emailModalError.value = ''
  try {
    const ok = await userStore.updateEmail(newEmail.value.trim())
    if (ok) {
      showEmailModal.value = false
      router.push('/')
      return true
    } else {
      emailModalError.value = '更新失败，请重试'
      return false
    }
  } catch {
    emailModalError.value = '网络错误'
    return false
  } finally {
    emailModalLoading.value = false
  }
}

function switchMode() {
  if (isLoading.value) return
  isLoginMode.value = !isLoginMode.value
  username.value = ''
  email.value = ''
  password.value = ''
  showEmailModal.value = false
}
</script>

<template>
  <div class="auth-page creative-theme">
    <div class="auth-layout">
      <section class="auth-story" aria-labelledby="auth-story-title">
        <span class="auth-kicker">A LITTLE WORLD OF OUR OWN <span aria-hidden="true">✦</span></span>
        <h1 id="auth-story-title">把喜欢的世界，<br>写成自己的故事<span>。</span></h1>
        <p>收藏心动，分享灵感。<br>和同样热爱创作的人，在这里相遇。</p>
        <div class="auth-art" aria-hidden="true">
          <div class="auth-orbit"></div>
          <div class="auth-art-card auth-art-back"><span>EVERY LITTLE IDEA</span><svg viewBox="0 0 120 100" fill="none"><path d="M18 68C40 2 76 2 103 68M24 62C54 96 76 80 100 38" stroke="currentColor" stroke-width="1.5"/><circle cx="60" cy="45" r="22" stroke="currentColor" stroke-width="1.5"/></svg><b>starts with you.</b></div>
          <div class="auth-art-card auth-art-front"><span>CREATE / COLLECT / CONNECT</span><strong>Make<br>your own<span>✧</span></strong><span class="auth-art-caption">每一份热爱，都有回响</span></div>
          <span class="auth-spark auth-spark-one">✦</span><span class="auth-spark auth-spark-two">✧</span>
        </div>
        <RouterLink to="/" class="auth-home-link">← 返回创作首页</RouterLink>
      </section>

      <section class="auth-card" aria-labelledby="auth-title">
        <span class="auth-card-kicker">YOUR CREATIVE CORNER <span aria-hidden="true">✦</span></span>
        <h2 id="auth-title"><Transition name="auth-copy" mode="out-in"><span :key="isLoginMode ? 'login' : 'register'">{{ isLoginMode ? '欢迎回来' : '初次见面，请多指教' }}<i>。</i></span></Transition></h2>
        <p class="auth-card-desc"><Transition name="auth-copy" mode="out-in"><span :key="isLoginMode ? 'login' : 'register'">{{ isLoginMode ? '登录，继续你的创作漫游。' : '创建账号，让灵感有一个小小的归处。' }}</span></Transition></p>
        <div class="auth-mode" :class="{ 'is-register': !isLoginMode }" role="group" aria-label="账号操作">
          <button type="button" :class="{ 'is-active': isLoginMode }" :aria-pressed="isLoginMode" :disabled="isLoading" @click="!isLoginMode && switchMode()">登录</button>
          <button type="button" :class="{ 'is-active': !isLoginMode }" :aria-pressed="!isLoginMode" :disabled="isLoading" @click="isLoginMode && switchMode()">注册</button>
        </div>
        <form class="auth-form" @submit.prevent="handleSubmit">
          <div class="auth-field"><label for="auth-username">用户名</label><input id="auth-username" v-model="username" type="text" placeholder="怎么称呼你？" autocomplete="username" required minlength="2" maxlength="32" :disabled="isLoading" /></div>
          <div class="auth-email-slot" :class="{ 'is-open': !isLoginMode }" :aria-hidden="isLoginMode" :inert="isLoginMode"><div class="auth-field"><label for="auth-email">邮箱</label><input id="auth-email" v-model="email" type="email" placeholder="用于账号联系，不会公开" autocomplete="email" :required="!isLoginMode" maxlength="254" :disabled="isLoading || isLoginMode" /></div></div>
          <div class="auth-field"><label for="auth-password">密码<span>至少 6 位</span></label><input id="auth-password" v-model="password" type="password" placeholder="请输入密码" :autocomplete="isLoginMode ? 'current-password' : 'new-password'" required minlength="6" :disabled="isLoading" /></div>
          <button type="submit" :disabled="isLoading" class="auth-submit"><span v-if="isLoading" class="auth-spinner"></span>{{ isLoading ? '处理中…' : (isLoginMode ? '登录，开启创作漫游' : '注册，加入这个小世界') }}<span v-if="!isLoading" aria-hidden="true">↗</span></button>
        </form>
        <p class="auth-switch">{{ isLoginMode ? '还没有账号？' : '已有账号？' }}<button type="button" :disabled="isLoading" @click="switchMode">{{ isLoginMode ? '在这里安放你的灵感' : '欢迎回来，去登录' }} <span aria-hidden="true">→</span></button></p>
        <div class="auth-card-note"><span aria-hidden="true">✧</span> 每一份创作，都值得被看见。</div>
      </section>
    </div>
    <a-modal v-model:visible="showEmailModal" title="为账号添一份联系" modal-class="auth-email-modal creative-theme" :mask-closable="false" :closable="false" :esc-to-close="false">
      <a-typography-text type="secondary" class="email-modal-desc">你的账号尚未绑定邮箱。设置邮箱后，就可以继续创作漫游。</a-typography-text>
      <a-typography-text v-if="emailModalError" type="danger" class="email-modal-error">{{ emailModalError }}</a-typography-text>
      <a-input v-model="newEmail" placeholder="请输入邮箱地址" :maxlength="254" class="email-modal-input" :disabled="emailModalLoading" @keyup.enter="submitEmail" />
      <template #footer><a-button type="primary" :loading="emailModalLoading" @click="submitEmail">保存邮箱</a-button></template>
    </a-modal>
  </div>
</template>

<style scoped>
.auth-page { padding: 70px 32px 80px; background: var(--creative-canvas); }
.auth-layout { max-width: 1000px; margin: 0 auto; display: grid; grid-template-columns: 1fr 430px; gap: 80px; align-items: center; }
.auth-kicker, .auth-card-kicker { display: block; font-size: 9px; letter-spacing: .16em; color: var(--creative-muted); }
.auth-kicker > span, .auth-card-kicker > span { margin-left: 8px; color: var(--creative-accent); }
.auth-story h1 { margin: 22px 0 18px; font-size: clamp(28px, 3.2vw, 39px); line-height: 1.6; letter-spacing: .02em; font-weight: 650; color: var(--creative-ink); }
.auth-story h1 > span, .auth-card h2 i { color: var(--creative-accent); }
.auth-story > p { font-size: 13px; color: var(--creative-muted); line-height: 2; }
.auth-art { position: relative; height: 240px; margin: 30px 0 22px; max-width: 360px; }
.auth-orbit { position: absolute; width: 260px; height: 180px; top: 32px; left: 28px; border: 1px solid var(--creative-line); border-radius: 50%; transform: rotate(-24deg); }
.auth-art-card { position: absolute; width: 168px; height: 192px; padding: 18px; border: 1px solid var(--creative-line); border-radius: 12px; box-shadow: var(--creative-shadow); }
.auth-art-card > span { font-size: 7px; letter-spacing: .07em; }
.auth-art-back { top: 20px; left: 10px; background: var(--creative-soft); color: var(--creative-accent); transform: rotate(-10deg); }
.auth-art-back svg { display: block; width: 100%; height: 105px; margin-top: 8px; }
.auth-art-back b { font-family: Georgia, serif; font-size: 18px; font-style: italic; font-weight: 400; }
.auth-art-front { top: 36px; left: 148px; background: var(--creative-paper); color: var(--creative-muted); transform: rotate(8deg); }
.auth-art-front strong { display: block; margin: 28px 0 20px; font-family: Georgia, serif; font-size: 30px; line-height: 1.05; font-weight: 400; color: var(--creative-ink); }
.auth-art-front strong > span { position: absolute; right: 20px; top: 82px; font-size: 34px; color: var(--creative-accent); }
.auth-art-caption { color: var(--creative-accent); }
.auth-spark { position: absolute; color: var(--creative-accent); }
.auth-spark-one { top: 0; left: 266px; font-size: 28px; }
.auth-spark-two { bottom: 0; left: 32px; font-size: 19px; }
.auth-home-link { font-size: 12px; text-decoration: none; color: var(--creative-muted); }
.auth-home-link:hover { color: var(--creative-accent); }
.auth-card { padding: 36px; border: 1px solid var(--creative-line); border-radius: 24px; background: var(--creative-paper); box-shadow: var(--creative-shadow); }
.auth-card h2 { margin: 14px 0 8px; font-size: 24px; font-weight: 650; line-height: 1.5; color: var(--creative-ink); }
.auth-card-desc { margin: 0; color: var(--creative-muted); font-size: 12px; line-height: 1.8; }
.auth-mode { position: relative; display: flex; gap: 4px; padding: 4px; margin: 26px 0 24px; border: 1px solid var(--creative-line); border-radius: 999px; background: var(--creative-canvas); }
.auth-mode button { position: relative; z-index: 1; flex: 1; padding: 9px; border: 0; border-radius: 999px; background: transparent; color: var(--creative-muted); font-size: 12px; cursor: pointer; transition: color .18s, background .18s, box-shadow .18s; }
.auth-mode .is-active { color: var(--creative-accent); background: transparent; font-weight: 600; }
.auth-form { display: flex; flex-direction: column; gap: 0; }
.auth-form > .auth-field { margin-bottom: 20px; }
.auth-field { display: flex; flex-direction: column; gap: 9px; }
.auth-field label { display: flex; align-items: center; justify-content: space-between; color: var(--creative-ink); font-size: 12px; font-weight: 600; }
.auth-field label > span { font-size: 10px; font-weight: 400; color: var(--creative-muted); }
.auth-field input { width: 100%; min-height: 44px; padding: 11px 13px; border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); color: var(--creative-ink); font-size: 12px; outline: none; transition: border-color .18s, box-shadow .18s; }
.auth-field input::placeholder { color: var(--creative-muted); }
.auth-field input:focus { background: var(--creative-paper); border-color: var(--creative-accent); box-shadow: 0 0 0 3px color-mix(in srgb, var(--creative-accent) 9%, transparent); }
.auth-submit { display: flex; align-items: center; justify-content: center; gap: 10px; width: 100%; margin-top: 4px; min-height: 44px; padding: 12px 15px; border: 0; border-radius: 999px; background: var(--creative-accent); color: var(--color-on-primary); font-size: 12px; font-weight: 600; cursor: pointer; transition: background .18s, box-shadow .18s; }
.auth-submit:hover { background: var(--creative-accent-hover); box-shadow: 0 5px 16px color-mix(in srgb, var(--creative-accent) 18%, transparent); }
.auth-submit > span:last-child:not(.auth-spinner) { font-size: 17px; }
.auth-spinner { width: 14px; height: 14px; border: 2px solid currentColor; border-right-color: transparent; border-radius: 50%; animation: auth-spin .8s linear infinite; }
@keyframes auth-spin { to { transform: rotate(360deg); } }
.auth-switch { display: flex; flex-wrap: wrap; justify-content: center; gap: 5px; margin: 24px 0 0; color: var(--creative-muted); font-size: 11px; line-height: 1.8; }
.auth-switch button { border: 0; padding: 0; background: transparent; color: var(--creative-accent); font-size: inherit; cursor: pointer; }
.auth-card-note { display: flex; justify-content: center; gap: 7px; margin-top: 24px; padding-top: 20px; border-top: 1px solid var(--creative-line); color: var(--creative-muted); font-size: 10px; }
.auth-card-note > span { color: var(--creative-accent); }
button:disabled { cursor: not-allowed; opacity: .6; }
.email-modal-desc { display: block; }
.email-modal-error { display: block; margin: 12px 0; }
.email-modal-input { margin-top: 16px; }
:global(.auth-email-modal) { border: 1px solid var(--creative-line); border-radius: 20px; background: var(--creative-paper); }
:global(.auth-email-modal .arco-modal-title) { color: var(--creative-ink); }
:global(.auth-email-modal .arco-modal-header), :global(.auth-email-modal .arco-modal-footer) { border-color: var(--creative-line); }
:global(.auth-email-modal .arco-input-wrapper) { border: 1px solid var(--creative-line); border-radius: 10px; background: var(--creative-canvas); }
:global(.auth-email-modal .arco-btn-primary) { border-radius: 999px; }
@media (max-width: 1000px) { .auth-layout { grid-template-columns: 1fr 400px; gap: 40px; } .auth-art { transform: scale(.9); transform-origin: left center; } }
@media (max-width: 768px) { .auth-page { padding: 32px 20px 44px; } .auth-layout { grid-template-columns: minmax(0, 1fr); gap: 28px; max-width: 430px; } .auth-story h1 { margin: 14px 0 10px; font-size: 26px; } .auth-story > p { font-size: 12px; } .auth-art { display: none; } .auth-home-link { display: inline-block; margin-top: 15px; font-size: 11px; } .auth-card { padding: 28px 24px; border-radius: 20px; } .auth-card h2 { font-size: 23px; } }

.auth-mode::before { content: ''; position: absolute; top: 4px; bottom: 4px; left: 4px; width: calc((100% - 12px) / 2); border-radius: 999px; background: var(--creative-paper); box-shadow: 0 2px 8px color-mix(in srgb, var(--creative-accent) 8%, transparent); transition: transform .28s cubic-bezier(.22, 1, .36, 1); }
.auth-mode.is-register::before { transform: translateX(calc(100% + 4px)); }
.auth-email-slot { display: grid; grid-template-rows: 0fr; opacity: 0; margin-bottom: 0; transition: grid-template-rows .28s cubic-bezier(.22, 1, .36, 1), opacity .2s ease, margin-bottom .28s ease; }
.auth-email-slot > .auth-field { min-height: 0; overflow: hidden; }
.auth-email-slot.is-open { grid-template-rows: 1fr; opacity: 1; margin-bottom: 20px; }
.auth-card h2 { min-height: 36px; }
.auth-card h2 i { font-style: normal; }
.auth-card-desc { min-height: 22px; }
.auth-copy-enter-active, .auth-copy-leave-active { display: inline-block; transition: opacity .12s ease, transform .12s ease; }
.auth-copy-enter-from { opacity: 0; transform: translateY(4px); }
.auth-copy-leave-to { opacity: 0; transform: translateY(-4px); }
</style>

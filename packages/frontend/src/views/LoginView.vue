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

async function submitEmail() {
  if (!EMAIL_RE.test(newEmail.value.trim())) {
    emailModalError.value = '请输入有效的邮箱地址'
    return
  }
  emailModalLoading.value = true
  emailModalError.value = ''
  try {
    const ok = await userStore.updateEmail(newEmail.value.trim())
    if (ok) {
      showEmailModal.value = false
      router.push('/')
    } else {
      emailModalError.value = '更新失败，请重试'
    }
  } catch {
    emailModalError.value = '网络错误'
  } finally {
    emailModalLoading.value = false
  }
}

function switchMode() {
  isLoginMode.value = !isLoginMode.value
  username.value = ''
  email.value = ''
  password.value = ''
  showEmailModal.value = false
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-[400px]">
      <div class="text-center mb-6">
        <div class="w-14 h-14 mx-auto mb-3 bg-[var(--theme-primary)] rounded-2xl flex items-center justify-center shadow-lg shadow-[var(--theme-primary)]/20">
          <svg class="w-8 h-8 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <path d="M12 6v6l4 2" />
          </svg>
        </div>
        <h1 class="text-xl font-bold theme-text">
          {{ isLoginMode ? '欢迎回来' : '创建账号' }}
        </h1>
      </div>

      <div class="theme-card rounded-xl p-6 shadow-sm theme-border">
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium theme-text-secondary mb-1.5">用户名</label>
            <input
              v-model="username"
              type="text"
              placeholder="请输入用户名"
              required
              minlength="2"
              maxlength="32"
              class="w-full px-3.5 py-2.5 text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-[var(--theme-primary)] focus:ring-2 focus:ring-[var(--theme-primary)]/10 transition-all"
            />
          </div>

          <div v-if="!isLoginMode">
            <label class="block text-sm font-medium theme-text-secondary mb-1.5">邮箱</label>
            <input
              v-model="email"
              type="email"
              placeholder="请输入邮箱地址"
              required
              maxlength="254"
              class="w-full px-3.5 py-2.5 text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-[var(--theme-primary)] focus:ring-2 focus:ring-[var(--theme-primary)]/10 transition-all"
            />
          </div>

          <div>
            <label class="block text-sm font-medium theme-text-secondary mb-1.5">密码</label>
            <input
              v-model="password"
              type="password"
              placeholder="请输入密码"
              required
              minlength="6"
              class="w-full px-3.5 py-2.5 text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-[var(--theme-primary)] focus:ring-2 focus:ring-[var(--theme-primary)]/10 transition-all"
            />
          </div>

          <button
            type="submit"
            :disabled="isLoading"
            class="w-full py-2.5 text-sm font-semibold text-white bg-[var(--theme-primary)] rounded-lg hover:brightness-90 active:scale-[0.98] transition-all disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <span v-if="isLoading" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
            {{ isLoading ? '处理中...' : (isLoginMode ? '登录' : '注册') }}
          </button>
        </form>

        <p class="mt-5 pt-4 border-t theme-border text-center text-sm theme-text-secondary">
          {{ isLoginMode ? '还没有账号？' : '已有账号？' }}
          <button @click="switchMode" class="font-semibold text-[var(--theme-primary)] hover:underline ml-0.5">
            {{ isLoginMode ? '立即注册' : '去登录' }}
          </button>
        </p>
      </div>

      <div class="text-center mt-4">
        <button @click="router.push('/')" class="text-sm theme-text-secondary hover:text-[var(--theme-primary)] transition-colors">
          返回首页
        </button>
      </div>
    </div>

    <!-- 缺邮箱强制补填弹窗 -->
    <Teleport to="body">
      <div v-if="showEmailModal" class="fixed inset-0 z-[2000] flex items-center justify-center bg-black/70">
        <div class="w-[90%] max-w-[380px] bg-[var(--theme-surface)] rounded-xl p-6 shadow-2xl">
          <h3 class="text-base font-bold theme-text mb-2">请设置邮箱</h3>
          <p class="text-sm theme-text-secondary mb-4">
            你的账号尚未绑定邮箱。登录后需要设置邮箱才能继续使用。
          </p>
          <div v-if="emailModalError" class="text-sm text-[var(--theme-danger)] mb-3">{{ emailModalError }}</div>
          <input
            v-model="newEmail"
            type="email"
            placeholder="请输入邮箱地址"
            maxlength="254"
            class="w-full px-3.5 py-2.5 text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-[var(--theme-primary)] mb-4"
            @keyup.enter="submitEmail"
          />
          <button
            type="button"
            :disabled="emailModalLoading"
            class="w-full py-2.5 text-sm font-semibold text-white bg-[var(--theme-primary)] rounded-lg hover:brightness-90 transition-all disabled:opacity-60"
            @click="submitEmail"
          >
            {{ emailModalLoading ? '保存中...' : '确认' }}
          </button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

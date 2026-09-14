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

async function submitEmail(): Promise<boolean> {
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
        <div class="w-14 h-14 mx-auto mb-3 bg-[rgb(var(--primary-6))] rounded-2xl flex items-center justify-center shadow-lg shadow-[rgb(var(--primary-6))]/20">
          <svg class="w-8 h-8 text-white" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10" />
            <path d="M12 6v6l4 2" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-[var(--color-text-1)]">
          {{ isLoginMode ? '欢迎回来' : '创建账号' }}
        </h1>
      </div>

      <div class="bg-[var(--color-bg-2)] rounded-xl p-6 shadow-sm">
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-[var(--color-text-2)] mb-1.5">用户名</label>
            <input
              v-model="username"
              type="text"
              placeholder="请输入用户名"
              required
              minlength="2"
              maxlength="32"
              class="w-full px-3.5 py-2.5 text-sm text-[var(--color-text-1)] rounded-lg bg-[var(--color-bg-2)] focus:outline-none focus:border-[rgb(var(--primary-6))] focus:ring-2 focus:ring-[rgb(var(--primary-6))]/10 transition-all"
            />
          </div>

          <div v-if="!isLoginMode">
            <label class="block text-sm font-medium text-[var(--color-text-2)] mb-1.5">邮箱</label>
            <input
              v-model="email"
              type="email"
              placeholder="请输入邮箱地址"
              required
              maxlength="254"
              class="w-full px-3.5 py-2.5 text-sm text-[var(--color-text-1)] rounded-lg bg-[var(--color-bg-2)] focus:outline-none focus:border-[rgb(var(--primary-6))] focus:ring-2 focus:ring-[rgb(var(--primary-6))]/10 transition-all"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-[var(--color-text-2)] mb-1.5">密码</label>
            <input
              v-model="password"
              type="password"
              placeholder="请输入密码"
              required
              minlength="6"
              class="w-full px-3.5 py-2.5 text-sm text-[var(--color-text-1)] rounded-lg bg-[var(--color-bg-2)] focus:outline-none focus:border-[rgb(var(--primary-6))] focus:ring-2 focus:ring-[rgb(var(--primary-6))]/10 transition-all"
            />
          </div>

          <button
            type="submit"
            :disabled="isLoading"
            class="w-full py-2.5 text-sm font-semibold text-white bg-[rgb(var(--primary-6))] rounded-lg hover:brightness-90 active:scale-[0.98] transition-all disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <span v-if="isLoading" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
            {{ isLoading ? '处理中...' : (isLoginMode ? '登录' : '注册') }}
          </button>
        </form>

        <p class="mt-5 pt-4 border-t border-[var(--color-border-2)] text-center text-sm text-[var(--color-text-2)]">
          {{ isLoginMode ? '还没有账号？' : '已有账号？' }}
          <button @click="switchMode" class="font-semibold text-[rgb(var(--primary-6))] hover:underline ml-0.5">
            {{ isLoginMode ? '立即注册' : '去登录' }}
          </button>
        </p>
      </div>

      <div class="text-center mt-4">
        <button @click="router.push('/')" class="text-sm text-[var(--color-text-2)] hover:text-[rgb(var(--primary-6))] transition-colors">
          返回首页
        </button>
      </div>
    </div>

    <!-- 缺邮箱强制补填弹窗 -->
    <a-modal
      v-model:visible="showEmailModal"
      title="请设置邮箱"
      :mask-closable="false"
      :closable="false"
      :esc-to-close="false"
    >
      <a-typography-text type="secondary" class="email-modal-desc">
        你的账号尚未绑定邮箱。登录后需要设置邮箱才能继续使用。
      </a-typography-text>

      <a-typography-text v-if="emailModalError" type="danger" class="email-modal-error">
        {{ emailModalError }}
      </a-typography-text>

      <a-input
        v-model="newEmail"
        placeholder="请输入邮箱地址"
        :maxlength="254"
        class="email-modal-input"
        @keyup.enter="submitEmail"
      />

      <template #footer>
        <a-button type="primary" :loading="emailModalLoading" @click="submitEmail">确认</a-button>
      </template>
    </a-modal>
  </div>
</template>

<style scoped>
.email-modal-desc {
  display: block;
}

.email-modal-error {
  display: block;
  margin: 12px 0;
}

.email-modal-input {
  margin-top: 16px;
}
</style>

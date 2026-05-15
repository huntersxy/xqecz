<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { authApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const isLoginMode = ref(true)
const username = ref('')
const password = ref('')
const message = ref('')
const isLoading = ref(false)

async function handleSubmit() {
  if (!username.value.trim() || !password.value.trim()) {
    message.value = '请填写用户名和密码'
    return
  }
  
  if (!isLoginMode.value) {
    if (username.value.trim().length < 3) {
      message.value = '用户名长度不能少于3个字符'
      return
    }
    if (username.value.trim().length > 50) {
      message.value = '用户名长度不能超过50个字符'
      return
    }
    if (password.value.length < 6) {
      message.value = '密码长度不能少于6个字符'
      return
    }
  }
  
  isLoading.value = true
  message.value = ''
  
  try {
    if (isLoginMode.value) {
      const success = await userStore.login(username.value, password.value)
      if (success) {
        router.push('/')
      } else {
        message.value = '用户名或密码错误'
      }
    } else {
      const res = await authApi.register(username.value, password.value)
      if (res.code === 200) {
        message.value = '注册成功，请登录'
        isLoginMode.value = true
        username.value = ''
        password.value = ''
      } else {
        message.value = res.message || '注册失败'
      }
    }
  } catch {
    message.value = '网络错误，请稍后重试'
  } finally {
    isLoading.value = false
  }
}

function toggleMode() {
  isLoginMode.value = !isLoginMode.value
  message.value = ''
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-3 sm:p-5 md:p-8">
    <div class="w-full max-w-sm sm:max-w-[420px] theme-card rounded-xl shadow-lg shadow-black/5 theme-border overflow-hidden">
      <div class="flex items-center px-3 sm:px-4 py-2 sm:py-2.5 theme-header-bg">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm theme-text-secondary font-medium">{{ isLoginMode ? '登录' : '注册' }}</div>
      </div>

      <div class="p-4 sm:p-6 md:p-8">
        <div class="text-center mb-4 sm:mb-5 md:mb-7">
          <div class="w-12 h-12 sm:w-14 sm:h-14 md:w-16 md:h-16 mx-auto mb-3 sm:mb-4 bg-(--theme-primary) rounded-xl flex items-center justify-center shadow-lg shadow-(--theme-primary)/30">
            <svg class="w-6 h-6 sm:w-7 sm:h-7 md:w-9 md:h-9 text-(--theme-on-primary)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <path d="M12 6v6l4 2" />
            </svg>
          </div>
          <h1 class="text-lg sm:text-xl md:text-xl font-bold theme-text mb-1 sm:mb-2">小泉动漫</h1>
          <p class="text-xs sm:text-sm theme-text-secondary">发现精彩二创内容</p>
        </div>

        <form @submit.prevent="handleSubmit" class="mb-4 sm:mb-5">
          <div class="mb-3 sm:mb-4 md:mb-4.5">
            <label class="flex items-center gap-1.5 sm:gap-2 text-xs sm:text-sm font-medium theme-text-secondary mb-1.5 sm:mb-2">
              <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
                <circle cx="12" cy="7" r="4" />
              </svg>
              用户名
            </label>
            <input
              v-model="username"
              type="text"
              class="w-full px-3 sm:px-4 py-2 sm:py-3 text-xs sm:text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-(--theme-primary) focus:ring-3 focus:ring-(--theme-primary)/10 transition-all min-h-[44px] sm:min-h-[48px]"
              placeholder="请输入用户名"
              required
            />
          </div>

          <div class="mb-3 sm:mb-4 md:mb-4.5">
            <label class="flex items-center gap-1.5 sm:gap-2 text-xs sm:text-sm font-medium theme-text-secondary mb-1.5 sm:mb-2">
              <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
                <path d="M7 11V7a5 5 0 0 1 10 0v4" />
              </svg>
              密码
            </label>
            <input
              v-model="password"
              type="password"
              class="w-full px-3 sm:px-4 py-2 sm:py-3 text-xs sm:text-sm theme-text theme-border rounded-lg theme-surface focus:outline-none focus:border-(--theme-primary) focus:ring-3 focus:ring-(--theme-primary)/10 transition-all min-h-[44px] sm:min-h-[48px]"
              placeholder="请输入密码"
              required
            />
          </div>

          <div v-if="message" class="px-3 sm:px-4 py-2 sm:py-3 rounded-lg text-xs sm:text-sm text-center mb-3 sm:mb-4" :class="{ 'bg-(--theme-danger)/10 text-(--theme-danger) theme-border border-(--theme-danger)/30': message.includes('错误') || message.includes('失败'), 'bg-(--theme-success)/10 text-(--theme-success) theme-border border-(--theme-success)/30': message.includes('成功') }">
            {{ message }}
          </div>

          <button
            type="submit"
            class="w-full py-2.5 sm:py-3 md:py-3.5 text-sm sm:text-base font-semibold flex items-center justify-center gap-2 bg-(--theme-primary) text-(--theme-on-primary) rounded-lg hover:brightness-90 transition-colors disabled:opacity-70 disabled:cursor-not-allowed min-h-[44px] sm:min-h-[48px]"
            :disabled="isLoading"
          >
            <span v-if="isLoading" class="w-4 h-4 sm:w-4.5 sm:h-4.5 border-2 border-(--theme-on-primary)/30 border-t-(--theme-on-primary) rounded-full animate-spin"></span>
            {{ isLoading ? '处理中...' : (isLoginMode ? '登录' : '注册') }}
          </button>
        </form>

        <div class="text-center text-xs sm:text-sm theme-text-secondary mb-3 sm:mb-4">
          <span>{{ isLoginMode ? '还没有账号？' : '已有账号？' }}</span>
          <button @click="toggleMode" class="text-(--theme-primary) font-semibold ml-1 px-2 py-1 rounded hover:bg-(--theme-primary)/10 transition-all">
            {{ isLoginMode ? '立即注册' : '立即登录' }}
          </button>
        </div>

        <div class="text-center">
          <button @click="router.push('/')" class="inline-flex items-center gap-1.5 px-3 sm:px-4 py-1.5 sm:py-2 theme-surface theme-border rounded-lg text-xs sm:text-sm theme-text-secondary hover:bg-[var(--theme-hover-bg)] hover:border-(--theme-primary) hover:text-(--theme-primary) hover:-translate-y-0.5 hover:shadow-lg transition-all min-h-[40px] sm:min-h-[44px]">
            <svg class="w-3 h-3 sm:w-3.5 sm:h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5" />
              <polyline points="12 19 5 12 12 5" />
            </svg>
            返回首页
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

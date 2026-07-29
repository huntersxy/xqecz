import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User } from '@/types'
import { authApi } from '@/api'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = ref(false)
  const isLoading = ref(false)
  const needsEmail = ref(false)

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    if (res.code === 200) {
      user.value = res.data.user
      isLoggedIn.value = true
      needsEmail.value = !!res.data.needs_email
      return true
    }
    return false
  }

  async function updateEmail(email: string) {
    const res = await authApi.updateEmail(email)
    if (res.code === 200) {
      needsEmail.value = false
      return true
    }
    return false
  }

  async function logout() {
    await authApi.logout()
    user.value = null
    isLoggedIn.value = false
    needsEmail.value = false
  }

  async function checkAuth() {
    isLoading.value = true
    try {
      const res = await authApi.getMe()
      if (res.code === 200) {
        user.value = res.data
        isLoggedIn.value = true
        needsEmail.value = !res.data.email
      }
    } catch {
      // API 调用失败时保持当前状态不变
    } finally {
      isLoading.value = false
    }
  }

  return {
    user,
    isLoggedIn,
    isLoading,
    needsEmail,
    login,
    logout,
    checkAuth,
    updateEmail,
  }
})

import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User } from '@/types'
import { authApi } from '@/api'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = ref(false)
  const isLoading = ref(false)
  const needsEmail = ref(false)
  let authRequest: Promise<void> | undefined
  let loginDay = ''
  const today = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Shanghai' }).format(new Date())

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    if (res.code === 200) {
      user.value = res.data.user
      isLoggedIn.value = true
      needsEmail.value = !!res.data.needs_email
      loginDay = today()
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
    loginDay = ''
  }

  function checkAuth(): Promise<void> {
    if (authRequest) return authRequest
    authRequest = loadAuth().finally(() => { authRequest = undefined })
    return authRequest
  }
  async function loadAuth() {
    isLoading.value = true
    try {
      const res = await authApi.getMe()
      if (res.code === 200) {
        user.value = res.data
        isLoggedIn.value = true
        needsEmail.value = !res.data.email
        loginDay = today()
      }
    } catch {
    } finally {
      isLoading.value = false
    }
  }

  function refreshDailyLogin() {
    if (isLoggedIn.value && loginDay !== today()) return checkAuth()
  }

  return {
    user,
    isLoggedIn,
    isLoading,
    needsEmail,
    login,
    logout,
    checkAuth,
    refreshDailyLogin,
    updateEmail,
  }
})

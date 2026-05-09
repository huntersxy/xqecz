import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { User } from '@/types'
import { authApi } from '@/api'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = ref(false)
  const isLoading = ref(false)

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    if (res.code === 200) {
      user.value = res.data.user
      isLoggedIn.value = true
      return true
    }
    return false
  }

  async function logout() {
    await authApi.logout()
    user.value = null
    isLoggedIn.value = false
  }

  async function checkAuth() {
    isLoading.value = true
    try {
      const res = await authApi.getMe()
      if (res.code === 200) {
        user.value = res.data
        isLoggedIn.value = true
      } else {
        user.value = null
        isLoggedIn.value = false
      }
    } catch (error) {
      user.value = null
      isLoggedIn.value = false
    } finally {
      isLoading.value = false
    }
  }

  function setUser(userData: User) {
    user.value = userData
    isLoggedIn.value = true
  }

  return {
    user,
    isLoggedIn,
    isLoading,
    login,
    logout,
    checkAuth,
    setUser
  }
})
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
  <div class="login-container">
    <div class="mac-window login-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">{{ isLoginMode ? '登录' : '注册' }}</div>
      </div>
      
      <div class="login-content">
        <div class="login-header">
          <div class="logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
          <h1 class="app-name">小泉动漫</h1>
          <p class="app-slogan">发现精彩二创内容</p>
        </div>
        
        <form @submit.prevent="handleSubmit" class="login-form">
          <div class="form-group">
            <label class="form-label">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                <circle cx="12" cy="7" r="4"/>
              </svg>
              用户名
            </label>
            <input 
              v-model="username" 
              type="text" 
              class="mac-input"
              placeholder="请输入用户名"
              required 
            />
          </div>
          
          <div class="form-group">
            <label class="form-label">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
                <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
              </svg>
              密码
            </label>
            <input 
              v-model="password" 
              type="password" 
              class="mac-input"
              placeholder="请输入密码"
              required 
            />
          </div>
          
          <div v-if="message" class="message-box" :class="{ error: message.includes('错误') || message.includes('失败'), success: message.includes('成功') }">
            {{ message }}
          </div>
          
          <button type="submit" class="submit-btn mac-btn primary-btn" :disabled="isLoading">
            <span v-if="isLoading" class="loading-spinner"></span>
            {{ isLoading ? '处理中...' : (isLoginMode ? '登录' : '注册') }}
          </button>
        </form>
        
        <div class="toggle-mode">
          <span>{{ isLoginMode ? '还没有账号？' : '已有账号？' }}</span>
          <button @click="toggleMode" class="toggle-btn">
            {{ isLoginMode ? '立即注册' : '立即登录' }}
          </button>
        </div>
        
        <div class="back-home">
          <button @click="router.push('/')" class="back-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5"/>
              <polyline points="12 19 5 12 12 5"/>
            </svg>
            返回首页
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.login-window {
  width: 100%;
  max-width: 400px;
  background: rgba(255, 255, 255, 0.85);
  border-radius: 16px;
  box-shadow: 
    0 8px 32px rgba(0, 0, 0, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(255, 255, 255, 0.4);
  overflow: hidden;
}

@supports (backdrop-filter: blur(20px)) or (-webkit-backdrop-filter: blur(20px)) {
  .login-window {
    background: rgba(255, 255, 255, 0.3);
    -webkit-backdrop-filter: blur(20px);
    backdrop-filter: blur(20px);
  }
}

.login-content {
  padding: 40px 32px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.logo {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  background: linear-gradient(135deg, #007aff, #0056b3);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 32px rgba(0, 122, 255, 0.3);
}

.logo svg {
  width: 36px;
  height: 36px;
  color: white;
}

.app-name {
  font-size: 24px;
  font-weight: 700;
  color: #1a1a1a;
  margin: 0 0 8px 0;
}

.app-slogan {
  font-size: 14px;
  color: #888;
  margin: 0;
}

.login-form {
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 500;
  color: #666;
  margin-bottom: 8px;
}

.form-label svg {
  width: 16px;
  height: 16px;
}

.mac-input {
  width: 100%;
  padding: 12px 16px;
  font-size: 14px;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 10px;
  background: white;
  transition: all 0.2s ease;
}

.mac-input:focus {
  outline: none;
  border-color: #007aff;
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.mac-input::placeholder {
  color: #aaa;
}

.message-box {
  padding: 12px 16px;
  border-radius: 10px;
  font-size: 13px;
  margin-bottom: 16px;
  text-align: center;
}

.message-box.error {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.message-box.success {
  background: #ecfdf5;
  color: #059669;
  border: 1px solid #a7f3d0;
}

.submit-btn {
  width: 100%;
  padding: 14px;
  font-size: 15px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.loading-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.toggle-mode {
  text-align: center;
  font-size: 13px;
  color: #666;
  margin-bottom: 20px;
}

.toggle-btn {
  background: none;
  border: none;
  color: #007aff;
  font-weight: 600;
  cursor: pointer;
  margin-left: 4px;
}

.toggle-btn:hover {
  text-decoration: underline;
}

.back-home {
  text-align: center;
}

.back-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: rgba(0, 0, 0, 0.05);
  border: none;
  border-radius: 8px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
  transition: all 0.2s ease;
}

.back-btn:hover {
  background: rgba(0, 0, 0, 0.1);
}

.back-btn svg {
  width: 14px;
  height: 14px;
}

@media screen and (max-width: 768px) {
  .login-container {
    padding: 12px;
    align-items: flex-start;
    padding-top: calc(60px + 20px);
  }

  .login-window {
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  }

  .login-content {
    padding: 32px 24px;
  }

  .logo {
    width: 56px;
    height: 56px;
    margin-bottom: 12px;
  }

  .logo svg {
    width: 32px;
    height: 32px;
  }

  .app-name {
    font-size: 20px;
    margin-bottom: 6px;
  }

  .app-slogan {
    font-size: 13px;
  }

  .login-header {
    margin-bottom: 24px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-label {
    font-size: 14px;
    margin-bottom: 6px;
  }

  .form-label svg {
    width: 18px;
    height: 18px;
  }

  .mac-input {
    padding: 14px 16px;
    font-size: 16px;
    min-height: 50px;
    border-radius: 12px;
  }

  .message-box {
    padding: 14px 16px;
    font-size: 14px;
  }

  .submit-btn {
    padding: 16px;
    font-size: 16px;
    border-radius: 12px;
    min-height: 52px;
  }

  .loading-spinner {
    width: 20px;
    height: 20px;
  }

  .toggle-mode {
    font-size: 14px;
    margin-bottom: 16px;
  }

  .back-btn {
    padding: 12px 20px;
    font-size: 14px;
    border-radius: 10px;
  }

  .back-btn svg {
    width: 16px;
    height: 16px;
  }
}

@media screen and (max-width: 480px) {
  .login-container {
    padding: 8px;
  }

  .login-content {
    padding: 24px 20px;
  }

  .logo {
    width: 48px;
    height: 48px;
  }

  .logo svg {
    width: 28px;
    height: 28px;
  }

  .app-name {
    font-size: 18px;
  }
}
</style>

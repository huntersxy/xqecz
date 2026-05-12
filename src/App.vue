<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useUserStore } from './stores/user'
import { onMounted, ref } from 'vue'
import axios from 'axios'

const userStore = useUserStore()
const isMobileMenuOpen = ref(false)
const isMobileUA = ref(false)

// 构建日期（构建时注入）
const buildDate = import.meta.env.VITE_BUILD_DATE || new Date().toISOString().split('T')[0]
// 当前年份
const currentYear = new Date().getFullYear()
// 是否显示ICP备案号（仅在xiey.work域名下显示）
const showICP = window.location.hostname.endsWith('xiey.work')

function isMobileDevice() {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile|mobile|CriOS/i.test(navigator.userAgent)
}

function toggleMobileMenu() {
  isMobileMenuOpen.value = !isMobileMenuOpen.value
}

function closeMobileMenu() {
  isMobileMenuOpen.value = false
}

async function loadRandomBackground() {
  try {
    const response = await axios.get('https://rimg.xiey.work/', {
      params: { t: Date.now() }
    })
    if (response.data && response.data.images && response.data.images.length > 0) {
      const randomIndex = Math.floor(Math.random() * response.data.images.length)
      const randomImage = response.data.images[randomIndex]
      const imageUrl = `https://rimg.xiey.work${randomImage}?t=${Date.now()}`
      const img = new Image()
      img.onload = () => {
        document.body.style.backgroundImage = `linear-gradient(135deg, rgba(245, 247, 250, 0.25) 0%, rgba(228, 232, 236, 0.25) 100%), url('${imageUrl}'), #f5f7fa`
      }
      img.onerror = () => {
        console.error('Failed to load random background, using default')
      }
      img.src = imageUrl
    }
  } catch (error) {
    console.error('Failed to load random background:', error)
  }
}

onMounted(() => {
  if (!userStore.isLoggedIn) {
    userStore.checkAuth()
  }
  loadRandomBackground()
  isMobileUA.value = isMobileDevice()
  if (isMobileUA.value) {
    document.body.classList.add('mobile-ua')
  }
})
</script>

<template>
  <header class="mac-header" :class="{ 'no-blur': isMobileUA }">
    <nav class="mac-nav">
      <div class="nav-left">
        <RouterLink to="/" class="nav-brand">
          <img src="/F775F3831CE0AAF6B17116666DD812F8.png" alt="小泉动漫二创站" class="brand-logo" />
        </RouterLink>
      </div>

      <div class="nav-center">
        <RouterLink to="/" class="nav-link" @click="closeMobileMenu">
          <svg class="link-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9z"/>
            <polyline points="9 22 9 12 15 12 15 22"/>
          </svg>
          <span>首页</span>
        </RouterLink>
        <RouterLink v-if="userStore.isLoggedIn" to="/upload" class="nav-link" @click="closeMobileMenu">
          <svg class="link-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          <span>上传内容</span>
        </RouterLink>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin" class="nav-link" @click="closeMobileMenu">
          <svg class="link-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>后台管理</span>
        </RouterLink>
        <RouterLink v-if="userStore.user?.is_admin || userStore.user?.IsAdmin" to="/admin/reports" class="nav-link" @click="closeMobileMenu">
          <svg class="link-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
          </svg>
          <span>举报管理</span>
        </RouterLink>
      </div>

      <div class="nav-right">
        <template v-if="userStore.isLoggedIn">
          <div class="user-info">
            <svg class="user-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <span class="user-name">{{ userStore.user?.username || userStore.user?.Username }}</span>
            <span v-if="userStore.user?.is_admin" class="admin-badge">管理员</span>
          </div>
          <button @click="userStore.logout(); closeMobileMenu()" class="mac-btn logout-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
            <span>退出</span>
          </button>
        </template>
        <template v-else>
          <RouterLink to="/login" class="mac-btn login-btn" @click="closeMobileMenu">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 2h-3a5 5 0 0 0-5 5v4H7v6h6v4a5 5 0 0 0 5 5h3"/>
              <polyline points="23 21 12 12 23 3"/>
            </svg>
            <span>登录</span>
          </RouterLink>
        </template>

        <button @click="toggleMobileMenu" class="mobile-menu-btn">
          <svg v-if="!isMobileMenuOpen" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"/>
            <line x1="3" y1="6" x2="21" y2="6"/>
            <line x1="3" y1="18" x2="21" y2="18"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </nav>
  </header>

  <main>
    <RouterView />
  </main>

  <div v-if="isMobileMenuOpen" class="mobile-menu-overlay" @click="closeMobileMenu">
    <div class="mobile-menu" @click.stop>
      <div class="mobile-menu-header">
        <span class="mobile-menu-title">菜单</span>
        <button @click="closeMobileMenu" class="mobile-menu-close">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div class="mobile-menu-content">
        <RouterLink to="/" class="mobile-nav-link" @click="closeMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9z"/>
            <polyline points="9 22 9 12 15 12 15 22"/>
          </svg>
          <span>首页</span>
        </RouterLink>

        <RouterLink v-if="userStore.isLoggedIn" to="/upload" class="mobile-nav-link" @click="closeMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19" />
            <line x1="5" y1="12" x2="19" y2="12" />
          </svg>
          <span>上传内容</span>
        </RouterLink>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin" class="mobile-nav-link" @click="closeMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>管理后台</span>
        </RouterLink>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin/reports" class="mobile-nav-link" @click="closeMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
          </svg>
          <span>举报管理</span>
        </RouterLink>
        <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="mobile-nav-link" @click="closeMobileMenu">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 16l-4-4m0 0L7 10m4 6l4-4m-4 6l4-4"/>
          </svg>
          <span>登录</span>
        </RouterLink>
        <div v-if="userStore.isLoggedIn" class="mobile-user-section">
          <div class="mobile-user-info">
            <svg class="mobile-user-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <div class="mobile-user-details">
              <span class="mobile-user-name">{{ userStore.user?.username || userStore.user?.Username }}</span>
              <span v-if="userStore.user?.is_admin" class="mobile-admin-badge">管理员</span>
            </div>
          </div>
          <button @click="userStore.logout(); closeMobileMenu()" class="mobile-logout-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
            <span>退出登录</span>
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- 页脚版权信息 -->
  <footer class="site-footer">
    <div class="footer-content">
      <div class="footer-left">
        <span class="copyright">© {{ currentYear }} 小泉动漫二创站</span>
        <span class="license">CC BY-NC 4.0 非商业使用</span>
        <span v-if="showICP" class="icp">桂ICP备2024031550号</span>
      </div>
      <div class="footer-right">
        <span class="build-info">构建时间: {{ buildDate }}</span>
      </div>
    </div>
  </footer>
</template>

<style scoped>
.mac-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: rgba(255, 255, 255, 0.98);
}

.mac-nav {
  max-width: 1400px;
  margin: 0 auto;
  padding: 8px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.nav-left {
  display: flex;
  align-items: center;
}

.nav-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
  color: #1a1a1a;
  font-weight: 600;
  font-size: 15px;
}

.brand-icon {
  width: 24px;
  height: 24px;
  color: #007aff;
}

.brand-logo {
  height: 32px;
  width: auto;
}

.nav-center {
  display: flex;
  gap: 8px;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  text-decoration: none;
  color: #333;
  border-radius: 8px;
  transition: all 0.2s ease;
  font-size: 14px;
}

.nav-link:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #007aff;
}

.nav-link.router-link-exact-active {
  background: rgba(0, 122, 255, 0.1);
  color: #007aff;
}

.link-icon {
  width: 16px;
  height: 16px;
}

.nav-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: rgba(0, 0, 0, 0.04);
  border-radius: 20px;
}

.user-icon {
  width: 18px;
  height: 18px;
  color: #666;
}

.user-name {
  font-size: 13px;
  color: #333;
}

.admin-badge {
  font-size: 10px;
  padding: 2px 6px;
  background: linear-gradient(135deg, #ff6b6b, #ee5a5a);
  color: white;
  border-radius: 10px;
  font-weight: 500;
}

.logout-btn,
.login-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  padding: 6px 14px;
}

.logout-btn svg,
.login-btn svg {
  width: 14px;
  height: 14px;
}

.mobile-menu-btn {
  display: none;
  width: 40px;
  height: 40px;
  justify-content: center;
  align-items: center;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #333;
  transition: background 0.2s ease;
}

.mobile-menu-btn:hover {
  background: rgba(0, 0, 0, 0.06);
}

.mobile-menu-btn svg {
  width: 24px;
  height: 24px;
}

.mobile-menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.mobile-menu {
  width: 85%;
  max-width: 320px;
  background: white;
  height: 100%;
  display: flex;
  flex-direction: column;
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}

.mobile-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.mobile-menu-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.mobile-menu-close {
  width: 36px;
  height: 36px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: rgba(0, 0, 0, 0.05);
  border: none;
  border-radius: 8px;
  cursor: pointer;
  color: #666;
}

.mobile-menu-close svg {
  width: 20px;
  height: 20px;
}

.mobile-menu-content {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}

.mobile-nav-link {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  text-decoration: none;
  color: #1a1a1a;
  border-radius: 12px;
  transition: all 0.2s ease;
  font-size: 16px;
  margin-bottom: 6px;
  background: rgba(255, 255, 255, 0.9);
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.mobile-nav-link:hover {
  background: rgba(0, 122, 255, 0.06);
}

.mobile-nav-link.router-link-exact-active {
  background: linear-gradient(135deg, rgba(0, 122, 255, 0.15) 0%, rgba(0, 122, 255, 0.08) 100%);
  color: #007aff;
  border-color: rgba(0, 122, 255, 0.2);
}

.mobile-nav-link svg {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.mobile-user-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.mobile-user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 10px;
  margin-bottom: 8px;
}

.mobile-user-icon {
  width: 24px;
  height: 24px;
  color: #007aff;
}

.mobile-user-details {
  flex: 1;
}

.mobile-user-name {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
}

.mobile-admin-badge {
  display: inline-block;
  font-size: 11px;
  padding: 2px 8px;
  background: linear-gradient(135deg, #ff6b6b, #ee5a5a);
  color: white;
  border-radius: 10px;
  font-weight: 500;
  margin-top: 4px;
}

.mobile-logout-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 14px;
  background: rgba(220, 38, 38, 0.1);
  border: none;
  border-radius: 10px;
  color: #dc2626;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s ease;
}

.mobile-logout-btn:hover {
  background: rgba(220, 38, 38, 0.15);
}

.mobile-logout-btn svg {
  width: 18px;
  height: 18px;
}

/* 页脚样式 */
.site-footer {
  margin-top: 40px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.85);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  backdrop-filter: blur(10px);
}

.footer-content {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.copyright {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}

.license {
  font-size: 13px;
  color: #999;
  padding: 4px 10px;
  background: rgba(0, 0, 0, 0.03);
  border-radius: 4px;
}

.icp {
  font-size: 13px;
  color: #999;
}

.footer-right {
  font-size: 13px;
  color: #999;
}

.build-info {
  font-family: monospace;
}

main {
  padding: 20px;
  min-height: calc(100vh - 200px);
}

@media screen and (max-width: 768px) {
  .mac-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
  }

  .mac-nav {
    padding: 8px 12px;
  }

  .nav-center {
    display: none;
  }

  .nav-right .user-info,
  .nav-right .logout-btn,
  .nav-right .login-btn {
    display: none;
  }

  .mobile-menu-btn {
    display: flex;
  }

  .brand-logo {
    height: 28px;
  }

  main {
    padding: 12px;
    padding-top: 60px;
    min-height: calc(100vh - 180px);
  }

  .site-footer {
    margin-top: 20px;
    padding: 16px;
  }

  .footer-content {
    flex-direction: column;
    text-align: center;
    gap: 8px;
  }

  .footer-left {
    flex-direction: column;
    gap: 8px;
  }

  .copyright {
    font-size: 13px;
  }

  .license {
    font-size: 12px;
  }

  .footer-right {
    font-size: 12px;
  }
}

@media screen and (min-width: 769px) {
  .mobile-menu-overlay {
    display: none;
  }
}
</style>

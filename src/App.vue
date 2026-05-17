<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
import { useUserStore } from './stores/user'
import { useThemeStore } from './stores/theme'
import { onMounted, ref, watch, onUnmounted } from 'vue'
import logoImg from '@/assets/logo.webp'

const userStore = useUserStore()
const themeStore = useThemeStore()
const isMobileMenuOpen = ref(false)
const isMobileUA = ref(false)
const navExpanded = ref(false)
let autoCollapseTimer: ReturnType<typeof setTimeout> | null = null

const buildDate = import.meta.env.VITE_BUILD_DATE || new Date().toISOString().split('T')[0]
const currentYear = new Date().getFullYear()
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

function resetAutoCollapse() {
  if (autoCollapseTimer) clearTimeout(autoCollapseTimer)
  autoCollapseTimer = setTimeout(() => {
    navExpanded.value = false
  }, 5000)
}

function toggleNav() {
  navExpanded.value = !navExpanded.value
  if (navExpanded.value) resetAutoCollapse()
}

watch(() => themeStore.currentTheme, (newTheme) => {
  if (newTheme !== 'immersiveMusic') {
    navExpanded.value = false
    if (autoCollapseTimer) clearTimeout(autoCollapseTimer)
  }
})

onMounted(() => {
  if (!userStore.isLoggedIn) {
    userStore.checkAuth()
  }
  isMobileUA.value = isMobileDevice()
  if (isMobileUA.value) {
    document.body.classList.add('mobile-ua')
  }
})

onUnmounted(() => {
  if (autoCollapseTimer) clearTimeout(autoCollapseTimer)
})
</script>

<template>
  <!-- 沉浸音乐主题悬浮按钮 -->
  <div v-if="themeStore.currentTheme === 'immersiveMusic' && !navExpanded" 
       class="floating-menu-btn fixed top-6 right-6 z-[999999] w-16 h-16 rounded-full bg-white border-4 border-blue-500 cursor-pointer flex items-center justify-center hover:scale-110 hover:shadow-2xl transition-all duration-300 shadow-xl"
       @click="toggleNav"
       style="pointer-events: auto !important; opacity: 1 !important; display: flex !important;">
    <svg class="w-8 h-8 text-gray-800" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
      <line x1="3" y1="6" x2="21" y2="6"/>
      <line x1="3" y1="12" x2="21" y2="12"/>
      <line x1="3" y1="18" x2="21" y2="18"/>
    </svg>
  </div>

  <header v-show="themeStore.currentTheme !== 'immersiveMusic' || navExpanded" 
           class="sticky top-0 z-100 bg-[var(--theme-surface)] liquid-glass-nav transition-all duration-300" 
           :class="{ '!backdrop-blur-0': isMobileUA, '!fixed top-4 left-4 right-4 max-w-[1400px] mx-auto rounded-xl shadow-2xl !z-[99998]': themeStore.currentTheme === 'immersiveMusic' }">
    <nav class="max-w-[1400px] mx-auto px-4 py-2 flex justify-between items-center">
      <div class="flex items-center gap-3">
        <RouterLink to="/" class="flex items-center gap-2 no-underline font-semibold text-[15px] text-[var(--theme-text)]">
          <img :src="logoImg" alt="小泉动漫二创站" class="h-8 w-auto" />
        </RouterLink>
        <button v-if="themeStore.currentTheme === 'immersiveMusic'" 
                @click="toggleNav" 
                class="w-8 h-8 flex items-center justify-center rounded-full hover:bg-[var(--theme-hover-bg)] transition-colors">
          <svg class="w-5 h-5 theme-text-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>

      <div class="flex gap-2">
        <RouterLink to="/" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-1.5 px-4 py-2 no-underline rounded-lg text-[14px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)]">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9z"/>
            <polyline points="9 22 9 12 15 12 15 22"/>
          </svg>
          <span>首页</span>
        </RouterLink>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-1.5 px-4 py-2 no-underline rounded-lg text-[14px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)]">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>后台管理</span>
        </RouterLink>
        <RouterLink to="/theme" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-1.5 px-4 py-2 no-underline rounded-lg text-[14px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)]">
          <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M12 1v6m0 6v6m4.28-13.34a4 4 0 0 1 0 5.66m-4.28 5.66a4 4 0 0 0 0 5.66M21 12H3"></path>
          </svg>
          <span>主题设置</span>
        </RouterLink>
      </div>

      <div class="flex items-center gap-3">
        <template v-if="userStore.isLoggedIn">
          <div class="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--theme-hover-bg)]">
            <svg class="w-[18px] h-[18px] text-[var(--theme-text-secondary)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <span class="text-[13px] text-[var(--theme-text)]">{{ userStore.user?.username }}</span>
            <span v-if="userStore.user?.is_admin" class="text-[10px] px-1.5 py-0.5 bg-gradient-to-br from-[#ff6b6b] to-[#ee5a5a] text-white rounded-lg font-medium">管理员</span>
          </div>
          <button @click="userStore.logout(); closeMobileMenu()" class="flex items-center gap-1.5 text-[13px] px-3.5 py-1.5 text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)]">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
              <polyline points="16 17 21 12 16 7"/>
              <line x1="21" y1="12" x2="9" y2="12"/>
            </svg>
            <span>退出</span>
          </button>
        </template>
        <template v-else>
          <RouterLink to="/login" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-1.5 text-[13px] px-3.5 py-1.5 text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] hover:text-[var(--theme-primary)]">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 2h-3a5 5 0 0 0-5 5v4H7v6h6v4a5 5 0 0 0 5 5h3"/>
              <polyline points="23 21 12 12 23 3"/>
            </svg>
            <span>登录</span>
          </RouterLink>
        </template>

        <button @click="toggleMobileMenu" class="hidden w-10 h-10 justify-center items-center bg-transparent border-none rounded-lg cursor-pointer text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)]">
          <svg v-if="!isMobileMenuOpen" class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"/>
            <line x1="3" y1="6" x2="21" y2="6"/>
            <line x1="3" y1="18" x2="21" y2="18"/>
          </svg>
          <svg v-else class="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
    </nav>
  </header>

  <main class="px-5 py-5 min-h-[calc(100vh-200px)]">
    <Suspense>
      <RouterView />
      <template #fallback>
        <div class="flex flex-col items-center justify-center py-[60px] px-5">
          <div class="w-10 h-10 border-3 border-[color-mix(in_srgb,var(--theme-primary)_20%,transparent)] border-t-[var(--theme-primary)] rounded-full animate-spin"></div>
          <p class="mt-4 text-[var(--theme-text-secondary)] text-[14px]">加载中...</p>
        </div>
      </template>
    </Suspense>
  </main>

  <div v-if="isMobileMenuOpen" class="fixed inset-0 bg-black/50 z-[1000] flex justify-end animate-[fadeIn_0.2s_ease]" @click="closeMobileMenu">
    <div class="w-[85%] max-w-[320px] bg-[var(--theme-surface)] h-full flex flex-col animate-[slideIn_0.3s_ease]" @click.stop>
      <div class="flex justify-between items-center px-4 py-5 border-b border-[var(--theme-card-border)]">
        <span class="text-[16px] font-semibold text-[var(--theme-text)]">菜单</span>
        <button @click="closeMobileMenu" class="w-9 h-9 flex justify-center items-center bg-[var(--theme-hover-bg)] border-none rounded-lg cursor-pointer text-[var(--theme-text-secondary)]">
          <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div class="flex-1 p-3 overflow-y-auto">
        <RouterLink to="/" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-4 px-4 py-4 no-underline rounded-xl text-[16px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] mb-1.5 bg-[var(--theme-surface)] border border-[var(--theme-card-border)]">
          <svg class="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9z"/>
            <polyline points="9 22 9 12 15 12 15 22"/>
          </svg>
          <span>首页</span>
        </RouterLink>

        <RouterLink v-if="userStore.isLoggedIn" to="/admin" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-4 px-4 py-4 no-underline rounded-xl text-[16px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] mb-1.5 bg-[var(--theme-surface)] border border-[var(--theme-card-border)]">
          <svg class="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 0 0 2.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 0 0 1.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 0 0-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 0 0-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 0 0-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 0 0-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 0 0 1.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>管理后台</span>
        </RouterLink>
        <RouterLink to="/theme" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-4 px-4 py-4 no-underline rounded-xl text-[16px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] mb-1.5 bg-[var(--theme-surface)] border border-[var(--theme-card-border)]">
          <svg class="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"></circle>
            <path d="M12 1v6m0 6v6m4.28-13.34a4 4 0 0 1 0 5.66m-4.28 5.66a4 4 0 0 0 0 5.66M21 12H3"></path>
          </svg>
          <span>主题设置</span>
        </RouterLink>
        <RouterLink v-if="!userStore.isLoggedIn" to="/login" @click.native="resetAutoCollapse" @click="closeMobileMenu" class="flex items-center gap-4 px-4 py-4 no-underline rounded-xl text-[16px] text-[var(--theme-text)] transition-all duration-200 hover:bg-[var(--theme-hover-bg)] mb-1.5 bg-[var(--theme-surface)] border border-[var(--theme-card-border)]">
          <svg class="w-5 h-5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 16l-4-4m0 0L7 10m4 6l4-4"/>
          </svg>
          <span>登录</span>
        </RouterLink>
        <div v-if="userStore.isLoggedIn" class="mt-4 pt-4 border-t border-[var(--theme-card-border)]">
          <div class="flex items-center gap-3 px-4 py-3 rounded-xl bg-[var(--theme-hover-bg)] mb-2">
            <svg class="w-6 h-6 text-[var(--theme-primary)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
              <circle cx="12" cy="7" r="4"/>
            </svg>
            <div class="flex-1">
              <span class="block text-[15px] font-semibold text-[var(--theme-text)]">{{ userStore.user?.username }}</span>
              <span v-if="userStore.user?.is_admin" class="inline-block text-[11px] px-2 py-0.5 bg-gradient-to-br from-[#ff6b6b] to-[#ee5a5a] text-white rounded-lg font-medium mt-1">管理员</span>
            </div>
          </div>
          <button @click="userStore.logout(); closeMobileMenu()" class="flex items-center justify-center gap-2 w-full py-3.5 bg-[color-mix(in_srgb,var(--theme-danger)_10%,transparent)] border-none rounded-xl text-[var(--theme-danger)] text-[15px] font-medium cursor-pointer transition-all duration-200 hover:bg-[color-mix(in_srgb,var(--theme-danger)_15%,transparent)]">
            <svg class="w-[18px] h-[18px]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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

  <footer v-if="themeStore.currentTheme !== 'immersiveMusic'" class="mt-10 py-5 bg-[var(--theme-surface)] border-t border-[var(--theme-card-border)] backdrop-blur-[10px]">
    <div class="max-w-[1400px] mx-auto flex justify-between items-center flex-wrap gap-3 px-4">
      <div class="flex items-center gap-4 flex-wrap">
        <span class="text-[14px] text-[var(--theme-text-secondary)] font-medium">© {{ currentYear }} 小泉动漫二创站</span>
        <span class="text-[13px] text-[var(--theme-text-secondary)] px-2.5 py-1 bg-[var(--theme-hover-bg)] rounded">CC BY-NC 4.0 非商业使用</span>
        <span v-if="showICP" class="text-[13px] text-[var(--theme-text-secondary)]">桂 ICP 备 2024031550 号</span>
      </div>
      <div class="text-[13px] text-[var(--theme-text-secondary)] font-mono">构建时间：{{ buildDate }}</div>
    </div>
  </footer>
</template>

<style scoped>
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from { transform: translateX(100%); }
  to { transform: translateX(0); }
}

html.theme-liquidGlass header.liquid-glass-nav {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.8), rgba(255, 255, 255, 0.6)) !important;
  backdrop-filter: blur(16px) !important;
  -webkit-backdrop-filter: blur(16px) !important;
  border-bottom: 1px solid rgba(0, 0, 0, 0.1);
}

html.theme-liquidGlass footer {
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.75), rgba(255, 255, 255, 0.55)) !important;
  backdrop-filter: blur(16px) !important;
  -webkit-backdrop-filter: blur(16px) !important;
  border-top: 1px solid rgba(0, 0, 0, 0.1);
}

html.theme-liquidGlass footer .text-\[var\(--theme-text-secondary\)\] {
  color: #333333 !important;
}

@media screen and (max-width: 768px) {
  header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
  }

  nav {
    padding: 8px 12px;
  }

  nav > div:nth-child(2) {
    display: none;
  }

  nav > div:nth-child(3) > :not(:last-child) {
    display: none;
  }

  nav > div:nth-child(3) > :last-child {
    display: flex;
  }

  nav > div:nth-child(1) img {
    height: 28px;
  }

  main {
    padding: 12px;
    padding-top: 60px;
    min-height: calc(100vh - 180px);
  }

  footer {
    margin-top: 20px;
    padding: 16px;
  }

  footer > div {
    flex-direction: column;
    text-align: center;
    gap: 8px;
  }

  footer > div > div:first-child {
    flex-direction: column;
    gap: 8px;
  }
}

@media screen and (min-width: 769px) {
  .fixed.floating-menu-btn {
    display: none;
  }
}
</style>

<script setup lang="ts">
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useUserStore } from './stores/user'
import { useThemeStore } from './stores/theme'
import { useGlobalSearch } from './composables/useGlobalSearch'
import { Toaster } from 'vue-sonner'
import logoImg from '@/assets/logo.webp'
import {
  UserOutlined,
  LogoutOutlined,
  HomeOutlined,
  SettingOutlined,
  BgColorsOutlined,
  LoginOutlined,
  MenuOutlined,
  CloseOutlined,
  CloudUploadOutlined,
} from '@ant-design/icons-vue'
import type { MenuProps } from 'ant-design-vue'
import { theme as antTheme } from 'ant-design-vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import { getAvatarUrl } from '@/utils'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()
const isMobileMenuOpen = ref(false)
const isMobileUA = ref(false)

// 全局搜索：把输入框和瀑布流查询接起来
const { searchKeyword, triggerSearch } = useGlobalSearch()
const searchInput = ref(searchKeyword.value)
function onGlobalSearch() {
  searchKeyword.value = searchInput.value.trim()
  triggerSearch()
}

const userAvatarUrl = computed(() => userStore.user?.email ? getAvatarUrl(userStore.user.email) : '')

// 全局 antd 暗色适配
const antThemeConfig = computed(() => ({
  algorithm: themeStore.mode === 'dark' ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
  token: { colorPrimary: '#6366f1', borderRadius: 6 },
}))

const buildDate = import.meta.env.VITE_BUILD_DATE || new Date().toISOString().split('T')[0]
const currentYear = new Date().getFullYear()
const showICP = globalThis.location.hostname.endsWith('xiey.work')

function isMobileDevice() {
  return /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini|Mobile|mobile|CriOS/i.test(
    navigator.userAgent,
  )
}

const navItems = computed<MenuProps['items']>(() => {
  const items: MenuProps['items'] = [
    { key: '/', icon: () => h(HomeOutlined), label: '首页' },
    { key: '/quick-upload', icon: () => h(CloudUploadOutlined), label: '快速上传' },
  ]
  if (userStore.isLoggedIn) {
    items.push({ key: '/admin', icon: () => h(SettingOutlined), label: '后台管理' })
  }
  if (!userStore.isLoggedIn) {
    items.push({ key: '/login', icon: () => h(LoginOutlined), label: '登录' })
  }
  return items
})

const selectedKeys = computed(() => {
  const p = route.path
  const items = navItems.value || []
  for (let i = items.length - 1; i >= 0; i--) {
    const item = items[i]
    if (item && 'key' in item) {
      const key = item.key as string
      if (p === key || p.startsWith(key + '/')) return [key]
    }
  }
  return ['/']
})

function onNavClick(info: { key: string | number }) {
  router.push(String(info.key))
  isMobileMenuOpen.value = false
}

function handleLogout() {
  userStore.logout()
  isMobileMenuOpen.value = false
}

onMounted(() => {
  if (!userStore.isLoggedIn) userStore.checkAuth()
  isMobileUA.value = isMobileDevice()
  if (isMobileUA.value) document.body.classList.add('mobile-ua')
})
</script>

<template>
  <a-config-provider :theme="antThemeConfig">
  <header class="app-header" role="banner">
    <div class="app-header-inner">
      <RouterLink to="/" class="app-logo" @click="isMobileMenuOpen = false" aria-label="返回首页">
        <img :src="logoImg" alt="小泉动漫二创站" class="app-logo-img" />
      </RouterLink>

      <a-menu
        mode="horizontal"
        :selected-keys="selectedKeys"
        :items="navItems"
        class="app-nav-menu"
        @click="onNavClick"
        role="navigation"
        aria-label="主导航"
      />

      <!-- 全局搜索框（首页瀑布流使用） -->
      <a-input-search
        v-if="route.path === '/'"
        v-model:value="searchInput"
        placeholder="搜索作品/标签/作者..."
        class="app-search-input"
        @search="onGlobalSearch"
      />

      <div class="app-header-right">
        <!-- 暗/亮模式切换 -->
        <a-dropdown :trigger="['click']" placement="bottomRight">
          <a-button type="text" class="app-theme-btn" aria-label="切换显示模式">
            <BgColorsOutlined />
          </a-button>
          <template #overlay>
            <div class="app-theme-dropdown">
              <div class="app-theme-label">显示模式</div>
              <div class="app-theme-options">
                <button
                  :class="['app-theme-option', { active: themeStore.mode === 'light' }]"
                  @click="themeStore.setMode('light')"
                >日间</button>
                <button
                  :class="['app-theme-option', { active: themeStore.mode === 'dark' }]"
                  @click="themeStore.setMode('dark')"
                >暗色</button>
              </div>
            </div>
          </template>
        </a-dropdown>

        <template v-if="userStore.isLoggedIn">
          <a-dropdown :trigger="['click']">
            <div
              class="app-user-trigger"
              role="button"
              tabindex="0"
              aria-label="用户菜单"
              aria-haspopup="true"
              @keydown.enter.prevent
              @keydown.space.prevent
            >
              <a-avatar size="small" :src="userAvatarUrl">
                <template v-if="!userAvatarUrl" #icon><UserOutlined /></template>
              </a-avatar>
              <span class="app-username">{{ userStore.user?.username }}</span>
              <a-tag
                v-if="userStore.user?.is_admin"
                color="red"
                :bordered="false"
                style="font-size: 11px; padding: 0 4px; margin-left: 2px"
                >管理员</a-tag
              >
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item key="logout" @click="handleLogout">
                  <LogoutOutlined /> 退出登录
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </template>
        <template v-else>
          <RouterLink
            to="/login"
            class="app-login-link"
            @click="isMobileMenuOpen = false"
            aria-label="登录"
          >
            <LoginOutlined /> 登录
          </RouterLink>
        </template>

        <a-button
          v-if="!route.path.startsWith('/admin')"
          class="app-mobile-menu-btn"
          type="text"
          @click="isMobileMenuOpen = !isMobileMenuOpen"
          :aria-label="isMobileMenuOpen ? '关闭菜单' : '打开菜单'"
          :aria-expanded="isMobileMenuOpen"
          aria-controls="mobile-menu-drawer"
        >
          <CloseOutlined v-if="isMobileMenuOpen" />
          <MenuOutlined v-else />
        </a-button>
      </div>
    </div>
  </header>

  <a-drawer
    v-if="!route.path.startsWith('/admin')"
    :open="isMobileMenuOpen"
    placement="right"
    :closable="false"
    @close="isMobileMenuOpen = false"
    :body-style="{ padding: 0 }"
    :width="280"
    id="mobile-menu-drawer"
    aria-label="移动端导航菜单"
  >
    <div class="mobile-drawer-header">
      <span class="mobile-drawer-title">菜单</span>
      <a-button type="text" size="small" @click="isMobileMenuOpen = false" aria-label="关闭菜单"
        ><CloseOutlined
      /></a-button>
    </div>
    <div class="mobile-drawer-user" v-if="userStore.isLoggedIn">
      <a-avatar style="background-color: #1677ff"
        ><template #icon><UserOutlined /></template
      ></a-avatar>
      <div class="mobile-drawer-user-info">
        <span class="mobile-drawer-username">{{ userStore.user?.username }}</span>
        <a-tag v-if="userStore.user?.is_admin" color="red" size="small">管理员</a-tag>
      </div>
    </div>
    <a-menu
      mode="inline"
      :selected-keys="selectedKeys"
      :items="navItems"
      @click="onNavClick"
      style="border: none"
    />
    <div v-if="userStore.isLoggedIn" class="mobile-drawer-footer">
      <a-button danger block @click="handleLogout" aria-label="退出登录"
        ><LogoutOutlined /> 退出登录</a-button
      >
    </div>
  </a-drawer>

  <main :class="route.path.startsWith('/admin') ? 'admin-main' : ''" role="main">
    <ErrorBoundary>
      <RouterView v-slot="{ Component }">
        <Transition name="route-fade" mode="out-in">
          <Suspense>
            <component :is="Component" />
            <template #fallback>
              <output
                class="flex flex-col items-center justify-center py-[60px] px-5"
                aria-live="polite"
              >
                <div
                  class="w-10 h-10 border-3 border-[color-mix(in_srgb,var(--theme-primary)_20%,transparent)] border-t-[var(--theme-primary)] rounded-full animate-spin"
                  aria-hidden="true"
                ></div>
                <p class="mt-4 text-[var(--theme-text-secondary)] text-[14px]">加载中...</p>
              </output>
            </template>
          </Suspense>
        </Transition>
      </RouterView>
    </ErrorBoundary>
  </main>

  <Toaster position="top-center" />
  <ConfirmDialog />

  <footer v-if="!route.path.startsWith('/admin')" class="app-footer" role="contentinfo">
    <div class="app-footer-inner">
      <div class="app-footer-left">
        <span class="app-footer-text">© {{ currentYear }} 小泉动漫二创站</span>
        <span class="app-footer-badge">CC BY-NC 4.0 非商业使用</span>
        <span v-if="showICP" class="app-footer-text">桂 ICP 备 2024031550 号</span>
      </div>
      <div class="app-footer-text" style="font-family: monospace">构建时间：{{ buildDate }}</div>
    </div>
  </footer>
  </a-config-provider>
</template>

<style scoped>
.app-header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--theme-surface);
  padding: 0;
  height: auto;
  line-height: normal;
  border-bottom: 1px solid var(--theme-card-border);
}

.app-header-inner {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 16px;
  display: flex;
  align-items: center;
  height: 48px;
}

.app-logo {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-right: 24px;
}

.app-logo-img {
  height: 32px;
  width: auto;
}

.app-nav-menu {
  flex: 1;
  min-width: 0;
  border-bottom: none !important;
  line-height: 46px;
  background: transparent;
}

:deep(.app-nav-menu.ant-menu) {
  overflow: visible;
}

:deep(.app-nav-menu.ant-menu-overflow) {
  overflow: visible !important;
}

:deep(.app-nav-menu.ant-menu-overflow-item) {
  overflow: visible !important;
}

:deep(.app-nav-menu .ant-menu-overflow-item-rest) {
  display: none !important;
}

:deep(.app-nav-menu .ant-menu-item) {
  color: var(--theme-text);
  white-space: nowrap;
}

:deep(.app-nav-menu .ant-menu-item:hover) {
  color: var(--theme-primary);
}

:deep(.app-nav-menu .ant-menu-item-selected) {
  color: var(--theme-primary);
}

.app-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.app-search-input {
  width: 280px;
}
.app-search-input :deep(.ant-input-search-button) {
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-theme-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  color: var(--theme-text);
}

.app-theme-btn:hover {
  color: var(--theme-primary);
  background: var(--theme-hover-bg);
}

.app-theme-dropdown {
  background: var(--theme-surface);
  border: 1px solid var(--theme-card-border);
  border-radius: 8px;
  padding: 12px;
  min-width: 200px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.app-theme-label {
  font-size: 12px;
  color: var(--theme-text-secondary);
  margin-bottom: 8px;
}

.app-theme-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.app-theme-option {
  padding: 4px 12px;
  font-size: 13px;
  border: 1px solid var(--theme-card-border);
  border-radius: 16px;
  background: transparent;
  color: var(--theme-text);
  cursor: pointer;
  transition: background 0.2s, color 0.2s, border-color 0.2s;
}

.app-theme-option:hover {
  border-color: var(--theme-primary);
  color: var(--theme-primary);
}

.app-theme-option.active {
  background: var(--theme-primary);
  border-color: var(--theme-primary);
  color: var(--theme-on-primary);
}

.app-user-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  background: var(--theme-hover-bg);
  cursor: pointer;
  transition: background 0.2s;
}

.app-user-trigger:hover {
  background: var(--theme-card-border);
}

.app-username {
  font-size: 13px;
  color: var(--theme-text);
}

.app-login-link {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--theme-text);
  text-decoration: none;
  padding: 4px 10px;
  border-radius: 6px;
  transition: background 0.2s;
}

.app-login-link:hover {
  background: var(--theme-hover-bg);
}

.app-mobile-menu-btn {
  display: none;
}

.mobile-drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.mobile-drawer-title {
  font-size: 16px;
  font-weight: 600;
}

.mobile-drawer-user {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.mobile-drawer-user-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mobile-drawer-username {
  font-size: 15px;
  font-weight: 600;
}

.mobile-drawer-footer {
  padding: 16px;
  border-top: 1px solid #f0f0f0;
}

:deep(.mobile-drawer-footer .ant-btn) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  vertical-align: middle;
}

:deep(.mobile-drawer-footer .anticon) {
  display: inline-flex;
  align-items: center;
  vertical-align: -0.125em;
}

.app-footer {
  margin-top: 40px;
  padding: 20px 0;
  background: var(--theme-surface);
  border-top: 1px solid var(--theme-card-border);
}

.app-footer-inner {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding: 0 16px;
}

.app-footer-left {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.app-footer-text {
  font-size: 13px;
  color: var(--theme-text-secondary);
}

.app-footer-badge {
  font-size: 13px;
  color: var(--theme-text-secondary);
  padding: 2px 10px;
  background: var(--theme-hover-bg);
  border-radius: 4px;
}



@media (max-width: 768px) {
  .app-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
  }

  .app-nav-menu {
    display: none !important;
  }

  .app-header-right {
    flex: 1;
    justify-content: flex-end;
    gap: 4px;
  }

  .app-mobile-menu-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    padding: 0;
  }

  :deep(.app-mobile-menu-btn .anticon) {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 18px;
  }

  .app-logo-img {
    height: 28px;
  }

  .app-logo {
    display: none;
  }

  .app-search-input {
    flex: 1;
    width: auto;
    min-width: 0;
    margin-right: 8px;
  }

  .app-username {
    display: none;
  }

  .app-user-trigger {
    padding: 4px 8px;
  }

  main {
    min-height: calc(100vh - 180px);
  }

  main.admin-main {
    min-height: 0;
    padding-top: 48px;
  }

  .app-footer {
    margin-top: 20px;
    padding: 16px;
  }

  .app-footer-inner {
    flex-direction: column;
    text-align: center;
  }

  .app-footer-left {
    flex-direction: column;
    gap: 8px;
  }
}

/* 路由切换淡入淡出 */
.route-fade-enter-active,
.route-fade-leave-active {
  transition: opacity 0.2s ease;
}

.route-fade-enter-from,
.route-fade-leave-to {
  opacity: 0;
}
</style>

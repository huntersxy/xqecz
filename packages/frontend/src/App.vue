<script setup lang="ts">
import { RouterLink, RouterView, useRoute, useRouter } from 'vue-router'
import { useUserStore } from './stores/user'
import { useThemeStore } from './stores/theme'
import { useGlobalSearch } from './composables/useGlobalSearch'
import { Toaster } from 'vue-sonner'
import logoImg from '@/assets/logo.webp'
import {
  Layout,
  LayoutHeader,
  Menu,
  MenuItem,
  Dropdown,
  Doption,
  Avatar,
  Button,
  Tag,
  InputSearch,
  Drawer,
  Space,
  Divider,
  TypographyText,
  TypographyTitle,
} from '@arco-design/web-vue'
// Arco 图标（太阳/月亮，用于暗色切换）
import {
  IconSun,
  IconMoon,
  IconHome,
  IconSettings,
  IconUpload,
  IconUser,
  IconPoweroff,
  IconMenu,
  IconClose,
} from '@arco-design/web-vue/es/icon'
import type { Component } from 'vue'
import ErrorBoundary from './components/ErrorBoundary.vue'
import { getAvatarUrl } from '@/utils'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const themeStore = useThemeStore()
const isMobileMenuOpen = ref(false)

// 全局搜索：把输入框和瀑布流查询接起来
const { searchKeyword, triggerSearch } = useGlobalSearch()
const searchInput = ref(searchKeyword.value)
function onGlobalSearch() {
  searchKeyword.value = searchInput.value.trim()
  triggerSearch()
}

const userAvatarUrl = computed(() => userStore.user?.email ? getAvatarUrl(userStore.user.email) : '')

const buildDate = import.meta.env.VITE_BUILD_DATE || new Date().toISOString().split('T')[0]
const currentYear = new Date().getFullYear()
const showICP = globalThis.location.hostname.endsWith('xiey.work')

interface NavItem {
  key: string
  icon: Component
  label: string
}

const navItems = computed<NavItem[]>(() => {
  const items: NavItem[] = [
    { key: '/', icon: IconHome, label: '首页' },
    { key: '/quick-upload', icon: IconUpload, label: '快速上传' },
  ]
  if (userStore.isLoggedIn) {
    items.push({ key: '/admin', icon: IconSettings, label: '后台管理' })
  }
  if (!userStore.isLoggedIn) {
    items.push({ key: '/login', icon: IconUser, label: '登录' })
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

function onNavClick(key: string | number) {
  router.push(String(key))
  isMobileMenuOpen.value = false
}

function goLogin() {
  isMobileMenuOpen.value = false
  router.push('/login')
}

function handleLogout() {
  userStore.logout()
  isMobileMenuOpen.value = false
}

// 响应式断点：≤768px 视为移动端（与 Arco 断点 sm=768 对齐）。
// 移动端不渲染横向 Menu，改为汉堡按钮打开 Drawer，从根本上避免横向菜单溢出「…」抖动。
const isMobile = ref(false)
function updateBreakpoint() {
  isMobile.value = window.matchMedia('(max-width: 768px)').matches
}

onMounted(() => {
  if (!userStore.isLoggedIn) userStore.checkAuth()
  updateBreakpoint()
  window.addEventListener('resize', updateBreakpoint)
})
onBeforeUnmount(() => window.removeEventListener('resize', updateBreakpoint))
</script>

<template>
  <Layout class="app-layout">
      <LayoutHeader class="app-header">
        <div class="app-header-inner">
          <RouterLink v-if="!isMobile" to="/" class="app-logo" @click="isMobileMenuOpen = false" aria-label="返回首页">
            <img :src="logoImg" alt="小泉动漫二创站" class="app-logo-img" />
          </RouterLink>

          <!-- 桌面端：用 Space + text Button 做导航，避免 Arco 横向 Menu 的「…」折叠 -->
          <Space
            v-if="!isMobile"
            class="app-nav-space"
            :size="4"
            align="center"
            :wrap="false"
          >
            <Button
              v-for="item in navItems"
              :key="item.key"
              type="text"
              :class="['app-nav-item', { 'app-nav-item-active': selectedKeys.includes(item.key) }]"
              @click="onNavClick(item.key)"
            >
              <template #icon><component :is="item.icon" /></template>
              {{ item.label }}
            </Button>
          </Space>

          <!-- 移动端首页：搜索框占满中间区域 -->
          <InputSearch
            v-else-if="isMobile && route.path === '/'"
            v-model="searchInput"
            placeholder="搜索..."
            class="app-search-input-mobile"
            @search="onGlobalSearch"
            allow-clear
          />

          <!-- 移动端非首页：弹性占位把右侧操作区推到最右 -->
          <div v-else class="app-header-spacer" />

          <!-- 桌面端首页：横向菜单右侧的搜索框（固定宽度，百分比+上限） -->
          <InputSearch
            v-if="!isMobile && route.path === '/'"
            v-model="searchInput"
            placeholder="搜索作品/标签/作者..."
            class="app-search-input"
            @search="onGlobalSearch"
            allow-clear
          />

          <Space class="app-header-right" :size="12" align="center">
            <!-- 暗/亮模式直接切换：暗色时显示太阳（点击回到日间），日间显示月亮（点击进入暗色） -->
            <Button
              type="text"
              class="app-theme-btn"
              :aria-label="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到暗色模式'"
              @click="themeStore.toggleMode()"
            >
              <IconSun v-if="themeStore.mode === 'dark'" />
              <IconMoon v-else />
            </Button>

            <template v-if="userStore.isLoggedIn">
              <Dropdown trigger="click">
                <div
                  class="app-user-trigger"
                  role="button"
                  tabindex="0"
                  aria-label="用户菜单"
                  aria-haspopup="true"
                  @keydown.enter.prevent
                  @keydown.space.prevent
                >
                  <Avatar :size="28" :image-url="userAvatarUrl" class="app-avatar-primary">
                    <IconUser v-if="!userAvatarUrl" />
                  </Avatar>
                  <span class="app-username">{{ userStore.user?.username }}</span>
                  <Tag
                    v-if="userStore.user?.is_admin"
                    color="red"
                    :bordered="false"
                    class="app-admin-tag"
                    >管理员</Tag
                  >
                </div>
                <template #content>
                  <Doption @click="handleLogout">
                    <template #icon><IconPoweroff /></template>
                    退出登录
                  </Doption>
                </template>
              </Dropdown>
            </template>
            <template v-else>
              <RouterLink
                to="/login"
                class="app-login-link"
                @click="isMobileMenuOpen = false"
                aria-label="登录"
              >
                <IconUser /> 登录
              </RouterLink>
            </template>

            <!-- 仅移动端显示汉堡按钮，打开抽屉导航 -->
            <Button
              v-if="isMobile && !route.path.startsWith('/admin')"
              class="app-mobile-menu-btn"
              type="text"
              @click="isMobileMenuOpen = !isMobileMenuOpen"
              :aria-label="isMobileMenuOpen ? '关闭菜单' : '打开菜单'"
              :aria-expanded="isMobileMenuOpen"
              aria-controls="mobile-menu-drawer"
            >
              <IconClose v-if="isMobileMenuOpen" />
              <IconMenu v-else />
            </Button>
          </Space>
        </div>
      </LayoutHeader>

      <main :class="route.path.startsWith('/admin') ? 'admin-main' : ''" role="main">
        <ErrorBoundary>
          <RouterView v-slot="{ Component }">
            <KeepAlive :include="['HomeView']">
              <component :is="Component" />
            </KeepAlive>
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
          <div class="app-footer-text app-footer-mono">构建时间：{{ buildDate }}</div>
        </div>
      </footer>
    </Layout>

    <Drawer
      v-if="!route.path.startsWith('/admin')"
      v-model:visible="isMobileMenuOpen"
      placement="left"
      :closable="true"
      :footer="false"
      title="小泉动漫二创站"
      :width="280"
      class="app-mobile-drawer"
      @cancel="isMobileMenuOpen = false"
      aria-label="移动端导航菜单"
    >
      <!-- 用户区 -->
      <div v-if="userStore.isLoggedIn" class="drawer-user">
        <Space :size="12" align="center">
          <Avatar :size="44" :image-url="userAvatarUrl" class="app-avatar-primary">
            <IconUser v-if="!userAvatarUrl" />
          </Avatar>
          <div class="drawer-user-meta">
            <TypographyTitle :heading="6" class="drawer-user-name">
              {{ userStore.user?.username }}
              <Tag v-if="userStore.user?.is_admin" color="red" size="small">管理员</Tag>
            </TypographyTitle>
            <TypographyText type="secondary" class="drawer-user-sub">欢迎回来</TypographyText>
          </div>
        </Space>
      </div>
      <Button v-else type="outline" long @click="goLogin">
        <template #icon><IconUser /></template>
        登录 / 注册
      </Button>

      <Divider />

      <!-- 导航 -->
      <Menu
        mode="vertical"
        :selected-keys="selectedKeys"
        @menu-item-click="onNavClick"
        :accordion="false"
        class="app-menu-flat"
      >
        <MenuItem v-for="item in navItems" :key="item.key">
          <template #icon><component :is="item.icon" /></template>
          {{ item.label }}
        </MenuItem>
      </Menu>

      <Divider />

      <!-- 退出 -->
      <Button v-if="userStore.isLoggedIn" status="danger" long @click="handleLogout">
        <template #icon><IconPoweroff /></template>
        退出登录
      </Button>
    </Drawer>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background: transparent; /* 透明，露出 body::before 固定壁纸 */
}

.app-layout > main {
  flex: 1 0 auto;
  background: transparent; /* 透明，露出 body::before 固定壁纸 */
}

.app-header {
  position: sticky;
  top: 0;
  z-index: 100;
  height: 56px;
  padding: 0;
  line-height: normal;
  background: var(--color-bg-2);
  border-bottom: 1px solid var(--color-border-2);
}

.app-header-inner {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 16px;
  display: flex;
  align-items: center;
  height: 56px;
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

.app-nav-space {
  /* 桌面端导航：占满中间区域，把搜索框和右侧操作区推到右侧。
     flex-shrink:0 保证导航按钮不被压，空间紧时由搜索框先缩。 */
  flex: 1 0 auto;
  min-width: 0;
}

.app-nav-item {
  color: var(--color-text-1) !important;
  white-space: nowrap;
}
.app-nav-item:hover {
  color: rgb(var(--primary-6)) !important;
  background: var(--color-fill-2) !important;
}
/* 激活高亮态：浅主色背景胶囊 + 主色文字 + 加粗，确保清晰可辨 */
.app-nav-item.app-nav-item-active,
.app-nav-item.app-nav-item-active:hover {
  color: rgb(var(--primary-6)) !important;
  background: rgb(var(--primary-1)) !important;
  font-weight: 600;
}
/* 暗色下浅蓝底在深色导航栏上对比偏弱，改用主色发光块 */
body[arco-theme='dark'] .app-nav-item.app-nav-item-active,
body[arco-theme='dark'] .app-nav-item.app-nav-item-active:hover {
  background: rgba(var(--primary-6), 0.18) !important;
  color: rgb(var(--primary-4)) !important;
}

.app-header-right {
  flex-shrink: 0;
  /* 搜索框（或导航占位）与右侧操作区之间留出间距 */
  margin-left: 12px;
}

.app-search-input {
  /* 可收缩：空间变窄时搜索框先让位（flex-shrink:1 + min-width:0），菜单保持完整 */
  flex: 0 1 30%;
  width: auto;
  max-width: 360px;
  min-width: 0;
}
.app-search-input :deep(.arco-input-search-btn) {
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-search-input-mobile {
  flex: 1;
  min-width: 0;
  max-width: 100%;
}
.app-search-input-mobile :deep(.arco-input-search-btn) {
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-header-spacer {
  flex: 1;
  min-width: 0;
}

.app-theme-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  color: var(--color-text-1);
  font-size: 18px;
}

.app-theme-btn:hover {
  color: rgb(var(--primary-6));
  background: var(--color-fill-2);
}

:deep(.app-theme-btn .arco-icon) {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.app-user-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 20px;
  background: var(--color-fill-2);
  cursor: pointer;
  transition: background 0.2s;
}

.app-user-trigger:hover {
  background: var(--color-border-2);
}

.app-username {
  font-size: 13px;
  color: var(--color-text-1);
}

.app-login-link {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--color-text-1);
  text-decoration: none;
  padding: 4px 10px;
  border-radius: 6px;
  transition: background 0.2s;
}

.app-login-link:hover {
  background: var(--color-fill-2);
}

.app-mobile-menu-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
}

:deep(.app-mobile-menu-btn .arco-icon) {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

/* 移动端抽屉（侧边栏） */
.app-mobile-drawer :deep(.arco-drawer-body) {
  padding: 12px 16px 16px;
}

.drawer-user {
  padding: 4px 4px 12px;
}

.drawer-user-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.drawer-user-name {
  margin: 0 !important;
  font-weight: 600;
  color: var(--color-text-1);
  display: flex;
  align-items: center;
  gap: 6px;
}

.drawer-user-sub {
  font-size: 12px;
}

:deep(.app-mobile-drawer .arco-divider) {
  margin: 12px 0;
}

:deep(.app-mobile-drawer .arco-menu) {
  background: transparent;
}

.app-footer {
  margin-top: 40px;
  padding: 20px 0;
  background: var(--color-bg-2);
  border-top: 1px solid var(--color-border-2);
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
  color: var(--color-text-2);
}

.app-footer-mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.app-avatar-primary {
  background-color: rgb(var(--primary-6));
}

.app-admin-tag {
  font-size: 11px;
  padding: 0 4px;
  margin-left: 2px;
}

.app-menu-flat {
  border: none;
}

.app-footer-badge {
  font-size: 13px;
  color: var(--color-text-2);
  padding: 2px 10px;
  background: var(--color-fill-2);
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

  .app-logo-img {
    height: 28px;
  }

  .app-username,
  .app-user-trigger .arco-tag {
    display: none;
  }

  .app-user-trigger {
    padding: 4px;
  }

  main {
    min-height: calc(100vh - 180px);
    padding-top: 56px;
  }

  main.admin-main {
    min-height: 0;
    padding-top: 56px;
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

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { IconArrowRight, IconBarChart, IconHome, IconMenu, IconMoon, IconPlus, IconSettings, IconSun, IconUpload, IconClose } from '@arco-design/web-vue/es/icon'
import { useThemeStore } from '@/stores/theme'
import { useUserStore } from '@/stores/user'
import { useGlobalSearch } from '@/composables/useGlobalSearch'
import { revealOriginFrom, withRevealTransition } from '@/utils/viewTransition'
import UserAvatar from '@/components/UserAvatar.vue'

const route = useRoute()
const router = useRouter()
const themeStore = useThemeStore()
const userStore = useUserStore()
const mobileOpen = ref(false)
const searchDraft = ref('')
const { searchKeyword, triggerSearch } = useGlobalSearch()
watch(searchKeyword, (value) => {
  searchDraft.value = value
})

const pageTab = computed(() =>
  ['/ideas', '/characters', '/polls'].some((path) => route.path === path || route.path.startsWith(`${path}/`))
    ? 'discover'
    : 'home',
)

function closeMobile() {
  mobileOpen.value = false
}

function toggleTheme(event: MouseEvent) {
  const target = ((event.target as Element | null)?.closest?.('.public-shell-theme') as Element | null) ?? null
  withRevealTransition(() => themeStore.toggleMode(), revealOriginFrom(event, target))
}

function submitSearch() {
  searchKeyword.value = searchDraft.value.trim()
  triggerSearch()
  closeMobile()
  if (route.path !== '/') void router.push('/')
}
</script>

<template>
  <div class="public-shell creative-theme" :class="{ 'is-mobile-open': mobileOpen }">
    <aside class="public-shell-sidebar" aria-label="作品广场导航">
      <div class="public-shell-brand-row">
        <button
          class="public-shell-theme"
          type="button"
          :aria-label="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到暗色模式'"
          @click="toggleTheme"
        >
          <IconSun v-if="themeStore.mode === 'dark'" />
          <IconMoon v-else />
        </button>
        <RouterLink to="/" class="public-shell-brand" @click="closeMobile">
          <strong>小泉宇宙</strong>
          <span>二创画廊</span>
        </RouterLink>
      </div>

      <div class="public-shell-primary" aria-label="主要入口">
        <RouterLink to="/studio" active-class="is-active" exact-active-class="is-active" @click="closeMobile">
          <IconUpload />
          <span>工作台</span>
        </RouterLink>
        <RouterLink to="/" active-class="is-active" exact-active-class="is-active" @click="closeMobile">
          <IconHome />
          <span>作品广场</span>
        </RouterLink>
      </div>

      <nav class="public-shell-nav" aria-label="发现">
        <p>发现</p>
        <RouterLink to="/" exact-active-class="is-active" @click="closeMobile">
          <IconHome />
          <span>作品首页</span>
        </RouterLink>
        <RouterLink to="/ideas" active-class="is-active" @click="closeMobile">
          <IconPlus />
          <span>创意广场</span>
        </RouterLink>
        <RouterLink to="/characters" active-class="is-active" @click="closeMobile">
          <IconSettings />
          <span>角色设定</span>
        </RouterLink>
        <RouterLink to="/polls" active-class="is-active" @click="closeMobile">
          <IconBarChart />
          <span>投票广场</span>
        </RouterLink>
      </nav>

      <div class="public-shell-library">
        <p>我的库</p>
        <RouterLink v-if="userStore.isLoggedIn" to="/admin" active-class="is-active" @click="closeMobile">
          <IconSettings />
          <span>我的内容</span>
        </RouterLink>
        <RouterLink v-else to="/login" class="public-shell-library-note" @click="closeMobile">
          <span>登录后查看收藏和发布</span>
          <IconArrowRight />
        </RouterLink>
      </div>

      <div class="public-shell-bottom">
        <RouterLink to="/quick-upload" class="public-shell-publish" @click="closeMobile">
          <IconUpload />
          <span>发布作品</span>
        </RouterLink>
        <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="public-shell-user" @click="closeMobile">
          <span class="public-shell-user-mark"><IconSettings /></span>
          <span><strong>登录 / 注册</strong><small>保存你的创作</small></span>
        </RouterLink>
        <RouterLink v-else to="/admin" class="public-shell-user" @click="closeMobile">
          <UserAvatar :size="32" :email="userStore.user?.email" :username="userStore.user?.username" />
          <span><strong>{{ userStore.user?.username }}</strong><small>创作者空间</small></span>
        </RouterLink>
      </div>
    </aside>

    <button v-if="mobileOpen" class="public-shell-scrim" type="button" aria-label="关闭导航" @click="closeMobile"></button>

    <div class="public-shell-main">
      <header class="public-shell-topbar">
        <button class="public-shell-menu" type="button" :aria-label="mobileOpen ? '关闭导航' : '打开导航'" @click="mobileOpen = !mobileOpen">
          <IconClose v-if="mobileOpen" />
          <IconMenu v-else />
        </button>
        <nav class="public-shell-tabs" aria-label="页面视图">
          <RouterLink to="/" :class="{ 'is-active': pageTab === 'home' }">作品</RouterLink>
          <RouterLink to="/ideas" :class="{ 'is-active': pageTab === 'discover' }">发现</RouterLink>
        </nav>
        <div class="public-shell-actions">
          <label class="public-shell-search">
            <span aria-hidden="true">⌕</span>
            <input v-model="searchDraft" type="search" placeholder="搜索作品、提示词、标签或用户" @keydown.enter.prevent="submitSearch" />
            <kbd>/</kbd>
          </label>
          <button class="public-shell-theme public-shell-theme-main" type="button" :aria-label="themeStore.mode === 'dark' ? '切换到日间模式' : '切换到暗色模式'" @click="toggleTheme">
            <IconSun v-if="themeStore.mode === 'dark'" />
            <IconMoon v-else />
          </button>
          <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="public-shell-login">登录</RouterLink>
          <RouterLink v-else to="/admin" class="public-shell-avatar" aria-label="打开我的内容">
            <UserAvatar :size="32" :email="userStore.user?.email" :username="userStore.user?.username" />
          </RouterLink>
        </div>
      </header>
      <div class="public-shell-content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.public-shell {
  --shell-paper: #fffdf9;
  --shell-canvas: #ebe4e1;
  --shell-soft: #f1e9e6;
  --shell-ink: #201b1c;
  --shell-muted: #817779;
  --shell-line: rgba(52, 40, 42, .14);
  --shell-accent: #6b183b;
  position: relative;
  display: flex;
  min-height: 100dvh;
  background: var(--shell-canvas);
  color: var(--shell-ink);
}
body[arco-theme='dark'] .public-shell {
  --shell-paper: #2b2427;
  --shell-canvas: #211b1e;
  --shell-soft: #342b2f;
  --shell-ink: #f8f0ed;
  --shell-muted: #c0b1b4;
  --shell-line: rgba(250, 240, 237, .16);
  --shell-accent: #f0a8bb;
}
.public-shell-sidebar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  flex: 0 0 280px;
  flex-direction: column;
  width: 280px;
  height: 100dvh;
  padding: 24px 20px 20px;
  background: var(--shell-paper);
  border-right: 1px solid var(--shell-line);
}
.public-shell-brand-row,
.public-shell-primary,
.public-shell-nav,
.public-shell-library,
.public-shell-bottom { display: grid; }
.public-shell-brand-row { grid-template-columns: 32px 1fr; gap: 10px; align-items: center; }
.public-shell-theme {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--shell-accent);
  cursor: pointer;
  transition: background .18s ease, transform .18s ease;
}
.public-shell-theme:hover { background: var(--shell-soft); transform: rotate(-12deg); }
.public-shell-theme:active { transform: rotate(-12deg) scale(.93); }
.public-shell-brand { display: flex; flex-direction: column; color: var(--shell-ink); text-decoration: none; }
.public-shell-brand strong { font: 600 20px/1.1 var(--creative-title-font); letter-spacing: .04em; }
.public-shell-brand span { margin-top: 4px; color: var(--shell-muted); font-size: 10px; letter-spacing: .14em; }
.public-shell-primary {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 3px;
  margin-top: 32px;
  padding: 3px;
  border: 1px solid var(--shell-line);
  border-radius: 11px;
  background: var(--shell-soft);
}
.public-shell-primary a,
.public-shell-nav a,
.public-shell-library a,
.public-shell-user,
.public-shell-publish { text-decoration: none; }
.public-shell-primary a {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  min-height: 38px;
  border-radius: 8px;
  color: var(--shell-muted);
  font-size: 11px;
  transition: background .18s ease, color .18s ease, transform .18s ease;
}
.public-shell-primary a:hover { color: var(--shell-ink); }
.public-shell-primary a.is-active { background: var(--shell-ink); color: var(--shell-paper); }
.public-shell-primary a:active { transform: scale(.98); }
.public-shell-nav,
.public-shell-library { gap: 4px; margin-top: 34px; }
.public-shell-library { margin-top: 28px; }
.public-shell-nav p,
.public-shell-library p { margin: 0 0 7px 10px; color: var(--shell-muted); font-size: 10px; letter-spacing: .14em; }
.public-shell-nav a,
.public-shell-library a {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding: 0 10px;
  border-radius: 9px;
  color: var(--shell-muted);
  font-size: 12px;
  transition: background .18s ease, color .18s ease, transform .18s ease;
}
.public-shell-nav a:hover,
.public-shell-library a:hover { color: var(--shell-ink); background: var(--shell-soft); transform: translateX(2px); }
.public-shell-nav a.is-active,
.public-shell-library a.is-active { color: var(--shell-accent); background: color-mix(in srgb, var(--shell-accent) 12%, var(--shell-paper)); font-weight: 600; }
.public-shell-nav a:active,
.public-shell-library a:active { transform: translateX(1px) scale(.98); }
.public-shell-library-note { justify-content: space-between; line-height: 1.5; }
.public-shell-bottom { gap: 16px; margin-top: auto; }
.public-shell-publish {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 44px;
  border-radius: 10px;
  background: var(--shell-ink);
  color: var(--shell-paper);
  font-size: 12px;
  font-weight: 600;
  transition: transform .18s ease, box-shadow .18s ease;
}
.public-shell-publish:hover { transform: translateY(-2px); box-shadow: 0 9px 22px color-mix(in srgb, var(--shell-ink) 22%, transparent); }
.public-shell-publish:active { transform: scale(.98); }
.public-shell-user { display: flex; align-items: center; gap: 10px; min-width: 0; color: var(--shell-ink); }
.public-shell-user-mark { display: grid; place-items: center; width: 32px; height: 32px; border-radius: 50%; background: var(--shell-soft); color: var(--shell-accent); }
.public-shell-user > span:last-child { display: flex; flex-direction: column; min-width: 0; }
.public-shell-user strong { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 12px; }
.public-shell-user small { margin-top: 3px; color: var(--shell-muted); font-size: 10px; }
.public-shell-main { flex: 1; min-width: 0; background: var(--shell-canvas); }
.public-shell-topbar { display: flex; align-items: center; gap: 24px; min-height: 76px; padding: 0 44px; border-bottom: 1px solid var(--shell-line); background: color-mix(in srgb, var(--shell-canvas) 94%, transparent); }
.public-shell-tabs { display: flex; align-items: center; gap: 24px; }
.public-shell-tabs a { position: relative; padding: 7px 0; color: var(--shell-muted); font: 500 22px/1 var(--creative-title-font); text-decoration: none; }
.public-shell-tabs a::after { content: ''; position: absolute; left: 0; right: 0; bottom: 0; height: 2px; border-radius: 2px; background: var(--shell-accent); transform: scaleX(0); transition: transform .2s ease; }
.public-shell-tabs a.is-active { color: var(--shell-ink); }
.public-shell-tabs a.is-active::after { transform: scaleX(1); }
.public-shell-actions { display: flex; align-items: center; gap: 12px; min-width: 0; margin-left: auto; }
.public-shell-search { display: flex; align-items: center; gap: 8px; width: min(330px, 32vw); min-height: 40px; padding: 0 12px; border: 1px solid var(--shell-line); border-radius: 999px; background: var(--shell-paper); color: var(--shell-muted); }
.public-shell-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--shell-ink); font: inherit; font-size: 11px; }
.public-shell-search kbd { padding: 2px 5px; border: 1px solid var(--shell-line); border-radius: 4px; color: var(--shell-muted); font-size: 10px; }
.public-shell-theme-main { flex: 0 0 auto; border: 1px solid var(--shell-line); }
.public-shell-login { display: inline-flex; align-items: center; min-height: 36px; padding: 0 14px; border: 1px solid var(--shell-line); border-radius: 999px; color: var(--shell-accent); font-size: 11px; text-decoration: none; }
.public-shell-avatar { display: inline-flex; align-items: center; border-radius: 50%; transition: transform .18s ease; }
.public-shell-avatar:hover { transform: translateY(-2px); }
.public-shell-content { max-width: 1480px; min-height: calc(100dvh - 76px); margin: 0 auto; padding: 28px 44px 64px; }
.public-shell-menu,
.public-shell-scrim { display: none; }
@media (max-width: 820px) {
  .public-shell-sidebar { position: fixed; inset: 0 auto 0 0; width: min(82vw, 300px); transform: translateX(-104%); transition: transform .24s cubic-bezier(.22,1,.36,1); box-shadow: 12px 0 32px rgba(31,24,27,.14); }
  .public-shell.is-mobile-open .public-shell-sidebar { transform: translateX(0); }
  .public-shell-scrim { position: fixed; inset: 0; z-index: 25; display: block; border: 0; background: rgba(32,25,27,.32); }
  .public-shell-topbar { position: sticky; top: 0; z-index: 20; gap: 12px; min-height: 60px; padding: 0 16px; background: var(--shell-canvas); }
  .public-shell-menu { display: grid; place-items: center; flex: 0 0 auto; width: 34px; height: 34px; padding: 0; border: 0; border-radius: 50%; background: transparent; color: var(--shell-muted); }
  .public-shell-tabs { gap: 16px; }
  .public-shell-tabs a { font-size: 19px; }
  .public-shell-actions { gap: 8px; }
  .public-shell-search { width: min(42vw, 240px); min-height: 36px; }
  .public-shell-search input { font-size: 10px; }
  .public-shell-search kbd { display: none; }
  .public-shell-content { min-height: calc(100dvh - 60px); padding: 22px 18px 48px; }
}
@media (prefers-reduced-motion: reduce) {
  .public-shell *, .public-shell *::before, .public-shell *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; }
}
</style>

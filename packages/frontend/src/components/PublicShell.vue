<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { IconArrowRight, IconBarChart, IconMenu, IconMoon, IconPlus, IconSettings, IconSun, IconUpload, IconUser, IconClose } from '@arco-design/web-vue/es/icon'
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
const searchInput = ref<HTMLInputElement | null>(null)
const { searchKeyword, triggerSearch } = useGlobalSearch()
watch(searchKeyword, (value) => {
  searchDraft.value = value
})

// 顶栏只切换「创作 / 浏览」两种模式；发现类页面是浏览模式下的子页面，
// 不在这里再放一份目的地。这样 /studio 与 / 各自只有一个入口。
const pageTab = computed(() => {
  if (route.path === '/studio' || route.path.startsWith('/studio/')) return 'studio'
  if (route.path === '/login' || route.path.startsWith('/quick-upload')) return 'none'
  return 'browse'
})

function closeMobile() {
  mobileOpen.value = false
}

function handlePublish() {
  closeMobile()
  void router.push('/quick-upload')
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

/**
 * 顶栏的 "/" 提示必须真的可用：输入框旁的 kbd 是承诺，不是装饰。
 * 只在焦点不在可编辑控件时接管，避免用户在搜索框或正文里打字时被抢走斜杠。
 */
function isEditableTarget(target: EventTarget | null) {
  const el = target as HTMLElement | null
  if (!el) return false
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || el.isContentEditable
}

function onGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && mobileOpen.value) {
    closeMobile()
    return
  }
  if (event.key !== '/' || event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return
  if (isEditableTarget(event.target)) return
  event.preventDefault()
  searchInput.value?.focus()
  searchInput.value?.select()
}

onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKeydown))
</script>

<template>
  <div class="public-shell creative-theme" :class="{ 'is-mobile-open': mobileOpen }">
    <aside class="public-shell-sidebar" aria-label="作品广场导航">
      <button
        v-if="mobileOpen"
        class="public-shell-drawer-close"
        type="button"
        aria-label="关闭导航"
        @click="closeMobile"
      >
        <IconClose />
      </button>
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

      <nav class="public-shell-nav" aria-label="发现">
        <p>发现</p>
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
        <button class="public-shell-publish" type="button" @click="handlePublish">
          <IconUpload />
          <span>发布作品</span>
        </button>
        <RouterLink v-if="!userStore.isLoggedIn" to="/login" class="public-shell-user" @click="closeMobile">
          <span class="public-shell-user-mark"><IconUser /></span>
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
        <nav class="public-shell-tabs" aria-label="主要入口">
          <RouterLink to="/studio" :class="{ 'is-active': pageTab === 'studio' }">工作台</RouterLink>
          <RouterLink to="/" :class="{ 'is-active': pageTab === 'browse' }">作品广场</RouterLink>
        </nav>
        <label class="public-shell-search">
          <span aria-hidden="true">⌕</span>
          <input
            ref="searchInput"
            v-model="searchDraft"
            type="search"
            aria-label="搜索作品、提示词、标签或用户"
            placeholder="搜索作品、提示词、标签或用户"
            @keydown.enter.prevent="submitSearch"
            @keydown.esc.prevent="searchInput?.blur()"
          />
          <kbd aria-hidden="true">/</kbd>
        </label>
        <div class="public-shell-actions">
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
.public-shell-nav a,
.public-shell-library a,
.public-shell-user,
.public-shell-publish { text-decoration: none; }
.public-shell-nav,
.public-shell-library { gap: 4px; margin-top: 28px; }
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
.public-shell-tabs a { position: relative; padding: 7px 0; color: var(--shell-muted); font: 500 22px/1 var(--creative-title-font); text-decoration: none; white-space: nowrap; }
.public-shell-tabs a::after { content: ''; position: absolute; left: 0; right: 0; bottom: 0; height: 2px; border-radius: 2px; background: var(--shell-accent); transform: scaleX(0); transition: transform .2s ease; }
.public-shell-tabs a.is-active { color: var(--shell-ink); }
.public-shell-tabs a.is-active::after { transform: scaleX(1); }
.public-shell-actions { display: flex; flex: 0 0 auto; align-items: center; gap: 12px; min-width: 0; }
.public-shell-search {
  display: flex;
  flex: 0 1 330px;
  align-items: center;
  gap: 8px;
  width: 330px;
  min-width: 120px;
  min-height: 40px;
  /* 搜索是顶栏唯一的自适应项：窄屏挤压它而不是把标签与操作挤出视口。 */
  margin-left: auto;
  padding: 0 12px;
  border: 1px solid var(--shell-line);
  border-radius: 999px;
  background: var(--shell-paper);
  color: var(--shell-muted);
}
/* 桌面端主题开关已在侧栏品牌行；顶栏再放一个会重复同一操作。 */
@media (min-width: 821px) {
  .public-shell-theme-main { display: none; }
}
.public-shell-search input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--shell-ink); font: inherit; font-size: 11px; }
.public-shell-search kbd { padding: 2px 5px; border: 1px solid var(--shell-line); border-radius: 4px; color: var(--shell-muted); font-size: 10px; }
.public-shell-theme-main { flex: 0 0 auto; border: 1px solid var(--shell-line); }
.public-shell-login { display: inline-flex; align-items: center; min-height: 36px; padding: 0 14px; border: 1px solid var(--shell-line); border-radius: 999px; color: var(--shell-accent); font-size: 11px; text-decoration: none; }
.public-shell-avatar { display: inline-flex; align-items: center; border-radius: 50%; transition: transform .18s ease; }
.public-shell-avatar:hover { transform: translateY(-2px); }
.public-shell-content { max-width: 1480px; min-height: calc(100dvh - 76px); margin: 0 auto; padding: 28px 44px 64px; }
.public-shell-menu,
.public-shell-scrim,
.public-shell-drawer-close { display: none; }
@media (max-width: 820px) {
  .public-shell-sidebar { position: fixed; inset: 0 auto 0 0; width: min(82vw, 300px); overflow-y: auto; overscroll-behavior: contain; transform: translateX(-104%); transition: transform .24s cubic-bezier(.22,1,.36,1); box-shadow: 12px 0 32px rgba(31,24,27,.14); }
  .public-shell.is-mobile-open .public-shell-sidebar { transform: translateX(0); }
  .public-shell-scrim { position: fixed; inset: 0; z-index: 25; display: block; border: 0; background: rgba(32,25,27,.32); }
  /* 手机顶栏分两行：第一行导航与账号，第二行搜索。早先单行里搜索、主题和登录
     会把 "作品 / 发现" 挤成竖排并溢出视口。 */
  .public-shell-topbar {
    position: sticky;
    top: 0;
    z-index: 20;
    display: grid;
    grid-template-columns: 34px minmax(0, 1fr) auto;
    grid-template-areas:
      'menu tabs actions'
      'search search search';
    align-items: center;
    gap: 8px 12px;
    min-height: 0;
    padding: 8px 16px 10px;
    background: var(--shell-canvas);
  }
  .public-shell-menu { display: grid; grid-area: menu; place-items: center; width: 34px; height: 34px; padding: 0; border: 0; border-radius: 50%; background: transparent; color: var(--shell-muted); }
  /* 抽屉盖住了顶栏的菜单按钮，抽屉内必须有可见的关闭入口。 */
  .public-shell-drawer-close { position: absolute; top: 18px; right: 16px; z-index: 1; display: grid; place-items: center; width: 32px; height: 32px; padding: 0; border: 1px solid var(--shell-line); border-radius: 50%; background: var(--shell-paper); color: var(--shell-muted); cursor: pointer; }
  .public-shell-tabs { grid-area: tabs; gap: 16px; }
  .public-shell-tabs a { font-size: 18px; }
  .public-shell-actions { grid-area: actions; gap: 8px; }
  .public-shell-search { grid-area: search; width: 100%; max-width: none; min-width: 0; min-height: 36px; margin-left: 0; }
  .public-shell-search input { font-size: 10px; }
  .public-shell-search kbd { display: none; }
  .public-shell-content { min-height: calc(100dvh - 104px); padding: 22px 18px 48px; }
}
@media (prefers-reduced-motion: reduce) {
  .public-shell *, .public-shell *::before, .public-shell *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; }
}
</style>

import type { SiteConfig } from '@/types'

export const DEFAULT_SITE_TEXTS: Record<string, string> = {
  'nav.discover': '发现',
  'nav.ideas': '创意广场',
  'nav.characters': '角色设定',
  'nav.polls': '投票广场',
  'nav.friends': '友情链接',
  'nav.publish': '发布作品',
  'nav.login': '登录 / 注册',
  'nav.loginHint': '保存你的创作',
  'nav.memberHint': '创作者空间',
  'nav.workspace': '工作台',
  'nav.gallery': '作品广场',
  'nav.searchPlaceholder': '搜索作品、提示词、标签或用户',

  'home.featuredLabel': 'CURATED WORKS',
  'home.featuredTitle': '正在被看见',
  'home.featuredRefresh': '换一批推荐作品',
  'home.featuredEmpty': '推荐作品正在整理中',
  'home.recommended': '推荐作品',
  'home.latest': '最新作品',
  'home.popular': '热门作品',
  'home.searchResults': '搜索结果',
  'home.empty': '还没有作品出现在这里',
  'home.emptyFiltered': '没有找到符合条件的作品',
  'home.publishFirst': '发布第一件作品',
  'home.loading': '正在加载作品',
  'home.loadMore': '加载更多作品',
  'home.end': '已经看到这里了',
  'home.clearFilter': '清除筛选',
  'home.allTags': '全部',
  'home.sortRecommended': '推荐',
  'home.sortLatest': '最新',
  'home.sortPopular': '热门',
  'home.filterLabel': '作品筛选',
  'home.sortLabel': '作品排序',
  'home.tagsLabel': '标签筛选',

  'login.storyKicker': 'A LITTLE UNIVERSE OF OUR OWN',
  'login.storyTitle': '在{name}，\n写下你的\n第一颗星球',
  'login.storyText': '上传你的作品，点亮你的星球。\n和第一批造物主，在这里相遇。',
  'login.cardKicker': 'YOUR CREATIVE CORNER',
  'login.titleLogin': '欢迎回来',
  'login.titleRegister': '欢迎来到{name}',
  'login.descLogin': '登录，继续点亮你的星球。',
  'login.descRegister': '创建你的造物主身份，点亮属于你的那颗星球。',
  'login.submitLogin': '登录，回到{name}',
  'login.submitRegister': '注册，成为造物主',
  'login.switchToRegister': '现在就加入宇宙',
  'login.switchToLogin': '欢迎回来，去登录',
  'login.note': '每一颗星球，都值得被看见。',
  'login.homeLink': '返回创作首页',
  'login.emailTitle': '为账号添一份联系',
  'login.emailDesc': '你的账号尚未绑定邮箱。设置邮箱后，就可以继续点亮你的星球。',
  'login.emailSubmit': '保存邮箱',
  'login.artKickerFirst': 'EVERY LITTLE IDEA',
  'login.artLine1': 'starts with you.',
  'login.artKickerSecond': 'CREATE / COLLECT / CONNECT',
  'login.artTitle': 'Make\nyour own',
  'login.artCaption': '每一颗星球，都会被点亮',

  'upload.kicker': 'PUBLISH A PIECE',
  'upload.tipKicker': 'A SMALL REMINDER',
  'upload.title': '发布一份作品',
  'upload.subtitle': '把你的画面、文字和灵感，留在{name}作品广场。',
  'upload.back': '返回作品广场',
  'upload.tip': '先把想法留下，\n细节之后再慢慢完善',
  'upload.notice': '发布后，作品会出现在作品广场。',

  'detail.loading': '正在穿越{name}…',
  'detail.errorTitle': '信号暂时离开了轨道',
  'detail.errorHint': '内容不存在或网络异常，请稍后再试',

  'characters.title': '角色设定库',
  'characters.search': '搜索角色名称与简介',
  'characters.detailMeta': '{title} · 角色设定 · {name}',
  'characters.indexMeta': '角色设定库 · {name}',
  'characters.description': '浏览官方角色设定、外观特征与授权参考图。',

  'ideas.title': '创意广场',
  'polls.title': '投票广场',

  'footer.copyright': '© {name} · 小泉动漫二创站',
}

export const DEFAULT_SITE_CONFIG: SiteConfig = {
  name: '小泉宇宙',
  short_name: '小泉宇宙',
  tagline: '二创画廊',
  browser_title: '{name} · 小泉动漫二创站',
  description: '小泉动漫二创内容分享平台：上传你的作品，点亮属于你的那颗星球。',
  logo_url: '',
  favicon_url: '/favicon.webp',
  friend_links: [
    {
      name: '蓝色大肥鱼',
      url: 'https://xn--pssy23gqgbz2d718b.com/',
      icon_url: '/friend-links/blue-fat-fish.webp',
    },
  ],
  texts: { ...DEFAULT_SITE_TEXTS },
}

export function renderSiteText(template: string, config: Pick<SiteConfig, 'name' | 'short_name' | 'tagline'>) {
  return template
    .replaceAll('{name}', config.name)
    .replaceAll('{short_name}', config.short_name)
    .replaceAll('{tagline}', config.tagline)
}

export function siteTextLines(template: string, config: Pick<SiteConfig, 'name' | 'short_name' | 'tagline'>) {
  return renderSiteText(template, config)
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

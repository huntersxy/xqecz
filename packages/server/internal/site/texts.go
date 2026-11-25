package site

type TextElement struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Group     string `json:"group"`
	Default   string `json:"default"`
	Multiline bool   `json:"multiline"`
	MaxLength int    `json:"max_length"`
}

var textElements = []TextElement{
	{Key: "nav.discover", Label: "导航分组名称", Group: "导航", Default: "发现", MaxLength: 30},
	{Key: "nav.ideas", Label: "创意广场入口", Group: "导航", Default: "创意广场", MaxLength: 30},
	{Key: "nav.characters", Label: "角色设定入口", Group: "导航", Default: "角色设定", MaxLength: 30},
	{Key: "nav.polls", Label: "投票广场入口", Group: "导航", Default: "投票广场", MaxLength: 30},
	{Key: "nav.friends", Label: "友情链接标题", Group: "导航", Default: "友情链接", MaxLength: 30},
	{Key: "nav.publish", Label: "发布作品按钮", Group: "导航", Default: "发布作品", MaxLength: 30},
	{Key: "nav.login", Label: "登录注册入口", Group: "导航", Default: "登录 / 注册", MaxLength: 30},
	{Key: "nav.loginHint", Label: "登录入口说明", Group: "导航", Default: "保存你的创作", MaxLength: 40},
	{Key: "nav.memberHint", Label: "登录后入口说明", Group: "导航", Default: "创作者空间", MaxLength: 40},
	{Key: "nav.workspace", Label: "工作台标签", Group: "导航", Default: "工作台", MaxLength: 20},
	{Key: "nav.gallery", Label: "作品广场标签", Group: "导航", Default: "作品广场", MaxLength: 20},
	{Key: "nav.searchPlaceholder", Label: "搜索框提示", Group: "导航", Default: "搜索作品、提示词、标签或用户", MaxLength: 60},

	{Key: "home.featuredLabel", Label: "精选区小标题", Group: "首页", Default: "CURATED WORKS", MaxLength: 40},
	{Key: "home.featuredTitle", Label: "精选区主标题", Group: "首页", Default: "正在被看见", MaxLength: 40},
	{Key: "home.featuredRefresh", Label: "换一批推荐（无障碍）", Group: "首页", Default: "换一批推荐作品", MaxLength: 40},
	{Key: "home.featuredEmpty", Label: "精选区空状态", Group: "首页", Default: "推荐作品正在整理中", MaxLength: 60},
	{Key: "home.recommended", Label: "推荐作品标题", Group: "首页", Default: "推荐作品", MaxLength: 30},
	{Key: "home.latest", Label: "最新作品标题", Group: "首页", Default: "最新作品", MaxLength: 30},
	{Key: "home.popular", Label: "热门作品标题", Group: "首页", Default: "热门作品", MaxLength: 30},
	{Key: "home.searchResults", Label: "搜索结果标题", Group: "首页", Default: "搜索结果", MaxLength: 30},
	{Key: "home.empty", Label: "空画廊提示", Group: "首页", Default: "还没有作品出现在这里", MaxLength: 60},
	{Key: "home.emptyFiltered", Label: "筛选无结果提示", Group: "首页", Default: "没有找到符合条件的作品", MaxLength: 60},
	{Key: "home.publishFirst", Label: "发布首件作品按钮", Group: "首页", Default: "发布第一件作品", MaxLength: 30},
	{Key: "home.loading", Label: "加载作品提示", Group: "首页", Default: "正在加载作品", MaxLength: 40},
	{Key: "home.loadMore", Label: "加载更多提示", Group: "首页", Default: "加载更多作品", MaxLength: 40},
	{Key: "home.end", Label: "列表到底提示", Group: "首页", Default: "已经看到这里了", MaxLength: 40},
	{Key: "home.clearFilter", Label: "清除筛选按钮", Group: "首页", Default: "清除筛选", MaxLength: 30},
	{Key: "home.allTags", Label: "全部标签按钮", Group: "首页", Default: "全部", MaxLength: 20},
	{Key: "home.sortRecommended", Label: "推荐排序标签", Group: "首页", Default: "推荐", MaxLength: 20},
	{Key: "home.sortLatest", Label: "最新排序标签", Group: "首页", Default: "最新", MaxLength: 20},
	{Key: "home.sortPopular", Label: "热门排序标签", Group: "首页", Default: "热门", MaxLength: 20},
	{Key: "home.filterLabel", Label: "筛选区无障碍说明", Group: "首页", Default: "作品筛选", MaxLength: 40},
	{Key: "home.sortLabel", Label: "排序区无障碍说明", Group: "首页", Default: "作品排序", MaxLength: 40},
	{Key: "home.tagsLabel", Label: "标签区无障碍说明", Group: "首页", Default: "标签筛选", MaxLength: 40},

	{Key: "login.storyKicker", Label: "登录页故事小标题", Group: "登录", Default: "A LITTLE UNIVERSE OF OUR OWN", MaxLength: 80},
	{Key: "login.storyTitle", Label: "登录页故事标题（每行一句）", Group: "登录", Default: "在{name}，\n写下你的\n第一颗星球", Multiline: true, MaxLength: 200},
	{Key: "login.storyText", Label: "登录页故事说明", Group: "登录", Default: "上传你的作品，点亮你的星球。\n和第一批造物主，在这里相遇。", Multiline: true, MaxLength: 300},
	{Key: "login.cardKicker", Label: "登录卡片小标题", Group: "登录", Default: "YOUR CREATIVE CORNER", MaxLength: 80},
	{Key: "login.titleLogin", Label: "登录标题", Group: "登录", Default: "欢迎回来", MaxLength: 40},
	{Key: "login.titleRegister", Label: "注册标题", Group: "登录", Default: "欢迎来到{name}", MaxLength: 40},
	{Key: "login.descLogin", Label: "登录说明", Group: "登录", Default: "登录，继续点亮你的星球。", MaxLength: 100},
	{Key: "login.descRegister", Label: "注册说明", Group: "登录", Default: "创建你的造物主身份，点亮属于你的那颗星球。", MaxLength: 100},
	{Key: "login.submitLogin", Label: "登录按钮", Group: "登录", Default: "登录，回到{name}", MaxLength: 40},
	{Key: "login.submitRegister", Label: "注册按钮", Group: "登录", Default: "注册，成为造物主", MaxLength: 40},
	{Key: "login.switchToRegister", Label: "切换到注册", Group: "登录", Default: "现在就加入宇宙", MaxLength: 40},
	{Key: "login.switchToLogin", Label: "切换到登录", Group: "登录", Default: "欢迎回来，去登录", MaxLength: 40},
	{Key: "login.note", Label: "登录卡片页脚", Group: "登录", Default: "每一颗星球，都值得被看见。", MaxLength: 60},
	{Key: "login.homeLink", Label: "返回首页链接", Group: "登录", Default: "返回创作首页", MaxLength: 30},
	{Key: "login.emailTitle", Label: "绑定邮箱弹窗标题", Group: "登录", Default: "为账号添一份联系", MaxLength: 40},
	{Key: "login.emailDesc", Label: "绑定邮箱弹窗说明", Group: "登录", Default: "你的账号尚未绑定邮箱。设置邮箱后，就可以继续点亮你的星球。", MaxLength: 120},
	{Key: "login.emailSubmit", Label: "保存邮箱按钮", Group: "登录", Default: "保存邮箱", MaxLength: 20},
	{Key: "login.artKickerFirst", Label: "登录插画小标题一", Group: "登录", Default: "EVERY LITTLE IDEA", MaxLength: 60},
	{Key: "login.artLine1", Label: "登录插画短句一", Group: "登录", Default: "starts with you.", MaxLength: 60},
	{Key: "login.artKickerSecond", Label: "登录插画小标题二", Group: "登录", Default: "CREATE / COLLECT / CONNECT", MaxLength: 60},
	{Key: "login.artTitle", Label: "登录插画主文案（可换行）", Group: "登录", Default: "Make\nyour own", Multiline: true, MaxLength: 80},
	{Key: "login.artCaption", Label: "登录插画说明", Group: "登录", Default: "每一颗星球，都会被点亮", MaxLength: 60},

	{Key: "upload.kicker", Label: "发布页小标题", Group: "发布", Default: "PUBLISH A PIECE", MaxLength: 60},
	{Key: "upload.tipKicker", Label: "发布页提示卡小标题", Group: "发布", Default: "A SMALL REMINDER", MaxLength: 60},
	{Key: "upload.title", Label: "发布页标题", Group: "发布", Default: "发布一份作品", MaxLength: 40},
	{Key: "upload.subtitle", Label: "发布页说明", Group: "发布", Default: "把你的画面、文字和灵感，留在{name}作品广场。", MaxLength: 120},
	{Key: "upload.back", Label: "返回作品广场链接", Group: "发布", Default: "返回作品广场", MaxLength: 30},
	{Key: "upload.tip", Label: "发布页提示卡（可换行）", Group: "发布", Default: "先把想法留下，\n细节之后再慢慢完善", Multiline: true, MaxLength: 120},
	{Key: "upload.notice", Label: "发布页页脚说明", Group: "发布", Default: "发布后，作品会出现在作品广场。", MaxLength: 80},

	{Key: "detail.loading", Label: "作品加载提示", Group: "作品页", Default: "正在穿越{name}…", MaxLength: 60},
	{Key: "detail.errorTitle", Label: "作品错误标题", Group: "作品页", Default: "信号暂时离开了轨道", MaxLength: 40},
	{Key: "detail.errorHint", Label: "作品错误说明", Group: "作品页", Default: "内容不存在或网络异常，请稍后再试", MaxLength: 80},

	{Key: "characters.title", Label: "角色库标题", Group: "角色页", Default: "角色设定库", MaxLength: 40},
	{Key: "characters.search", Label: "角色搜索提示", Group: "角色页", Default: "搜索角色名称与简介", MaxLength: 50},
	{Key: "characters.detailMeta", Label: "角色详情浏览器标题", Group: "角色页", Default: "{title} · 角色设定 · {name}", MaxLength: 120},
	{Key: "characters.indexMeta", Label: "角色库浏览器标题", Group: "角色页", Default: "角色设定库 · {name}", MaxLength: 120},
	{Key: "characters.description", Label: "角色库描述", Group: "角色页", Default: "浏览官方角色设定、外观特征与授权参考图。", MaxLength: 120},

	{Key: "ideas.title", Label: "创意广场标题", Group: "发现页", Default: "创意广场", MaxLength: 40},
	{Key: "polls.title", Label: "投票广场标题", Group: "发现页", Default: "投票广场", MaxLength: 40},

	{Key: "footer.copyright", Label: "页脚版权文案", Group: "页脚", Default: "© {name} · 小泉动漫二创站", MaxLength: 160},
}

func Elements() []TextElement {
	out := make([]TextElement, len(textElements))
	copy(out, textElements)
	return out
}

func textElement(key string) (TextElement, bool) {
	for _, element := range textElements {
		if element.Key == key {
			return element, true
		}
	}
	return TextElement{}, false
}

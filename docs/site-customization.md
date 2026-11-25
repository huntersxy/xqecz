# 站点外观与文案配置

站点名称、Logo、图标、友情链接和前台文案统一保存在 `site_settings` 表的单行 JSON 中，由后台「系统 → 站点外观」可视化编辑。服务端启动时加载到内存缓存，保存后立即更新缓存并清理被替换的本地素材。

## 数据与接口

- 表结构：`packages/server/internal/migrate/sql/2026-11-21-add-site-settings.sql`（内嵌进二进制，随部署自动应用）
- 存储服务：`packages/server/internal/site/site.go`
- 文案目录：`packages/server/internal/site/texts.go`
- HTTP 入口：`packages/server/internal/modules/site/site.go`

公开接口（无需登录）：

- `GET /api/site/config`：返回已合并默认值、已替换占位符的站点配置。
- `GET /api/site/manifest`：按当前配置动态生成 Web App Manifest，名称、图标与快捷入口随配置变化。

管理员接口：

- `GET /api/admin/site/settings`：返回当前配置与文案元素目录 `elements`。
- `PUT /api/admin/site/settings`：保存配置，返回保存后的完整配置和目录。
- `POST /api/admin/site/assets/:kind`：上传 `logo` 或 `favicon`，返回 `/images/site/...` 地址。单张不超过 4MB，仅接受 PNG/JPEG/WebP/GIF；favicon 会裁切为 256×256 PNG，Logo 会转为 WebP 并限制最长边 1600px。

素材保存在 `IMAGES_DIR/site/`，通过已有的 `/images/*filepath` 路由公开访问。备份站点外观时需同时包含 `site_settings` 表与 `data/images/site/`。

## 文本元素

`texts` 保存的是「与默认值不同的覆盖项」。保存时服务端会：

- 丢弃未知 key；
- 把单行元素中的换行折叠为空格；
- 丢弃空值和与默认值相同的值，使默认文案后续仍可统一调整；
- 校验每个元素的 `max_length`（按 Unicode 字符数）。

前台通过 `useSiteStore().text(key)`、`format(key, values)`、`lines(key)` 读取文案。内置兜底值位于 `packages/frontend/src/utils/siteText.ts`，在接口失败或首屏加载前使用。

支持占位符：

- `{name}`：站点名称
- `{short_name}`：短名称
- `{tagline}`：站点标语

页面级占位符（例如角色详情标题里的 `{title}`）由调用方通过 `format()` 替换。

### 新增一个文本元素

1. 在 `texts.go` 的 `textElements` 中追加 `Key/Label/Group/Default/MaxLength`。
2. 在 `siteText.ts` 的 `DEFAULT_SITE_TEXTS` 中加入同名兜底值。
3. 在页面中使用 `siteStore.text('key')`。
4. 补充或更新对应测试。

后台表单由 `elements` 目录驱动，不需要改管理端模板。

## 前台应用流程

`main.ts` 在挂载前调用 `useSiteStore().load()`（不阻塞首屏）。加载完成后会更新：

- `document.title`
- `meta[name=description]`、`meta[name=application-name]`
- `link[rel=icon]`、`link[rel=apple-touch-icon]`
- `link[rel=manifest]` 指向 `/api/site/manifest`

`index.html` 保留默认标题、描述和图标，作为接口不可用时的静态兜底。

## 校验规则

- 站点名称必填，最长 40 字；短名称 ≤ 20；标语 ≤ 40；浏览器标题 ≤ 120；描述 ≤ 300。
- Logo、图标、友链图标只允许站内 `/...` 路径或 HTTP(S) 绝对地址。
- 友情链接最多 12 条，名称必填，URL 必须为 HTTP(S)。
- 保存时若替换了 Logo 或图标，旧的本地上传素材会被删除；外链和默认静态资源不受影响。

## 排查

- 首页仍显示旧品牌：检查 `/api/site/config` 是否可达，以及 `GET /api/admin/site/settings` 返回的 `texts` 是否包含目标 key。
- 上传成功但前台没变：上传只返回地址，需要在后台点击「保存更改」。
- 图标更新后浏览器仍显示旧图：`/images` 响应带 30 天缓存，可更换 favicon 文件名（重新上传会自动换名）或强制刷新。
- 表结构不会自动出现：`site_settings` 由内嵌迁移创建（见 [数据库迁移](database-migrations.md)），推送部署会自动应用；本机可用 `go -C packages/server run ./cmd/server migrate` 手动补齐。

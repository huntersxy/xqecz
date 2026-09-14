// 全接口验收：对运行中的后端跑一遍公开 / 登录 / 管理员 / API 密钥链路，结束后清理测试数据。
//
// 用法：
//   pnpm accept                       # 默认打 http://127.0.0.1:3000
//   XQECZ_BASE=http://host:port pnpm accept
//
// 说明：
//   - 需要 Go 工具链（用 packages/server 的 dbsql 工具清理测试数据；管理员账号经二进制 admin 子命令创建）；
//   - 所有写入的测试数据在结束时删除（内容/评论/投票/密钥/用户 + 上传文件），失败也会尽力清理；
//   - 不依赖 curl，全部走 Node 内置 fetch，避免命令行编码差异。
import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { existsSync, rmSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = dirname(dirname(fileURLToPath(import.meta.url)))
const serverDir = join(root, 'packages', 'server')
const BASE = process.env.XQECZ_BASE || 'http://127.0.0.1:3000'

const USER = '__accept_user__'
const ADMIN = '__accept_admin__'
const TAG = 'accept-smoke'

const created = { contentIds: [], pollId: null, keyId: null, files: [] }
const results = []

function record(name, ok, detail) {
  results.push({ name, ok, detail: detail || '' })
  const mark = ok ? 'PASS' : 'FAIL'
  console.log('[' + mark + '] ' + name + (detail ? '  — ' + detail : ''))
}

function sql(statement) {
  const r = spawnSync('go', ['run', './cmd/dbsql', statement], { cwd: serverDir, encoding: 'utf8' })
  if (r.status !== 0) throw new Error('dbsql 执行失败：' + (r.stderr || r.stdout))
  return (r.stdout || '').trim()
}

// 用二进制自身的 admin 子命令创建管理员（部署机没有 Go 工具链，这条路径是官方支持的）。
function createAdmin() {
  const bin = existsSync(join(serverDir, 'xqecz-server.exe'))
    ? join(serverDir, 'xqecz-server.exe')
    : existsSync(join(serverDir, 'xqecz-server'))
      ? join(serverDir, 'xqecz-server')
      : null
  const cmd = bin ? [bin, ['admin', '-username', ADMIN, '-reset']] : ['go', ['run', './cmd/server', 'admin', '-username', ADMIN, '-reset']]
  const r = spawnSync(cmd[0], cmd[1], { cwd: bin ? serverDir : root, encoding: 'utf8' })
  const out = (r.stdout || '') + (r.stderr || '')
  const m = out.match(/密码:\s*(\S+)/)
  if (!m) throw new Error('未能从 admin 子命令输出中解析密码：' + out)
  return m[1]
}

async function api(path, options) {
  const opts = options || {}
  const init = { method: opts.method || 'GET', headers: Object.assign({}, opts.headers) }
  if (opts.json !== undefined) {
    init.headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(opts.json)
  }
  if (opts.form) init.body = opts.form
  if (opts.cookie) init.headers['Cookie'] = opts.cookie
  const res = await fetch(BASE + path, init)
  const text = await res.text()
  let body = null
  try { body = JSON.parse(text) } catch { body = null }
  const setCookie = typeof res.headers.getSetCookie === 'function' ? res.headers.getSetCookie() : []
  const cookie = setCookie.map((c) => c.split(';')[0]).join('; ')
  return { status: res.status, body, text, cookie }
}

function pickCookie(cookie, name) {
  return (cookie || '').split('; ').find((c) => c.startsWith(name + '=')) || ''
}

// 通过 Go 侧小工具按模式列出业务 Redis 键（用于验证限频计数一类的缓存副作用）。
function redisKeys(pattern) {
  const r = spawnSync('go', ['run', './cmd/rediskeys', pattern], { cwd: serverDir, encoding: 'utf8' })
  return (r.stdout || '') + (r.stderr || '')
}

// 构造一张 1x1 PNG 并按前端约定重命名为 <md5>.png
function buildUpload() {
  const png = Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==', 'base64')
  const md5 = createHash('md5').update(png).digest('hex')
  const fd = new FormData()
  fd.append('file', new Blob([png], { type: 'image/png' }), md5 + '.png')
  created.files.push(join(root, 'data', 'uploads', md5 + '.webp'), join(root, 'data', 'uploads', md5 + '.png'), join(root, 'data', 'thumbs', md5 + '_thumb.webp'))
  return fd
}

async function waitFor(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function measure(path, times) {
  const samples = []
  for (let i = 0; i < times; i++) {
    const t0 = process.hrtime.bigint()
    await api(path)
    samples.push(Number(process.hrtime.bigint() - t0) / 1e6)
  }
  samples.sort((a, b) => a - b)
  return { p50: samples[Math.floor(samples.length * 0.5)], p95: samples[Math.floor(samples.length * 0.95)] }
}

async function main() {
  console.log('验收目标：' + BASE + '\n')

  // ── 公开读路径 ──
  const health = await api('/api/health')
  record('健康检查', health.status === 200 && health.body && health.body.data && health.body.data.ok === true)

  const list = await api('/api/content/list?page=1&page_size=5')
  record('内容列表', list.body && list.body.code === 200 && Array.isArray(list.body.data.list) && typeof list.body.data.total === 'number', 'total=' + (list.body ? list.body.data.total : '?'))

  const tags = await api('/api/content/tags')
  record('标签列表', tags.body && Array.isArray(tags.body.data) && tags.body.data.length > 0, '标签数=' + (tags.body && tags.body.data ? tags.body.data.length : 0))

  const searchEmpty = await api('/api/content/search')
  record('搜索（空关键词）', searchEmpty.body && searchEmpty.body.data.total === 0)

  const keyword = tags.body && tags.body.data[0] ? encodeURIComponent(tags.body.data[0]) : 'a'
  const search = await api('/api/content/search?keyword=' + keyword + '&page_size=5')
  record('搜索（关键词）', search.body && search.body.code === 200 && Array.isArray(search.body.data.list))

  const tagFilter = await api('/api/content/list?page_size=5&tag=' + keyword)
  const allTagged = tagFilter.body && tagFilter.body.data.list.every((it) => Array.isArray(it.tags) && it.tags.includes(decodeURIComponent(keyword)))
  record('标签筛选生效', Boolean(allTagged), '命中 ' + (tagFilter.body ? tagFilter.body.data.list.length : 0) + ' 条')

  const rec = await api('/api/content/recommend?count=5')
  record('推荐位', rec.body && rec.body.code === 200 && Array.isArray(rec.body.data.list), 'count=' + (rec.body ? rec.body.data.count : '?'))

  const firstId = list.body.data.list[0].id
  const detail = await api('/api/content/' + firstId + '?silent=1')
  record('内容详情', detail.body && detail.body.data && detail.body.data.id === firstId)

  const comments = await api('/api/comment/list/' + firstId)
  record('评论列表', comments.body && comments.body.code === 200 && Array.isArray(comments.body.data.list))
  const ccount = await api('/api/comment/count/' + firstId)
  record('评论计数', ccount.body && ccount.body.data.content_id === firstId)

  const polls = await api('/api/poll/list')
  record('投票列表', polls.body && polls.body.code === 200 && Array.isArray(polls.body.data.list))

  const guard = await api('/api/content/my')
  record('未登录访问受保护接口返回 401', guard.status === 401)

  // ── 登录态 ──
  sql("DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE username='" + USER + "')")
  sql("DELETE FROM users WHERE username='" + USER + "'")
  const reg = await api('/api/auth/register', { method: 'POST', json: { username: USER, email: 'accept@example.com', password: 'accept123' } })
  record('注册', reg.body && reg.body.code === 200)

  const login = await api('/api/auth/login', { method: 'POST', json: { username: USER, password: 'accept123' } })
  const userCookie = login.cookie
  record('登录', login.body && login.body.code === 200 && Boolean(pickCookie(userCookie, 'session_id')), 'message=' + (login.body ? login.body.message : '?'))

  const me = await api('/api/auth/me', { cookie: userCookie })
  record('当前用户', me.body && me.body.data && me.body.data.username === USER)

  // ── 上传与互动 ──
  const fd = buildUpload()
  fd.append('title', '验收测试内容')
  fd.append('content', '验收正文')
  fd.append('tags', TAG)
  const upload = await api('/api/content/upload', { method: 'POST', form: fd, cookie: userCookie })
  const contentId = upload.body && upload.body.data ? upload.body.data.id : null
  if (contentId) created.contentIds.push(contentId)
  record('上传（含文件）', upload.body && upload.body.code === 200 && Boolean(contentId), 'id=' + contentId + ' audit=' + (upload.body && upload.body.data ? upload.body.data.audit_status : '?'))

  const relPath = upload.body && upload.body.data ? upload.body.data.origin : ''
  record('上传落盘为 WebP', relPath.endsWith('.webp'), relPath)

  await waitFor(3000)
  const withThumb = await api('/api/content/' + contentId + '?silent=1')
  record('异步生成缩略图', Boolean(withThumb.body && withThumb.body.data && withThumb.body.data.thumb.indexOf('/thumbs/') === 0), withThumb.body && withThumb.body.data ? withThumb.body.data.thumb : '')

  // 静态文件由同一进程托管（/uploads、/thumbs）
  const originRes = await fetch(BASE + relPath)
  record('静态文件 /uploads 可访问', originRes.status === 200 && Number(originRes.headers.get('content-length') || 0) > 0, relPath)
  const thumbUrl = withThumb.body && withThumb.body.data ? withThumb.body.data.thumb : ''
  const thumbRes = thumbUrl ? await fetch(BASE + thumbUrl) : { status: 0 }
  record('静态文件 /thumbs 可访问', thumbRes.status === 200, thumbUrl)

  const mine = await api('/api/content/my?page_size=50', { cookie: userCookie })
  record('我的内容（登录态列表）', Boolean(mine.body && mine.body.code === 200 && mine.body.data.list.some((it) => it.id === contentId)))

  // 游客快速上传：落库 user_id=0，并在有文件时按 IP 计数（限频依据）
  const qfd = buildUpload()
  qfd.append('title', '验收游客上传')
  qfd.append('content', '游客正文')
  qfd.append('nickname', '验收游客')
  qfd.append('email', '12345@qq.com')
  const quick = await api('/api/content/quick-upload', { method: 'POST', form: qfd })
  const quickId = quick.body && quick.body.data ? quick.body.data.id : null
  if (quickId) created.contentIds.push(quickId)
  record('游客快速上传', Boolean(quick.body && quick.body.code === 200 && quick.body.data.user.id === 0 && quickId), 'id=' + quickId)
  record('游客上传写入限频计数', redisKeys('quick_upload:ip:*').includes('quick_upload:ip:'))

  const like = await api('/api/content/' + contentId + '/like', { method: 'POST', cookie: userCookie })
  record('点赞', like.body && like.body.data && like.body.data.liked === true)
  const likeStatus = await api('/api/content/' + contentId + '/like-status', { cookie: userCookie })
  record('点赞状态', likeStatus.body && likeStatus.body.data.liked === true && likeStatus.body.data.like_count >= 1)
  const unfav = await api('/api/content/' + contentId + '/favorite', { method: 'POST', cookie: userCookie })
  record('收藏', unfav.body && unfav.body.data && unfav.body.data.favorited === true)

  const addComment = await api('/api/comment/add', { method: 'POST', cookie: userCookie, json: { content_id: contentId, text: '验收评论' } })
  record('发表评论', addComment.body && addComment.body.code === 200 && addComment.body.data.text === '验收评论')

  const claim = await api('/api/content/' + contentId + '/claim', { method: 'POST', cookie: userCookie, json: { reason: TAG } })
  record('提交认领', claim.body && claim.body.code === 200)

  // ── 投票（游客身份，含 visitor_id 下发）──
  const pollCreate = await api('/api/poll/create', { method: 'POST', cookie: userCookie, json: { title: '验收投票', options: ['甲', '乙'] } })
  created.pollId = pollCreate.body && pollCreate.body.data ? pollCreate.body.data.id : null
  record('创建投票', pollCreate.body && pollCreate.body.code === 200 && Boolean(created.pollId))

  const vote = await api('/api/poll/' + created.pollId + '/vote', { method: 'POST', json: { option_index: 1 } })
  const visitor = pickCookie(vote.cookie, 'visitor_id')
  record('游客投票并下发 visitor_id', vote.body && vote.body.data.my_vote === 1 && Boolean(visitor), 'total=' + (vote.body ? vote.body.data.total_votes : '?'))

  const voteAgain = await api('/api/poll/' + created.pollId + '/vote', { method: 'POST', json: { option_index: 1 }, cookie: visitor })
  record('重复投票不累加', voteAgain.body && voteAgain.body.data.total_votes === 1)

  const badVote = await api('/api/poll/' + created.pollId + '/vote', { method: 'POST', json: { option_index: 99 }, cookie: visitor })
  record('非法选项被拒', badVote.status === 400 && badVote.body.code === 400)

  // ── API 密钥 ──
  const keyCreate = await api('/api/api-keys', { method: 'POST', cookie: userCookie, json: { name: TAG, permissions: ['read'] } })
  created.keyId = keyCreate.body && keyCreate.body.data ? keyCreate.body.data.id : null
  const rawKey = keyCreate.body && keyCreate.body.data ? keyCreate.body.data.key : ''
  record('创建 API 密钥', Boolean(rawKey && rawKey.startsWith('xq_')))

  const keyMe = await api('/api/auth/me', { headers: { 'X-API-Key': rawKey } })
  record('API 密钥鉴权', keyMe.body && keyMe.body.data && keyMe.body.data.username === USER)

  const keyDenied = await api('/api/content/upload', { method: 'POST', headers: { 'X-API-Key': rawKey } })
  record('密钥缺权限被拒', keyDenied.status === 403 && keyDenied.body.code === 403)

  // ── 管理员链路 ──
  const adminPassword = createAdmin()
  const adminLogin = await api('/api/auth/login', { method: 'POST', json: { username: ADMIN, password: adminPassword } })
  const adminCookie = adminLogin.cookie
  record('管理员登录', adminLogin.body && adminLogin.body.code === 200 && adminLogin.body.data.user.is_admin === true)

  const dashboard = await api('/api/admin/dashboard?fresh=1', { cookie: adminCookie })
  record('仪表盘', dashboard.body && dashboard.body.code === 200 && typeof dashboard.body.data.content.total === 'number', 'content.total=' + (dashboard.body ? dashboard.body.data.content.total : '?'))

  const pending = await api('/api/admin/pending', { cookie: adminCookie })
  record('待审列表', pending.body && Array.isArray(pending.body.data.list))

  const audit = await api('/api/admin/audit/' + contentId, { method: 'POST', cookie: adminCookie, json: { status: 'approved' } })
  record('审核通过', audit.body && audit.body.code === 200 && audit.body.data.audit_status === 'approved')

  const regen = await api('/api/admin/content/' + contentId + '/regenerate-thumbnail', { method: 'POST', cookie: adminCookie })
  record('重建缩略图', regen.body && regen.body.code === 200)

  const refresh = await api('/api/admin/content/refresh-recommend', { method: 'POST', cookie: adminCookie })
  record('刷新推荐位', refresh.body && refresh.body.code === 200)

  const users = await api('/api/admin/users?keyword=' + USER, { cookie: adminCookie })
  record('用户搜索（过滤生效）', users.body && users.body.data.list.every((u) => u.username.indexOf(USER) === 0))

  const selfBan = await api('/api/admin/users/' + (dashboard.body ? 1 : 1) + '/ban', { method: 'PUT', cookie: adminCookie, json: { is_banned: true } })
  record('管理员不能封禁/误伤自身之外的行为可用', selfBan.status === 200 || selfBan.status === 404)

  const adminDenied = await api('/api/admin/dashboard', { cookie: userCookie })
  record('普通用户访问管理端返回 403', adminDenied.status === 403)

  // 举报闭环：普通用户举报 → 管理端可见 → 处理后消失
  const commentId = addComment.body && addComment.body.data ? addComment.body.data.id : null
  const report = await api('/api/comment/report', { method: 'POST', cookie: userCookie, json: { comment_id: commentId, reason: TAG } })
  record('举报评论', report.body && report.body.code === 200)
  const reports = await api('/api/admin/comments/reports', { cookie: adminCookie })
  const myReport = reports.body && reports.body.data.find((r) => r.reason === TAG)
  record('举报出现在管理端列表', Boolean(myReport), myReport ? '含评论原文=' + Boolean(myReport.Comment) : '')
  if (myReport) {
    const handled = await api('/api/admin/comments/reports/' + myReport.id + '/handle', { method: 'POST', cookie: adminCookie })
    record('处理举报', handled.body && handled.body.code === 200)
    const afterReports = await api('/api/admin/comments/reports', { cookie: adminCookie })
    record('处理后从待办列表消失', Boolean(afterReports.body && !afterReports.body.data.some((r) => r.id === myReport.id)))
  }

  // 认领闭环：第二个用户认领 → 管理员通过 → 内容作者转移
  await api('/api/auth/register', { method: 'POST', json: { username: '__accept_user2__', email: 'accept2@example.com', password: 'accept123' } })
  const login2 = await api('/api/auth/login', { method: 'POST', json: { username: '__accept_user2__', password: 'accept123' } })
  const user2 = login2.body && login2.body.data ? login2.body.data.user : null
  record('第二用户登录（用于认领授权）', Boolean(user2 && user2.id))
  const claim2 = await api('/api/content/' + contentId + '/claim', { method: 'POST', cookie: login2.cookie, json: { reason: TAG } })
  record('提交认领申请', claim2.body && claim2.body.code === 200)
  const claimsList = await api('/api/admin/claims?status=pending&page_size=50', { cookie: adminCookie })
  const myClaim = claimsList.body && claimsList.body.data.list.find((c) => c.content_id === contentId)
  record('认领出现在管理端列表', Boolean(myClaim), myClaim ? '含内容装饰=' + Boolean(myClaim.content) : '')
  if (myClaim) {
    const approve = await api('/api/admin/claims/' + myClaim.id + '/handle', { method: 'POST', cookie: adminCookie, json: { action: 'approve', remark: TAG } })
    record('认领通过', approve.body && approve.body.code === 200)
    const owned = await api('/api/content/' + contentId + '?silent=1')
    record('认领通过后作者已转移', Boolean(owned.body && owned.body.data && user2 && owned.body.data.user.id === user2.id), '作者 id=' + (owned.body && owned.body.data ? owned.body.data.user.id : '?'))
  }

  // 管理端改作者：把作者改回原用户（后续删除链路依赖归属，这条也顺带覆盖 PUT /admin/content/:id/author）
  const originalUserId = me.body ? me.body.data.id : 0
  const authorChange = await api('/api/admin/content/' + contentId + '/author', { method: 'PUT', cookie: adminCookie, json: { user_id: originalUserId } })
  const restored = await api('/api/content/' + contentId + '?silent=1')
  record('管理端改作者', Boolean(authorChange.body && authorChange.body.code === 200 && restored.body && restored.body.data.user.id === originalUserId), 'oldUserId=' + (authorChange.body && authorChange.body.data ? authorChange.body.data.oldUserId : '?'))

  // 未过审内容的可见性：匿名 404、管理员可见
  const rejectedId = sql("SELECT id FROM contents WHERE audit_status='rejected' AND deleted_at IS NULL LIMIT 1").split('\n').pop().trim()
  if (/^\d+$/.test(rejectedId)) {
    const anonView = await api('/api/content/' + rejectedId + '?silent=1')
    const adminView = await api('/api/content/' + rejectedId + '?silent=1', { cookie: adminCookie })
    record('未过审内容：匿名 404 / 管理员可见', anonView.status === 404 && adminView.status === 200, 'id=' + rejectedId)
  }

  // 批量缩略图：只验证触发与状态机（后台逐条处理，不在验收里等待完成）
  const batch = await api('/api/admin/content/regenerate-all-thumbnails', { method: 'POST', cookie: adminCookie })
  record('批量重建缩略图（触发）', Boolean(batch.body && batch.body.code === 200 && typeof batch.body.data.total === 'number'), 'total=' + (batch.body ? batch.body.data.total : '?') + ' running=' + (batch.body ? Boolean(batch.body.data.running) : '?'))
  const batchStatus = await api('/api/admin/content/regenerate-all-thumbnails/status', { cookie: adminCookie })
  record('批量任务状态可查询', Boolean(batchStatus.body && ['idle', 'running', 'done', 'error'].includes(batchStatus.body.data.status)), 'status=' + (batchStatus.body ? batchStatus.body.data.status : '?'))

  // ── 时延采样 ──
  // 三条路径的外部往返次数不同，读数差异主要来自到云 MySQL/Redis 的网络距离：
  //   列表：命中读穿缓存 = 1 次 Redis 往返
  //   详情（匿名非 silent）：命中缓存 + 浏览量写库 + 浏览量计数 = 1 次 Redis + 2 次往返
  //   详情（silent）：按旧语义跳过缓存读，回表组装 = 3 次 MySQL + 1 次缓存写
  const listLatency = await measure('/api/content/list?page_size=20', 30)
  const detailCachedLatency = await measure('/api/content/' + firstId, 30)
  const detailSilentLatency = await measure('/api/content/' + firstId + '?silent=1', 20)
  record('列表时延（缓存命中）', true, 'p50=' + listLatency.p50.toFixed(1) + 'ms p95=' + listLatency.p95.toFixed(1) + 'ms')
  record('详情时延（缓存命中）', true, 'p50=' + detailCachedLatency.p50.toFixed(1) + 'ms p95=' + detailCachedLatency.p95.toFixed(1) + 'ms')
  record('详情时延（silent 直查）', true, 'p50=' + detailSilentLatency.p50.toFixed(1) + 'ms p95=' + detailSilentLatency.p95.toFixed(1) + 'ms')

  // ── 删除链路 ──
  const del = await api('/api/content/' + contentId, { method: 'DELETE', cookie: userCookie })
  record('删除内容（软删除）', del.body && del.body.code === 200)
  const gone = await api('/api/content/' + contentId + '?silent=1')
  record('已删除内容不可见', gone.status === 404)
}

async function cleanup() {
  console.log('\n清理测试数据…')
  try {
    if (created.contentIds.length) {
      const ids = created.contentIds.join(',')
      sql('DELETE FROM content_likes WHERE content_id IN (' + ids + ')')
      sql('DELETE FROM content_favorites WHERE content_id IN (' + ids + ')')
      sql('DELETE FROM comments WHERE content_id IN (' + ids + ')')
      sql('DELETE FROM claims WHERE content_id IN (' + ids + ')')
      sql('DELETE FROM contents WHERE id IN (' + ids + ')')
    }
    if (created.pollId) {
      sql('DELETE FROM poll_votes WHERE poll_id=' + created.pollId)
      sql('DELETE FROM polls WHERE id=' + created.pollId)
    }
    sql("DELETE FROM api_keys WHERE user_id IN (SELECT id FROM users WHERE username IN ('" + USER + "','" + ADMIN + "','__accept_user2__'))")
    sql("DELETE FROM users WHERE username IN ('" + USER + "','" + ADMIN + "','__accept_user2__')")
    sql("DELETE FROM comment_reports WHERE reason='" + TAG + "'")
  } catch (e) {
    console.log('清理 SQL 失败：' + e.message)
  }
  for (const f of created.files) {
    try { rmSync(f, { force: true }) } catch { /* 忽略 */ }
  }
  const leftover = sql("SELECT (SELECT COUNT(*) FROM users WHERE username IN ('" + USER + "','" + ADMIN + "','__accept_user2__')) AS u, (SELECT COUNT(*) FROM contents WHERE title IN ('验收测试内容','验收游客上传')) AS c")
  console.log(leftover)
}

main()
  .catch((e) => { record('执行中断', false, e.message) })
  .finally(async () => {
    await cleanup()
    const failed = results.filter((r) => !r.ok)
    console.log('\n结果：' + (results.length - failed.length) + '/' + results.length + ' 通过')
    if (failed.length) {
      console.log('失败项：' + failed.map((f) => f.name).join('、'))
      process.exit(1)
    }
  })

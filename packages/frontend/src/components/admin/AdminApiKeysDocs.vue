<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import python from 'highlight.js/lib/languages/python'
import javascript from 'highlight.js/lib/languages/javascript'
import powershell from 'highlight.js/lib/languages/powershell'
import json from 'highlight.js/lib/languages/json'
import 'highlight.js/styles/github-dark.css'
import { toast } from '@/composables/useToast'
import { scrollBehavior } from '@/utils'

hljs.registerLanguage('bash', bash)
hljs.registerLanguage('python', python)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('powershell', powershell)
hljs.registerLanguage('json', json)

const activeLang = ref('curl')
const langLabels: Record<string, string> = {
  curl: 'curl',
  python: 'Python',
  node: 'Node.js',
  powershell: 'PowerShell',
}
const langClass: Record<string, string> = {
  curl: 'bash',
  python: 'python',
  node: 'javascript',
  powershell: 'powershell',
}

// 示例统一以“上传 → 列表 → 删除”完整链路为例；密钥换成你自己的。
const codeExamples: Record<string, string> = {
  curl: `# 1. 上传内容（multipart/form-data）
curl -X POST "https://xq.xiey.work/api/content/upload" \\
  -H "X-API-Key: xq_你的完整密钥" \\
  -F "title=我的作品" \\
  -F "content=**Markdown** 描述" \\
  -F "tags=AI,风景" \\
  -F "file=@./work.png"

# 2. 读取列表（公开，无需密钥）
curl "https://xq.xiey.work/api/content/list?page=1&page_size=20"

# 3. 删除内容（id 取上传返回的 data.id）
curl -X DELETE "https://xq.xiey.work/api/content/815" \\
  -H "X-API-Key: xq_你的完整密钥"`,
  python: `import requests

BASE = "https://xq.xiey.work/api"
HEADERS = {"X-API-Key": "xq_你的完整密钥"}

# 1. 上传内容（multipart/form-data）
resp = requests.post(
    BASE + "/content/upload",
    headers=HEADERS,
    data={
        "title": "我的作品",
        "content": "**Markdown** 描述",
        "tags": "AI,风景",
    },
    files={"file": open("work.png", "rb")},
)
data = resp.json()
content_id = data["data"]["id"]   # bigint 字段返回字符串
print(data["code"], content_id)

# 2. 读取列表
lst = requests.get(BASE + "/content/list",
                   params={"page": 1, "page_size": 20}).json()
print(lst["data"]["total"], lst["data"]["list"][0]["title"])

# 3. 删除内容
deleted = requests.delete(BASE + "/content/" + content_id, headers=HEADERS)
print(deleted.json())   # {"code": 200, "message": "已删除", "data": null}`,
  node: `import { readFile } from 'node:fs/promises'

const BASE = 'https://xq.xiey.work/api'
const HEADERS = { 'X-API-Key': 'xq_你的完整密钥' }

// 1. 上传内容（multipart/form-data；Content-Type 由 fetch 自动带 boundary）
const file = new Blob([await readFile('./work.png')], { type: 'image/png' })
const form = new FormData()
form.append('title', '我的作品')
form.append('content', '**Markdown** 描述')
form.append('tags', 'AI,风景')
form.append('file', file, 'work.png')

const resp = await fetch(BASE + '/content/upload', {
  method: 'POST',
  headers: HEADERS,
  body: form,
})
const data = await resp.json()
const contentId = data.data?.id

// 2. 读取列表（公开）
const list = await fetch(BASE + '/content/list?page=1&page_size=20')
  .then((r) => r.json())
console.log(list.data.total)

// 3. 删除内容
const deleted = await fetch(BASE + '/content/' + contentId, {
  method: 'DELETE',
  headers: HEADERS,
}).then((r) => r.json())
console.log(deleted)`,
  powershell: `$base = 'https://xq.xiey.work/api'
$headers = @{ 'X-API-Key' = 'xq_你的完整密钥' }

# 1. 上传内容（multipart/form-data，PowerShell 7+ 的 -Form）
$resp = Invoke-RestMethod -Uri ($base + '/content/upload') -Method Post \`
  -Headers $headers -Form @{
    title   = '我的作品'
    content = '**Markdown** 描述'
    tags    = 'AI,风景'
    file    = Get-Item '.\\work.png'
  }
$id = $resp.data.id
Write-Host ('code=' + $resp.code + ' id=' + $id)

# 2. 读取列表（公开）
$list = Invoke-RestMethod -Uri ($base + '/content/list?page=1&page_size=20')
Write-Host ('total=' + $list.data.total)

# 3. 删除内容
$deleted = Invoke-RestMethod -Uri ($base + '/content/' + $id) -Method Delete -Headers $headers
$deleted | ConvertTo-Json -Depth 5`,
}

// 目录与正文一一对应：正文连续渲染，目录只负责定位与高亮。
const tocItems = [
  { id: 'sec-auth', label: '认证与权限' },
  { id: 'sec-response', label: '统一响应格式' },
  { id: 'ep-list', label: 'GET  /content/list' },
  { id: 'ep-detail', label: 'GET  /content/:id' },
  { id: 'ep-upload', label: 'POST /content/upload' },
  { id: 'ep-update', label: 'PUT  /content/:id' },
  { id: 'ep-delete', label: 'DELETE /content/:id' },
  { id: 'sec-examples', label: '语言示例' },
]

const activeToc = ref('sec-auth')

// 点击目录后短暂锁住高亮，避免平滑滚动途中被中间章节抢回去
let lockUntil = 0
let observer: IntersectionObserver | null = null
const visibleSections = new Set<string>()


interface EndpointParam { name: string; loc?: string; type: string; required: string; descHtml: string }
interface EndpointCard {
  id: string; method: string; methodCls: string; path: string; badge: string; badgeCls: string
  descHtml: string; noteHtml?: string
  params?: { title: string; head: string[]; hasLoc: boolean; rows: EndpointParam[] }
  response?: string
}

const endpointCards: EndpointCard[] = [
  {
    id: 'ep-list', method: 'GET', methodCls: 'is-get', path: '/api/content/list',
    badge: '公开', badgeCls: 'is-public',
    descHtml: `分页获取已审核内容列表。`,
    params: { title: '请求参数（query）', head: ['参数', '类型', '必填', '说明'], hasLoc: false, rows: [
      { name: 'page', type: 'number', required: '否', descHtml: `页码，默认 1` },
      { name: 'page_size', type: 'number', required: '否', descHtml: `每页条数，默认 20，最大 100` },
      { name: 'sort_by', type: 'string', required: '否', descHtml: `created_at / view_count / id，默认 created_at` },
      { name: 'order', type: 'string', required: '否', descHtml: `desc / asc，默认 desc` },
      { name: 'tag', type: 'string', required: '否', descHtml: `标签过滤，多个用英文逗号分隔` },
      { name: 'keyword', type: 'string', required: '否', descHtml: `标题关键词（完整搜索见 <code>/api/content/search</code>）` },
    ] },
    response: `{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": "814",
        "title": "晓晓自拍",
        "text": "",
        "thumb": "/thumbs/xxx_thumb.webp",
        "img": "/uploads/xxx.webp",
        "user": { "id": "2", "username": "huntersxy" },
        "tags": [],
        "like_count": 0,
        "audit_status": "approved",
        "created_at": "2026-08-01T17:00:28.227Z"
      }
    ],
    "total": 309,
    "page": 1,
    "page_size": 20,
    "total_page": 16
  }
}`,
  },
  {
    id: 'ep-detail', method: 'GET', methodCls: 'is-get', path: '/api/content/:id',
    badge: '公开', badgeCls: 'is-public',
    descHtml: `获取单个内容详情，<code>data</code> 为单个内容对象（字段同列表项）；
            可附加 <code>?silent=1</code> 跳过浏览量计数。`,
    params: { title: '请求参数（path / query）', head: ['参数', '位置', '类型', '必填', '说明'], hasLoc: true, rows: [
      { name: 'id', loc: 'path', type: 'number', required: '是', descHtml: `内容 ID` },
      { name: 'silent', loc: 'query', type: 'string', required: '否', descHtml: `传 1 时跳过浏览量计数` },
    ] },
  },
  {
    id: 'ep-upload', method: 'POST', methodCls: 'is-post', path: '/api/content/upload',
    badge: '需 upload 权限', badgeCls: 'is-auth',
    descHtml: `上传内容，请求体为 <code>multipart/form-data</code>（文本字段与文件可同时携带）。`,
    noteHtml: `上传后原样保留源文件并立即生成缩略图；后台压缩任务会择机用 TinyPNG 压缩原图（就地替换，压缩后后缀不变，GIF 与小于 400KB 的不处理）。`,
    params: { title: '请求参数（form-data）', head: ['字段', '类型', '必填', '说明'], hasLoc: false, rows: [
      { name: 'title', type: 'string', required: '是', descHtml: `标题，1-200 字` },
      { name: 'content', type: 'string', required: '否', descHtml: `Markdown 描述；与 file 至少填一项` },
      { name: 'file', type: 'file', required: '否', descHtml: `图片 / 视频文件（≤20MB）；上传不做转码，压缩由后台任务完成；与 content 至少填一项` },
      { name: 'tags', type: 'string', required: '否', descHtml: `标签，多个用英文逗号分隔，如 <code>AI,风景</code>` },
    ] },
    response: `{
  "code": 200,
  "message": "上传成功",
  "data": {
    "id": "815",
    "title": "我的作品",
    "text": "**Markdown** 描述",
    "thumb": "",
    "img": "",
    "user": { "id": "51", "username": "my_tool" },
    "tags": ["AI", "风景"],
    "like_count": 0,
    "audit_status": "pending",
    "created_at": "2026-08-02T01:12:24.000Z"
  }
}`,
  },
  {
    id: 'ep-update', method: 'PUT', methodCls: 'is-put', path: '/api/content/:id',
    badge: '需 upload 权限', badgeCls: 'is-auth',
    descHtml: `编辑自己的内容，请求体同上传（<code>multipart/form-data</code>）；
            仅能修改密钥所属账号自己的内容，至少提供一个待修改字段。`,
    params: { title: '请求参数（path / form-data）', head: ['字段', '位置', '类型', '必填', '说明'], hasLoc: true, rows: [
      { name: 'id', loc: 'path', type: 'number', required: '是', descHtml: `要修改的内容 ID` },
      { name: 'title', loc: 'form', type: 'string', required: '否', descHtml: `新标题` },
      { name: 'content', loc: 'form', type: 'string', required: '否', descHtml: `新 Markdown 描述` },
      { name: 'tags', loc: 'form', type: 'string', required: '否', descHtml: `新标签，逗号分隔` },
      { name: 'file', loc: 'form', type: 'file', required: '否', descHtml: `替换媒体文件（≤20MB）` },
    ] },
  },
  {
    id: 'ep-delete', method: 'DELETE', methodCls: 'is-delete', path: '/api/content/:id',
    badge: '需 delete 权限', badgeCls: 'is-auth',
    descHtml: `删除内容（物理删除，不可恢复），仅能删除密钥所属账号自己的内容。响应示例：`,
    response: `{
  "code": 200,
  "message": "已删除",
  "data": null
}`,
  },
]

function scrollToSection(id: string) {
  activeToc.value = id
  lockUntil = Date.now() + 700
  nextTick(() => {
    document.getElementById(id)?.scrollIntoView({ behavior: scrollBehavior(), block: 'start' })
  })
}

// 按目录顺序取第一个进入视口的章节作为当前高亮项
function syncActiveFromViewport() {
  if (Date.now() < lockUntil) return
  const firstVisible = tocItems.find((item) => visibleSections.has(item.id))
  if (firstVisible) activeToc.value = firstVisible.id
}

function setupObserver() {
  // jsdom 与老浏览器没有 IntersectionObserver：保持点击切换，不抛错
  if (typeof IntersectionObserver === 'undefined') return
  observer?.disconnect()
  visibleSections.clear()
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) visibleSections.add(entry.target.id)
        else visibleSections.delete(entry.target.id)
      })
      syncActiveFromViewport()
    },
    { rootMargin: '-96px 0px -55% 0px', threshold: 0 },
  )
  tocItems.forEach((item) => {
    const el = document.getElementById(item.id)
    if (el) observer?.observe(el)
  })
}

function highlightBlocks() {
  document.querySelectorAll('.api-docs pre:not(.no-hl) code').forEach((el) => {
    const node = el as HTMLElement
    // 连续渲染下同一节点会被多次收集，已高亮过就跳过，避免 hljs 重复告警
    if (node.dataset.highlighted) return
    hljs.highlightElement(node)
  })
}

watch(activeLang, () => nextTick(highlightBlocks))
onMounted(() => {
  nextTick(() => {
    highlightBlocks()
    setupObserver()
  })
})
onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
  visibleSections.clear()
})

function copyActiveExample() {
  const code = codeExamples[activeLang.value]
  if (!code) return
  navigator.clipboard
    ?.writeText(code)
    .then(() => toast.success('示例已复制'))
    .catch(() => { /* 剪贴板不可用时静默 */ })
}

function copyFrame(e: MouseEvent) {
  const btn = e.currentTarget as HTMLElement
  const code = btn.closest('.code-frame')?.querySelector('code')?.textContent?.trim() ?? ''
  if (!code) return
  navigator.clipboard
    ?.writeText(code)
    .then(() => toast.success('已复制'))
    .catch(() => { /* 剪贴板不可用时静默 */ })
}
</script>

<template>
  <div class="api-docs">
    <div class="api-overview">
      <h3 class="api-docs-head">API 使用说明</h3>
      <p class="api-docs-intro">
        密钥用于第三方应用以编程方式调用本站内容接口。请求统一走
        <code>https://xq.xiey.work/api</code>（本地开发为
        <code>http://localhost:3000/api</code>），受保护接口通过请求头
        <code>X-API-Key</code> 认证，所有响应统一为
        <code>{ code, message, data }</code> 结构。
      </p>
      <a-alert type="warning" class="api-alert">
        <template #title>密钥安全</template>
        完整密钥仅在创建时展示一次，关闭后无法再次查看；请复制后妥善保存，
        一旦泄露请立即在本页「撤销」该密钥。
      </a-alert>
    </div>

    <div class="api-body">
      <main class="api-main">
        <!-- 认证与权限 -->
        <section id="sec-auth" class="api-section">
          <h4 class="api-section-title">认证与权限</h4>
          <p class="api-desc">
            在请求头中携带密钥即可完成认证（密钥形如 <code>xq_</code> 开头，共 51 位）：
          </p>
          <div class="code-frame">
            <div class="code-frame-head">
              <span class="code-frame-label">请求头示例</span>
              <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
            </div>
            <pre class="api-code no-hl"><code>X-API-Key: xq_你的完整密钥</code></pre>
          </div>
          <div class="api-table-wrap">
            <table class="api-table">
              <thead>
                <tr><th>权限</th><th>允许的接口</th></tr>
              </thead>
              <tbody>
                <tr>
                  <td><code>upload</code> 上传内容</td>
                  <td>
                    <code>POST /api/content/upload</code>、
                    <code>PUT /api/content/:id</code>
                  </td>
                </tr>
                <tr>
                  <td><code>delete</code> 删除内容</td>
                  <td><code>DELETE /api/content/:id</code>（仅能删除密钥所属账号自己的内容）</td>
                </tr>
                <tr>
                  <td><code>read</code> 读取内容</td>
                  <td>列表 / 详情为公开接口，无需密钥；该权限为后续私有接口预留</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 统一响应格式 -->
        <section id="sec-response" class="api-section">
          <h4 class="api-section-title">统一响应格式</h4>
          <p class="api-desc">
            所有接口返回统一结构：<code>code</code> 为状态码、<code>message</code>
            为提示、<code>data</code> 为业务数据（失败时通常为 null）。
          </p>
          <div class="code-frame">
            <div class="code-frame-head">
              <span class="code-frame-label">通用响应结构</span>
              <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
            </div>
            <pre class="api-code"><code class="language-json">{
  "code": 200,
  "message": "ok",
  "data": { }
}</code></pre>
          </div>
          <div class="api-table-wrap">
            <table class="api-table">
              <thead>
                <tr><th>code</th><th>含义</th></tr>
              </thead>
              <tbody>
                <tr><td>400</td><td>请求参数不合法（缺 title、正文与文件均为空等）</td></tr>
                <tr><td>401</td><td>未登录，或 X-API-Key 缺失 / 无效 / 已撤销</td></tr>
                <tr><td>403</td><td>密钥缺少所需权限，或无权操作该内容</td></tr>
                <tr><td>404</td><td>内容不存在</td></tr>
                <tr><td>413</td><td>上传文件超过 20MB 限制</td></tr>
                <tr><td>429</td><td>触发频率限制（快速上传 1 小时 20 次）</td></tr>
                <tr><td>500</td><td>服务器内部错误</td></tr>
              </tbody>
            </table>
          </div>
          <p class="api-note">
            注意字段类型：<code>id</code>、<code>user.id</code>、<code>file_size</code>
            等 bigint 字段在 JSON 中返回字符串；<code>created_at</code> /
            <code>updated_at</code> 为 ISO 8601 时间字符串。
          </p>
        </section>

        <!-- 5 个接口卡片连续排列，目录负责定位 -->
        <section v-for="card in endpointCards" :id="card.id" :key="card.id" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method" :class="card.methodCls">{{ card.method }}</span>
            <code class="api-ep-path">{{ card.path }}</code>
            <span class="api-ep-badge" :class="card.badgeCls">{{ card.badge }}</span>
          </div>
          <p class="api-desc" v-html="card.descHtml"></p>
          <p v-if="card.noteHtml" class="api-note" v-html="card.noteHtml"></p>
          <div v-if="card.params" class="api-ep-block">
            <div class="api-ep-label">{{ card.params.title }}</div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr><th v-for="h in card.params.head" :key="h">{{ h }}</th></tr>
                </thead>
                <tbody>
                  <tr v-for="row in card.params.rows" :key="row.name">
                    <td><code>{{ row.name }}</code></td>
                    <td v-if="card.params.hasLoc">{{ row.loc }}</td>
                    <td>{{ row.type }}</td>
                    <td>{{ row.required }}</td>
                    <td v-html="row.descHtml"></td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div v-if="card.response" class="api-ep-block">
            <div class="api-ep-label">响应示例</div>
            <div class="code-frame">
              <div class="code-frame-head">
                <span class="code-frame-label">响应示例</span>
                <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
              </div>
              <pre class="api-json"><code class="language-json">{{ card.response }}</code></pre>
            </div>
          </div>
        </section>

        <!-- 语言示例 -->
        <section id="sec-examples" class="api-section">
          <div class="api-examples-head">
            <h4 class="api-section-title">语言示例</h4>
            <a-button size="mini" @click="copyActiveExample">复制当前示例</a-button>
          </div>
          <p class="api-desc">以「上传 → 列表 → 删除」完整链路为例（把密钥换成你自己的）：</p>
          <a-tabs v-model:active-key="activeLang" class="api-code-tabs" type="card-gutter">
            <a-tab-pane v-for="(code, lang) in codeExamples" :key="lang" :title="langLabels[lang]">
              <pre class="api-code api-code-example"><code :class="'language-' + langClass[lang]">{{ code }}</code></pre>
            </a-tab-pane>
          </a-tabs>
        </section>
      </main>

      <!-- 右侧吸顶目录：IntersectionObserver 高亮当前章节 -->
      <aside class="api-toc" aria-label="文档目录">
        <div class="api-toc-group">目录</div>
        <button
          v-for="(item, index) in tocItems"
          :key="item.id"
          type="button"
          class="api-toc-item"
          :class="{ 'is-active': activeToc === item.id }"
          :aria-current="activeToc === item.id ? 'true' : undefined"
          @click="scrollToSection(item.id)"
        >
          <span class="api-toc-index">{{ index + 1 }}</span>
          {{ item.label }}
        </button>
      </aside>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use './admin' as *;

// 文档页只换皮肤：正文结构、接口内容与代码示例保持不变。
.api-docs {
  padding: var(--adm-gap-xl, 24px);
  color: var(--adm-ink, var(--admin-text));
}

.api-overview {
  margin-bottom: var(--adm-gap-xl, 24px);
  padding-bottom: var(--adm-gap-lg, 16px);
  border-bottom: 1px solid var(--adm-line, var(--admin-border-soft));
}

.api-docs-head {
  margin: 0 0 var(--adm-gap-sm, 8px);
  font-family: var(--site-title-font);
  font-size: var(--adm-font-xl, 21px);
  font-weight: 600;
  line-height: 1.35;
  letter-spacing: -0.01em;
  color: var(--adm-ink, var(--admin-text));
}

.api-docs-intro {
  max-width: 760px;
  margin: 0;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.9;
  color: var(--adm-muted, var(--admin-text-2));

  code {
    padding: 1px 5px;
    border-radius: 4px;
    background: var(--adm-soft, var(--admin-fill));
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 0.92em;
    color: var(--adm-ink, var(--admin-text));
  }
}

.api-alert {
  margin-top: var(--adm-gap-md, 12px);
  border-radius: var(--adm-r-control, 10px);
}

/* ── 双栏：正文 + 右侧吸顶目录 ── */
.api-body {
  display: flex;
  align-items: flex-start;
  gap: var(--adm-gap-xl, 24px);
}

.api-main {
  flex: 1;
  min-width: 0;
}

.api-toc {
  position: sticky;
  top: var(--adm-gap-lg, 16px);
  flex: none;
  width: 212px;
  max-height: calc(100vh - 120px);
  overflow-y: auto;
  padding: var(--adm-gap-md, 12px) var(--adm-gap-sm, 8px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
  scrollbar-width: thin;
}

.api-toc-group {
  padding: 8px 12px 6px;
  font-size: var(--adm-font-xs, 12px);
  font-weight: 600;
  letter-spacing: 0.08em;
  color: var(--adm-muted, var(--admin-text-3));
}

.api-toc-item {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-sm, 8px);
  width: 100%;
  padding: 8px 12px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.5;
  text-align: left;
  color: var(--adm-muted, var(--admin-text-2));
  cursor: pointer;
  transition:
    background var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1)),
    color var(--adm-dur-fast, 120ms) var(--adm-ease-std, cubic-bezier(0.4, 0, 0.2, 1));

  @include focus-ring(2px);

  &:hover {
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-ink, var(--admin-text));
  }

  &.is-active {
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    font-weight: 600;
    color: var(--adm-accent, var(--admin-primary));
  }
}

.api-toc-index {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 6px;
  background: var(--adm-soft, var(--admin-fill));
  font-size: 11px;
  font-weight: 600;
  color: var(--adm-muted, var(--admin-text-3));
  @include numeric;
}

.api-toc-item.is-active .api-toc-index {
  background: var(--adm-accent, var(--admin-primary));
  color: #f7f1e8;
}

/* ── 章节与接口卡片 ── */
.api-section {
  margin-bottom: var(--adm-gap-xl, 24px);
  scroll-margin-top: var(--adm-gap-lg, 16px);
}

.api-section-title {
  position: relative;
  margin: 0 0 var(--adm-gap-md, 12px);
  padding-left: 12px;
  font-family: var(--site-title-font);
  font-size: var(--adm-font-lg, 17px);
  font-weight: 600;
  line-height: 1.4;
  color: var(--adm-ink, var(--admin-text));

  &::before {
    content: "";
    position: absolute;
    top: 3px;
    bottom: 3px;
    left: 0;
    width: 3px;
    border-radius: 2px;
    background: var(--adm-accent, var(--admin-primary));
  }
}

.api-desc {
  margin: var(--adm-gap-sm, 8px) 0;
  font-size: var(--adm-font-sm, 13px);
  line-height: 1.9;
  color: var(--adm-muted, var(--admin-text-2));

  code {
    padding: 1px 5px;
    border-radius: 4px;
    background: var(--adm-soft, var(--admin-fill));
    color: var(--adm-ink, var(--admin-text));
    font-size: 0.92em;
    word-break: break-all;
  }
}

.api-endpoint-card {
  margin-bottom: var(--adm-gap-xl, 24px);
  padding: var(--adm-gap-lg, 16px) var(--adm-gap-xl, 24px) var(--adm-gap-xl, 24px);
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-card, var(--admin-radius));
  background: var(--adm-surface, var(--admin-surface));
  scroll-margin-top: var(--adm-gap-lg, 16px);
}

.api-ep-head {
  display: flex;
  align-items: center;
  gap: var(--adm-gap-md, 12px);
  margin-bottom: var(--adm-gap-md, 12px);
  padding-bottom: var(--adm-gap-md, 12px);
  border-bottom: 1px solid var(--admin-border-soft);
}

.api-ep-path {
  min-width: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-md, 15px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
  word-break: break-all;
}

/* 方法徽标：12% 色底 + 同色文字 + 描边，不用高饱和实心块 */
.api-method {
  flex: none;
  padding: 3px 10px;
  border: 1px solid transparent;
  border-radius: 7px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: var(--adm-font-xs, 12px);
  font-weight: 700;
  letter-spacing: 0.03em;

  &.is-get {
    color: var(--adm-success, rgb(var(--success-6)));
    background: color-mix(in srgb, var(--adm-success, #16a34a) 12%, transparent);
    border-color: color-mix(in srgb, var(--adm-success, #16a34a) 32%, transparent);
  }

  &.is-post {
    color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    border-color: color-mix(in srgb, var(--adm-accent, var(--admin-primary)) 32%, transparent);
  }

  &.is-put {
    color: var(--adm-warning, rgb(var(--warning-6)));
    background: color-mix(in srgb, var(--adm-warning, #d97706) 12%, transparent);
    border-color: color-mix(in srgb, var(--adm-warning, #d97706) 32%, transparent);
  }

  &.is-delete {
    color: var(--adm-danger, rgb(var(--danger-6)));
    background: color-mix(in srgb, var(--adm-danger, #dc2626) 12%, transparent);
    border-color: color-mix(in srgb, var(--adm-danger, #dc2626) 32%, transparent);
  }
}

.api-ep-badge {
  margin-left: auto;
  padding: 3px 10px;
  border: 1px solid transparent;
  border-radius: var(--adm-r-pill, 999px);
  font-size: var(--adm-font-xs, 12px);
  font-weight: 600;
  white-space: nowrap;

  &.is-public {
    color: var(--adm-muted, var(--admin-text-2));
    background: var(--adm-soft, var(--admin-fill));
    border-color: var(--adm-line, var(--admin-border-soft));
  }

  &.is-auth {
    color: var(--adm-accent, var(--admin-primary));
    background: var(--adm-accent-soft, var(--admin-primary-soft));
    border-color: color-mix(in srgb, var(--adm-accent, var(--admin-primary)) 26%, transparent);
  }
}

.api-ep-block {
  margin-top: var(--adm-gap-lg, 16px);
}

.api-ep-label {
  margin-bottom: var(--adm-gap-sm, 8px);
  font-size: var(--adm-font-sm, 13px);
  font-weight: 600;
  color: var(--adm-ink, var(--admin-text));
}

/* ── 表格：圆角容器，去双线外框 ── */
.api-table-wrap {
  overflow: hidden;
  border: 1px solid var(--adm-line, var(--admin-border-soft));
  border-radius: var(--adm-r-control, 12px);
}

.api-table {
  width: 100%;
  margin: 0;
  border: none;
  border-collapse: collapse;
  font-size: var(--adm-font-xs, 12px);

  th,
  td {
    padding: 10px 14px;
    border: none;
    border-right: 1px solid var(--admin-border-soft);
    border-bottom: 1px solid var(--admin-border-soft);
    text-align: left;
    vertical-align: top;
    line-height: 1.6;
  }

  th {
    background: var(--adm-soft, var(--admin-fill));
    font-weight: 600;
    white-space: nowrap;
    color: var(--adm-muted, var(--admin-text-2));
  }

  td {
    color: var(--adm-ink, var(--admin-text));
  }

  th:last-child,
  td:last-child {
    border-right: none;
  }

  tbody tr:last-child td {
    border-bottom: none;
  }

  tbody tr:hover {
    background: var(--adm-soft, var(--admin-fill));
  }

  code {
    padding: 1px 4px;
    border-radius: 4px;
    background: var(--adm-soft, var(--admin-fill));
    font-size: 0.94em;
    word-break: break-all;
  }
}

/* ── 代码块：暖调深色恒底，不随亮/暗主题翻转 ── */
.code-frame {
  margin: var(--adm-gap-sm, 8px) 0 var(--adm-gap-md, 12px);
  overflow: hidden;
  border: 1px solid #3a322c;
  border-radius: var(--adm-r-control, 12px);
  background: #1a1614;
}

.code-frame-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-sm, 8px);
  padding: 6px 8px 6px 14px;
  border-bottom: 1px solid #3a322c;
  background: #241f1b;
}

.code-frame-label {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.05em;
  color: #a89c8c;
}

.code-frame-copy {
  flex: none;
  min-height: var(--adm-touch, 36px);
  color: #a89c8c;

  &:hover {
    background: rgba(255, 255, 255, 0.06);
    color: #ece5da;
  }
}

.api-code,
.api-json {
  max-width: 100%;
  margin: 0;
  padding: 16px 18px;
  overflow: auto;
  border: 0;
  border-radius: 0;
  background: transparent;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12.5px;
  line-height: 1.65;
  color: #ece5da;

  code {
    padding: 0;
    background: transparent;
    font-family: inherit;
    color: inherit;
  }
}

/* 字符串值用暖白，避免 github-dark 的冷色铺在暖底上 */
.api-code :deep(.hljs-string),
.api-json :deep(.hljs-string) {
  color: #f2e8d8;
}

.api-json {
  max-height: 360px;
}

.api-note {
  margin-top: var(--adm-gap-md, 12px);
  padding: 10px 14px;
  border-left: 3px solid var(--adm-accent, var(--admin-primary));
  border-radius: 0 10px 10px 0;
  background: var(--adm-accent-soft, var(--admin-primary-soft));
  font-size: var(--adm-font-xs, 12px);
  line-height: 1.7;
  color: var(--adm-muted, var(--admin-text-2));

  code {
    padding: 1px 4px;
    border-radius: 4px;
    background: var(--adm-surface, var(--admin-surface));
    color: var(--adm-ink, var(--admin-text));
  }
}

/* ── 语言示例 ── */
.api-examples-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--adm-gap-md, 12px);
  margin-bottom: var(--adm-gap-xs, 4px);
}

.api-code-tabs {
  margin-top: var(--adm-gap-xs, 4px);

  :deep(.arco-tabs-pane) {
    padding-top: var(--adm-gap-xs, 4px);
  }
}

.api-code-example {
  margin: var(--adm-gap-sm, 8px) 0 var(--adm-gap-xs, 4px);
  border: 1px solid #3a322c;
  border-radius: var(--adm-r-control, 12px);
  background: #1a1614;
  white-space: pre;
}

/* ── ≤820：目录变吸顶横向条，置顶 ── */
@media (max-width: 820px) {
  .api-body {
    flex-direction: column;
    gap: var(--adm-gap-lg, 16px);
  }

  .api-toc {
    order: -1;
    top: 0;
    z-index: 2;
    display: flex;
    flex-wrap: nowrap;
    gap: 2px;
    width: 100%;
    max-height: none;
    padding: 6px;
    overflow-x: auto;
    overflow-y: hidden;
  }

  .api-toc-group {
    display: none;
  }

  .api-toc-item {
    flex: none;
    width: auto;
    min-height: 40px;
    padding: 4px 10px;
    font-size: var(--adm-font-xs, 12px);
    white-space: nowrap;
  }

  .api-toc-index {
    display: none;
  }

  .api-endpoint-card {
    padding: var(--adm-gap-lg, 16px);
  }

  .api-table-wrap {
    overflow-x: auto;
  }
}

@media (max-width: 768px) {
  .api-docs {
    padding: var(--adm-gap-lg, 16px);
  }

  .api-toc-item {
    min-height: 44px;
  }

  .api-code,
  .api-json {
    padding: 12px 14px;
    font-size: 12px;
  }
}
</style>

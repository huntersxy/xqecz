<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import python from 'highlight.js/lib/languages/python'
import javascript from 'highlight.js/lib/languages/javascript'
import powershell from 'highlight.js/lib/languages/powershell'
import json from 'highlight.js/lib/languages/json'
import 'highlight.js/styles/github-dark.css'
import { toast } from '@/composables/useToast'

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

// 目录即“分页”列表：一次只展示一个条目，通过侧栏或上/下页切换。
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
const activeIndex = computed(() =>
  Math.max(0, tocItems.findIndex((item) => item.id === activeToc.value)),
)

function scrollToSection(id: string) {
  activeToc.value = id
  nextTick(() => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function goPrev() {
  const prev = tocItems[activeIndex.value - 1]
  if (prev) scrollToSection(prev.id)
}

function goNext() {
  const next = tocItems[activeIndex.value + 1]
  if (next) scrollToSection(next.id)
}

function highlightBlocks() {
  document.querySelectorAll('.api-docs pre:not(.no-hl) code').forEach((el) => {
    hljs.highlightElement(el as HTMLElement)
  })
}

watch(activeToc, () => nextTick(highlightBlocks))
watch(activeLang, () => nextTick(highlightBlocks))
onMounted(() => nextTick(highlightBlocks))

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
      <aside class="api-toc">
        <div class="api-toc-group">目录</div>
        <button
          v-for="(item, index) in tocItems"
          :key="item.id"
          type="button"
          class="api-toc-item"
          :class="{ 'is-active': activeToc === item.id }"
          @click="scrollToSection(item.id)"
        >
          <span class="api-toc-index">{{ index + 1 }}</span>
          {{ item.label }}
        </button>
      </aside>

      <main class="api-main">
        <!-- 认证与权限 -->
        <section v-if="activeToc === 'sec-auth'" id="sec-auth" class="api-section">
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
        <section v-else-if="activeToc === 'sec-response'" id="sec-response" class="api-section">
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

        <!-- GET list -->
        <section v-else-if="activeToc === 'ep-list'" id="ep-list" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method is-get">GET</span>
            <code class="api-ep-path">/api/content/list</code>
            <span class="api-ep-badge is-public">公开</span>
          </div>
          <p class="api-desc">分页获取已审核内容列表。</p>
          <div class="api-ep-block">
            <div class="api-ep-label">请求参数（query）</div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr><th>参数</th><th>类型</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr><td><code>page</code></td><td>number</td><td>否</td><td>页码，默认 1</td></tr>
                <tr><td><code>page_size</code></td><td>number</td><td>否</td><td>每页条数，默认 20，最大 100</td></tr>
                <tr><td><code>sort_by</code></td><td>string</td><td>否</td><td>created_at / view_count / id，默认 created_at</td></tr>
                <tr><td><code>order</code></td><td>string</td><td>否</td><td>desc / asc，默认 desc</td></tr>
                <tr><td><code>tag</code></td><td>string</td><td>否</td><td>标签过滤，多个用英文逗号分隔</td></tr>
                <tr><td><code>type</code></td><td>string</td><td>否</td><td>内容类型：image / text</td></tr>
                  <tr><td><code>keyword</code></td><td>string</td><td>否</td><td>标题关键词（完整搜索见 <code>/api/content/search</code>）</td></tr>
                </tbody>
              </table>
            </div>
          </div>
          <div class="api-ep-block">
            <div class="api-ep-label">响应示例</div>
            <div class="code-frame">
              <div class="code-frame-head">
                <span class="code-frame-label">响应示例</span>
                <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
              </div>
              <pre class="api-json"><code class="language-json">{
  "code": 200,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": "814",
        "title": "晓晓自拍",
        "type": "image",
        "text": "",
        "thumb": "/thumbs/xxx_thumb.jpg",
        "img": "/images/xxx_tinified.webp",
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
}</code></pre>
            </div>
          </div>
        </section>

        <!-- GET detail -->
        <section v-else-if="activeToc === 'ep-detail'" id="ep-detail" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method is-get">GET</span>
            <code class="api-ep-path">/api/content/:id</code>
            <span class="api-ep-badge is-public">公开</span>
          </div>
          <p class="api-desc">
            获取单个内容详情，<code>data</code> 为单个内容对象（字段同列表项）；
            可附加 <code>?silent=1</code> 跳过浏览量计数。
          </p>
          <div class="api-ep-block">
            <div class="api-ep-label">请求参数（path / query）</div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr><th>参数</th><th>位置</th><th>类型</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr><td><code>id</code></td><td>path</td><td>number</td><td>是</td><td>内容 ID</td></tr>
                  <tr><td><code>silent</code></td><td>query</td><td>string</td><td>否</td><td>传 1 时跳过浏览量计数</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <!-- POST upload -->
        <section v-else-if="activeToc === 'ep-upload'" id="ep-upload" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method is-post">POST</span>
            <code class="api-ep-path">/api/content/upload</code>
            <span class="api-ep-badge is-auth">需 upload 权限</span>
          </div>
          <p class="api-desc">上传内容，请求体为 <code>multipart/form-data</code>（文本字段与文件可同时携带）。</p>
          <p class="api-note">非 GIF 图片上传后由服务端本地无损转为 WebP 作为新原图（源文件删除），随后进入缩略图与压缩链路。</p>
          <div class="api-ep-block">
            <div class="api-ep-label">请求参数（form-data）</div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr><th>字段</th><th>类型</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr><td><code>title</code></td><td>string</td><td>是</td><td>标题，1-200 字</td></tr>
                <tr><td><code>content</code></td><td>string</td><td>否</td><td>Markdown 描述；与 file 至少填一项</td></tr>
                <tr><td><code>file</code></td><td>file</td><td>否</td><td>图片 / 视频文件（≤20MB）；非 GIF 图片自动无损转为 WebP 原图；与 content 至少填一项</td></tr>
                  <tr><td><code>tags</code></td><td>string</td><td>否</td><td>标签，多个用英文逗号分隔，如 <code>AI,风景</code></td></tr>
                </tbody>
              </table>
            </div>
          </div>
          <div class="api-ep-block">
            <div class="api-ep-label">响应示例</div>
            <div class="code-frame">
              <div class="code-frame-head">
                <span class="code-frame-label">响应示例</span>
                <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
              </div>
              <pre class="api-json"><code class="language-json">{
  "code": 200,
  "message": "上传成功",
  "data": {
    "id": "815",
    "title": "我的作品",
    "type": "text",
    "text": "**Markdown** 描述",
    "thumb": "",
    "img": "",
    "user": { "id": "51", "username": "my_tool" },
    "tags": ["AI", "风景"],
    "like_count": 0,
    "audit_status": "pending",
    "created_at": "2026-08-02T01:12:24.000Z"
  }
}</code></pre>
            </div>
          </div>
        </section>

        <!-- PUT update -->
        <section v-else-if="activeToc === 'ep-update'" id="ep-update" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method is-put">PUT</span>
            <code class="api-ep-path">/api/content/:id</code>
            <span class="api-ep-badge is-auth">需 upload 权限</span>
          </div>
          <p class="api-desc">
            编辑自己的内容，请求体同上传（<code>multipart/form-data</code>）；
            仅能修改密钥所属账号自己的内容，至少提供一个待修改字段。
          </p>
          <div class="api-ep-block">
            <div class="api-ep-label">请求参数（path / form-data）</div>
            <div class="api-table-wrap">
              <table class="api-table">
                <thead>
                  <tr><th>字段</th><th>位置</th><th>类型</th><th>必填</th><th>说明</th></tr>
              </thead>
              <tbody>
                <tr><td><code>id</code></td><td>path</td><td>number</td><td>是</td><td>要修改的内容 ID</td></tr>
                <tr><td><code>title</code></td><td>form</td><td>string</td><td>否</td><td>新标题</td></tr>
                <tr><td><code>content</code></td><td>form</td><td>string</td><td>否</td><td>新 Markdown 描述</td></tr>
                <tr><td><code>tags</code></td><td>form</td><td>string</td><td>否</td><td>新标签，逗号分隔</td></tr>
                  <tr><td><code>file</code></td><td>form</td><td>file</td><td>否</td><td>替换媒体文件（≤20MB）</td></tr>
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <!-- DELETE -->
        <section v-else-if="activeToc === 'ep-delete'" id="ep-delete" class="api-endpoint-card">
          <div class="api-ep-head">
            <span class="api-method is-delete">DELETE</span>
            <code class="api-ep-path">/api/content/:id</code>
            <span class="api-ep-badge is-auth">需 delete 权限</span>
          </div>
          <p class="api-desc">软删除内容，仅能删除密钥所属账号自己的内容。响应示例：</p>
          <div class="api-ep-block">
            <div class="api-ep-label">响应示例</div>
            <div class="code-frame">
              <div class="code-frame-head">
                <span class="code-frame-label">响应示例</span>
                <a-button size="mini" type="text" class="code-frame-copy" @click="copyFrame">复制</a-button>
              </div>
              <pre class="api-json"><code class="language-json">{
  "code": 200,
  "message": "已删除",
  "data": null
}</code></pre>
            </div>
          </div>
        </section>

        <!-- 语言示例 -->
        <section v-else id="sec-examples" class="api-section">
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

        <!-- 分页切换 -->
        <div class="api-pager">
          <a-button size="small" :disabled="activeIndex <= 0" @click="goPrev">‹ 上一步</a-button>
          <span class="api-pager-info">{{ activeIndex + 1 }} / {{ tocItems.length }}</span>
          <a-button size="small" :disabled="activeIndex >= tocItems.length - 1" @click="goNext">下一步 ›</a-button>
        </div>
      </main>
    </div>
  </div>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.api-docs {
  margin-top: 12px;
  color: $admin-text;
}

.api-overview {
  margin-bottom: 18px;
}

.api-docs-head {
  margin: 0 0 8px;
  font-size: $admin-font-lg;
  font-weight: 700;
}

.api-docs-intro {
  margin: 0 0 12px;
  font-size: $admin-font-sm;
  line-height: 1.7;
  color: $admin-text-2;

  code {
    padding: 1px 5px;
    border-radius: 4px;
    background: $admin-fill;
    font-size: 0.92em;
  }
}

.api-alert {
  margin-bottom: 0;
}

/* ── 双栏：左侧目录 + 右侧内容 ── */
.api-body {
  display: flex;
  align-items: flex-start;
  gap: 18px;
}

.api-toc {
  position: sticky;
  top: 12px;
  flex-shrink: 0;
  width: 200px;
  max-height: calc(100vh - 120px);
  overflow-y: auto;
  padding: 10px 8px;
  border: 1px solid $admin-border-soft;
  border-radius: 12px;
  background: $admin-surface;
}

.api-toc-group {
  padding: 8px 10px 4px;
  font-size: $admin-font-xs;
  font-weight: 600;
  color: $admin-text-3;
}

.api-toc-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 10px;
  border: none;
  border-radius: 8px;
  background: transparent;
  text-align: left;
  font-size: $admin-font-sm;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: $admin-text-2;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;

  &:hover {
    background: $admin-fill;
    color: $admin-text;
  }

  &.is-active {
    background: $admin-primary-soft;
    color: $admin-primary;
    font-weight: 600;
  }
}

.api-toc-index {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  border-radius: 5px;
  background: $admin-fill;
  color: $admin-text-3;
  font-size: 11px;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.api-toc-item.is-active .api-toc-index {
  background: $admin-primary;
  color: #fff;
}

.api-main {
  flex: 1;
  min-width: 0;
}

.api-section {
  min-height: 60vh;
}

.api-section-title {
  margin: 0 0 10px;
  font-size: $admin-font-md;
  font-weight: 700;
}

.api-desc {
  margin: 6px 0;
  font-size: $admin-font-sm;
  line-height: 1.7;
  color: $admin-text-2;

  code {
    padding: 1px 5px;
    border-radius: 4px;
    background: $admin-fill;
    font-size: 0.92em;
  }
}

/* ── 接口卡片（Apifox 风格）── */
.api-endpoint-card {
  min-height: 60vh;
  padding: 16px 18px;
  border: 1px solid $admin-border-soft;
  border-radius: 12px;
  background: $admin-surface;
}

.api-ep-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 12px;
  border-bottom: 1px solid $admin-border-soft;
}

.api-ep-path {
  font-size: $admin-font-base;
  font-weight: 700;
  word-break: break-all;
}

.api-ep-badge {
  margin-left: auto;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: $admin-font-xs;
  font-weight: 600;
  white-space: nowrap;

  &.is-public {
    background: $admin-fill;
    color: $admin-text-2;
  }

  &.is-auth {
    background: $admin-primary-soft;
    color: $admin-primary;
  }
}

.api-method {
  flex-shrink: 0;
  padding: 3px 10px;
  border-radius: 6px;
  font-size: $admin-font-xs;
  font-weight: 700;
  color: #fff;
  letter-spacing: 0.03em;

  &.is-get { background: rgb(var(--success-6)); }
  &.is-post { background: rgb(var(--primary-6)); }
  &.is-put { background: rgb(var(--warning-6)); }
  &.is-delete { background: rgb(var(--danger-6)); }
}

.api-ep-block {
  margin-top: 14px;
}

.api-ep-label {
  margin-bottom: 8px;
  font-size: $admin-font-sm;
  font-weight: 700;
  color: $admin-text;
}

/* ── 表格 / 代码（深色高亮，贴合 Apifox）── */
.api-table {
  width: 100%;
  border-collapse: collapse;
  font-size: $admin-font-xs;

  th,
  td {
    padding: 8px 10px;
    border: 1px solid $admin-border-soft;
    text-align: left;
    vertical-align: top;
    line-height: 1.6;
  }

  th {
    background: $admin-fill;
    font-weight: 600;
    white-space: nowrap;
  }

  code {
    padding: 1px 4px;
    border-radius: 4px;
    background: $admin-fill;
    font-size: 0.94em;
    word-break: break-all;
  }
}

.api-code,
.api-json {
  margin: 8px 0 12px;
  padding: 12px 14px;
  border: 1px solid #30363d;
  border-radius: 10px;
  background: #0d1117;
  /* 背景恒为黑色，文字固定浅色，不随 admin 主题变化 */
  color: #e6edf3;
  overflow: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, 'Courier New', monospace;
  font-size: 12.5px;
  line-height: 1.65;

  code {
    font-family: inherit;
    background: transparent;
    padding: 0;
    color: inherit;
  }
}

/* 请求地址等字符串值固定白色（github-dark 默认的浅蓝在黑色背景上对比不足） */
.api-code :deep(.hljs-string),
.api-json :deep(.hljs-string) {
  color: #ffffff;
}

.api-json {
  max-height: 340px;
  margin-top: 0;
}

.api-note {
  margin-top: 10px;
  padding: 8px 12px;
  border-left: 3px solid $admin-primary;
  border-radius: 0 8px 8px 0;
  background: $admin-primary-soft;
  font-size: $admin-font-xs;
  line-height: 1.7;
}

/* ── 语言示例 Tab ── */
.api-examples-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.api-code-tabs {
  margin-top: 6px;

  :deep(.arco-tabs-pane) {
    padding-top: 4px;
  }
}

.api-code-example {
  margin: 8px 0 0;
  white-space: pre;
}

/* ── 分页 ── */
.api-pager {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid $admin-border-soft;
}

.api-pager-info {
  font-size: $admin-font-xs;
  color: $admin-text-3;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 1100px) {
  .api-body {
    flex-direction: column;
  }

  .api-toc {
    position: static;
    width: 100%;
    max-height: none;
    display: flex;
    flex-wrap: wrap;
    gap: 2px;

    .api-toc-group {
      display: none;
    }
  }

  .api-toc-item {
    width: auto;
    padding: 4px 10px;
    font-size: $admin-font-xs;
  }

  .api-toc-index {
    display: none;
  }
}

/* ═══ 重新设计：整体留白、圆角与代码块质感 ═══ */
.api-docs {
  margin-top: 0;
  padding: 24px 28px 32px;
}

.api-overview {
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid $admin-border-soft;
}

.api-docs-head {
  margin-bottom: 10px;
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.api-docs-intro {
  max-width: 760px;
  font-size: 13.5px;
  line-height: 1.9;
}

.api-alert {
  margin-top: 14px;
  border-radius: 10px;
}

.api-body {
  gap: 24px;
}

.api-toc {
  width: 212px;
  padding: 12px 10px;
  border-radius: 14px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.api-toc-group {
  padding: 12px 12px 6px;
  font-size: 11px;
  letter-spacing: 0.08em;
}

.api-toc-item {
  padding: 8px 12px;
  border-radius: 9px;
  font-size: 13px;
  line-height: 1.5;
}

.api-toc-index {
  width: 20px;
  height: 20px;
  border-radius: 6px;
  font-size: 11px;
}

.api-main {
  padding-top: 2px;
}

.api-section {
  margin-bottom: 28px;
}

.api-section-title {
  position: relative;
  margin-bottom: 14px;
  padding-left: 12px;
  font-size: 16px;

  &::before {
    content: '';
    position: absolute;
    left: 0;
    top: 2px;
    bottom: 2px;
    width: 3px;
    border-radius: 2px;
    background: $admin-primary;
  }
}

.api-desc {
  margin: 8px 0;
  font-size: 13px;
  line-height: 1.9;
}

.api-endpoint-card {
  margin-bottom: 28px;
  padding: 20px 24px 24px;
  border-radius: 14px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.api-ep-head {
  gap: 12px;
  padding-bottom: 14px;
  margin-bottom: 14px;
}

.api-ep-path {
  font-size: 15px;
}

.api-method {
  padding: 4px 12px;
  border-radius: 7px;
  font-size: 12px;
}

.api-ep-badge {
  padding: 3px 10px;
}

.api-ep-block {
  margin-top: 18px;
}

.api-ep-label {
  margin-bottom: 10px;
  font-size: 13px;
}

/* 表格：圆角容器，去掉双线外框 */
.api-table-wrap {
  border: 1px solid $admin-border-soft;
  border-radius: 12px;
  overflow: hidden;
}

.api-table {
  margin: 0;
  border: none;
}

.api-table th,
.api-table td {
  padding: 10px 14px;
  border: none;
  border-right: 1px solid $admin-border-soft;
  border-bottom: 1px solid $admin-border-soft;
}

.api-table th:last-child,
.api-table td:last-child {
  border-right: none;
}

.api-table tbody tr:last-child td {
  border-bottom: none;
}

.api-table tbody tr:hover {
  background: $admin-fill;
}

/* 代码块：标题栏 + 内容体（深色恒底） */
.code-frame {
  margin: 10px 0 14px;
  border: 1px solid #30363d;
  border-radius: 12px;
  overflow: hidden;
  background: #0d1117;
}

.code-frame-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px 6px 14px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
}

.code-frame-label {
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.05em;
  color: #8b949e;
}

.code-frame-copy {
  color: #8b949e;

  &:hover {
    color: #e6edf3;
    background: rgba(255, 255, 255, 0.06);
  }
}

.api-code,
.api-json {
  margin: 0;
  padding: 16px 18px;
  border: none;
  border-radius: 0;
  background: transparent;
}

.api-json {
  max-height: 360px;
}

/* 语言示例 Tab 内的代码块独立成框 */
.api-code-example {
  margin: 10px 0 4px;
  border: 1px solid #30363d;
  border-radius: 12px;
  background: #0d1117;
}

.api-note {
  margin-top: 14px;
  padding: 10px 14px;
  border-radius: 0 10px 10px 0;
}

.api-examples-head {
  margin-bottom: 4px;
}

.api-pager {
  margin-top: 24px;
  padding-top: 20px;
  gap: 20px;
}

@media (max-width: 1100px) {
  .api-docs {
    padding: 18px 16px 24px;
  }

  .api-toc {
    box-shadow: none;
  }
}
</style>

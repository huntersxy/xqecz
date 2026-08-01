<template>
  <div class="api-docs">
    <a-divider class="api-docs-divider" />

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

    <h4>一、认证与权限</h4>
    <p>在请求头中携带密钥即可完成认证（密钥形如 <code>xq_</code> 开头，共 51 位）：</p>
    <pre class="api-code"><code>X-API-Key: xq_你的完整密钥</code></pre>
    <p>创建密钥时勾选的权限与接口的对应关系：</p>
    <table class="api-table">
      <thead>
        <tr><th>权限</th><th>允许的接口</th></tr>
      </thead>
      <tbody>
        <tr>
          <td><code>upload</code>（上传内容）</td>
          <td>
            <code>POST /api/content/upload</code>、
            <code>POST /api/content/upload-image</code>、
            <code>PUT /api/content/:id</code>
          </td>
        </tr>
        <tr>
          <td><code>delete</code>（删除内容）</td>
          <td><code>DELETE /api/content/:id</code>（仅能删除密钥所属账号自己的内容）</td>
        </tr>
        <tr>
          <td><code>read</code>（读取内容）</td>
          <td>列表 / 详情为公开接口，无需密钥即可访问；该权限为后续私有接口预留</td>
        </tr>
      </tbody>
    </table>

    <h4>二、统一响应格式</h4>
    <pre class="api-code"><code>{
  "code": 200,        // 200 成功；非 200 见下方错误码
  "message": "ok",    // 人类可读的提示信息
  "data": { ... }     // 业务数据；失败时通常为 null
}</code></pre>
    <table class="api-table">
      <thead>
        <tr><th>code</th><th>含义</th></tr>
      </thead>
      <tbody>
        <tr><td>400</td><td>请求参数不合法（缺 title、正文与文件均为空等）</td></tr>
        <tr><td>401</td><td>未登录，或 X-API-Key 缺失 / 无效 / 已撤销</td></tr>
        <tr><td>403</td><td>密钥缺少所需权限，或无权操作该内容</td></tr>
        <tr><td>404</td><td>内容不存在</td></tr>
        <tr><td>429</td><td>触发频率限制（快速上传 1 小时 20 次）</td></tr>
        <tr><td>500</td><td>服务器内部错误</td></tr>
      </tbody>
    </table>
    <p class="api-note">
      注意字段类型：<code>id</code>、<code>user.id</code>、<code>file_size</code>
      等 bigint 字段在 JSON 中返回<b>字符串</b>；<code>created_at</code> /
      <code>updated_at</code> 为 ISO 8601 时间字符串。
    </p>

    <h4>三、接口明细</h4>

    <div class="api-endpoint">
      <span class="api-method is-get">GET</span>
      <code>/api/content/list</code>
      <span class="api-public">公开</span>
    </div>
    <p class="api-desc">分页获取已审核内容列表。</p>
    <table class="api-table">
      <thead>
        <tr><th>参数</th><th>位置</th><th>类型</th><th>必填</th><th>说明</th></tr>
      </thead>
      <tbody>
        <tr><td><code>page</code></td><td>query</td><td>number</td><td>否</td><td>页码，默认 1</td></tr>
        <tr><td><code>page_size</code></td><td>query</td><td>number</td><td>否</td><td>每页条数，默认 20，最大 100</td></tr>
        <tr><td><code>sort_by</code></td><td>query</td><td>string</td><td>否</td><td>created_at / view_count / id，默认 created_at</td></tr>
        <tr><td><code>order</code></td><td>query</td><td>string</td><td>否</td><td>desc / asc，默认 desc</td></tr>
        <tr><td><code>tag</code></td><td>query</td><td>string</td><td>否</td><td>标签过滤，多个用英文逗号分隔</td></tr>
        <tr><td><code>type</code></td><td>query</td><td>string</td><td>否</td><td>内容类型：image / text</td></tr>
        <tr><td><code>keyword</code></td><td>query</td><td>string</td><td>否</td><td>标题关键词（完整搜索见 <code>/api/content/search</code>）</td></tr>
      </tbody>
    </table>
    <p class="api-desc">返回示例：</p>
    <pre class="api-code"><code>{
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

    <div class="api-endpoint">
      <span class="api-method is-get">GET</span>
      <code>/api/content/:id</code>
      <span class="api-public">公开</span>
    </div>
    <p class="api-desc">
      获取单个内容详情，<code>data</code> 为单个内容对象（字段同列表项）；
      可附加 <code>?silent=1</code> 跳过浏览量计数。
    </p>

    <div class="api-endpoint">
      <span class="api-method is-post">POST</span>
      <code>/api/content/upload</code>
      <a-tag size="small" :bordered="false" color="arcoblue">需 upload 权限</a-tag>
    </div>
    <p class="api-desc">上传内容，请求体为 <code>multipart/form-data</code>（文本字段与文件可同时携带）。</p>
    <table class="api-table">
      <thead>
        <tr><th>字段</th><th>类型</th><th>必填</th><th>说明</th></tr>
      </thead>
      <tbody>
        <tr><td><code>title</code></td><td>string</td><td>是</td><td>标题，1-200 字</td></tr>
        <tr><td><code>content</code></td><td>string</td><td>否</td><td>Markdown 描述；与 file 至少填一项</td></tr>
        <tr><td><code>file</code></td><td>file</td><td>否</td><td>图片 / 视频文件（≤20MB）；与 content 至少填一项</td></tr>
        <tr><td><code>tags</code></td><td>string</td><td>否</td><td>标签，多个用英文逗号分隔，如 <code>AI,风景</code></td></tr>
      </tbody>
    </table>
    <p class="api-desc">返回示例：</p>
    <pre class="api-code"><code>{
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

    <div class="api-endpoint">
      <span class="api-method is-put">PUT</span>
      <code>/api/content/:id</code>
      <a-tag size="small" :bordered="false" color="arcoblue">需 upload 权限</a-tag>
    </div>
    <p class="api-desc">
      编辑自己的内容，请求体同上传（<code>multipart/form-data</code>）；
      仅能修改密钥所属账号自己的内容，至少提供一个待修改字段。
    </p>

    <div class="api-endpoint">
      <span class="api-method is-delete">DELETE</span>
      <code>/api/content/:id</code>
      <a-tag size="small" :bordered="false" color="red">需 delete 权限</a-tag>
    </div>
    <p class="api-desc">软删除内容，仅能删除密钥所属账号自己的内容。返回示例：</p>
    <pre class="api-code"><code>{
  "code": 200,
  "message": "已删除",
  "data": null
}</code></pre>

    <div class="api-endpoint">
      <span class="api-method is-post">POST</span>
      <code>/api/content/upload-image</code>
      <a-tag size="small" :bordered="false" color="arcoblue">需 upload 权限</a-tag>
    </div>
    <p class="api-desc">
      富文本图片上传（Markdown 编辑器插图用），请求体 <code>multipart/form-data</code>，
      仅 <code>file</code> 字段。返回 <code>data.image_url</code> 可直接用于
      <code>![alt](url)</code>。
    </p>

    <h4>四、常用语言示例</h4>
    <p class="api-desc">以「上传 → 列表 → 删除」完整链路为例（把密钥换成你自己的）：</p>

    <div class="api-lang-label">curl</div>
    <pre class="api-code"><code># 1. 上传内容（multipart/form-data）
curl -X POST "https://xq.xiey.work/api/content/upload" \
  -H "X-API-Key: xq_你的完整密钥" \
  -F "title=我的作品" \
  -F "content=**Markdown** 描述" \
  -F "tags=AI,风景" \
  -F "file=@./work.png"

# 2. 读取列表（公开，无需密钥）
curl "https://xq.xiey.work/api/content/list?page=1&page_size=20"

# 3. 删除内容（id 取上传返回的 data.id）
curl -X DELETE "https://xq.xiey.work/api/content/815" \
  -H "X-API-Key: xq_你的完整密钥"</code></pre>

    <div class="api-lang-label">Python（requests）</div>
    <pre class="api-code"><code>import requests

BASE = "https://xq.xiey.work/api"
HEADERS = {"X-API-Key": "xq_你的完整密钥"}

# 1. 上传内容（multipart/form-data）
resp = requests.post(
    f"{BASE}/content/upload",
    headers=HEADERS,
    data={
        "title": "我的作品",
        "content": "**Markdown** 描述",
        "tags": "AI,风景",
    },
    files={"file": open("work.png", "rb")},
)
data = resp.json()
content_id = data["data"]["id"]          # 字符串类型的 bigint
print(data["code"], content_id)

# 2. 读取列表
lst = requests.get(
    f"{BASE}/content/list",
    params={"page": 1, "page_size": 20},
).json()
print(lst["data"]["total"], lst["data"]["list"][0]["title"])

# 3. 删除内容
deleted = requests.delete(f"{BASE}/content/{content_id}", headers=HEADERS)
print(deleted.json())                    # {"code": 200, "message": "已删除", "data": null}</code></pre>

    <div class="api-lang-label">Node.js（fetch，需 Node 18+）</div>
    <pre class="api-code"><code>import { readFile } from 'node:fs/promises'

const BASE = 'https://xq.xiey.work/api'
const HEADERS = { 'X-API-Key': 'xq_你的完整密钥' }

// 1. 上传内容（multipart/form-data；Content-Type 由 fetch 自动带 boundary）
const file = new Blob([await readFile('./work.png')], { type: 'image/png' })
const form = new FormData()
form.append('title', '我的作品')
form.append('content', '**Markdown** 描述')
form.append('tags', 'AI,风景')
form.append('file', file, 'work.png')

const resp = await fetch(`${BASE}/content/upload`, {
  method: 'POST',
  headers: HEADERS,
  body: form,
})
const data = await resp.json()
const contentId = data.data?.id

// 2. 读取列表（公开）
const list = await fetch(`${BASE}/content/list?page=1&page_size=20`).then((r) => r.json())
console.log(list.data.total)

// 3. 删除内容
const deleted = await fetch(`${BASE}/content/${contentId}`, {
  method: 'DELETE',
  headers: HEADERS,
}).then((r) => r.json())
console.log(deleted)</code></pre>

    <div class="api-lang-label">PowerShell（7+ 支持 -Form）</div>
    <pre class="api-code"><code>$base = 'https://xq.xiey.work/api'
$headers = @{ 'X-API-Key' = 'xq_你的完整密钥' }

# 1. 上传内容（multipart/form-data）
$resp = Invoke-RestMethod -Uri "$base/content/upload" -Method Post `
  -Headers $headers -Form @{
    title   = '我的作品'
    content = '**Markdown** 描述'
    tags    = 'AI,风景'
    file    = Get-Item '.\work.png'
  }
$id = $resp.data.id
Write-Host "code=$($resp.code) id=$id"

# 2. 读取列表（公开）
$list = Invoke-RestMethod -Uri "$base/content/list?page=1&page_size=20"
Write-Host "total=$($list.data.total)"

# 3. 删除内容
$deleted = Invoke-RestMethod -Uri "$base/content/$id" -Method Delete -Headers $headers
$deleted | ConvertTo-Json -Depth 5</code></pre>
  </div>
</template>

<style lang="scss" scoped>
@use './admin' as *;

.api-docs {
  margin-top: 8px;
  color: $admin-text;
}

.api-docs-divider {
  margin: 8px 0 20px;
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
  margin-bottom: 20px;
}

h4 {
  margin: 22px 0 8px;
  font-size: $admin-font-md;
  font-weight: 700;
}

.api-docs > p {
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

.api-code {
  margin: 8px 0 12px;
  padding: 12px 14px;
  border: 1px solid $admin-border-soft;
  border-radius: 10px;
  background: $admin-surface;
  overflow-x: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, 'Courier New', monospace;
  font-size: 12.5px;
  line-height: 1.65;
  color: $admin-text;

  code {
    font-family: inherit;
  }
}

.api-table {
  width: 100%;
  margin: 8px 0 14px;
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

.api-note {
  padding: 8px 12px;
  border-left: 3px solid $admin-primary;
  border-radius: 0 8px 8px 0;
  background: $admin-primary-soft;
}

.api-endpoint {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 18px;
  padding: 10px 12px;
  border: 1px solid $admin-border-soft;
  border-radius: 10px;
  background: $admin-surface;

  > code {
    font-size: $admin-font-sm;
    font-weight: 600;
    word-break: break-all;
  }
}

.api-method {
  padding: 2px 8px;
  border-radius: 6px;
  font-size: $admin-font-xs;
  font-weight: 700;
  color: #fff;

  &.is-get { background: rgb(var(--success-6)); }
  &.is-post { background: rgb(var(--primary-6)); }
  &.is-put { background: rgb(var(--warning-6)); }
  &.is-delete { background: rgb(var(--danger-6)); }
}

.api-public {
  margin-left: auto;
  font-size: $admin-font-xs;
  color: $admin-text-3;
}

.api-desc {
  margin: 6px 0;
  font-size: $admin-font-sm;
  line-height: 1.7;
  color: $admin-text-2;
}

.api-lang-label {
  margin: 16px 0 4px;
  font-size: $admin-font-sm;
  font-weight: 700;
  color: $admin-text;
}
</style>

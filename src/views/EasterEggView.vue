<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import type { Content } from '@/types'

function getImageUrl(image: string | undefined, filePath: string | undefined): string {
  if (image) {
    return image.replace(/http:\/\/localhost:8080/, 'https://xqapi.xiey.work')
  }
  if (filePath) {
    return `https://xqapi.xiey.work/uploads/${filePath}`
  }
  return ''
}

function getPreviewText(text: string, maxLength: number = 120): string {
  if (!text) return ''
  const plainText = text
    .replace(/#{1,6}\s/g, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/```[\s\S]*?```/g, '[代码块]')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/[-*+]\s/g, '')
    .replace(/\n/g, ' ')
    .trim()
  return plainText.length > maxLength ? plainText.substring(0, maxLength) + '...' : plainText
}

const router = useRouter()
const contents = ref<Content[]>([])
const total = ref(0)
const page = ref(1)
const totalPages = ref(1)
const showQrModal = ref(false)

const mysteryMembers = ref([
  {
    id: 1,
    avatar: '🐉',
    nickname: 'SCPLin',
    role: '成员',
    signature: '龙王',
    tags: ['ESFP', '话痨', '轻松活泼']
  },
  {
    id: 2,
    avatar: '📖',
    nickname: '东文闻妙喵',
    role: '成员',
    signature: '评论家',
    tags: ['INTJ', '深度分析', '严谨']
  },
  {
    id: 3,
    avatar: '💭',
    nickname: '折光记',
    role: '成员',
    signature: '沉默终结者',
    tags: ['ENTJ', '深度话题', '高回复']
  },
  {
    id: 4,
    avatar: '🔧',
    nickname: 'brief',
    role: '成员',
    signature: '技术专家',
    tags: ['ISTP', '技术宅', '低调']
  },
  {
    id: 5,
    avatar: '✨',
    nickname: '未闻的星间',
    role: '成员',
    signature: '互动达人',
    tags: ['ENFJ', '善于交流', '社交牛']
  },
  {
    id: 6,
    avatar: '😂',
    nickname: '黎旭',
    role: '成员',
    signature: '表情包军火库',
    tags: ['ISFP', '表情包大师', '活泼']
  },
  {
    id: 7,
    avatar: '🌙',
    nickname: '饮水思源',
    role: '成员',
    signature: '夜猫子',
    tags: ['INFP', '沉思者', '安静']
  },
  {
    id: 8,
    avatar: '☀️',
    nickname: 'ᴍɪɴɢ.',
    role: '成员',
    signature: '阳角',
    tags: ['ESTJ', '高冷', '影响力']
  }
])

const displayNumber = '1098940107'

async function copyNumber() {
  window.open('https://qun.qq.com/universal-share/share?ac=1&authKey=agCMbFW0oz7gtdv%2BZON30E2Ml/XHV8T88it3K5CUu3RyWR2/6ZowlelOJgGgHEzT&busi_data=eyJncm91cENvZGUiOiIxMDk4OTQwMTA3IiwidG9rZW4iOiJMeU9lclU1dVNiMGI1QmEvam9oNkFVbW1EU29VNDdKQ3QwdW4xb1lTei9GUS9MNmJ3QnhvV1NHdmZLallCK2U0IiwidWluIjoiMTgzMzUwNTk3MiJ9&data=ThBmm-0JfST_YYCUEq8TuTwHjyFzlWtiWU_Lqz2AA1EY_1oGQhG-jMIOaUXRbf_VxIVhRwRPatY9PHDlbFFXuNY7Z5ex-ceKTfli8ZFJSyY&svctype=5&tempid=h5_group_info', '_blank')
}

// API调用已注释，暂时不获取数据
async function loadContents() {
  // try {
  //   const res = await contentApi.list({ page: page.value, page_size: pageSize.value })
  //   if (res.code === 200) {
  //     contents.value = res.data.list
  //     total.value = res.data.total
  //     totalPages.value = res.data.total_page
  //   }
  // } catch (error) {
  //   console.error('加载彩蛋内容失败', error)
  // }
  // 暂时使用模拟数据
  contents.value = []
  total.value = 0
  totalPages.value = 1
}

function goToDetail(content: Content) {
  const id = content.id || content.ID
  if (id) {
    router.push(`/content/${id}`)
  }
}

function goBack() {
  router.push('/')
}

function openQrModal() {
  showQrModal.value = true
}

function closeQrModal() {
  showQrModal.value = false
}

function goToPage(p: number) {
  if (p >= 1 && p <= totalPages.value) {
    page.value = p
    loadContents()
  }
}

onMounted(() => {
  loadContents()
})
</script>

<template>
  <div class="easter-egg-container">
    <div class="mac-window egg-window">
      <div class="mac-title-bar">
        <div class="mac-dot mac-dot-red"></div>
        <div class="mac-dot mac-dot-yellow"></div>
        <div class="mac-dot mac-dot-green"></div>
        <div class="window-title">🎁 彩蛋空间</div>
      </div>

      <div class="content-area">
        <div class="header-section">
          <div class="egg-banner">
            <div class="egg-icon">🎉</div>
            <h1 class="egg-title">欢迎来到彩蛋空间</h1>
            <p class="egg-subtitle">你发现了一个隐藏的秘密区域！</p>
          </div>

          <button @click="goBack" class="mac-btn back-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7"/>
            </svg>
            返回主页
          </button>
        </div>

        <div class="text-showcase-section">
          <div class="section-header">
            <h2 class="section-title">📝 文字展示区</h2>
          </div>
          <div class="text-content">
            <div class="text-block">
              <h3 class="text-title">👋 关于我</h3>
              <p class="text-paragraph">大家好！我是 <strong>汐兮雨</strong>，一个小泉动漫的粉丝。</p>
              <p class="text-paragraph">在这里偷偷回答一下泉姐问为什么发私信的不是我，因为我之前发送过消息了发不了新的了😭</p>
              <p class="text-paragraph">欢迎关注我的社交平台：</p>
              <div class="social-links">
                <a href="https://space.bilibili.com/98560079" target="_blank" class="social-link">
                  <svg class="social-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19.54 4.46c-1.3-1.3-3.02-2-4.86-2H9.32c-1.84 0-3.56.7-4.86 2L.46 9.32c-1.3 1.3-2 3.02-2 4.86v1.36c0 1.84.7 3.56 2 4.86l4.86 4.86c1.3 1.3 3.02 2 4.86 2h1.36c1.84 0 3.56-.7 4.86-2l4.86-4.86c1.3-1.3 2-3.02 2-4.86v-1.36c0-1.84-.7-3.56-2-4.86l-4.86-4.86zM12 16.5c-2.49 0-4.5-2.01-4.5-4.5s2.01-4.5 4.5-4.5 4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z"/>
                  </svg>
                  <span>Bilibili</span>
                </a>
                <a href="https://github.com/huntersxy" target="_blank" class="social-link">
                  <svg class="social-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
                  </svg>
                  <span>GitHub</span>
                </a>
              </div>
            </div>

            <div class="text-block">
              <h3 class="text-title">🌟 关于小泉动漫</h3>
              <p class="text-paragraph">原创手绘✍️制作成本比较大，谢谢大家点赞关注！全村的希望！喜欢的宝子们，可以充电支持一下！每天不定期1～10000更新。</p>
              <p class="text-paragraph">欢迎关注小泉动漫的B站账号：</p>
              <div class="social-links">
                <a href="https://space.bilibili.com/3546778043419394" target="_blank" class="social-link">
                  <svg class="social-icon" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19.54 4.46c-1.3-1.3-3.02-2-4.86-2H9.32c-1.84 0-3.56.7-4.86 2L.46 9.32c-1.3 1.3-2 3.02-2 4.86v1.36c0 1.84.7 3.56 2 4.86l4.86 4.86c1.3 1.3 3.02 2 4.86 2h1.36c1.84 0 3.56-.7 4.86-2l4.86-4.86c1.3-1.3 2-3.02 2-4.86v-1.36c0-1.84-.7-3.56-2-4.86l-4.86-4.86zM12 16.5c-2.49 0-4.5-2.01-4.5-4.5s2.01-4.5 4.5-4.5 4.5 2.01 4.5 4.5-2.01 4.5-4.5 4.5z"/>
                  </svg>
                  <span>小泉动漫</span>
                </a>
              </div>
            </div>

            <div class="text-block">
              <h3 class="text-title">🎊 加入我们</h3>
              <p class="text-paragraph">小泉动漫二创的QQ交流群：小泉动漫同人二创群🎊📢！</p>
              <div class="qq-group">
                <div class="qq-qrcode-wrapper" @click="openQrModal">
                  <img src="/qrcode_1778246119604.jpg" alt="QQ群二维码" class="qq-qrcode" />
                  <div class="qrcode-overlay">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="11" cy="11" r="8"/>
                      <path d="m21 21-4.35-4.35"/>
                      <path d="M11 8v6M8 11h6"/>
                    </svg>
                    <span>查看大图</span>
                  </div>
                </div>
                <div class="qq-info">
                  <div class="qq-header">
                    <div class="qq-avatar">
                      <svg viewBox="0 0 24 24" fill="currentColor">
                        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 1.34 3 3s-1.34 3-3 3-3-1.34-3-3 1.34-3 3-3zm0 14.2c-2.5 0-4.71-1.28-6-3.22.03-1.99 4-3.08 6-3.08 1.99 0 5.97 1.09 6 3.08-1.29 1.94-3.5 3.22-6 3.22z"/>
                      </svg>
                    </div>
                    <div class="qq-title-text">
                      <span class="qq-name">小泉动漫同人二创群</span>
                      <span class="qq-desc">欢迎加入一起交流</span>
                    </div>
                  </div>
                  <div class="qq-number-section">
                    <span class="qq-number-label">群号</span>
                    <span class="qq-number">{{ displayNumber }}</span>
                  </div>
                  <button class="copy-btn" @click="copyNumber">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M15 10l4.553-2.276A1 1 0 0 1 21 8.618v6.764a1 1 0 0 1-1.447.894L15 14M5 18h8a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2z"/>
                    </svg>
                    加入群聊
                  </button>
                </div>
              </div>
              <p class="text-paragraph" style="margin-top: 12px;">期待你的加入，一起交流分享！🎉</p>
            </div>
          </div>
        </div>

        <div class="mystery-members-section">
          <div class="section-header">
            <h2 class="section-title">👥 神秘群员</h2>
            <span class="section-count">共 {{ mysteryMembers.length }} 位</span>
          </div>
          <div class="mystery-grid">
            <div
              v-for="member in mysteryMembers"
              :key="member.id"
              class="mystery-card mac-card"
            >
              <div class="mystery-avatar">{{ member.avatar }}</div>
              <div class="mystery-info">
                <div class="mystery-header">
                  <h3 class="mystery-name">{{ member.nickname }}</h3>
                  <span v-if="member.role !== '成员'" :class="['mystery-role', member.role]">
                    {{ member.role }}
                  </span>
                </div>
                <p class="mystery-signature">{{ member.signature }}</p>
                <div class="mystery-tags">
                  <span v-for="tag in member.tags" :key="tag" class="mystery-tag">
                    {{ tag }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="content-section">
          <div class="section-header">
            <h2 class="section-title">神秘内容列表</h2>
            <span class="section-count">共 {{ total }} 条</span>
          </div>

          <div class="content-grid">
            <div
              v-for="content in contents"
              :key="content.id || content.ID"
              @click="goToDetail(content)"
              class="content-card mac-card egg-card"
            >
              <div class="card-media">
                <template v-if="(content.type || content.Type) === 'image'">
                  <img
                    :src="getImageUrl(content.image, content.file_path || content.FilePath)"
                    alt="内容图片"
                    class="card-image"
                  />
                </template>
                <template v-else-if="(content.type || content.Type) === 'video'">
                  <img
                    :src="getImageUrl(content.image, content.thumb_path || content.file_path || content.FilePath)"
                    alt="视频封面"
                    class="card-image"
                  />
                  <div class="play-overlay">
                    <svg class="play-icon" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8 5v14l11-7z"/>
                    </svg>
                  </div>
                </template>
                <template v-else>
                  <div class="text-preview">
                    <p class="preview-text">{{ getPreviewText(content.content || content.Content || '暂无内容') }}</p>
                  </div>
                </template>
              </div>

              <div class="card-info">
                <h3 class="card-title">{{ content.title || content.Title }}</h3>
                <div class="card-meta">
                  <span class="meta-item">
                    <svg class="meta-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                      <circle cx="12" cy="7" r="4"/>
                    </svg>
                    {{ content.user?.username || content.User?.Username }}
                  </span>
                  <div class="tags-wrapper">
                    <span v-for="tag in (content.tags || content.Tags || [])" :key="tag" class="meta-tag">
                      {{ tag }}
                    </span>
                  </div>
                </div>
                <div class="card-type">
                  <span :class="['type-badge', (content.type || content.Type)]">
                    {{ (content.type || content.Type) === 'video' ? '视频' : (content.type || content.Type) === 'image' ? '图片' : '文字' }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="contents.length === 0" class="empty-state">
            <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            <p>暂无彩蛋内容</p>
          </div>
        </div>

        <div v-if="totalPages > 1" class="pagination-section">
          <button
            @click="goToPage(page - 1)"
            :disabled="page <= 1"
            class="mac-btn pagination-btn"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M15 19l-7-7 7-7"/>
            </svg>
            上一页
          </button>
          <span class="pagination-info">第 {{ page }} / {{ totalPages }} 页</span>
          <button
            @click="goToPage(page + 1)"
            :disabled="page >= totalPages"
            class="mac-btn pagination-btn"
          >
            下一页
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M9 5l7 7-7 7"/>
            </svg>
          </button>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showQrModal" class="qr-modal-overlay" @click="closeQrModal">
        <div class="qr-modal" @click.stop>
          <button class="qr-modal-close" @click="closeQrModal">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
          <div class="qr-modal-content">
            <img src="/qrcode_1778246119604.jpg" alt="QQ群二维码" class="qr-modal-image" />
            <div class="qr-modal-info">
              <p class="qr-title">小泉动漫同人二创群</p>
              <p class="qr-number">QQ群号：1098940107</p>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.easter-egg-container {
  min-height: 100vh;
  height: 100%;
  width: 100%;
  padding: 20px;
  display: flex;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: fixed;
  top: 0;
  left: 0;
  overflow-y: auto;
}

.egg-window {
  width: 100%;
  max-width: 1200px;
  min-height: 80vh;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: white;
}

.mac-title-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  padding: 10px 16px;
  background: linear-gradient(180deg, rgba(0,0,0,0.08) 0%, rgba(0,0,0,0.02) 100%);
  border-radius: 12px 12px 0 0;
}

.window-title {
  margin-left: 16px;
  font-size: 13px;
  color: #666;
  font-weight: 500;
}

.content-area {
  padding: 24px;
  flex: 1;
  overflow-y: auto;
  background: white;
}

.header-section {
  text-align: center;
  margin-bottom: 32px;
}

.egg-banner {
  margin-bottom: 20px;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.8) 0%, rgba(118, 75, 162, 0.8) 100%);
  border-radius: 12px;
  padding: 20px;
}

.egg-icon {
  font-size: 64px;
  margin-bottom: 12px;
  animation: bounce 2s ease-in-out infinite;
}

@keyframes bounce {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.egg-title {
  font-size: 28px;
  color: #fff;
  margin: 0 0 8px 0;
  text-shadow: 0 2px 10px rgba(0,0,0,0.2);
}

.egg-subtitle {
  font-size: 16px;
  color: rgba(255,255,255,0.8);
  margin: 0;
}

.back-btn {
  margin-top: 20px;
  background: #0071e3;
  border: none;
  color: #fff;
  padding: 10px 24px;
  transition: all 0.2s ease;
}

.back-btn:hover {
  background: #0077ed;
  transform: scale(1.02);
}

.back-btn svg {
  width: 16px;
  height: 16px;
  margin-right: 8px;
}

.text-showcase-section {
  margin-bottom: 24px;
}

.text-content {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 12px;
  padding: 20px;
}

.text-block {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.text-block:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.text-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 12px 0;
}

.text-paragraph {
  font-size: 14px;
  color: #666;
  line-height: 1.7;
  margin: 0 0 10px 0;
}

.text-paragraph:last-child {
  margin-bottom: 0;
}

.text-list {
  margin: 0;
  padding-left: 20px;
}

.text-list li {
  font-size: 14px;
  color: #666;
  line-height: 1.8;
  margin-bottom: 8px;
}

.text-list li:last-child {
  margin-bottom: 0;
}

.inline-code {
  background: rgba(0, 0, 0, 0.08);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Fira Code', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  color: #d97706;
}

.social-links {
  display: flex;
  gap: 12px;
  margin-top: 12px;
}

.social-link {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: rgba(0, 0, 0, 0.05);
  border-radius: 8px;
  color: #333;
  text-decoration: none;
  transition: all 0.2s ease;
}

.social-link:hover {
  background: rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.social-icon {
  width: 18px;
  height: 18px;
}

.social-link span {
  font-size: 13px;
  font-weight: 500;
}

.qq-group {
  display: flex;
  align-items: stretch;
  gap: 20px;
  margin-top: 12px;
  max-width: 500px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  padding: 16px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.qq-qrcode-wrapper {
  position: relative;
  cursor: pointer;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: white;
  padding: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.qq-qrcode {
  width: 120px;
  height: 120px;
  border-radius: 6px;
  display: block;
  transition: transform 0.2s ease;
}

.qq-qrcode-wrapper:hover .qq-qrcode {
  transform: scale(1.02);
}

.qrcode-overlay {
  position: absolute;
  top: 8px;
  left: 8px;
  right: 8px;
  bottom: 8px;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.2s ease;
  border-radius: 6px;
}

.qq-qrcode-wrapper:hover .qrcode-overlay {
  opacity: 1;
}

.qrcode-overlay svg {
  width: 24px;
  height: 24px;
  color: #333;
  margin-bottom: 4px;
}

.qrcode-overlay span {
  font-size: 12px;
  color: #333;
}

.qq-info {
  display: flex;
  flex-direction: column;
  justify-content: center;
  flex: 1;
  min-width: 0;
  gap: 10px;
}

.qq-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.qq-avatar {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.qq-avatar svg {
  width: 24px;
  height: 24px;
  color: #333;
}

.qq-title-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.qq-name {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.qq-desc {
  font-size: 12px;
  color: #888;
}

.qq-number-section {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f5f5f7;
  border-radius: 8px;
  padding: 10px 14px;
}

.qq-number-label {
  font-size: 12px;
  color: #86868b;
  flex-shrink: 0;
}

.qq-number {
  font-size: 16px;
  font-weight: 600;
  color: #1d1d1f;
  font-family: 'SF Mono', 'Fira Code', 'Monaco', monospace;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 16px;
  background: #0071e3;
  border: none;
  border-radius: 8px;
  color: white;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.copy-btn:hover {
  background: #0077ed;
}

.copy-btn:active {
  transform: scale(0.98);
}

.copy-btn svg {
  width: 16px;
  height: 16px;
}

.content-section {
  margin-bottom: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: rgba(255,255,255,0.1);
  border-radius: 10px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.section-count {
  font-size: 13px;
  color: #666;
}

.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 20px;
}

.content-card {
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  background: rgba(255,255,255,0.95);
}

.content-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 32px rgba(0,0,0,0.2);
}

.egg-card {
  border-radius: 12px;
}

.card-media {
  position: relative;
  width: 100%;
  padding-top: 62.5%;
  background: #f8f9fa;
  overflow: hidden;
}

.card-image {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.play-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 48px;
  height: 48px;
  background: rgba(0, 0, 0, 0.6);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.play-icon {
  width: 20px;
  height: 20px;
  color: #333;
  margin-left: 3px;
}

.text-preview {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  padding: 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.preview-text {
  font-size: 13px;
  color: #64748b;
  line-height: 1.5;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.card-info {
  padding: 16px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 10px 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #888;
}

.meta-icon {
  width: 14px;
  height: 14px;
}

.tags-wrapper {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.meta-tag {
  padding: 2px 6px;
  background: rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  font-size: 11px;
  color: #666;
}

.card-type {
  display: flex;
  gap: 8px;
}

.type-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

.type-badge.video {
  background: #fff0e6;
  color: #d97706;
}

.type-badge.image {
  background: #ecfdf5;
  color: #059669;
}

.type-badge.text {
  background: #f0f9ff;
  color: #0284c7;
}

.mystery-members-section {
  margin-bottom: 24px;
}

.mystery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.mystery-card {
  display: flex;
  gap: 14px;
  padding: 16px;
  background: rgba(255, 255, 255, 0.85);
  border-radius: 12px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.mystery-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.12);
}

.mystery-avatar {
  width: 52px;
  height: 52px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.mystery-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.mystery-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.mystery-name {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0;
}

.mystery-role {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.mystery-role.群主 {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: #fff;
}

.mystery-role.管理员 {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: #fff;
}

.mystery-signature {
  font-size: 13px;
  color: #666;
  margin: 0;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.mystery-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: auto;
}

.mystery-tag {
  padding: 2px 8px;
  background: rgba(102, 126, 234, 0.1);
  color: #667eea;
  border-radius: 8px;
  font-size: 11px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #666;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
}

.pagination-section {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(255,255,255,0.1);
}

.pagination-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: white;
  border: 1px solid rgba(0,0,0,0.1);
  color: #333;
}

.pagination-btn:hover:not(:disabled) {
  background: #f5f5f5;
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination-btn svg {
  width: 14px;
  height: 14px;
}

.pagination-info {
  font-size: 14px;
  color: #666;
}

@media screen and (max-width: 768px) {
  .easter-egg-container {
    padding: 0;
    min-height: calc(100vh - 60px);
    position: relative;
  }

  .egg-window {
    border-radius: 0;
    box-shadow: none;
    min-height: calc(100vh - 60px);
    max-height: none;
  }

  .mac-title-bar {
    display: none;
  }

  .content-area {
    padding: 16px;
  }

  .header-section {
    margin-bottom: 20px;
  }

  .egg-icon {
    font-size: 48px;
  }

  .egg-banner {
    padding: 16px;
  }

  .egg-title {
    font-size: 22px;
  }

  .egg-subtitle {
    font-size: 14px;
  }

  .content-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
  }

  .card-media {
    padding-top: 56.25%;
  }

  .play-overlay {
    width: 40px;
    height: 40px;
  }

  .play-icon {
    width: 16px;
    height: 16px;
  }

  .card-info {
    padding: 12px;
  }

  .card-title {
    font-size: 14px;
    margin-bottom: 6px;
  }

  .card-meta {
    gap: 6px;
    margin-bottom: 6px;
  }

  .meta-item {
    font-size: 11px;
  }

  .meta-icon {
    width: 12px;
    height: 12px;
  }

  .meta-tag {
    padding: 1px 5px;
    font-size: 10px;
  }

  .type-badge {
    padding: 2px 6px;
    font-size: 10px;
  }

  .section-header {
    margin-bottom: 12px;
    padding: 10px 12px;
  }

  .section-title {
    font-size: 16px;
  }

  .section-count {
    font-size: 12px;
  }

  .pagination-section {
    gap: 12px;
    padding-top: 12px;
  }

  .pagination-btn {
    min-width: 80px;
    padding: 10px 12px;
  }

  .pagination-info {
    font-size: 13px;
  }

  .empty-state {
    padding: 40px 16px;
  }

  .empty-icon {
    width: 48px;
    height: 48px;
  }
}

@media screen and (max-width: 480px) {
  .content-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .mystery-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .mystery-card {
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 16px;
  }

  .mystery-header {
    flex-direction: column;
    gap: 6px;
  }

  .mystery-tags {
    justify-content: center;
  }

  .card-info {
    padding: 10px;
  }

  .card-title {
    font-size: 13px;
  }

  .qq-group {
    flex-direction: column;
    align-items: center;
    text-align: center;
    max-width: 100%;
    padding: 16px;
    background: rgba(255, 255, 255, 0.9);
  }

  .qq-qrcode-wrapper {
    width: 130px;
    height: 130px;
    padding: 6px;
  }

  .qq-qrcode {
    width: 100%;
    height: 100%;
  }

  .qq-info {
    width: 100%;
    margin-top: 14px;
    align-items: center;
  }

  .qq-header {
    flex-direction: column;
    gap: 8px;
  }

  .qq-name {
    font-size: 15px;
  }

  .qq-number-section {
    width: 100%;
    justify-content: center;
  }

  .qq-number {
    font-size: 18px;
  }

  .copy-btn {
    width: 100%;
    margin-top: 4px;
  }

  .social-links {
    flex-wrap: wrap;
    justify-content: center;
  }

  .text-content {
    padding: 16px;
  }

  .text-title {
    font-size: 16px;
  }

  .text-paragraph {
    font-size: 13px;
  }
}

@media screen and (max-width: 360px) {
  .qq-qrcode-wrapper {
    width: 120px;
    height: 120px;
  }

  .qq-number {
    font-size: 20px;
  }

  .qq-name {
    font-size: 13px;
  }

  .egg-icon {
    font-size: 40px;
  }

  .egg-title {
    font-size: 18px;
  }

  .back-btn {
    padding: 8px 16px;
    font-size: 14px;
  }
}

.qr-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.qr-modal {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  max-width: 90vw;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  animation: scaleIn 0.3s ease;
}

@keyframes scaleIn {
  from {
    transform: scale(0.9);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

.qr-modal-close {
  align-self: flex-end;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 8px;
  color: #666;
  transition: color 0.2s ease;
}

.qr-modal-close:hover {
  color: #000;
}

.qr-modal-close svg {
  width: 24px;
  height: 24px;
}

.qr-modal-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.qr-modal-image {
  max-width: 400px;
  max-height: 70vh;
  width: 100%;
  height: auto;
  border-radius: 12px;
  border: 4px solid #f0f0f0;
}

.qr-modal-info {
  text-align: center;
  margin-top: 20px;
}

.qr-title {
  font-size: 20px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 8px 0;
}

.qr-number {
  font-size: 16px;
  color: #666;
  margin: 0;
  font-family: 'Fira Code', 'Monaco', monospace;
}
</style>

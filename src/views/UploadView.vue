<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { contentApi } from '@/api'

const router = useRouter()
const userStore = useUserStore()

const message = ref('')
const uploadProgress = ref(0)
const isUploading = ref(false)

const uploadForm = ref({
  title: '',
  type: 'image' as 'video' | 'image' | 'text' | 'link',
  content: '',
  url: '',
  tags: [] as string[],
  file: undefined as File | undefined,
  filePath: '',
})

const filePreview = ref<string>('')
const allTags = ref<string[]>([])
const customTagInput = ref('')
const isDragging = ref(false)

const imageUploadInput = ref<HTMLInputElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const agreeUpload = ref(false)

watch(
  () => uploadForm.value.type,
  () => {
    agreeUpload.value = false
    clearFilePreview()
  }
)

function resetForm() {
  agreeUpload.value = false
  uploadForm.value = {
    title: '',
    type: 'image',
    content: '',
    url: '',
    tags: [],
    file: undefined,
    filePath: '',
  }
  filePreview.value = ''
  if (fileInputRef.value) fileInputRef.value.value = ''
}

function insertMarkdown(prefix: string, suffix: string = '') {
  const textarea = document.querySelector('.upload-textarea') as HTMLTextAreaElement
  if (!textarea) return
  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const text = uploadForm.value.content || ''
  uploadForm.value.content =
    text.substring(0, start) + prefix + text.substring(start, end) + suffix + text.substring(end)
  textarea.focus()
  const newPos = start + prefix.length + (end - start) + suffix.length
  setTimeout(() => {
    textarea.setSelectionRange(newPos, newPos)
  }, 0)
}

function triggerImageUpload() {
  imageUploadInput.value?.click()
}

async function handleImageUpload(event: Event) {
  const target = event.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
  if (!validTypes.includes(file.type)) {
    message.value = '只支持 JPG、PNG、GIF、WebP 格式的图片'
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    message.value = '图片大小不能超过10MB'
    return
  }
  try {
    message.value = '图片上传中...'
    const res = await contentApi.uploadImage(file)
    if (res.code === 200) {
      insertMarkdown(`![${file.name}](${res.data.image_url})`, '')
      message.value = '图片上传成功'
    } else {
      message.value = res.message || '上传失败'
    }
  } catch {
    message.value = '上传失败，请稍后重试'
  } finally {
    target.value = ''
  }
}

function toggleTag(tag: string) {
  const index = uploadForm.value.tags.indexOf(tag)
  if (index > -1) {
    uploadForm.value.tags.splice(index, 1)
  } else {
    uploadForm.value.tags.push(tag)
  }
}

function removeTag(tag: string) {
  const index = uploadForm.value.tags.indexOf(tag)
  if (index > -1) {
    uploadForm.value.tags.splice(index, 1)
  }
}

function addCustomTag() {
  const tag = customTagInput.value.trim()
  if (!tag) return
  if (uploadForm.value.tags.includes(tag)) {
    message.value = '该标签已存在'
    return
  }
  uploadForm.value.tags.push(tag)
  customTagInput.value = ''
}

function handleCustomTagKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter') {
    event.preventDefault()
    addCustomTag()
  }
}

function handleDragOver(event: DragEvent) {
  event.preventDefault()
  isDragging.value = true
}

function handleDragLeave(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
}

function handleDrop(event: DragEvent) {
  event.preventDefault()
  isDragging.value = false
  const files = event.dataTransfer?.files
  if (files && files.length > 0) {
    processFile(files[0])
  }
}

function processFile(file: File) {
  const isImage = uploadForm.value.type === 'image'
  const isVideo = uploadForm.value.type === 'video'

  if (isImage) {
    const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
    if (!validTypes.includes(file.type)) {
      message.value = '只支持 JPG、PNG、GIF、WebP 格式的图片'
      return
    }
    if (file.size > 10 * 1024 * 1024) {
      message.value = '图片大小不能超过10MB'
      return
    }
  }

  if (isVideo) {
    if (!file.type.startsWith('video/')) {
      message.value = '请上传视频文件'
      return
    }
    if (file.size > 100 * 1024 * 1024) {
      message.value = '视频大小不能超过100MB'
      return
    }
  }

  uploadForm.value.file = file
  const reader = new FileReader()
  reader.onload = (e) => {
    filePreview.value = e.target?.result as string
  }
  reader.readAsDataURL(file)
}

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    processFile(target.files[0])
  }
}

function clearFilePreview() {
  filePreview.value = ''
  uploadForm.value.file = undefined
  if (fileInputRef.value) fileInputRef.value.value = ''
}

async function loadAllTags() {
  try {
    const res = await contentApi.getTags()
    if (res.code === 200) {
      allTags.value = res.data as string[]
    }
  } catch {
    console.error('加载标签失败')
  }
}

async function handleUpload() {
  if (!userStore.user) {
    message.value = '请先登录'
    return
  }
  try {
    const userId = userStore.user.id
    if (!uploadForm.value.title && uploadForm.value.type !== 'link') {
      message.value = '请填写标题'
      return
    }
    isUploading.value = true
    uploadProgress.value = 0
    const res = await contentApi.upload({ ...uploadForm.value, user_id: userId }, (percent) => {
      uploadProgress.value = percent
    })
    if (res.code === 200) {
      message.value = '上传成功！'
      resetForm()
    } else {
      message.value = res.message || '上传失败'
    }
  } catch {
    message.value = '上传失败'
  } finally {
    isUploading.value = false
    uploadProgress.value = 0
  }
}

onMounted(() => {
  loadAllTags()
})
</script>

<template>
  <div class="min-h-screen p-2 sm:p-5 flex justify-center">
    <div class="w-full max-w-[800px] overflow-hidden bg-white/75 rounded-xl sm:rounded-xl shadow-lg shadow-black/5 border border-white/40">
      <div class="flex items-center px-3 py-2 sm:px-4 sm:py-2.5 bg-gradient-to-b from-black/8 to-black/2">
        <div class="w-3 h-3 rounded-full bg-[#ff5f57] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#febc2e] mr-2"></div>
        <div class="w-3 h-3 rounded-full bg-[#28c840] mr-4"></div>
        <div class="text-xs sm:text-sm text-gray-500 font-medium">小泉动漫 - 发布内容</div>
      </div>

      <div class="p-4 sm:p-6">
        <div v-if="message" class="px-3 py-2 sm:px-4 sm:py-3 rounded-lg mb-3 sm:mb-4 flex justify-between items-center" :class="{ 'bg-emerald-100 text-emerald-600': message.includes('成功'), 'bg-red-100 text-red-600': message.includes('失败') }">
          <span class="text-sm">{{ message }}</span>
          <span class="text-lg font-bold cursor-pointer" @click="message = ''">×</span>
        </div>

        <div class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">标题</label>
          <input v-model="uploadForm.title" type="text" class="w-full px-3 py-2 sm:py-2.5 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10" placeholder="输入标题" />
        </div>

        <div class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">类型</label>
          <div class="flex flex-wrap gap-2 sm:gap-4">
            <label class="flex items-center gap-1 sm:gap-1.5 cursor-pointer text-xs sm:text-sm text-gray-800">
              <input type="radio" value="text" v-model="uploadForm.type" class="cursor-pointer" />
              <span>图文</span>
            </label>
            <label class="flex items-center gap-1 sm:gap-1.5 cursor-pointer text-xs sm:text-sm text-gray-800">
              <input type="radio" value="image" v-model="uploadForm.type" class="cursor-pointer" />
              <span>图片</span>
            </label>
            <label class="flex items-center gap-1 sm:gap-1.5 cursor-pointer text-xs sm:text-sm text-gray-800">
              <input type="radio" value="video" v-model="uploadForm.type" class="cursor-pointer" />
              <span>视频</span>
            </label>
            <label class="flex items-center gap-1 sm:gap-1.5 cursor-pointer text-xs sm:text-sm text-gray-800">
              <input type="radio" value="link" v-model="uploadForm.type" class="cursor-pointer" />
              <span>链接</span>
            </label>
          </div>
        </div>

        <div v-if="uploadForm.type === 'text'" class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">内容</label>
          <div class="flex gap-1.5 mb-2 p-2 bg-black/3 rounded-md">
            <button type="button" @click="insertMarkdown('## ')" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="标题">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 12h16M4 6h16M4 18h10" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('**', '**')" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="粗体">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('*', '*')" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="斜体">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="19" y1="4" x2="10" y2="4" />
                <line x1="14" y1="20" x2="5" y2="20" />
                <line x1="15" y1="4" x2="9" y2="20" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('- ')" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="列表">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="8" y1="6" x2="21" y2="6" />
                <line x1="8" y1="12" x2="21" y2="12" />
                <line x1="8" y1="18" x2="21" y2="18" />
                <line x1="3" y1="6" x2="3.01" y2="6" />
                <line x1="3" y1="12" x2="3.01" y2="12" />
                <line x1="3" y1="18" x2="3.01" y2="18" />
              </svg>
            </button>
            <button type="button" @click="insertMarkdown('`', '`')" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="代码">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="16 18 22 12 16 6" />
                <polyline points="8 6 2 12 8 18" />
              </svg>
            </button>
            <button type="button" @click="triggerImageUpload" class="w-9 h-9 sm:w-10 sm:h-10 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="上传图片">
              <svg class="w-4.5 h-4.5 sm:w-5 sm:h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </button>
          </div>
          <textarea
            v-model="uploadForm.content"
            class="w-full min-h-[150px] sm:min-h-[200px] resize-y font-mono text-xs sm:text-sm leading-relaxed border border-black/10 rounded-lg bg-white p-3 focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 upload-textarea"
            placeholder="支持Markdown格式"
          ></textarea>
        </div>

        <div v-else-if="uploadForm.type !== 'link'" class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">文件</label>
          <div
            class="border-2 border-dashed border-black/15 rounded-xl p-3 sm:p-5 md:p-8 bg-black/2 cursor-pointer transition-all text-center min-h-[120px] sm:min-h-[160px] md:min-h-[200px] flex flex-col items-center justify-center hover:border-blue-500/40 hover:bg-blue-500/4"
            :class="{ 'border-blue-500 bg-blue-500/8 scale-[1.01]': isDragging, 'border-solid border-blue-500/30 bg-blue-500/2': filePreview }"
            @dragover="handleDragOver"
            @dragleave="handleDragLeave"
            @drop="handleDrop"
            @click="fileInputRef?.click()"
          >
            <div v-if="!filePreview" class="flex flex-col items-center gap-2 sm:gap-3 md:gap-4">
              <div class="w-10 h-10 sm:w-12 sm:h-12 md:w-16 md:h-16 text-gray-400">
                <svg class="w-full h-full" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                  <polyline points="17 8 12 3 7 8" />
                  <line x1="12" y1="3" x2="12" y2="15" />
                </svg>
              </div>
              <p class="text-xs sm:text-sm md:text-base text-gray-600 m-0 px-2 sm:px-0">
                拖拽{{ uploadForm.type === 'image' ? '图片' : '视频' }}到此处，或<span class="text-blue-500 font-medium">点击上传</span>
              </p>
              <p class="text-xs text-gray-400 m-0 px-2 sm:px-0">
                {{ uploadForm.type === 'image' ? '支持 JPG、PNG、GIF、WebP，最大 10MB' : '支持常见视频格式，最大 100MB' }}
              </p>
            </div>
            <div v-else class="relative w-full flex flex-col items-center gap-2 sm:gap-3">
              <img v-if="uploadForm.type === 'image'" :src="filePreview" class="max-w-full max-h-32 sm:max-h-48 md:max-h-60 rounded-lg object-contain shadow-md shadow-black/10" />
              <video v-else-if="uploadForm.type === 'video'" :src="filePreview" class="max-w-full max-h-32 sm:max-h-48 md:max-h-60 rounded-lg" controls />
              <button type="button" @click.stop="clearFilePreview" class="absolute top-1 sm:top-2 right-1 sm:right-2 w-7 h-7 sm:w-8 sm:h-8 border-none bg-black/60 text-white rounded-full cursor-pointer flex items-center justify-center hover:bg-red-500/90 hover:scale-110 transition-all">
                <svg class="w-3.5 h-3.5 sm:w-4 sm:h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
              <div v-if="uploadForm.file" class="flex items-center flex-wrap justify-center gap-1.5 sm:gap-2 px-2 sm:px-3 py-1 sm:py-1.5 bg-blue-500/8 rounded-lg">
                <span class="text-xs sm:text-sm font-medium text-gray-800 truncate max-w-[150px] sm:max-w-[200px]">{{ uploadForm.file.name }}</span>
                <span class="text-xs text-gray-600">{{ (uploadForm.file.size / 1024 / 1024).toFixed(2) }} MB</span>
              </div>
            </div>
          </div>
          <input
            ref="fileInputRef"
            type="file"
            @change="handleFileChange"
            class="hidden"
            :accept="uploadForm.type === 'image' ? 'image/*' : 'video/*'"
          />
        </div>

        <div v-if="uploadForm.type === 'video'" class="border-t border-black/6 pt-2 sm:pt-3 mb-1">
          <label class="flex gap-2 text-xs sm:text-sm text-gray-600 leading-relaxed cursor-pointer items-start hover:text-gray-800">
            <input type="checkbox" v-model="agreeUpload" class="mt-0.5 shrink-0 cursor-pointer w-4 h-4 sm:w-4 sm:h-4" />
            <span class="text-xs sm:text-sm md:text-sm">自2026年5月14日起，本平台建议使用「链接」类型来代替视频类型，您可以在链接类型中填入您的B站视频链接。若您同意，但您继续上传大于15MB的视频，则视为委托站长通过二创站官方B站账号代为上传，原视频将替换为该B站链接。该官方账号不会产生任何收益，亦不会用于上传其他内容。勾选即表示您已阅读并同意上述条款。</span>
          </label>
        </div>
        <div class="pt-1 mb-2">
          <div class="text-xs sm:text-sm text-gray-500 leading-relaxed">上传至本站的内容如未另行标注版权信息，默认采用<a href="https://creativecommons.org/licenses/by/4.0/deed.zh-hans" target="_blank" rel="noopener" class="text-blue-500 no-underline hover:underline">知识共享署名 4.0 国际许可协议（CC BY 4.0）</a>进行授权。建议您在作品中添加水印或署名以保障自身权益。</div>
        </div>

        <div v-if="uploadForm.type === 'link'" class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">链接地址</label>
          <input v-model="uploadForm.url" type="url" class="w-full px-3 py-2 sm:py-2.5 border border-black/10 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10" placeholder="https://..." />
        </div>

        <div class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">已选标签</label>
          <div v-if="uploadForm.tags.length > 0" class="flex flex-wrap gap-1.5 sm:gap-2 mb-2 sm:mb-3">
            <span v-for="tag in uploadForm.tags" :key="tag" class="px-2 py-0.5 sm:px-2.5 sm:py-1 bg-blue-500/10 border border-blue-500/30 rounded-full text-xs sm:text-sm text-blue-500 flex items-center gap-1">
              {{ tag }}
              <span @click="removeTag(tag)" class="cursor-pointer font-bold opacity-60 hover:opacity-100">×</span>
            </span>
          </div>
          <div v-else class="text-gray-500 text-xs sm:text-sm py-1.5 sm:py-2">点击下方标签进行选择</div>
        </div>

        <div class="mb-4 sm:mb-5">
          <label class="block text-xs sm:text-sm font-semibold text-gray-900 mb-1.5 sm:mb-2">选择标签</label>
          <div class="flex gap-1.5 sm:gap-2 mb-2 sm:mb-3">
            <input
              v-model="customTagInput"
              type="text"
              class="flex-1 px-3 py-1.5 sm:py-2 border border-black/10 rounded-md text-xs sm:text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10"
              placeholder="输入自定义标签后按回车"
              @keydown="handleCustomTagKeydown"
            />
            <button @click="addCustomTag" class="px-3 py-1.5 sm:px-4 sm:py-2 bg-blue-500/95 text-white border border-blue-500/95 rounded-md text-xs sm:text-sm cursor-pointer hover:bg-blue-700/95 hover:border-blue-700/95 transition-all">添加</button>
          </div>
          <div class="bg-black/2 rounded-lg p-2 sm:p-3">
            <div class="flex flex-wrap gap-1.5 sm:gap-2">
              <span
                v-for="tag in allTags"
                :key="tag"
                class="px-2 py-0.5 sm:px-2.5 sm:py-1 bg-white/95 border border-black/15 rounded-full text-xs sm:text-sm text-gray-800 cursor-pointer hover:border-blue-500/30 hover:text-blue-500 transition-all"
                :class="{ 'bg-blue-500/10 border-blue-500/30 text-blue-500': uploadForm.tags.includes(tag) }"
                @click="toggleTag(tag)"
              >
                {{ tag }}
              </span>
            </div>
            <div v-if="allTags.length === 0" class="text-gray-500 text-xs sm:text-sm text-center py-3 sm:py-4">暂无可用标签</div>
          </div>
        </div>

        <div v-if="isUploading" class="mb-4 sm:mb-5">
          <div class="relative w-full h-4 sm:h-5 bg-black/6 rounded-full overflow-hidden">
            <div class="h-full bg-gradient-to-r from-blue-500 to-purple-500 rounded-full transition-all duration-300" :style="{ width: `${uploadProgress}%` }"></div>
            <span class="absolute inset-0 flex items-center justify-center text-xs font-medium text-white drop-shadow-md">{{ uploadProgress }}%</span>
          </div>
        </div>

        <div class="flex justify-end gap-2 sm:gap-3 pt-3 sm:pt-4 border-t border-black/6">
          <button @click="router.push('/')" class="px-3 py-2 sm:px-5 sm:py-2.5 bg-white/95 border border-black/10 rounded-lg text-xs sm:text-sm text-gray-800 hover:bg-blue-500/8 hover:text-blue-500 hover:border-blue-500/30 transition-all">取消</button>
          <button
            @click="handleUpload"
            class="px-3 py-2 sm:px-5 sm:py-2.5 bg-blue-500/95 text-white border border-blue-500/95 rounded-lg text-xs sm:text-sm cursor-pointer hover:bg-blue-700/95 hover:border-blue-700/95 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            :disabled="isUploading || (uploadForm.type === 'video' && uploadForm.file && uploadForm.file.size > 15 * 1024 * 1024 && !agreeUpload)"
          >
            {{ isUploading ? '上传中...' : '发布内容' }}
          </button>
        </div>
      </div>
    </div>

    <input
      ref="imageUploadInput"
      type="file"
      accept="image/jpeg,image/png,image/gif,image/webp"
      style="display: none"
      @change="handleImageUpload"
    />
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';

interface UploadForm {
  title: string;
  content: string;
  url: string;
  type: 'video' | 'image' | 'text' | 'link';
  tags: string[];
  file?: File;
}

interface Props {
  visible: boolean;
  uploadForm: UploadForm;
  allTags: string[];
  filePreview?: string;
  uploadProgress?: number;
  isUploading?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  allTags: () => [],
  filePreview: '',
  uploadProgress: 0,
  isUploading: false,
});

const mergedTags = computed(() => {
  const tagSet = new Set([...props.allTags, ...props.uploadForm.tags]);
  return Array.from(tagSet);
});

const agreeUpload = ref(false)

const emit = defineEmits<{
  close: [];
  submit: [];
  insertMarkdown: [prefix: string, suffix?: string];
  triggerImageUpload: [];
  handleImageUpload: [e: Event];
  handleFileChange: [e: Event];
  addTag: [target: string];
  removeTag: [tag: string];
  toggleTag: [tag: string];
  clearPreview: [];
}>();
</script>

<template>
  <div v-if="visible" class="fixed inset-0 flex items-center justify-center z-[9999] bg-black/50" @click.self="$emit('close')">
    <div class="w-[90%] max-w-[600px] bg-white/98 rounded-xl shadow-2xl overflow-hidden relative z-[10000]">
      <div class="flex justify-between items-center px-5 py-4 bg-gradient-to-b from-black/5 to-transparent border-b border-black/5">
        <h2 class="text-base font-semibold text-gray-900 m-0">上传内容</h2>
        <span class="text-2xl text-gray-400 cursor-pointer leading-none hover:text-blue-500 transition-colors" @click="$emit('close')">×</span>
      </div>
      <div class="p-5">
        <div class="mb-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">标题</label>
          <input v-model="uploadForm.title" type="text" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10" placeholder="输入标题" />
        </div>
        <div class="mb-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">类型</label>
          <div class="flex gap-4">
            <label class="flex items-center gap-1.5 cursor-pointer text-sm text-gray-700">
              <input type="radio" value="text" v-model="uploadForm.type" class="cursor-pointer" />
              <span>文字</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer text-sm text-gray-700">
              <input type="radio" value="image" v-model="uploadForm.type" class="cursor-pointer" />
              <span>图片</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer text-sm text-gray-700">
              <input type="radio" value="video" v-model="uploadForm.type" class="cursor-pointer" />
              <span>视频</span>
            </label>
            <label class="flex items-center gap-1.5 cursor-pointer text-sm text-gray-700">
              <input type="radio" value="link" v-model="uploadForm.type" class="cursor-pointer" />
              <span>链接</span>
            </label>
          </div>
        </div>
        <div v-if="uploadForm.type === 'text'" class="mb-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">内容</label>
          <div class="flex gap-1 mb-2 p-1.5 bg-black/3 rounded-md">
            <button type="button" @click="$emit('insertMarkdown', '## ')" class="w-8 h-8 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="标题">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 12h16M4 6h16M4 18h10" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '**', '**')" class="w-8 h-8 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="粗体">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '*', '*')" class="w-8 h-8 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="斜体">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="19" y1="4" x2="10" y2="4" />
                <line x1="14" y1="20" x2="5" y2="20" />
                <line x1="15" y1="4" x2="9" y2="20" />
              </svg>
            </button>
            <button type="button" @click="$emit('triggerImageUpload')" class="w-8 h-8 flex items-center justify-center bg-white border border-black/10 rounded-md text-gray-600 hover:text-blue-500 hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="上传图片">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </button>
          </div>
          <textarea v-model="uploadForm.content" class="w-full min-h-[200px] resize-y font-mono text-sm leading-relaxed box-border bg-white border border-gray-200 rounded-lg p-3 focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10 upload-textarea" placeholder="支持Markdown"></textarea>
        </div>
        <div v-else class="mb-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">文件</label>
          <input type="file" @change="$emit('handleFileChange', $event)" class="p-2 border border-gray-200 rounded-lg bg-white cursor-pointer" :accept="uploadForm.type === 'image' ? 'image/*' : 'video/*'" />
          <div v-if="filePreview" class="relative mt-3 rounded-lg overflow-hidden">
            <img v-if="uploadForm.type === 'image'" :src="filePreview" class="w-full max-h-[300px] object-contain" />
            <video v-else-if="uploadForm.type === 'video'" :src="filePreview" class="w-full max-h-[300px]" controls />
            <button type="button" @click="$emit('clearPreview')" class="absolute top-2 right-2 w-7 h-7 border-none bg-black/60 text-white rounded-full cursor-pointer text-lg flex items-center justify-center hover:bg-black/80 transition-colors">×</button>
          </div>
        </div>
        <div v-if="uploadForm.type === 'video'" class="border-t border-gray-200 pt-2.5 px-4 pb-0">
          <label class="flex gap-2 text-xs text-gray-600 leading-relaxed cursor-pointer items-start hover:text-gray-800">
            <input type="checkbox" v-model="agreeUpload" class="mt-0.5 shrink-0 cursor-pointer" />
            <span>自2026年5月14日起，本平台建议使用「链接」类型来代替视频类型，您可以在链接类型中填入您的B站视频链接。若您同意，但您继续上传大于15MB的视频，则视为委托站长通过二创站官方B站账号代为上传，原视频将替换为该B站链接。该官方账号不会产生任何收益，亦不会用于上传其他内容。勾选即表示您已阅读并同意上述条款。</span>
          </label>
        </div>
        <div class="border-t border-gray-200 pt-1 pb-0 mt-4">
          <div class="text-xs text-gray-500 leading-relaxed">上传至本站的内容如未另行标注版权信息，默认采用<a href="https://creativecommons.org/licenses/by/4.0/deed.zh-hans" target="_blank" rel="noopener" class="text-blue-500 no-underline hover:underline">知识共享署名 4.0 国际许可协议（CC BY 4.0）</a>进行授权。建议您在作品中添加水印或署名以保障自身权益。</div>
        </div>
        <div v-if="uploadForm.type === 'link'" class="mb-4 mt-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">链接地址</label>
          <input v-model="uploadForm.url" type="url" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-white focus:outline-none focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/10" placeholder="https://..." />
        </div>
        <div class="mb-4">
          <label class="block text-[13px] font-medium text-gray-700 mb-2">标签</label>
          <div class="mt-2">
            <div class="flex flex-wrap gap-2">
              <span v-for="tag in mergedTags" :key="tag"
                :class="['px-3 py-1 bg-white border cursor-pointer transition-all text-sm rounded-full', uploadForm.tags.includes(tag) ? 'bg-blue-100/50 border-blue-300 text-blue-600 hover:border-blue-400' : 'border-black/20 text-gray-700 hover:border-blue-300 hover:text-blue-500']"
                @click="$emit('toggleTag', tag)">{{ tag }}</span>
              <button type="button" @click="$emit('addTag', 'upload')" class="w-8 h-8 flex items-center justify-center bg-white border-dashed border-black/20 rounded-md text-gray-500 hover:border-blue-300 hover:text-blue-500 transition-all" title="添加标签">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>
          </div>
        </div>
        <div v-if="isUploading" class="mb-4">
          <div class="relative w-full h-5 bg-gray-200 rounded-full overflow-hidden">
            <div class="h-full bg-gradient-to-r from-blue-500 to-purple-500 rounded-full transition-all duration-300" :style="{ width: `${uploadProgress}%` }"></div>
            <span class="absolute inset-0 flex items-center justify-center text-xs font-medium text-white drop-shadow-md">{{ uploadProgress }}%</span>
          </div>
        </div>
      </div>
      <div class="flex justify-end gap-3 px-5 py-4 bg-black/3">
        <button @click="$emit('close')" class="px-4 py-2 bg-white border border-black/10 rounded-lg text-sm text-gray-700 hover:bg-gray-50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isUploading">取消</button>
        <button @click="$emit('submit')" class="px-4 py-2 bg-blue-500 border border-blue-500 rounded-lg text-sm text-white hover:bg-blue-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isUploading || (uploadForm.type === 'video' && uploadForm.file && uploadForm.file.size > 15 * 1024 * 1024 && !agreeUpload)">
          {{ isUploading ? '上传中...' : '提交' }}
        </button>
      </div>
    </div>
  </div>
</template>

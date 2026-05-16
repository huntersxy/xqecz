<script setup lang="ts">
import { computed } from 'vue';

interface EditForm {
  id: number;
  title: string;
  content: string;
  url: string;
  type: 'video' | 'image' | 'text' | 'link';
  tags: string[];
  file?: File;
}

interface Props {
  visible: boolean;
  editForm: EditForm;
  allTags: string[];
}

const props = withDefaults(defineProps<Props>(), {
  allTags: () => [],
});

const mergedTags = computed(() => {
  const tagSet = new Set([...props.allTags, ...props.editForm.tags]);
  return Array.from(tagSet);
});

const emit = defineEmits<{
  close: [];
  save: [];
  insertMarkdown: [prefix: string, suffix?: string];
  triggerImageUpload: [];
  handleImageUpload: [e: Event];
  handleFileChange: [e: Event];
  addTag: [target: string];
  removeTag: [tag: string];
  toggleTag: [tag: string];
}>();
</script>

<template>
  <div v-if="visible" class="fixed inset-0 flex items-center justify-center z-[9999] bg-[var(--theme-hover-bg)]0" @click.self="$emit('close')">
    <div class="w-[90%] max-w-[600px] bg-[var(--theme-surface)] rounded-xl shadow-2xl overflow-hidden relative z-[10000]">
      <div class="flex justify-between items-center px-5 py-4 bg-gradient-to-b from-black/5 to-transparent border-b border-black/5">
        <h2 class="text-base font-semibold theme-text m-0">编辑内容</h2>
        <span class="text-2xl theme-text-secondary cursor-pointer leading-none hover:text-[var(--theme-primary)] transition-colors" @click="$emit('close')">×</span>
      </div>
      <div class="p-5">
        <div class="mb-4">
          <label class="block text-[13px] font-medium theme-text mb-2">标题</label>
          <input v-model="editForm.title" type="text" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10" />
        </div>
        <div v-if="editForm.type === 'text'" class="mb-4">
          <label class="block text-[13px] font-medium theme-text mb-2">内容</label>
          <div class="flex gap-1 mb-2 p-1.5 bg-[var(--theme-hover-bg)] rounded-md">
            <button type="button" @click="$emit('insertMarkdown', '## ')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="标题">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M4 12h16M4 6h16M4 18h10" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '**', '**')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="粗体">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M6 4h8a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
                <path d="M6 12h9a4 4 0 0 1 4 4 4 4 0 0 1-4 4H6z" />
              </svg>
            </button>
            <button type="button" @click="$emit('insertMarkdown', '*', '*')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="斜体">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="19" y1="4" x2="10" y2="4" />
                <line x1="14" y1="20" x2="5" y2="20" />
                <line x1="15" y1="4" x2="9" y2="20" />
              </svg>
            </button>
            <button type="button" @click="$emit('triggerImageUpload')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-md theme-text-secondary hover:text-[var(--theme-primary)] hover:border-blue-300 transition-all active:bg-blue-50 active:shadow-inner active:ring-1 active:ring-blue-200" title="上传图片">
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
                <circle cx="8.5" cy="8.5" r="1.5" />
                <polyline points="21 15 16 10 5 21" />
              </svg>
            </button>
          </div>
          <textarea v-model="editForm.content" class="w-full min-h-[200px] resize-y font-mono text-sm leading-relaxed box-border bg-[var(--theme-surface)] border border-gray-200 rounded-lg p-3 focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10 edit-textarea"></textarea>
        </div>
        <div v-else class="mb-4">
          <label class="block text-[13px] font-medium theme-text mb-2">文件</label>
          <input type="file" @change="$emit('handleFileChange', $event)" class="p-2 border border-gray-200 rounded-lg bg-[var(--theme-surface)] cursor-pointer" />
        </div>
        <div v-if="editForm.type === 'link'" class="mb-4">
          <label class="block text-[13px] font-medium theme-text mb-2">链接地址</label>
          <input v-model="editForm.url" type="url" class="w-full px-3 py-2.5 border border-gray-200 rounded-lg text-sm bg-[var(--theme-surface)] focus:outline-none focus:border-[var(--theme-primary)]/50 focus:ring-2 focus:ring-[var(--theme-primary)]/10" placeholder="https://..." />
        </div>
        <div class="mb-4">
          <label class="block text-[13px] font-medium theme-text mb-2">标签</label>
          <div class="mt-2">
            <div class="flex flex-wrap gap-2">
              <span v-for="tag in mergedTags" :key="tag"
                :class="['px-3 py-1 bg-[var(--theme-surface)] border cursor-pointer transition-all text-sm rounded-full', editForm.tags.includes(tag) ? 'bg-[var(--theme-primary)]/10/50 border-blue-300 text-[var(--theme-primary)] hover:border-blue-400' : 'border-black/20 theme-text hover:border-blue-300 hover:text-[var(--theme-primary)]']"
                @click="$emit('toggleTag', tag)">{{ tag }}</span>
              <button type="button" @click="$emit('addTag', 'edit')" class="w-8 h-8 flex items-center justify-center bg-[var(--theme-surface)] border-dashed border-black/20 rounded-md theme-text-secondary hover:border-blue-300 hover:text-[var(--theme-primary)] transition-all" title="添加标签">
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
      <div class="flex justify-end gap-3 px-5 py-4 bg-[var(--theme-hover-bg)]">
        <button @click="$emit('close')" class="px-4 py-2 bg-[var(--theme-surface)] border border-[var(--theme-card-border)] rounded-lg text-sm theme-text hover:bg-gray-50 transition-colors">取消</button>
        <button @click="$emit('save')" class="px-4 py-2 bg-[var(--theme-primary)] border border-[var(--theme-primary)] rounded-lg text-sm text-white hover:bg-[var(--theme-primary)] transition-colors">保存</button>
      </div>
    </div>
  </div>
</template>

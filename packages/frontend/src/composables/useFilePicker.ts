import { ref } from 'vue'
import { toast } from './useToast'

/** 与后端保持一致：单文件最大 20MB。 */
export const MAX_FILE_SIZE = 20 * 1024 * 1024

/**
 * 文件选择/校验/预览共用逻辑。
 * @param onPick 校验通过后回调，由调用方保存文件并派生自己的状态。
 * @param options.allowVideo 管理端等旧入口可继续接收视频，公开创作入口只允许图片。
 */
export function useFilePicker(
  onPick: (file: File) => void,
  options: { allowVideo?: boolean } = {},
) {
  const filePreview = ref('')
  const fileInput = ref<HTMLInputElement | null>(null)
  const dragOver = ref(false)

  function pickFile(f: File | undefined) {
    if (!f) return
    const isImage = f.type.startsWith('image/')
    const isVideo = f.type.startsWith('video/')
    if (!isImage && !(options.allowVideo !== false && isVideo)) {
      toast.error(options.allowVideo === false ? '仅支持图片文件' : '仅支持图片或视频文件')
      return
    }
    if (f.size > MAX_FILE_SIZE) {
      toast.error('文件大小不能超过 20MB')
      return
    }
    onPick(f)
    const reader = new FileReader()
    reader.onload = (ev) => { filePreview.value = ev.target?.result as string }
    reader.readAsDataURL(f)
  }

  function onFileChange(e: Event) {
    pickFile((e.target as HTMLInputElement).files?.[0])
  }

  function onDrop(e: DragEvent) {
    dragOver.value = false
    pickFile(e.dataTransfer?.files?.[0])
  }

  function clearPreview() {
    filePreview.value = ''
    if (fileInput.value) fileInput.value.value = ''
  }

  return { filePreview, fileInput, dragOver, pickFile, onFileChange, onDrop, clearPreview }
}

package content

import (
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/store"
)

// fakeMirror 是测试用的镜像替身：只按固定前缀拼地址，能记下推送调用。
type fakeMirror struct {
	base     string
	pushed   []string
	archived []string
}

func (f *fakeMirror) PublicURL(rel string) string {
	if f.base == "" || rel == "" {
		return ""
	}
	return f.base + "/" + rel
}

func (f *fakeMirror) PushAsync(rel, absPath string) {
	f.pushed = append(f.pushed, rel)
}

// ArchiveOriginalAsync 记录归档调用：本包只关心「有没有被调用」，不校验 R2 侧细节。
func (f *fakeMirror) ArchiveOriginalAsync(rel, origPath string) {
	f.archived = append(f.archived, rel)
}

// TestDecorateWithMirrorExposesBackupURL 图片行的 R2 备份地址与主地址同路径、仅换 host。
func TestDecorateWithMirrorExposesBackupURL(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://file.example.com"}
	row := store.Content{
		ID: 1, Title: "i", FilePath: ptr("ab12.webp"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)

	if item.Img != "/uploads/ab12.webp" {
		t.Fatalf("主地址应为源站相对路径，实际 %q", item.Img)
	}
	if item.MirrorImg != "https://file.example.com/ab12.webp" {
		t.Fatalf("备份地址错误: %q", item.MirrorImg)
	}
	if item.MirrorVideo != "" {
		t.Fatalf("图片行不应有视频备份地址: %q", item.MirrorVideo)
	}
}

// TestDecorateWithMirrorVideo 视频行同理，走 mirror_video 字段。
func TestDecorateWithMirrorVideo(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://file.example.com"}
	row := store.Content{
		ID: 2, Title: "v", FilePath: ptr("clip.mp4"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)
	if item.Video != "/uploads/clip.mp4" || item.MirrorVideo != "https://file.example.com/clip.mp4" {
		t.Fatalf("视频字段错误: video=%q mirror=%q", item.Video, item.MirrorVideo)
	}
	if item.MirrorImg != "" {
		t.Fatalf("视频行不应有图片备份地址: %q", item.MirrorImg)
	}
}

// TestDecorateWithoutMirrorHasNoBackup 未接入 R2 时输出与旧实现逐字段一致（无备份字段）。
func TestDecorateWithoutMirrorHasNoBackup(t *testing.T) {
	now := time.Now()
	row := store.Content{
		ID: 3, Title: "i", FilePath: ptr("ab12.webp"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorate(row, nil, 0, false)
	if item.MirrorImg != "" || item.MirrorVideo != "" {
		t.Fatalf("未接入镜像时不应出现备份地址: img=%q video=%q", item.MirrorImg, item.MirrorVideo)
	}
	if item.Img != "/uploads/ab12.webp" {
		t.Fatalf("主地址不受影响，实际 %q", item.Img)
	}
}

// TestDecorateThumbOnlyRowHasNoMirror 只有缩略图的行不镜像（缩略图保持纯本地）。
func TestDecorateThumbOnlyRowHasNoMirror(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://file.example.com"}
	row := store.Content{
		ID: 4, Title: "t", ThumbPath: ptr("thumbs/ab12_thumb.webp"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)
	if item.MirrorImg != "" {
		t.Fatalf("仅缩略图的内容不应有 R2 备份地址: %q", item.MirrorImg)
	}
	if item.Img != "/thumbs/ab12_thumb.webp" || item.Thumb != "/thumbs/ab12_thumb.webp" {
		t.Fatalf("缩略图回退路径不应变化: img=%q thumb=%q", item.Img, item.Thumb)
	}
}

// TestPrepareUploadFilePushesToMirror 上传落盘后立即异步推 R2（原图先落本地，再推一份）。
func TestPrepareUploadFilePushesToMirror(t *testing.T) {
	mm := &fakeMirror{base: "https://file.example.com"}
	h := &Handler{mirror: mm}

	rel, size, abs := h.prepareUploadFile(&uploadedFile{
		AbsPath: "/tmp/data/uploads/ab12.webp", RelPath: "ab12.webp", Size: 1234, Mime: "image/webp",
	})
	if rel != "ab12.webp" || size != 1234 || abs != "/tmp/data/uploads/ab12.webp" {
		t.Fatalf("返回值不应改变: %q %d %q", rel, size, abs)
	}
	if len(mm.pushed) != 1 || mm.pushed[0] != "ab12.webp" {
		t.Fatalf("应推送一次且路径正确，实际 %v", mm.pushed)
	}
}

// TestPrepareUploadFileWithoutMirror 未接入 R2 时上传流程不受影响，且 nil 文件安全。
func TestPrepareUploadFileWithoutMirror(t *testing.T) {
	h := &Handler{}
	if rel, size, abs := h.prepareUploadFile(nil); rel != "" || size != 0 || abs != "" {
		t.Fatalf("nil 文件应返回零值: %q %d %q", rel, size, abs)
	}
	rel, _, _ := h.prepareUploadFile(&uploadedFile{AbsPath: "/x/a.webp", RelPath: "a.webp", Size: 1})
	if rel != "a.webp" {
		t.Fatalf("未接入镜像时行为应不变: %q", rel)
	}
}

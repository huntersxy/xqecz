package content

import (
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/store"
)

type fakeMirror struct {
	base     string
	base2    string
	pushed   []string
	archived []string
}

func (f *fakeMirror) R2URL(rel string) string {
	if f.base == "" || rel == "" {
		return ""
	}
	return f.base + "/" + rel
}

func (f *fakeMirror) OpenListURL(rel string) string {
	if f.base2 == "" || rel == "" {
		return ""
	}
	return f.base2 + "/" + rel
}

func (f *fakeMirror) PushAsync(rel, absPath string) {
	f.pushed = append(f.pushed, rel)
}

func (f *fakeMirror) ArchiveOriginalAsync(rel, origPath string) {
	f.archived = append(f.archived, rel)
}

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

func TestDecorateWithMirror2ExposesFallbackURL(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://r2.example.com", base2: "https://drive.example.com"}
	row := store.Content{
		ID: 5, Title: "i", FilePath: ptr("ab12.webp"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)

	if item.MirrorImg != "https://r2.example.com/ab12.webp" {
		t.Fatalf("主目标地址错误: %q", item.MirrorImg)
	}
	if item.Mirror2Img != "https://drive.example.com/ab12.webp" {
		t.Fatalf("首选镜像地址错误: %q", item.Mirror2Img)
	}
	if item.Mirror2Video != "" {
		t.Fatalf("图片行不应有视频首选镜像地址: %q", item.Mirror2Video)
	}
}

func TestDecorateMirror2EmptyWhenClosed(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://r2.example.com"}
	row := store.Content{
		ID: 6, Title: "i", FilePath: ptr("ab12.webp"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)

	if item.MirrorImg == "" {
		t.Fatal("R2 不应受首选镜像状态影响")
	}
	if item.Mirror2Img != "" {
		t.Fatalf("首选停用时不应下发地址: %q", item.Mirror2Img)
	}
}

func TestDecorateMirror2Video(t *testing.T) {
	now := time.Now()
	mm := &fakeMirror{base: "https://r2.example.com", base2: "https://drive.example.com"}
	row := store.Content{
		ID: 7, Title: "v", FilePath: ptr("clip.mp4"), Tags: "[]",
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now,
	}
	item := decorateWith(mm, row, nil, 0, false)

	if item.Mirror2Video != "https://drive.example.com/clip.mp4" {
		t.Fatalf("视频首选镜像地址错误: %q", item.Mirror2Video)
	}
	if item.Mirror2Img != "" {
		t.Fatalf("视频行不应有图片首选镜像地址: %q", item.Mirror2Img)
	}
}

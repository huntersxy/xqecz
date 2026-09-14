package content

import (
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/store"
)

func ptr[T any](v T) *T { return &v }

func TestIsVideoPath(t *testing.T) {
	cases := map[string]bool{
		"a.mp4": true, "b.WEBM": true, "c.mkv": true,
		"d.webp": false, "e.png": false, "noext": false, "f.gif": false,
	}
	for in, want := range cases {
		if got := IsVideoPath(in); got != want {
			t.Errorf("IsVideoPath(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFileURL(t *testing.T) {
	cases := map[string]string{
		"":                    "",
		"abc.webp":            "/uploads/abc.webp",
		"thumbs/a_thumb.webp": "/thumbs/a_thumb.webp",
		"images/a.webp":       "/images/a.webp",
		"/already/abs.webp":   "/already/abs.webp",
		"https://x/y.webp":    "https://x/y.webp",
	}
	for in, want := range cases {
		if got := FileURL(in); got != want {
			t.Errorf("FileURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMakeAvatarURL(t *testing.T) {
	qq := MakeAvatarURL("12345@qq.com", 80)
	if qq != "https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100" {
		t.Errorf("QQ 头像生成错误: %s", qq)
	}
	if got := MakeAvatarURL("a@b.com", 80); len(got) < 32 || got[:32] != "https://www.gravatar.com/avatar/" {
		t.Errorf("Gravatar 生成错误: %s", got)
	}
	if got := MakeAvatarURL("", 80); got != "" {
		t.Errorf("空邮箱应返回空串，实际 %q", got)
	}
}

func TestParseTags(t *testing.T) {
	if got := ParseTags(`["a","b"]`); len(got) != 2 || got[0] != "a" {
		t.Errorf("JSON 数组解析错误: %v", got)
	}
	if got := ParseTags("a, b ,c"); len(got) != 3 || got[2] != "c" {
		t.Errorf("逗号分隔解析错误: %v", got)
	}
	if got := ParseTags(""); len(got) != 0 {
		t.Errorf("空串应返回空切片: %v", got)
	}
}

func TestDecorateVideoAndImage(t *testing.T) {
	now := time.Now()
	video := store.Content{
		ID: 1, Title: "v", FilePath: ptr("v.mp4"), ThumbPath: ptr("thumbs/v_thumb.webp"),
		Tags: "[]", AuditStatus: "approved", CreatedAt: now, UpdatedAt: now, UserID: 2,
	}
	item := decorate(video, map[uint64]store.User{2: {ID: 2, Username: "u"}}, 3, false)
	if item.Video != "/uploads/v.mp4" {
		t.Errorf("视频行 video 字段错误: %q", item.Video)
	}
	if item.Img != "" {
		t.Errorf("视频行 img 应为空: %q", item.Img)
	}
	if item.Thumb != "/thumbs/v_thumb.webp" {
		t.Errorf("缩略图应取 thumb_path: %q", item.Thumb)
	}
	if item.LikeCount != 3 {
		t.Errorf("点赞数错误: %d", item.LikeCount)
	}
	if item.ViewCount != nil {
		t.Error("默认不应输出 view_count")
	}

	img := store.Content{
		ID: 2, Title: "i", FilePath: ptr("i.webp"), Tags: `["x"]`,
		AuditStatus: "approved", CreatedAt: now, UpdatedAt: now, UserID: 2,
	}
	item2 := decorate(img, nil, 0, true)
	if item2.Img != "/uploads/i.webp" || item2.Video != "" {
		t.Errorf("图片行字段错误: img=%q video=%q", item2.Img, item2.Video)
	}
	if item2.User.Username != "unknown" {
		t.Errorf("缺用户表时应回退 unknown: %+v", item2.User)
	}
	if item2.ViewCount == nil || *item2.ViewCount != 0 {
		t.Error("includeViewCount 时应输出 view_count")
	}
}

func TestDecorateGuest(t *testing.T) {
	row := store.Content{
		ID: 3, Title: "g", GuestNickname: ptr("游客甲"), GuestEmail: ptr("12345@qq.com"),
		Tags: "[]", AuditStatus: "pending", UserID: 0, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	item := decorate(row, nil, 0, false)
	if item.User.ID != 0 || item.User.Username != "游客甲" {
		t.Errorf("游客作者信息错误: %+v", item.User)
	}
	if item.AvatarURL != "https://q.qlogo.cn/headimg_dl?dst_uin=12345&spec=100" {
		t.Errorf("游客头像应取 guest_email: %q", item.AvatarURL)
	}
	if item.Origin != "" {
		t.Errorf("无文件时不应输出 origin: %q", item.Origin)
	}
}

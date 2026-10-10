package content

import (
	"encoding/json"
	"strings"

	"github.com/huntersxy/xqecz/server/internal/app"
	useravatar "github.com/huntersxy/xqecz/server/internal/avatar"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

// 视频扩展名集合：媒体类型以扩展名为唯一依据（内容不做分类）。
var videoExts = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".m4v": true, ".mkv": true,
	".avi": true, ".flv": true, ".ogv": true, ".wmv": true, ".3gp": true,
	".mpeg": true, ".mpg": true, ".ts": true, ".m2ts": true,
}

// IsVideoPath 判断文件是否为视频。
func IsVideoPath(p string) bool {
	dot := strings.LastIndex(p, ".")
	if dot < 0 {
		return false
	}
	return videoExts[strings.ToLower(p[dot:])]
}

// MediaTypeForPath 返回媒体类型（video / image）。
func MediaTypeForPath(p string) string {
	if IsVideoPath(p) {
		return "video"
	}
	return "image"
}

// FileURL 把存储的相对路径补全为前端可访问的 URL。
// thumbs/ 前缀对应缩略图目录；images/ 是历史遗留（旧 TinyPNG 压缩图，目录已清空，
// 仅保留兼容分支以读取尚未迁移的存量记录）；其余裸文件名一律落在 uploads。
func FileURL(rel string) string {
	if rel == "" {
		return ""
	}
	if strings.HasPrefix(rel, "http://") || strings.HasPrefix(rel, "https://") {
		return rel
	}
	if strings.HasPrefix(rel, "/") {
		return rel
	}
	if strings.HasPrefix(rel, "thumbs/") || strings.HasPrefix(rel, "images/") {
		return "/" + rel
	}
	return "/uploads/" + rel
}

// isMirrorablePath 判定该存储路径是否参与 R2 镜像：
// 只有 uploads 目录（裸文件名，含原图与压缩图）会被镜像，缩略图与历史 images/ 不镜像。
func isMirrorablePath(rel string) bool {
	if rel == "" {
		return false
	}
	return !strings.HasPrefix(rel, "thumbs/") && !strings.HasPrefix(rel, "images/")
}

// mirrorKey 把存储路径换算成「相对上传目录」的对象名（镜像侧的键）。
func mirrorKey(rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	// 裸文件名即上传目录下的文件；带 uploads/ 前缀的（若历史数据有）去掉前缀。
	return strings.TrimPrefix(rel, "uploads/")
}

// ParseTags 解析 tags 列（JSON 数组字符串，兼容历史逗号分隔写法）。
func ParseTags(raw string) []string {
	out := []string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var arr []string
	if err := json.Unmarshal([]byte(raw), &arr); err == nil {
		for _, t := range arr {
			if t != "" {
				out = append(out, t)
			}
		}
		return out
	}
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ParseTagsParam 解析查询参数里的标签（支持逗号分隔的多标签）。
func ParseTagsParam(raw string) []string {
	out := []string{}
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// UserBrief 是内容里内嵌的作者信息。
type UserBrief struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

// Item 是内容列表/详情的对外形状，与旧后端 decorateContent 的输出逐字段一致。
type Item struct {
	ID     uint64 `json:"id"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Thumb  string `json:"thumb"`
	Video  string `json:"video"`
	Img    string `json:"img"`
	Origin string `json:"origin,omitempty"`
	// MirrorImg / MirrorVideo 是同一份文件在**主目标**（R2）上的绝对地址（未启用时为空）。
	// 与源站不同 origin，无法用相对路径推导，因此这里给完整地址：
	// 换公开域名只改服务端 .env，前端不必重新构建。
	// 前端据此在「源站」与「镜像」之间做可达性选择，本地缩略图不受影响。
	MirrorImg   string `json:"mirror_img,omitempty"`
	MirrorVideo string `json:"mirror_video,omitempty"`
	// Mirror2Img / Mirror2Video 是同一份文件在自建 OpenList（首选镜像）上的地址。
	// 取值规则与 R2 那一组一致，只是对象落在自己的机器上。
	// 前端按 OpenList → R2 → 源站 逐级回退，任一为空即跳过该级。
	Mirror2Img   string    `json:"mirror2_img,omitempty"`
	Mirror2Video string    `json:"mirror2_video,omitempty"`
	FileSize     int64     `json:"file_size"`
	User         UserBrief `json:"user"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Tags         []string  `json:"tags"`
	LikeCount    int64     `json:"like_count"`
	ViewCount    *int64    `json:"view_count,omitempty"`
	AuditStatus  string    `json:"audit_status"`
	CreatedAt    web.Time  `json:"created_at"`
	UpdatedAt    web.Time  `json:"updated_at"`
}

// Page 是列表类接口的统一分页包装。
type Page struct {
	List      []Item `json:"list"`
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	TotalPage int    `json:"total_page"`
}

// decorate 把一行内容转换为对外形状（不带 R2 镜像地址，供内部/测试使用）。
func decorate(row store.Content, userMap map[uint64]store.User, likeCount int64, includeViewCount bool) Item {
	return decorateWith(nil, row, userMap, likeCount, includeViewCount)
}

// decorateWith 在 decorate 的基础上补 R2 备份地址。
// mirror 为 nil（未启用 R2）时与 decorate 完全等价。
// userMap 为批量预取的用户表（nil 时按需单查）；likeCount 为已统计的点赞数。
func decorateWith(mm app.MediaMirror, row store.Content, userMap map[uint64]store.User, likeCount int64, includeViewCount bool) Item {
	var author UserBrief
	var avatar string

	if row.GuestNickname != nil && *row.GuestNickname != "" {
		author = UserBrief{ID: 0, Username: *row.GuestNickname}
		if row.GuestEmail != nil {
			avatar = useravatar.URL(*row.GuestEmail, 80)
		}
	} else if u, ok := userMap[row.UserID]; ok {
		author = UserBrief{ID: u.ID, Username: u.Username}
		if u.Email != nil {
			avatar = useravatar.URL(*u.Email, 80)
		}
	} else {
		author = UserBrief{ID: row.UserID, Username: "unknown"}
	}

	filePath := deref(row.FilePath)
	thumbPath := deref(row.ThumbPath)
	isVideo := filePath != "" && IsVideoPath(filePath)

	img := ""
	if !isVideo && (filePath != "" || thumbPath != "") {
		if filePath != "" {
			img = FileURL(filePath)
		} else {
			img = FileURL(thumbPath)
		}
	}
	// 缩略图优先用生成的 thumb_path，未生成时回退原文件，避免破图。
	thumb := FileURL(thumbPath)
	if thumb == "" {
		thumb = FileURL(filePath)
	}

	item := Item{
		ID:          row.ID,
		Title:       row.Title,
		Text:        deref(row.Content),
		Thumb:       thumb,
		Video:       "",
		Img:         img,
		Origin:      FileURL(filePath),
		FileSize:    row.FileSize,
		User:        author,
		AvatarURL:   avatar,
		Tags:        ParseTags(row.Tags),
		LikeCount:   likeCount,
		AuditStatus: row.AuditStatus,
		CreatedAt:   web.TimeOf(row.CreatedAt),
		UpdatedAt:   web.TimeOf(row.UpdatedAt),
	}
	if isVideo {
		item.Video = FileURL(filePath)
	}
	// 镜像地址：与主地址同源同路径，只是换了 host。
	// 缩略图保持纯本地，不参与镜像，也就没有候选。
	if mm != nil && isMirrorablePath(filePath) {
		key := mirrorKey(filePath)
		if mirrorURL := mm.R2URL(key); mirrorURL != "" {
			if isVideo {
				item.MirrorVideo = mirrorURL
			} else {
				item.MirrorImg = mirrorURL
			}
		}
		// 首选地址独立取：未配置公开域名时它自己就是空串，
		// 不必（也不应）跟着 R2 那一级一起判空。
		if mirrorURL := mm.OpenListURL(key); mirrorURL != "" {
			if isVideo {
				item.Mirror2Video = mirrorURL
			} else {
				item.Mirror2Img = mirrorURL
			}
		}
	}
	if includeViewCount {
		v := row.ViewCount
		item.ViewCount = &v
	}
	return item
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

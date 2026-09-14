package content

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

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

var qqMailPattern = regexp.MustCompile(`^(\d{5,11})@qq\.com$`)

// MakeAvatarURL 由邮箱生成头像地址（不暴露原始邮箱）。
func MakeAvatarURL(email string, size int) string {
	if email == "" {
		return ""
	}
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if m := qqMailPattern.FindStringSubmatch(trimmed); m != nil {
		return "https://q.qlogo.cn/headimg_dl?dst_uin=" + m[1] + "&spec=100"
	}
	sum := md5.Sum([]byte(trimmed))
	return "https://www.gravatar.com/avatar/" + hex.EncodeToString(sum[:]) + "?d=identicon&s=" + itoa(size)
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
	ID          uint64    `json:"id"`
	Title       string    `json:"title"`
	Text        string    `json:"text"`
	Thumb       string    `json:"thumb"`
	Video       string    `json:"video"`
	Img         string    `json:"img"`
	Origin      string    `json:"origin,omitempty"`
	FileSize    int64     `json:"file_size"`
	User        UserBrief `json:"user"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Tags        []string  `json:"tags"`
	LikeCount   int64     `json:"like_count"`
	ViewCount   *int64    `json:"view_count,omitempty"`
	AuditStatus string    `json:"audit_status"`
	CreatedAt   web.Time  `json:"created_at"`
	UpdatedAt   web.Time  `json:"updated_at"`
}

// Page 是列表类接口的统一分页包装。
type Page struct {
	List      []Item `json:"list"`
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	TotalPage int    `json:"total_page"`
}

// decorate 把一行内容转换为对外形状。
// userMap 为批量预取的用户表（nil 时按需单查）；likeCount 为已统计的点赞数。
func decorate(row store.Content, userMap map[uint64]store.User, likeCount int64, includeViewCount bool) Item {
	var author UserBrief
	var avatar string

	if row.GuestNickname != nil && *row.GuestNickname != "" {
		author = UserBrief{ID: 0, Username: *row.GuestNickname}
		if row.GuestEmail != nil {
			avatar = MakeAvatarURL(*row.GuestEmail, 80)
		}
	} else if u, ok := userMap[row.UserID]; ok {
		author = UserBrief{ID: u.ID, Username: u.Username}
		if u.Email != nil {
			avatar = MakeAvatarURL(*u.Email, 80)
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

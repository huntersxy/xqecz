package content

import (
	"encoding/json"
	"strings"

	"github.com/huntersxy/xqecz/server/internal/app"
	useravatar "github.com/huntersxy/xqecz/server/internal/avatar"
	"github.com/huntersxy/xqecz/server/internal/media"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

var videoExts = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".m4v": true, ".mkv": true,
	".avi": true, ".flv": true, ".ogv": true, ".wmv": true, ".3gp": true,
	".mpeg": true, ".mpg": true, ".ts": true, ".m2ts": true,
}

func IsVideoPath(p string) bool {
	dot := strings.LastIndex(p, ".")
	if dot < 0 {
		return false
	}
	return videoExts[strings.ToLower(p[dot:])]
}

func MediaTypeForPath(p string) string {
	if IsVideoPath(p) {
		return "video"
	}
	return "image"
}

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

func isMirrorablePath(rel string) bool {
	if rel == "" {
		return false
	}
	return !strings.HasPrefix(rel, "thumbs/") && !strings.HasPrefix(rel, "images/")
}

func mirrorKey(rel string) string {
	rel = strings.TrimPrefix(rel, "/")
	return strings.TrimPrefix(rel, "uploads/")
}

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

func ParseTagsParam(raw string) []string {
	out := []string{}
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

type UserBrief struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}

type Item struct {
	ID           uint64    `json:"id"`
	Title        string    `json:"title"`
	Text         string    `json:"text"`
	Thumb        string    `json:"thumb"`
	AtlasThumb   string    `json:"atlas_thumb"`
	MontageThumb string    `json:"montage_thumb"`
	Video        string    `json:"video"`
	Img          string    `json:"img"`
	Origin       string    `json:"origin,omitempty"`
	MirrorImg    string    `json:"mirror_img,omitempty"`
	MirrorVideo  string    `json:"mirror_video,omitempty"`
	Mirror2Img   string    `json:"mirror2_img,omitempty"`
	Mirror2Video string    `json:"mirror2_video,omitempty"`
	FileSize     int64     `json:"file_size"`
	User         UserBrief `json:"user"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Tags         []string  `json:"tags"`
	LikeCount    int64     `json:"like_count"`
	ViewCount    *int64    `json:"view_count,omitempty"`
	AuditStatus  string    `json:"audit_status"`
	AIEnhanced   bool      `json:"ai_enhanced"`
	CreatedAt    web.Time  `json:"created_at"`
	UpdatedAt    web.Time  `json:"updated_at"`
}

type Page struct {
	List      []Item `json:"list"`
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PageSize  int    `json:"page_size"`
	TotalPage int    `json:"total_page"`
}

func decorate(row store.Content, userMap map[uint64]store.User, likeCount int64, includeViewCount bool) Item {
	return decorateWith(nil, row, userMap, likeCount, includeViewCount)
}

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
	thumb := FileURL(thumbPath)
	if thumb == "" {
		thumb = FileURL(filePath)
	}

	item := Item{
		ID:           row.ID,
		Title:        row.Title,
		Text:         deref(row.Content),
		Thumb:        thumb,
		AtlasThumb:   media.AtlasThumbnailURL(FileURL(thumbPath)),
		MontageThumb: media.MontageThumbnailURL(FileURL(thumbPath)),
		Video:        "",
		Img:          img,
		Origin:       FileURL(filePath),
		FileSize:     row.FileSize,
		User:         author,
		AvatarURL:    avatar,
		Tags:         ParseTags(row.Tags),
		LikeCount:    likeCount,
		AuditStatus:  row.AuditStatus,
		AIEnhanced:   row.AIEnhanced,
		CreatedAt:    web.TimeOf(row.CreatedAt),
		UpdatedAt:    web.TimeOf(row.UpdatedAt),
	}
	if isVideo {
		item.Video = FileURL(filePath)
	}
	if mm != nil && isMirrorablePath(filePath) {
		key := mirrorKey(filePath)
		if mirrorURL := mm.R2URL(key); mirrorURL != "" {
			if isVideo {
				item.MirrorVideo = mirrorURL
			} else {
				item.MirrorImg = mirrorURL
			}
		}
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

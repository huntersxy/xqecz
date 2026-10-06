package config

import (
	"os"
	"strings"
	"time"
)

// OpenListConfig 是自建媒体源（广州机的 OpenList）的连接参数。
//
// 与 R2Config 的形态差异都源于协议本身：OpenList 没有桶、没有 AK/SK，
// 只有「基址 + admin token + 一个挂载点」。挂载点（比如 `/` → 媒体目录）
// 配在 OpenList 服务端，这里只认管理 API 的入口。
type OpenListConfig struct {
	// Endpoint 是 OpenList 的基址，如 http://127.0.0.1:5244
	Endpoint string
	// Token 是 admin token，取自 data.db 的 x_setting_items.key='token'。
	Token string
	// Prefix 是对象在挂载内的前缀，需与 R2_PREFIX 保持一致，
	// 否则同一个内容在两边的对象名不同，前端换源会 404。
	Prefix string
	// PublicBase 是浏览器可直连的公开基址，**必须带免签直链前缀 `/d`**，
	// 如 https://drive.xiey.work/d。
	//
	// 不能只写 https://drive.xiey.work：那个地址在 OpenList 侧是 SPA 前端，
	// 任何未加前缀的路径都回 index.html（200 text/html），
	// ObjectURL 拼出的地址会「换源成功但图都是 HTML」。
	// 前缀与实例侧 sign_all=false 是配套的两件事：没有它 /d/ 裸访问一律 401。
	// 留空 = 只镜像、不给前端候选源。
	PublicBase string
	// SyncEvery 是回填周期。
	SyncEvery time.Duration
	// UploadTimeout 是单次上传超时。
	UploadTimeout time.Duration
}

// Enabled 表示基址与 token 齐备。
func (c OpenListConfig) Enabled() bool { return c.Endpoint != "" && c.Token != "" }

// Exposed 表示对象有浏览器可直连的公开地址。
func (c OpenListConfig) Exposed() bool { return c.Enabled() && c.PublicBase != "" }

// ObjectURL 返回对象的公开访问地址。
func (c OpenListConfig) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(c.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

// loadOpenList 读取环境变量。缺 Endpoint 或 Token 即整体停用
// （与 TinyPNG 缺 Key、R2 缺凭据同一套「要么齐备要么不做」的取向）。
func loadOpenList() OpenListConfig {
	c := OpenListConfig{
		Endpoint:      strings.TrimSpace(os.Getenv("OPENLIST_ENDPOINT")),
		Token:         strings.TrimSpace(os.Getenv("OPENLIST_TOKEN")),
		Prefix:        env("OPENLIST_PREFIX", "uploads"),
		PublicBase:    strings.TrimRight(strings.TrimSpace(os.Getenv("OPENLIST_PUBLIC_BASE")), "/"),
		SyncEvery:     time.Duration(envInt("OPENLIST_SYNC_INTERVAL_SECONDS", 300)) * time.Second,
		UploadTimeout: time.Duration(envInt("OPENLIST_UPLOAD_TIMEOUT_SECONDS", 120)) * time.Second,
	}
	if !c.Enabled() {
		return OpenListConfig{}
	}
	return c
}

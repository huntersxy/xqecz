package config

import (
	"fmt"
	"strings"
	"time"
)

// R2Config 是 Cloudflare R2（S3 兼容）镜像的配置。
//
// 设计取向与 TinyPNG 一致：凭据不全就整体停用，任务静默休眠、不报错，
// 这样本地开发、未开通 R2 的环境都能照常跑，接入与否只由 .env 决定。
type R2Config struct {
	AccountID string
	AccessKey string
	SecretKey string
	Bucket    string
	// Endpoint 默认由 AccountID 推导；私有化/测试可用 R2_ENDPOINT 覆盖。
	Endpoint string
	// PublicBase 是对象对浏览器公开访问的基址（自定义域名或 r2.dev），
	// 尾斜杠会被去掉；留空表示「只镜像不对外暴露」，前端拿不到备用源。
	PublicBase string
	// Prefix 是对象在桶内的公共前缀，便于与桶里其它用途的对象区分。
	Prefix string
	// SyncEvery 是回填任务扫描存量待同步文件的间隔。
	SyncEvery time.Duration
	// UploadTimeout 是单次 PUT 的超时上限。
	UploadTimeout time.Duration
}

// Enabled 表示是否具备完整凭据可以工作。
func (c R2Config) Enabled() bool {
	return c.AccountID != "" && c.AccessKey != "" && c.SecretKey != "" && c.Bucket != ""
}

// Exposed 表示对象是否有可供浏览器直连的公开地址。
// 只有凭据完备且配置了 PublicBase 时，详情页才存在「本地 vs R2」二选一。
func (c R2Config) Exposed() bool { return c.Enabled() && c.PublicBase != "" }

// ObjectURL 返回对象的公开访问地址；key 形如 "14_thumb.webp"（桶内名为 Prefix/key）。
func (c R2Config) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(c.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

// loadR2 从环境变量组装 R2 配置并做归一化（去空白、补默认值、去尾部斜杠）。
func loadR2() R2Config {
	var c R2Config
	c.AccountID = env("R2_ACCOUNT_ID", "")
	c.AccessKey = env("R2_ACCESS_KEY_ID", "")
	c.SecretKey = env("R2_SECRET_ACCESS_KEY", "")
	c.Bucket = env("R2_BUCKET", "")
	c.Prefix = strings.Trim(strings.TrimSpace(env("R2_PREFIX", "uploads")), "/")

	c.Endpoint = strings.TrimRight(env("R2_ENDPOINT", ""), "/")
	if c.Endpoint == "" && c.AccountID != "" {
		c.Endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)
	}
	c.PublicBase = strings.TrimRight(env("R2_PUBLIC_BASE", ""), "/")

	c.SyncEvery = time.Duration(envInt("R2_SYNC_INTERVAL_SECONDS", 300)) * time.Second
	c.UploadTimeout = time.Duration(envInt("R2_UPLOAD_TIMEOUT_SECONDS", 120)) * time.Second
	return c
}

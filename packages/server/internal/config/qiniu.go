package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// QiniuConfig 是七牛云 Kodo（S3 兼容）作为**一级替补**镜像的配置。
//
// 定位：R2 可达就用 R2；R2 不可达时用七牛；七牛也不可达才回源站。
// 推送走 S3 接口（AWS SigV4，复用 internal/r2），取图走 CDN 加速域名，
// 两者共用同一对 AK/SK——七牛是两套签名（SigV4 管数据面、Qiniu/QBox 管管理面）。
//
// 停用策略与 R2/TinyPNG 一致：凭据不全就整体休眠，本地开发不必配。
type QiniuConfig struct {
	AccessKey string
	SecretKey string
	// Bucket 是空间名；Region 是存储区域（决定 S3 端点）。
	Bucket string
	Region string
	// Endpoint 是推送用的 S3 端点，留空时由 Region 推导。
	Endpoint string
	// PublicBase 是浏览器取图的 CDN 加速域名（形如 https://img.xiey.work），
	// 与推送端点是两个不同的东西，切勿混填。
	PublicBase string
	// Prefix 是对象在桶内的公共前缀，与 R2 保持同名便于对账。
	Prefix string

	SyncEvery     time.Duration
	UploadTimeout time.Duration
	// QuotaEvery 是额度检查的间隔；统计数据本身有约 5 分钟延迟，
	// 间隔显著大于它才有意义。
	QuotaEvery time.Duration
	// Threshold 是额度触顶比例（0 表示用默认 0.9）。
	Threshold float64
	// EgressCapBytes 是外网流出的自设上限。
	// 七牛对该项**没有免费额度**，只能自己划线；0 表示不判定
	//（桶保持私有时 S3 直连走不通，也就没有这笔开销）。
	EgressCapBytes int64
	// TimeKey 是时间戳防盗链的密钥（控制台主密钥）。
	// 留空表示不对替补层地址签名——**一旦控制台开启了时间戳防盗链，
	// 这里就必须配上**，否则替补层发出去的地址一律 403。
	TimeKey string
}

// Enabled 表示凭据与空间齐备，可以推送。
func (c QiniuConfig) Enabled() bool {
	return c.AccessKey != "" && c.SecretKey != "" && c.Bucket != "" && c.Region != ""
}

// Exposed 表示存在可供浏览器直连的公开地址——只有这一项为真，
// 详情页才会把七牛作为候选源交给前端探测。
func (c QiniuConfig) Exposed() bool { return c.Enabled() && c.PublicBase != "" }

// CDNDomain 返回加速域名的主机名（fusion 流量查询的 domains 参数），
// 从 PublicBase 解析，不再单独配一个环境变量。
// 解析不出来时返回空串，此时 CDN 下载流量不参与额度判定。
func (c QiniuConfig) CDNDomain() string {
	u, err := url.Parse(c.PublicBase)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// ObjectURL 返回对象的公开访问地址；key 形如 "uploads/14.webp"（桶内名即 key）。
func (c QiniuConfig) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(c.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

// loadQiniu 从环境变量组装七牛配置并归一化。
//
// R2 用 AccountID 推导端点，七牛没有账号 ID 概念，改用**区域**推导：
// s3.cn-south-1.qiniucs.com 等，见 developer.qiniu.com/kodo/4088/s3-access-domainname。
func loadQiniu() QiniuConfig {
	var c QiniuConfig
	c.AccessKey = env("QINIU_ACCESS_KEY", "")
	c.SecretKey = env("QINIU_SECRET_KEY", "")
	c.Bucket = env("QINIU_BUCKET", "")
	c.Region = env("QINIU_REGION", "")
	c.Prefix = strings.Trim(strings.TrimSpace(env("QINIU_PREFIX", "uploads")), "/")

	c.Endpoint = strings.TrimRight(env("QINIU_ENDPOINT", ""), "/")
	if c.Endpoint == "" && c.Region != "" {
		c.Endpoint = fmt.Sprintf("https://s3.%s.qiniucs.com", c.Region)
	}
	c.PublicBase = strings.TrimRight(env("QINIU_PUBLIC_BASE", ""), "/")

	c.SyncEvery = time.Duration(envInt("QINIU_SYNC_INTERVAL_SECONDS", 300)) * time.Second
	c.UploadTimeout = time.Duration(envInt("QINIU_UPLOAD_TIMEOUT_SECONDS", 120)) * time.Second
	c.QuotaEvery = time.Duration(envInt("QINIU_QUOTA_INTERVAL_SECONDS", 600)) * time.Second
	c.Threshold = envFloat("QINIU_QUOTA_THRESHOLD", 0.9)

	// 以 GB 计的自设上限；0 = 不判定。负数按 0 处理。
	if gb := envFloat("QINIU_EGRESS_CAP_GB", 0); gb > 0 {
		c.EgressCapBytes = int64(gb * float64(1<<30))
	}
	c.TimeKey = strings.TrimSpace(os.Getenv("QINIU_TIME_KEY"))
	return c
}

// envFloat 解析浮点环境变量，解析失败或未设置时回落到 def。
func envFloat(key string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

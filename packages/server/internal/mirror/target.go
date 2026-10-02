package mirror

import (
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
)

// Target 描述一个镜像目标：怎么连（端点/凭据/桶）、怎么被访问（公开基址）。
//
// 与具体云厂商解耦——Cloudflare R2 与七牛 Kodo 都经由它接入，
// 同一套推送去重、归档与回填逻辑对两者一视同仁。
// 唯一的差别是签名区域：R2 固定 auto，七牛必须是真实 region
// （如 cn-south-1），否则 SigV4 必然校验失败。
type Target struct {
	// Name 是日志里的目标名（r2 / qiniu）。空值按 "r2" 处理。
	Name       string
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	Region     string
	Prefix     string
	PublicBase string
	// SyncEvery 是回填周期；两目标共用同一节奏，取首个非零值。
	SyncEvery time.Duration
	Timeout   time.Duration
}

// Enabled 表示凭据与桶齐备，可以推送。
func (t Target) Enabled() bool {
	return t.Endpoint != "" && t.AccessKey != "" && t.SecretKey != "" && t.Bucket != ""
}

// Exposed 表示对象有浏览器可直连的公开地址。
func (t Target) Exposed() bool { return t.Enabled() && t.PublicBase != "" }

// ObjectURL 返回桶内 key 的公开访问地址。
func (t Target) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(t.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

// LogName 返回日志用的目标名，未设置时回落为 "r2"（历史语义）。
func (t Target) LogName() string {
	if t.Name == "" {
		return "r2"
	}
	return t.Name
}

// TargetFromR2 把 R2 配置映射为镜像目标。
// R2 的签名区域固定 auto——签名用不到桶的位置，取值稳定即可。
// Endpoint 留空时按 AccountID 推导，与 config.loadR2 同一条规则：
// 否则只有 AccountID 的配置会被判成「未启用」，公开地址随之全空。
func TargetFromR2(c config.R2Config) Target {
	endpoint := c.Endpoint
	if endpoint == "" && c.AccountID != "" {
		endpoint = "https://" + c.AccountID + ".r2.cloudflarestorage.com"
	}
	return Target{
		Name:       "r2",
		Endpoint:   endpoint,
		AccessKey:  c.AccessKey,
		SecretKey:  c.SecretKey,
		Bucket:     c.Bucket,
		Region:     "auto",
		Prefix:     c.Prefix,
		PublicBase: c.PublicBase,
		SyncEvery:  c.SyncEvery,
		Timeout:    c.UploadTimeout,
	}
}

// TargetFromQiniu 把七牛配置映射为镜像目标。
// Region 必须是真实值（七牛 S3 端点按 region 推导，SigV4 也按它签名）。
func TargetFromQiniu(c config.QiniuConfig) Target {
	return Target{
		Name:       "qiniu",
		Endpoint:   c.Endpoint,
		AccessKey:  c.AccessKey,
		SecretKey:  c.SecretKey,
		Bucket:     c.Bucket,
		Region:     c.Region,
		Prefix:     c.Prefix,
		PublicBase: c.PublicBase,
		// 回填周期必须有值：worker 用它建 time.NewTicker，0 会直接 panic。
		SyncEvery: c.SyncEvery,
		Timeout:   c.UploadTimeout,
	}
}

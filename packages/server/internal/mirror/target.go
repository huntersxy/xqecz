package mirror

import (
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
)

// Driver 决定用什么协议连目标，也决定 Target.Enabled 的判定条件。
type Driver string

const (
	// DriverR2 是 S3 兼容对象存储，凭据 = 端点 + AK/SK + 桶。
	DriverR2 Driver = "r2"
	// DriverOpenList 是 OpenList 的 HTTP API，凭据 = 基址 + admin token。
	// 它没有桶的概念（"桶"是挂载点，落在服务端配置里），故不吃 AK/SK/Bucket。
	DriverOpenList Driver = "openlist"
)

// Target 描述一个镜像目标：怎么连（驱动/端点/凭据）、怎么被访问（公开基址）。
//
// 与具体云厂商解耦——Cloudflare R2 与 OpenList 都经由它接入，
// 同一套推送去重、归档与回填逻辑对两者一视同仁；差异全部收敛在
// Enabled（判据不同）与 Setup.newStore（协议不同）两处。
type Target struct {
	// Name 是日志里的目标名（r2 / openlist）。空值按 "r2" 处理。
	Name string
	// Driver 为空时按 DriverR2 处理，兼容既有配置。
	Driver    Driver
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	// Token 是 OpenList 的 admin token（OpenList 无 AK/SK）。
	Token      string
	Prefix     string
	PublicBase string
	// SyncEvery 是回填周期；两目标共用同一节奏，取首个非零值。
	SyncEvery time.Duration
	Timeout   time.Duration
}

// driver 返回归一化后的驱动名。
func (t Target) driver() Driver {
	if t.Driver == "" {
		return DriverR2
	}
	return t.Driver
}

// Enabled 表示连接参数齐备，可以推送。判据随驱动变化：
// S3 需要端点+AK/SK+桶；OpenList 只需要基址+token。
func (t Target) Enabled() bool {
	switch t.driver() {
	case DriverOpenList:
		return t.Endpoint != "" && t.Token != ""
	default:
		return t.Endpoint != "" && t.AccessKey != "" && t.SecretKey != "" && t.Bucket != ""
	}
}

// Exposed 表示对象有浏览器可直连的公开地址。
func (t Target) Exposed() bool { return t.Enabled() && t.PublicBase != "" }

// ObjectURL 返回对象 key 的公开访问地址。
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
		Driver:     DriverR2,
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

// TargetFromOpenList 把 OpenList 配置映射为镜像目标。
// Prefix 必须与 R2 保持一致，否则同一个内容在两边的对象路径不同，
// 前端换源会 404（历史上两家的 key 就是同一套 uploads/<name>）。
//
// PublicBase 必须带上实例的免签直链前缀（`/d`）：OpenList 的裸域名是 SPA，
// 未加前缀的路径一律回 index.html，ObjectURL 拼出的地址会变成 200 text/html。
func TargetFromOpenList(c config.OpenListConfig) Target {
	return Target{
		Name:       "openlist",
		Driver:     DriverOpenList,
		Endpoint:   c.Endpoint,
		Token:      c.Token,
		Prefix:     c.Prefix,
		PublicBase: c.PublicBase,
		// 回填周期必须有值：worker 用它建 time.NewTicker，0 会直接 panic。
		SyncEvery: c.SyncEvery,
		Timeout:   c.UploadTimeout,
	}
}

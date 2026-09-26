// Package mysqldsn 统一组装 MySQL/TiDB 连接串，供服务端与 cmd 下的排查工具复用。
//
// 单独成包是为了让「排查工具」不必依赖 gorm；更重要的是把 TLS 参数收敛到一处：
// 每个调用点各写一份 DSN 时，新增连接参数（如托管实例强制的加密连接）必然漏配。
package mysqldsn

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Options 描述一个 MySQL 连接目标。
type Options struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	// TLS 透传给驱动的 tls 参数："" 表示不加密（自建 MySQL 的既有行为），
	// "true" 加密并校验服务端证书，"skip-verify" 只加密不校验。
	// TiDB Cloud Serverless 拒绝明文连接（错误 1105 insecure transport），必须设为 "true"。
	TLS string
	// ParseTime 为 true 时驱动把 datetime 解析成 time.Time（业务读写需要）；
	// 逐字节搬运数据时应关闭，避免时区换算改写时间值。
	ParseTime bool
	// Timeout 是建立 TCP 连接的超时（秒）。0 表示交给驱动默认值。
	Timeout int
}

// Format 返回可直接交给 sql.Open("mysql", ...) 的 DSN。
func Format(o Options) string {
	c := mysql.NewConfig()
	c.User = o.User
	c.Passwd = o.Password
	c.Net = "tcp"
	c.Addr = fmt.Sprintf("%s:%d", o.Host, o.Port)
	c.DBName = o.Database
	c.ParseTime = o.ParseTime
	c.Loc = time.Local
	c.Params = map[string]string{"charset": "utf8mb4"}
	if o.Timeout > 0 {
		c.Timeout = time.Duration(o.Timeout) * time.Second
	}
	if o.TLS != "" {
		c.Params["tls"] = o.TLS
	}
	return c.FormatDSN()
}

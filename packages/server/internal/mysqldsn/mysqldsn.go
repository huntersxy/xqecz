package mysqldsn

import (
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Options struct {
	Host      string
	Port      int
	User      string
	Password  string
	Database  string
	TLS       string
	ParseTime bool
	Timeout   int
}

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

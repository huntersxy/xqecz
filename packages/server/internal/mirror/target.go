package mirror

import (
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
)

type Driver string

const (
	DriverR2       Driver = "r2"
	DriverOpenList Driver = "openlist"
)

type Target struct {
	Name       string
	Driver     Driver
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	Region     string
	Token      string
	Prefix     string
	PublicBase string
	SyncEvery  time.Duration
	Timeout    time.Duration
}

func (t Target) driver() Driver {
	if t.Driver == "" {
		return DriverR2
	}
	return t.Driver
}

func (t Target) Enabled() bool {
	switch t.driver() {
	case DriverOpenList:
		return t.Endpoint != "" && t.Token != ""
	default:
		return t.Endpoint != "" && t.AccessKey != "" && t.SecretKey != "" && t.Bucket != ""
	}
}

func (t Target) Exposed() bool { return t.Enabled() && t.PublicBase != "" }

func (t Target) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(t.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

func (t Target) LogName() string {
	if t.Name == "" {
		return "r2"
	}
	return t.Name
}

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

func TargetFromOpenList(c config.OpenListConfig) Target {
	return Target{
		Name:       "openlist",
		Driver:     DriverOpenList,
		Endpoint:   c.Endpoint,
		Token:      c.Token,
		Prefix:     c.Prefix,
		PublicBase: c.PublicBase,
		SyncEvery:  c.SyncEvery,
		Timeout:    c.UploadTimeout,
	}
}

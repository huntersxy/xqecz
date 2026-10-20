package config

import (
	"fmt"
	"strings"
	"time"
)

type R2Config struct {
	AccountID     string
	AccessKey     string
	SecretKey     string
	Bucket        string
	Endpoint      string
	PublicBase    string
	Prefix        string
	SyncEvery     time.Duration
	UploadTimeout time.Duration
}

func (c R2Config) Enabled() bool {
	return c.AccountID != "" && c.AccessKey != "" && c.SecretKey != "" && c.Bucket != ""
}

func (c R2Config) Exposed() bool { return c.Enabled() && c.PublicBase != "" }

func (c R2Config) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(c.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

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

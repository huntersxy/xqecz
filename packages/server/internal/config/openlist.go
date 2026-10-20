package config

import (
	"os"
	"strings"
	"time"
)

type OpenListConfig struct {
	Endpoint      string
	Token         string
	Prefix        string
	PublicBase    string
	SyncEvery     time.Duration
	UploadTimeout time.Duration
}

func (c OpenListConfig) Enabled() bool { return c.Endpoint != "" && c.Token != "" }

func (c OpenListConfig) Exposed() bool { return c.Enabled() && c.PublicBase != "" }

func (c OpenListConfig) ObjectURL(key string) string {
	base := strings.TrimRight(strings.TrimSpace(c.PublicBase), "/")
	if base == "" || key == "" {
		return ""
	}
	return base + "/" + strings.TrimPrefix(key, "/")
}

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

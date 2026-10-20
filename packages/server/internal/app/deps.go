package app

import (
	"github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/logx"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"gorm.io/gorm"
)

type MediaMirror interface {
	R2URL(rel string) string
	OpenListURL(rel string) string
	PushAsync(rel, absPath string)
	ArchiveOriginalAsync(rel, origPath string)
}

type Deps struct {
	Cfg    config.Config
	DB     *gorm.DB
	Redis  *cache.Client
	Mirror MediaMirror
	Vision *vision.Service
	AI     *aigen.Service
	Logs   *logx.Buffer
}

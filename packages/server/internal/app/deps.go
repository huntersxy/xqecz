// Package app 定义各层共享的依赖集，避免 web 与业务模块之间出现循环依赖。
package app

import (
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"gorm.io/gorm"
)

type Deps struct {
	Cfg   config.Config
	DB    *gorm.DB
	Redis *cache.Client
}

// Package app 定义各层共享的依赖集，避免 web 与业务模块之间出现循环依赖。
package app

import (
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"gorm.io/gorm"
)

// MediaMirror 是各业务模块对「媒体镜像」的全部需求：既能给出 R2 上的公开地址，
// 又能把本地文件推上去。用接口而非具体类型（mirror.Setup），
// 让 app 包不反向依赖业务包，也便于测试注入假实现。
type MediaMirror interface {
	// PublicURL 返回对象公开地址；未启用或未配置公开域名时返回空串。
	PublicURL(rel string) string
	// PushAsync 在后台把本地文件推送到 R2，失败只记日志（由回填任务兜底）。
	PushAsync(rel, absPath string)
}

type Deps struct {
	Cfg   config.Config
	DB    *gorm.DB
	Redis *cache.Client
	// Mirror 为 R2 媒体镜像；nil 表示未接入，所有镜像相关行为退化为空操作。
	Mirror MediaMirror
}

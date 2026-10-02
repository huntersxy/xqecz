// Package app 定义各层共享的依赖集，避免 web 与业务模块之间出现循环依赖。
package app

import (
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"gorm.io/gorm"
)

// MediaMirror 是各业务模块对「媒体镜像」的全部需求：既能给出对象存储上的公开地址，
// 又能把本地文件推上去。用接口而非具体类型（mirror.Chain），
// 让 app 包不反向依赖业务包，也便于测试注入假实现。
type MediaMirror interface {
	// PublicURL 返回 R2 那一级的对象公开地址（前端**次选**）；
	// 未启用或未配置公开域名时返回空串。
	PublicURL(rel string) string
	// Mirror2URL 返回七牛那一级的对象公开地址（前端**首选**）。
	// 额度触顶时返回空串——前端因此连候选都拿不到，不会被引到一个注定失败的地址上。
	// 与 R2 那级的区别只有两点：受额度闸门约束；地址带时间戳防盗链签名（若开启）。
	Mirror2URL(rel string) string
	// PushAsync 在后台把本地文件推送到各目标，失败只记日志（由回填任务兜底）。
	PushAsync(rel, absPath string)
	// ArchiveOriginalAsync 在后台把压缩前的原图归档到各目标的 original/ 命名空间，
	// 与现役对象各留一份；失败只记日志（由回填任务从垃圾桶兜底）。
	ArchiveOriginalAsync(rel, origPath string)
}

type Deps struct {
	Cfg   config.Config
	DB    *gorm.DB
	Redis *cache.Client
	// Mirror 为媒体镜像链（主 R2 + 替补七牛）；nil 表示未接入，
	// 所有镜像相关行为退化为空操作。
	Mirror MediaMirror
}

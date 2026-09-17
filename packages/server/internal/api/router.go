// Package api 组装 HTTP 路由与全部业务模块。
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/modules/admin"
	"github.com/huntersxy/xqecz/server/internal/modules/apikey"
	"github.com/huntersxy/xqecz/server/internal/modules/auth"
	"github.com/huntersxy/xqecz/server/internal/modules/comment"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/modules/poll"
	"github.com/huntersxy/xqecz/server/internal/web"
)

// New 创建 HTTP 引擎并挂载全部模块路由。
// opts 透传给内容模块（目前只承载 R2 媒体镜像）。
func New(deps app.Deps, opts ...content.Options) *gin.Engine {
	return web.New(deps,
		// 媒体目录：带 ?download=1 时以附件下载，文件名取内容标题。
		func(r *gin.Engine) { content.New(deps).RegisterMedia(r) },
		func(api *gin.RouterGroup) {
			api.GET("/health", func(c *gin.Context) { web.OK(c, gin.H{"ok": true}, "ok") })
			auth.Register(api, deps)
			contentHandler := content.Register(api, deps, opts...)
			comment.Register(api, deps)
			poll.Register(api, deps)
			apikey.Register(api, deps)
			admin.Register(api, deps, contentHandler)
		})
}

// Run 启动 HTTP 服务并阻塞至退出信号。
// 用闭包把可选项钉进构建函数，避免改动 web.Run 的签名。
func Run(deps app.Deps, opts ...content.Options) error {
	return web.Run(deps, func(d app.Deps) *gin.Engine { return New(d, opts...) })
}

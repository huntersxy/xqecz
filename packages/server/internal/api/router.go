package api

import (
	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/modules/admin"
	aigenhttp "github.com/huntersxy/xqecz/server/internal/modules/aigen"
	"github.com/huntersxy/xqecz/server/internal/modules/apikey"
	"github.com/huntersxy/xqecz/server/internal/modules/auth"
	"github.com/huntersxy/xqecz/server/internal/modules/comment"
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	creativehttp "github.com/huntersxy/xqecz/server/internal/modules/creative"
	"github.com/huntersxy/xqecz/server/internal/modules/poll"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func New(deps app.Deps, opts ...content.Options) *gin.Engine {
	return web.New(deps,
		func(r *gin.Engine) { content.New(deps).RegisterMedia(r) },
		func(api *gin.RouterGroup) {
			api.GET("/health", func(c *gin.Context) { web.OK(c, gin.H{"ok": true}, "ok") })
			auth.Register(api, deps)
			contentHandler := content.Register(api, deps, opts...)
			comment.Register(api, deps)
			poll.Register(api, deps)
			apikey.Register(api, deps)
			admin.Register(api, deps, contentHandler)
			aigenhttp.Register(api, deps, contentHandler)
			creativehttp.Register(api, deps)
		})
}

func Run(deps app.Deps, opts ...content.Options) error {
	return web.Run(deps, func(d app.Deps) *gin.Engine { return New(d, opts...) })
}

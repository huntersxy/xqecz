package web

import "github.com/gin-gonic/gin"

const ctxIdentity = "xqecz.identity"

// Identity 表示一次请求的调用者，来源为 Session Cookie 或 X-API-Key。
type Identity struct {
	UID         uint64
	Username    string
	IsAdmin     bool
	IsAPIKey    bool
	APIKeyID    uint64
	APIKeyPerms []string
}

func setIdentity(c *gin.Context, id Identity) { c.Set(ctxIdentity, id) }

// IdentityOf 取出当前请求身份；未认证时返回 false。
func IdentityOf(c *gin.Context) (Identity, bool) {
	v, ok := c.Get(ctxIdentity)
	if !ok {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}

// MustIdentity 供已挂 RequireAuth 的路由使用。
func MustIdentity(c *gin.Context) Identity {
	id, _ := IdentityOf(c)
	return id
}

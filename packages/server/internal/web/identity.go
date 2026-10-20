package web

import "github.com/gin-gonic/gin"

const ctxIdentity = "xqecz.identity"

type Identity struct {
	UID         uint64
	Username    string
	IsAdmin     bool
	IsAPIKey    bool
	APIKeyID    uint64
	APIKeyPerms []string
}

func setIdentity(c *gin.Context, id Identity) { c.Set(ctxIdentity, id) }

func IdentityOf(c *gin.Context) (Identity, bool) {
	v, ok := c.Get(ctxIdentity)
	if !ok {
		return Identity{}, false
	}
	id, ok := v.(Identity)
	return id, ok
}

func MustIdentity(c *gin.Context) Identity {
	id, _ := IdentityOf(c)
	return id
}

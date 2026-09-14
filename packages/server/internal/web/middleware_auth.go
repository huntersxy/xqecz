package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
)

const SessionCookie = "session_id"

// RequireAuth 与旧 AuthGuard 等价：优先校验 X-API-Key，其次校验 Session Cookie。
func RequireAuth(deps app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if raw := strings.TrimSpace(c.GetHeader("X-API-Key")); raw != "" {
			id, err := authenticateAPIKey(c.Request.Context(), deps, raw)
			if err != nil {
				Fail(c, 401, err.Error())
				return
			}
			setIdentity(c, id)
			c.Next()
			return
		}

		sid, _ := c.Cookie(SessionCookie)
		if sid == "" {
			Fail(c, 401, "未登录")
			return
		}
		uid, ok := deps.Redis.GetSession(c.Request.Context(), sid)
		if !ok {
			Fail(c, 401, "登录已过期")
			return
		}
		var user store.User
		if err := deps.DB.WithContext(c.Request.Context()).First(&user, uid).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				Fail(c, 401, "用户不存在")
				return
			}
			Fail(c, 500, "服务异常")
			return
		}
		if user.IsBanned == 1 {
			Fail(c, 401, "账号已被封禁")
			return
		}
		setIdentity(c, Identity{UID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin == 1})
		c.Next()
	}
}

// OptionalAuth 与旧 OptionalAuthGuard 等价：仅识别 Session，失败不阻断。
func OptionalAuth(deps app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid, _ := c.Cookie(SessionCookie)
		if sid == "" {
			c.Next()
			return
		}
		uid, ok := deps.Redis.GetSession(c.Request.Context(), sid)
		if !ok {
			c.Next()
			return
		}
		var user store.User
		if err := deps.DB.WithContext(c.Request.Context()).First(&user, uid).Error; err != nil || user.IsBanned == 1 {
			c.Next()
			return
		}
		setIdentity(c, Identity{UID: user.ID, Username: user.Username, IsAdmin: user.IsAdmin == 1})
		c.Next()
	}
}

// RequireAdmin 与旧 AdminGuard 等价。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := IdentityOf(c)
		if !ok || !id.IsAdmin {
			Fail(c, 403, "需要管理员权限")
			return
		}
		c.Next()
	}
}

// RequireAPIKeyPermission 与旧 ApiKeyPermissionGuard 等价：仅约束密钥调用，Session 用户不受限。
func RequireAPIKeyPermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := IdentityOf(c)
		if !ok || !id.IsAPIKey {
			c.Next()
			return
		}
		for _, p := range id.APIKeyPerms {
			if p == permission {
				c.Next()
				return
			}
		}
		Fail(c, 403, "API 密钥缺少 "+permission+" 权限")
	}
}

func authenticateAPIKey(ctx context.Context, deps app.Deps, raw string) (Identity, error) {
	sum := sha256.Sum256([]byte(raw))
	hash := hex.EncodeToString(sum[:])

	var row store.APIKey
	err := deps.DB.WithContext(ctx).Where("key_hash = ? AND is_active = 1", hash).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Identity{}, errors.New("API 密钥无效")
	}
	if err != nil {
		return Identity{}, errors.New("服务异常")
	}

	var perms []string
	if err := json.Unmarshal([]byte(row.Permissions), &perms); err != nil {
		perms = nil
	}

	// 最后使用时间每分钟至多落库一次，避免高频请求打爆 DB。
	if row.LastUsedAt == nil || time.Since(*row.LastUsedAt) > time.Minute {
		_ = deps.DB.WithContext(ctx).Model(&store.APIKey{}).
			Where("id = ?", row.ID).Update("last_used_at", time.Now()).Error
	}

	return Identity{UID: row.UserID, Username: "API", IsAPIKey: true, APIKeyID: row.ID, APIKeyPerms: perms}, nil
}

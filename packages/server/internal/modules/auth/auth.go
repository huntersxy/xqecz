// Package auth 实现注册、登录、会话与账户设置接口，行为与旧 NestJS 实现逐字段对齐。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	deps app.Deps
}

func New(deps app.Deps) *Handler { return &Handler{deps: deps} }

// Register 挂载 /auth 路由。
func Register(api *gin.RouterGroup, deps app.Deps) {
	h := New(deps)
	g := api.Group("/auth")
	g.POST("/register", h.register)
	g.POST("/login", h.login)
	g.POST("/logout", h.logout)
	g.POST("/change-password", web.RequireAuth(deps), h.changePassword)
	g.GET("/me", web.RequireAuth(deps), h.me)
	g.PUT("/email", web.RequireAuth(deps), h.updateEmail)
}

var usernamePattern = regexp.MustCompile("^[\\w\u4e00-\u9fff]+$")

type registerReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}

	switch n := runeLen(req.Username); {
	case n < 2 || n > 32:
		web.Fail(c, 400, "用户名长度需在 2 到 32 个字符之间")
		return
	case !usernamePattern.MatchString(strings.TrimSpace(req.Username)):
		web.Fail(c, 400, "用户名只能包含字母、数字、下划线或中文")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		web.Fail(c, 400, "请输入有效的邮箱地址")
		return
	}
	if runeLen(req.Password) < 6 {
		web.Fail(c, 400, "密码长度不能少于 6 个字符")
		return
	}

	db := h.deps.DB.WithContext(c.Request.Context())
	var count int64
	if err := db.Model(&store.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if count > 0 {
		web.Fail(c, 409, "用户名已存在")
		return
	}
	if err := db.Model(&store.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if count > 0 {
		web.Fail(c, 409, "邮箱已被注册")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	email := req.Email
	user := store.User{Username: req.Username, Email: &email, Password: string(hash)}
	if err := db.Create(&user).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, gin.H{"user_id": user.ID}, "注册成功")
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}

	var user store.User
	err := h.deps.DB.WithContext(c.Request.Context()).
		Where("username = ?", req.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 401, "用户名或密码错误")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		web.Fail(c, 401, "用户名或密码错误")
		return
	}
	if user.IsBanned == 1 {
		web.Fail(c, 403, "账号已被封禁")
		return
	}

	sid := newSessionID()
	if err := h.deps.Redis.SetSession(c.Request.Context(), sid, user.ID); err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.SetCookie(c, web.SessionCookie, sid, int(cache.SessionTTL/time.Second), true)

	payload := gin.H{
		"id":       user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin == 1,
	}
	if hasEmail(user) {
		payload["email"] = *user.Email
	}
	web.OK(c, gin.H{"user": payload, "needs_email": !hasEmail(user)}, "登录成功")
}

func (h *Handler) logout(c *gin.Context) {
	if sid, err := c.Cookie(web.SessionCookie); err == nil && sid != "" {
		_ = h.deps.Redis.DelSession(c.Request.Context(), sid)
	}
	web.SetCookie(c, web.SessionCookie, "", -1, true)
	web.OK(c, nil, "已退出登录")
}

func (h *Handler) me(c *gin.Context) {
	id := web.MustIdentity(c)
	var user store.User
	err := h.deps.DB.WithContext(c.Request.Context()).First(&user, id.UID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 401, "用户不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, userPayload(user), "ok")
}

type emailReq struct {
	Email string `json:"email"`
}

func (h *Handler) updateEmail(c *gin.Context) {
	id := web.MustIdentity(c)
	var req emailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		web.Fail(c, 400, "请输入有效的邮箱地址")
		return
	}

	db := h.deps.DB.WithContext(c.Request.Context())
	var other store.User
	err := db.Where("email = ?", req.Email).First(&other).Error
	if err == nil && other.ID != id.UID {
		web.Fail(c, 409, "邮箱已被使用")
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 500, "服务异常")
		return
	}

	if err := db.Model(&store.User{}).Where("id = ?", id.UID).Update("email", req.Email).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	// 邮箱参与头像 URL 生成，变化后需失效内容缓存。
	h.deps.Redis.ClearAllContentCaches(c.Request.Context())
	web.OK(c, nil, "邮箱更新成功")
}

type changePasswordReq struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (h *Handler) changePassword(c *gin.Context) {
	id := web.MustIdentity(c)
	var req changePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		web.Fail(c, 400, "请求参数格式错误")
		return
	}
	if runeLen(req.OldPassword) < 6 || runeLen(req.NewPassword) < 6 {
		web.Fail(c, 400, "密码长度不能少于 6 个字符")
		return
	}

	var user store.User
	err := h.deps.DB.WithContext(c.Request.Context()).First(&user, id.UID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		web.Fail(c, 401, "用户不存在")
		return
	}
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)) != nil {
		web.Fail(c, 401, "旧密码错误")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), 10)
	if err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	if err := h.deps.DB.WithContext(c.Request.Context()).Model(&store.User{}).
		Where("id = ?", id.UID).Update("password", string(hash)).Error; err != nil {
		web.Fail(c, 500, "服务异常")
		return
	}
	web.OK(c, nil, "密码修改成功")
}

func userPayload(u store.User) gin.H {
	out := gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"is_admin":   u.IsAdmin == 1,
		"is_banned":  u.IsBanned == 1,
		"created_at": web.TimeOf(u.CreatedAt),
		"updated_at": web.TimeOf(u.UpdatedAt),
	}
	if hasEmail(u) {
		out["email"] = *u.Email
	}
	return out
}

func hasEmail(u store.User) bool { return u.Email != nil && *u.Email != "" }

func newSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b)
}

func runeLen(s string) int { return len([]rune(strings.TrimSpace(s))) }

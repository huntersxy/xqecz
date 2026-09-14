package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope 前后端统一响应包装：{ code, message, data }，code == 200 表示成功。
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// OK 返回成功响应（HTTP 200）。
func OK(c *gin.Context, data any, message string) {
	if message == "" {
		message = "ok"
	}
	c.JSON(200, Envelope{Code: 200, Message: message, Data: data})
}

// SetCookie 写入 HttpOnly + SameSite=Lax 的 Cookie（与旧实现的下发参数一致）。
// maxAge 为秒；传负数表示立即失效。
func SetCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		SameSite: http.SameSiteLaxMode,
	})
}

// SoftFail 返回 HTTP 200 但业务码非 200 的响应。
// 与旧实现一致：参数校验类失败不改变 HTTP 状态，前端按 body.code 判定。
func SoftFail(c *gin.Context, code int, message string) {
	c.JSON(200, Envelope{Code: code, Message: message, Data: nil})
}

// Fail 返回业务错误：HTTP 状态码与 code 一致，message 供前端直接展示。
// 与旧实现一致，前端从 body.message 读取文案。
func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Envelope{Code: status, Message: message, Data: nil})
}

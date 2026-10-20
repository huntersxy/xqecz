package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func OK(c *gin.Context, data any, message string) {
	if message == "" {
		message = "ok"
	}
	c.JSON(200, Envelope{Code: 200, Message: message, Data: data})
}

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

func SoftFail(c *gin.Context, code int, message string) {
	c.JSON(200, Envelope{Code: code, Message: message, Data: nil})
}

func Fail(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, Envelope{Code: status, Message: message, Data: nil})
}

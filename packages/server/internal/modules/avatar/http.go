package avatar

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	core "github.com/huntersxy/xqecz/server/internal/avatar"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func Register(api *gin.RouterGroup, deps app.Deps) {
	if deps.Avatars == nil {
		return
	}
	api.GET(core.Route+":token", web.OptionalAuth(deps), func(c *gin.Context) {
		subject, id, valid := deps.Avatars.Verify(c.Param("token"))
		if !valid {
			c.Status(http.StatusNotFound)
			return
		}
		email := visibleEmail(c, deps, subject, id)
		if email == nil || strings.TrimSpace(*email) == "" {
			c.Status(http.StatusNotFound)
			return
		}
		image, err := deps.Avatars.Image(c.Request.Context(), *email)
		if err != nil {
			c.Status(http.StatusBadGateway)
			return
		}
		c.Header("Cross-Origin-Resource-Policy", "cross-origin")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, "image/png", image)
	})
}

func visibleEmail(c *gin.Context, deps app.Deps, subject core.Subject, id uint64) *string {
	db := deps.DB.WithContext(c.Request.Context())
	if subject == core.Guest {
		var row store.Content
		if err := db.Select("guest_email", "guest_nickname", "audit_status", "user_id").First(&row, id).Error; err != nil ||
			row.GuestNickname == nil || *row.GuestNickname == "" || !web.CanViewContent(c, row) {
			return nil
		}
		return row.GuestEmail
	}
	query := db.Select("email").Where("users.id = ?", id)
	viewer, signedIn := web.IdentityOf(c)
	if !signedIn || (!viewer.IsAdmin && viewer.UID != id) {
		works := db.Model(&store.Content{}).Select("1").
			Where("user_id = ? AND audit_status IN ?", id, store.PublicContentStatuses())
		comments := db.Model(&store.Comment{}).Select("1").
			Joins("JOIN contents ON contents.id = comments.content_id").
			Where("comments.user_id = ? AND comments.is_banned = 0 AND contents.audit_status IN ?", id, store.PublicContentStatuses())
		query = query.Where("EXISTS (?) OR EXISTS (?)", works, comments)
	}
	var user store.User
	if err := query.Take(&user).Error; err != nil {
		return nil
	}
	return user.Email
}

package content

import (
	"context"
	"gorm.io/gorm"
)

// PublishAI reuses normal content creation, review, thumbnail and cache behavior.
// The caller links its immutable asset inside this transaction to prevent reposts.
func (h *Handler) PublishAI(ctx context.Context, uid uint64, title, text, rel, absolute string, size int64, admin bool, link func(*gorm.DB, uint64) error) (Item, error) {
	status := "pending"
	if admin {
		status = "approved"
	}
	if h.mirror != nil {
		h.mirror.PushAsync(rel, absolute)
	}
	return h.createContent(ctx, createInput{Title: title, Content: text, FilePath: rel, FileSize: size, UserID: uid, Tags: []string{"AIGC"}, AuditStatus: status, AbsPath: absolute, Link: link})
}

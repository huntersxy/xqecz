package content

import (
	"context"
	"errors"

	"github.com/huntersxy/xqecz/server/internal/contenttags"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"gorm.io/gorm"
)

// PublishAI reuses normal content creation, review, thumbnail and cache behavior.
// The caller links its immutable asset inside this transaction to prevent reposts.
func (h *Handler) PublishAI(ctx context.Context, uid uint64, title, text string, tags []string, autoMetadata bool, rel, absolute string, size int64, admin bool, link func(*gorm.DB, uint64) error) (Item, error) {
	if autoMetadata {
		if h.deps.Vision == nil {
			return Item{}, errors.New("视觉生成服务暂不可用，请关闭自动标题与标签后手动填写")
		}
		title = vision.PlaceholderTitle
		tags = []string{vision.PlaceholderTag}
	}
	status := "pending"
	if admin {
		status = "approved"
	}
	if h.mirror != nil {
		h.mirror.PushAsync(rel, absolute)
	}
	return h.createContent(ctx, createInput{Title: title, Content: text, FilePath: rel, FileSize: size, UserID: uid, Tags: contenttags.WithAIGC(tags, true), AuditStatus: status, AbsPath: absolute, AutoMetadata: autoMetadata, Link: link})
}

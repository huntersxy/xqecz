package content

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/contenttags"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func (h *Handler) uploadMetadata(f *form, title string) (bool, string, []string, error) {
	ai := f.value("is_ai_generated")
	if ai != "" && ai != "true" && ai != "false" {
		return false, "", nil, errors.New("is_ai_generated 参数需为 true 或 false")
	}
	tags := contenttags.WithAIGC(f.parseTagsField(), ai == "true")
	v := f.value("auto_metadata")
	if v != "" && v != "true" && v != "false" {
		return false, "", nil, errors.New("auto_metadata 参数需为 true 或 false")
	}
	if v != "true" {
		return false, title, tags, nil
	}
	if f.File == nil || !strings.HasPrefix(f.File.Mime, "image/") || IsVideoPath(f.File.RelPath) {
		return false, "", nil, errors.New("自动标题与标签仅支持图片，请先选择图片")
	}
	if h.deps.Vision == nil {
		return false, "", nil, errors.New("视觉生成服务暂不可用，请手动填写标题和标签")
	}
	return true, vision.PlaceholderTitle, contenttags.WithAIGC([]string{vision.PlaceholderTag}, contenttags.HasAIGC(tags)), nil
}

func (h *Handler) metadataOptions(c *gin.Context) {
	ready := h.deps.Vision != nil && h.deps.Vision.Ready(c.Request.Context())
	web.OK(c, gin.H{"available": h.deps.Vision != nil, "ready": ready, "placeholder_title": vision.PlaceholderTitle, "placeholder_tag": vision.PlaceholderTag}, "ok")
}

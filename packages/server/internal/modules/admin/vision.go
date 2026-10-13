package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func (h *Handler) hasVision(c *gin.Context) bool {
	if h.deps.Vision == nil {
		web.SoftFail(c, 503, "视觉生成服务未初始化，请检查服务端日志和数据库权限")
		return false
	}
	return true
}
func (h *Handler) visionSettings(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	res, err := h.deps.Vision.Settings(c.Request.Context())
	if err != nil {
		web.Fail(c, 500, "配置读取失败")
		return
	}
	web.OK(c, res, "ok")
}
func (h *Handler) saveVisionSettings(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	var req vision.SettingsUpdate
	if c.ShouldBindJSON(&req) != nil {
		web.SoftFail(c, 400, "请求参数格式错误")
		return
	}
	res, err := h.deps.Vision.SaveSettings(c.Request.Context(), req)
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "视觉配置已保存")
}
func (h *Handler) testVision(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	res, err := h.deps.Vision.Test(c.Request.Context())
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "视觉请求与标签解析通过")
}
func (h *Handler) visionJobs(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	res, err := h.deps.Vision.Jobs(c.Request.Context())
	if err != nil {
		web.Fail(c, 500, "任务读取失败")
		return
	}
	list := make([]gin.H, 0, len(res))
	for _, job := range res {
		list = append(list, gin.H{"content_id": job.ContentID, "status": job.Status, "attempts": job.Attempts, "error": job.Error, "created_at": web.TimeOf(job.CreatedAt), "updated_at": web.TimeOf(job.UpdatedAt)})
	}
	web.OK(c, list, "ok")
}
func (h *Handler) retryVision(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	id, ok := pathID(c)
	if !ok {
		web.SoftFail(c, 400, "任务 ID 无效")
		return
	}
	if err := h.deps.Vision.Retry(c.Request.Context(), id); err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, nil, "任务已重新排队")
}
func (h *Handler) visionReferences(c *gin.Context) {
	if !h.hasVision(c) {
		return
	}
	res, err := h.deps.Vision.References(c.Request.Context())
	if err != nil {
		web.SoftFail(c, 400, err.Error())
		return
	}
	web.OK(c, res, "ok")
}

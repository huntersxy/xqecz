package aigen

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func (h *Handler) ideas(c *gin.Context)      { h.listIdeas(c, false) }
func (h *Handler) adminIdeas(c *gin.Context) { h.listIdeas(c, true) }
func (h *Handler) listIdeas(c *gin.Context, admin bool) {
	rows := []core.Idea{}
	q := h.db(c)
	if !admin {
		q = q.Where("status NOT IN ?", []string{"hidden", "withdrawn"})
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	if search := c.Query("search"); search != "" {
		q = q.Where("(title LIKE ? OR description LIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	if character := c.Query("character"); character != "" {
		id, err := strconv.ParseUint(character, 10, 64)
		if err != nil || id == 0 {
			fail(c, core.ErrInvalid)
			return
		}
		value := strconv.FormatUint(id, 10)
		q = q.Where("(character_ids = ? OR character_ids LIKE ? OR character_ids LIKE ? OR character_ids LIKE ?)", "["+value+"]", "["+value+",%", "%,"+value+",%", "%,"+value+"]")
	}
	if before, _ := strconv.ParseUint(c.Query("before_id"), 10, 64); before != 0 {
		q = q.Where("id < ?", before)
	}
	if err := q.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	out, err := h.ideaViews(c, rows)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, out)
}
func (h *Handler) author(c *gin.Context, id uint64) any {
	var user store.User
	if h.db(c).First(&user, id).Error != nil {
		return gin.H{"id": id, "username": "已注销用户"}
	}
	return gin.H{"id": id, "username": user.Username}
}
func (h *Handler) ideaView(c *gin.Context, row core.Idea) any {
	assets := []core.Asset{}
	var input core.Input
	_ = json.Unmarshal([]byte(row.Snapshot), &input)
	ids := []uint64{}
	for _, ref := range input.References {
		ids = append(ids, ref.AssetID)
	}
	if len(ids) > 0 {
		_ = h.db(c).Where("id IN ?", ids).Find(&assets).Error
	}
	if row.ReferenceSnapshot != "" {
		var snapshot []core.Asset
		if json.Unmarshal([]byte(row.ReferenceSnapshot), &snapshot) == nil {
			assets = snapshot
		}
	}
	results := []core.Asset{}
	if row.Status == "realized" {
		_ = h.db(c).Where("task_id = ?", row.TaskID).Find(&results).Error
	}
	currentCost, quoteErr := h.s.IdeaQuote(c.Request.Context(), row)
	if quoteErr != nil {
		currentCost = row.EstimatedCost
	}
	view := gin.H{"idea": row, "author": h.author(c, row.UserID), "sponsor": nil, "references": assets, "results": results, "current_cost": currentCost, "sponsorable": row.Status == "waiting" && quoteErr == nil}
	if row.SponsorID != 0 {
		view["sponsor"] = h.author(c, row.SponsorID)
	}
	// Moderation and billing notes stay in the administrator-only events route.
	return view
}
func (h *Handler) idea(c *gin.Context) {
	var row core.Idea
	if err := h.db(c).Where("id = ? AND status NOT IN ?", pathID(c), []string{"hidden", "withdrawn"}).First(&row).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, h.ideaView(c, row))
}
func (h *Handler) ideaAsset(c *gin.Context) {
	assetID, _ := strconv.ParseUint(c.Param("asset"), 10, 64)
	if !h.s.IdeaHasAsset(c.Request.Context(), pathID(c), assetID) {
		web.Fail(c, 404, "图片不存在")
		return
	}
	var row core.Asset
	if err := h.db(c).First(&row, assetID).Error; err != nil {
		fail(c, err)
		return
	}
	h.sendImage(c, row, false)
}
func (h *Handler) createIdea(c *gin.Context) {
	var req core.IdeaInput
	if bind(c, &req) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	row, err := h.s.CreateIdea(c.Request.Context(), uid(c), req)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, row)
}
func (h *Handler) sponsor(c *gin.Context) {
	var req struct {
		Key          string `json:"idempotency_key"`
		ExpectedCost int64  `json:"expected_cost"`
	}
	if bind(c, &req) != nil {
		fail(c, core.ErrInvalid)
		return
	}
	task, err := h.s.Sponsor(c.Request.Context(), uid(c), pathID(c), req.Key, req.ExpectedCost)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, task)
}
func (h *Handler) withdraw(c *gin.Context) {
	if err := h.s.HideIdea(c.Request.Context(), uid(c), pathID(c), false, ""); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) hide(c *gin.Context) {
	var req struct {
		Note string `json:"note"`
	}
	if bind(c, &req) != nil || len([]rune(req.Note)) > 300 {
		fail(c, core.ErrInvalid)
		return
	}
	if err := h.s.HideIdea(c.Request.Context(), uid(c), pathID(c), true, req.Note); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) deleteAdminIdea(c *gin.Context) {
	if err := h.s.DeleteIdea(c.Request.Context(), pathID(c)); err != nil {
		fail(c, err)
		return
	}
	ok(c, nil)
}
func (h *Handler) events(c *gin.Context) {
	rows := []core.IdeaEvent{}
	if err := h.db(c).Where("idea_id = ?", pathID(c)).Order("id asc").Find(&rows).Error; err != nil {
		fail(c, err)
		return
	}
	ok(c, rows)
}

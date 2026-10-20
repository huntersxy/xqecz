package aigen

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *Handler) publish(c *gin.Context) {
	var req struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Tags         []string `json:"tags"`
		AutoMetadata bool     `json:"auto_metadata"`
		Consent      bool     `json:"public_consent"`
	}
	if bind(c, &req) != nil || !req.Consent || (!req.AutoMetadata && len([]rune(strings.TrimSpace(req.Title))) < 1) || len([]rune(req.Title)) > 200 || len([]rune(req.Description)) > 10000 || len(req.Tags) > 16 {
		fail(c, core.ErrInvalid)
		return
	}
	if req.AutoMetadata && h.deps.Vision == nil {
		web.SoftFail(c, 400, "视觉生成服务暂不可用，请关闭自动标题与标签后手动填写")
		return
	}
	for _, tag := range req.Tags {
		if len([]rune(strings.TrimSpace(tag))) > 40 {
			fail(c, core.ErrInvalid)
			return
		}
	}
	var asset core.Asset
	if err := h.db(c).Where("id = ? AND user_id = ? AND archived = ? AND task_id <> 0", pathID(c), uid(c), false).First(&asset).Error; err != nil {
		fail(c, err)
		return
	}
	if asset.ContentID != nil {
		ok(c, gin.H{"content_id": *asset.ContentID})
		return
	}
	basePrompt, enhanced := h.taskPrompt(c, asset.TaskID)
	if strings.TrimSpace(req.Description) == "" {
		req.Description = basePrompt
	}
	source, err := h.s.AssetPath(asset)
	if err != nil {
		fail(c, err)
		return
	}
	if err = os.MkdirAll(h.deps.Cfg.UploadDir, 0o755); err != nil {
		fail(c, err)
		return
	}
	rel := fmt.Sprintf("ai-%d%s", asset.ID, filepath.Ext(source))
	target := filepath.Join(h.deps.Cfg.UploadDir, rel)
	if err = copyImage(source, target); err != nil {
		fail(c, err)
		return
	}
	var existing uint64
	already := errors.New("already linked")
	item, err := h.content.PublishAI(c.Request.Context(), uid(c), req.Title, req.Description, req.Tags, req.AutoMetadata, rel, target, asset.Bytes, web.MustIdentity(c).IsAdmin, enhanced, func(tx *gorm.DB, id uint64) error {
		var locked core.Asset
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, asset.ID).Error; err != nil {
			return err
		}
		if locked.ContentID != nil {
			existing = *locked.ContentID
			return already
		}
		return tx.Model(&locked).Update("content_id", id).Error
	})
	if errors.Is(err, already) {
		ok(c, gin.H{"content_id": existing})
		return
	}
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"content_id": item.ID})
}
func (h *Handler) taskPrompt(c *gin.Context, taskID uint64) (string, bool) {
	if taskID == 0 {
		return "", false
	}
	var task core.Task
	if err := h.db(c).Select("id", "prompt", "snapshot").Where("id = ?", taskID).First(&task).Error; err != nil {
		return "", false
	}
	var snapshot struct {
		Input core.Input `json:"input"`
	}
	if err := json.Unmarshal([]byte(task.Snapshot), &snapshot); err != nil {
		return task.Prompt, false
	}
	base := strings.TrimSpace(snapshot.Input.BasePrompt)
	if base == "" {
		base = task.Prompt
	}
	return base, snapshot.Input.EnhancePrompt
}

func copyImage(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.CreateTemp(filepath.Dir(target), ".ai-publish-*")
	if err != nil {
		return err
	}
	temp := out.Name()
	defer os.Remove(temp)
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if _, err = os.Stat(target); err == nil {
		return nil
	}
	return os.Rename(temp, target)
}

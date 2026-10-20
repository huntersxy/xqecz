package aigen

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/huntersxy/xqecz/server/internal/creative"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IdeaInput struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Input           Input    `json:"input"`
	CharacterIDs    []uint64 `json:"character_ids"`
	PublicConsent   bool     `json:"public_consent"`
	ReferenceRights bool     `json:"reference_rights"`
}

func (s *Service) CreateIdea(ctx context.Context, uid uint64, request IdeaInput) (Idea, error) {
	request.Title = strings.TrimSpace(request.Title)
	if !request.PublicConsent || !request.ReferenceRights || len([]rune(request.Title)) < 1 || len([]rune(request.Title)) > 150 || len([]rune(request.Description)) > 3000 || len(request.CharacterIDs) > 20 {
		return Idea{}, ErrInvalid
	}
	input := request.Input
	if input.Mode == "byok" {
		own, err := s.providerFor(ctx, uid, input)
		if err != nil {
			return Idea{}, err
		}
		input.ProviderID = own.AllowedProviderID
	}
	if len(request.CharacterIDs) == 0 {
		request.CharacterIDs = input.CharacterIDs
	}
	if len(request.CharacterIDs) > 20 {
		return Idea{}, ErrInvalid
	}
	for _, id := range request.CharacterIDs {
		var count int64
		if s.db.WithContext(ctx).Model(&creative.Resource{}).Where("id = ? AND kind = ? AND status = ?", id, "character", "published").Count(&count).Error != nil || count != 1 {
			return Idea{}, ErrInvalid
		}
	}
	input.Mode = "site"
	input.SessionID = 0
	input.Key = ""
	input.ParentAssetID = 0
	snap, cost, err := s.prepare(ctx, uid, input, prepareOptions{allowNoSiteKey: true})
	if err != nil {
		return Idea{}, err
	}
	for _, ref := range snap.References {
		if ref.Public && !ref.DownloadAllowed {
			return Idea{}, invalidInput("此素材未授权公开投稿参考图")
		}
	}
	snap.Input.ExpectedCost = cost
	snap.Input.ProviderRevision = snap.Provider.Revision
	snap.Input.CharacterIDs = request.CharacterIDs
	row := Idea{UserID: uid, Title: request.Title, Description: request.Description, Snapshot: encode(snap.Input), ReferenceSnapshot: encode(snap.References), Model: snap.Provider.Model, CharacterIDs: encode(request.CharacterIDs), Status: "waiting", EstimatedCost: cost}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&IdeaEvent{IdeaID: row.ID, ActorID: uid, Action: "submitted"}).Error
	})
	return row, err
}

func (s *Service) IdeaQuote(ctx context.Context, idea Idea) (int64, error) {
	var input Input
	if json.Unmarshal([]byte(idea.Snapshot), &input) != nil {
		return 0, ErrInvalid
	}
	snap, cost, err := s.prepare(ctx, idea.UserID, input, prepareOptions{historical: true})
	if err == nil && snap.Provider.Model != idea.Model {
		return 0, invalidInput("供应商模型已变更，请重新投稿")
	}
	return cost, err
}

func (s *Service) Sponsor(ctx context.Context, uid, id uint64, key string, expectedCost int64) (Task, error) {
	if len(key) < 16 || len(key) > 80 {
		return Task{}, ErrInvalid
	}
	var idea Idea
	if err := s.db.WithContext(ctx).First(&idea, id).Error; err != nil {
		return Task{}, err
	}
	var input Input
	if json.Unmarshal([]byte(idea.Snapshot), &input) != nil {
		return Task{}, ErrInvalid
	}
	input.Key = key
	input.Mode = "site"
	snap, cost, err := s.prepare(ctx, idea.UserID, input, prepareOptions{historical: true})
	if err != nil {
		return Task{}, err
	}
	if snap.Provider.Model != idea.Model || cost != expectedCost {
		return Task{}, ErrConflict
	}
	snap.Input.ExpectedCost = cost
	var task Task
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&idea, id).Error; err != nil {
			return err
		}
		account, err := lockAccount(tx, uid)
		if err != nil {
			return err
		}
		var prior Task
		if tx.Where("user_id = ? AND idempotency_key = ?", uid, key).First(&prior).Error == nil {
			if prior.IdeaID != id {
				return ErrConflict
			}
			task = prior
			return nil
		}
		if idea.Status != "waiting" {
			return ErrBusy
		}
		task, err = s.insertTask(tx, uid, account, snap, cost, id)
		if err != nil {
			return err
		}
		if err = tx.Model(&idea).Updates(map[string]any{"status": "generating", "sponsor_id": uid, "task_id": task.ID, "estimated_cost": cost}).Error; err != nil {
			return err
		}
		return tx.Create(&IdeaEvent{IdeaID: id, ActorID: uid, TaskID: task.ID, Action: "sponsored"}).Error
	})
	return task, err
}

func (s *Service) HideIdea(ctx context.Context, uid, id uint64, admin bool, note string) error {
	var hidden Idea
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var idea Idea
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&idea, id).Error; err != nil {
			return err
		}
		if !admin && idea.UserID != uid {
			return ErrInvalid
		}
		status := "withdrawn"
		if admin {
			status = "hidden"
		}
		if err := tx.Model(&idea).Update("status", status).Error; err != nil {
			return err
		}
		hidden = idea
		return tx.Create(&IdeaEvent{IdeaID: id, ActorID: uid, TaskID: idea.TaskID, Action: status, Note: note}).Error
	})
	if err != nil {
		return err
	}
	if hidden.TaskID != 0 {
		var task Task
		if s.db.WithContext(ctx).First(&task, hidden.TaskID).Error == nil && task.Status == "queued" {
			err = s.complete(task, "queued", "cancelled", "创意已撤回或下架，生成未执行", nil, false, uid)
			if errors.Is(err, ErrConflict) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (s *Service) DeleteIdea(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var idea Idea
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&idea, id).Error; err != nil {
			return err
		}
		if idea.TaskID != 0 {
			var task Task
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, idea.TaskID).Error; err != nil {
				return err
			}
			switch task.Status {
			case "queued", "running":
				return invalidInput("创意仍有执行中的生成任务，不能删除")
			case "unknown":
				return invalidInput("创意仍有待核查的生成任务，请先完成额度核查")
			}
			if err := tx.Model(&Task{}).Where("id = ? AND idea_id = ?", task.ID, idea.ID).Update("idea_id", 0).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("idea_id = ?", idea.ID).Delete(&IdeaEvent{}).Error; err != nil {
			return err
		}
		return tx.Delete(&idea).Error
	})
}

func (s *Service) IdeaHasAsset(ctx context.Context, ideaID, assetID uint64) bool {
	var idea Idea
	if s.db.WithContext(ctx).Where("id = ? AND status NOT IN ?", ideaID, []string{"hidden", "withdrawn"}).First(&idea).Error != nil {
		return false
	}
	var input Input
	_ = json.Unmarshal([]byte(idea.Snapshot), &input)
	for _, ref := range input.References {
		if ref.AssetID == assetID {
			return true
		}
	}
	if idea.Status == "realized" {
		var count int64
		if s.db.WithContext(ctx).Model(&Asset{}).Where("id = ? AND task_id = ?", assetID, idea.TaskID).Count(&count).Error == nil && count > 0 {
			return true
		}
	}
	return false
}

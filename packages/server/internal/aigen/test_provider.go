package aigen

import (
	"context"

	"gorm.io/gorm"
)

func (s *Service) QueueTest(ctx context.Context, uid, id uint64, site bool, key string) (Task, error) {
	if len(key) < 16 || len(key) > 80 {
		return Task{}, ErrInvalid
	}
	input := Input{ProviderID: id, Mode: "byok", Prompt: "A simple pink circle on a white background.", Count: 1, Key: key}
	if site {
		input.Mode = "site"
	}
	provider, err := s.providerFor(ctx, uid, input)
	if err != nil {
		return Task{}, err
	}
	input.Size = "auto"
	input.ProviderRevision = provider.Revision
	snap, _, err := s.prepare(ctx, uid, input, prepareOptions{})
	if err != nil {
		return Task{}, err
	}
	snap.Input.Mode = "test"
	var task Task
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := lockAccount(tx, uid)
		if err != nil {
			return err
		}
		if tx.Where("user_id = ? AND idempotency_key = ?", uid, key).First(&task).Error == nil {
			return nil
		}
		task, err = s.insertTask(tx, uid, account, snap, 0, 0)
		return err
	})
	return task, err
}

package aigen

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var grantBatchKey = regexp.MustCompile(`^[a-zA-Z0-9-]{16,80}$`)

func (s *Service) GrantBatch(ctx context.Context, actor uint64, ids []uint64, all bool, amount int64, key, note string) (int, error) {
	if amount < 1 || amount > 10000 || !grantBatchKey.MatchString(key) || len([]rune(note)) > 300 || (all && len(ids) > 0) || (!all && len(ids) == 0) {
		return 0, ErrInvalid
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) > 10000 || slices.Contains(ids, uint64(0)) {
		return 0, ErrInvalid
	}
	prefix := fmt.Sprintf("grant-batch:%d:%s:", actor, key)
	mode := "selected:"
	if all {
		mode = "all:"
	}
	count := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []store.User
		q := tx.Select("id").Order("id asc")
		if !all {
			q = q.Where("id IN ?", ids)
		}
		if err := q.Find(&users).Error; err != nil {
			return err
		}
		lockIDs := []uint64{actor}
		for _, user := range users {
			lockIDs = append(lockIDs, user.ID)
		}
		slices.Sort(lockIDs)
		lockIDs = slices.Compact(lockIDs)
		accounts := map[uint64]Account{}
		for _, id := range lockIDs {
			account, err := lockAccount(tx, id)
			if err != nil {
				return err
			}
			accounts[id] = account
		}
		var previous []Ledger
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("`key` LIKE ?", prefix+"%").Order("user_id asc").Find(&previous).Error; err != nil {
			return err
		}
		if len(previous) > 0 {
			priorIDs := []uint64{}
			for _, row := range previous {
				if !strings.HasPrefix(row.Key, prefix+mode) || row.Amount != amount || row.Note != note {
					return ErrConflict
				}
				priorIDs = append(priorIDs, row.UserID)
			}
			if !all && !slices.Equal(ids, priorIDs) {
				return ErrConflict
			}
			count = len(previous)
			return nil
		}
		if len(users) == 0 || (!all && len(users) != len(ids)) {
			return ErrInvalid
		}
		for _, user := range users {
			account := accounts[user.ID]
			if err := tx.Model(&Account{}).Where("user_id = ?", user.ID).Update("available", account.Available+amount).Error; err != nil {
				return err
			}
			if err := tx.Create(&Ledger{UserID: user.ID, ActorID: actor, Key: fmt.Sprintf("%s%s%d", prefix, mode, user.ID), Kind: "grant", Amount: amount, Note: note}).Error; err != nil {
				return err
			}
		}
		count = len(users)
		return nil
	})
	return count, err
}

package aigen

import (
	"bytes"
	"context"
	"errors"
	"github.com/huntersxy/xqecz/server/internal/store"
	"testing"
)

func TestGrantBatchAtomicityAndSnapshot(t *testing.T) {
	s, _, _, _ := testService(t)
	ctx := context.Background()
	if _, err := s.GrantBatch(ctx, 20, []uint64{1, 99999}, false, 5, "invalid-batch-key-01", ""); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if a, _ := lockAccount(s.db, 1); a.Available != 0 {
		t.Fatal("partial invalid grant", a)
	}
	count, err := s.GrantBatch(ctx, 20, []uint64{2, 1, 2}, false, 5, "selected-batch-key-01", "奖励")
	if err != nil || count != 2 {
		t.Fatal(count, err)
	}
	count, err = s.GrantBatch(ctx, 20, []uint64{1, 2}, false, 5, "selected-batch-key-01", "奖励")
	if err != nil || count != 2 {
		t.Fatal("retry", count, err)
	}
	if a, _ := lockAccount(s.db, 1); a.Available != 5 {
		t.Fatal("duplicate grant", a)
	}
	if _, err = s.GrantBatch(ctx, 20, []uint64{1}, false, 5, "selected-batch-key-01", "奖励"); !errors.Is(err, ErrConflict) {
		t.Fatal("changed recipients accepted", err)
	}
	count, err = s.GrantBatch(ctx, 20, nil, true, 3, "everyone-batch-key-01", "")
	if err != nil || count != 20 {
		t.Fatal(count, err)
	}
	if err = s.db.Create(&store.User{ID: 21, Username: "new-user"}).Error; err != nil {
		t.Fatal(err)
	}
	count, err = s.GrantBatch(ctx, 20, nil, true, 3, "everyone-batch-key-01", "")
	if err != nil || count != 20 {
		t.Fatal("all users retry changed snapshot", count, err)
	}
	if a, _ := lockAccount(s.db, 21); a.Available != 0 {
		t.Fatal("retry granted newly registered user", a)
	}
}

func TestNumberedReferencesPreserveAttachmentPositions(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	first, err := s.Upload(ctx, 1, bytes.NewReader(testPNG()))
	if err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID = 0
	if err = s.db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	input := inputFor(p, session, "numbered-reference-01")
	input.Prompt = "使用 @图片1 的人物和 @图片2 的服装"
	input.References = []Reference{{AssetID: first.ID}, {AssetID: second.ID}}
	snap, _, err := s.prepare(ctx, 1, input, prepareOptions{})
	if err != nil || len(snap.Files) != 2 || len(snap.References) != 2 {
		t.Fatal("numbered attachments were collapsed", len(snap.Files), err)
	}
}

func TestAdministratorQuotaExemption(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	if err := s.db.Model(&store.User{}).Where("id = ?", 1).Update("is_admin", 1).Error; err != nil {
		t.Fatal(err)
	}
	input := inputFor(p, session, "admin-zero-balance-01")
	cost, err := s.Quote(ctx, 1, input)
	if err != nil || cost != 0 {
		t.Fatal("administrator quote", cost, err)
	}
	input.ExpectedCost = 0
	task, err := s.Submit(ctx, 1, input)
	if err != nil || task.Cost != 0 {
		t.Fatal("administrator blocked by quota", task, err)
	}
	if a, _ := lockAccount(s.db, 1); a.Available != 0 || a.Reserved != 0 {
		t.Fatal("administrator quota changed", a)
	}
	ordinary := Session{UserID: 2, Name: "ordinary", Draft: "{}", Version: 1}
	s.db.Create(&ordinary)
	if _, err = s.Submit(ctx, 2, inputFor(p, ordinary, "ordinary-zero-balance")); !errors.Is(err, ErrQuota) {
		t.Fatal("ordinary quota bypassed", err)
	}
}

package aigen

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestDailyLoginCapAndRewardSeparation(t *testing.T) {
	s, _, _, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 17, 15, 59, 0, 0, time.UTC)
	login := func(at time.Time) {
		t.Helper()
		if err := GrantDailyLogin(ctx, s.db, 1, at, DailyCreditCap); err != nil {
			t.Fatal(err)
		}
	}
	login(now)
	login(now)
	if a := balance(t, s, 1); a.Available != 5 || a.DailyAvailable != 5 {
		t.Fatal("daily login must fill the daily quota once", a)
	}
	login(now.Add(time.Minute))
	if a := balance(t, s, 1); a.Available != 5 || a.DailyAvailable != 5 {
		t.Fatal("China midnight boundary", a)
	}
	for i := 1; i <= 7; i++ {
		login(now.AddDate(0, 0, i))
	}
	if a := balance(t, s, 1); a.Available != 5 || a.DailyAvailable != 5 {
		t.Fatal("daily cap", a)
	}
	grant(t, s, 1, 10)
	if err := s.db.Transaction(func(tx *gorm.DB) error { return RewardApproval(tx, 1, 42) }); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error { return RewardApproval(tx, 1, 42) }); err != nil {
		t.Fatal(err)
	}
	login(now.AddDate(0, 0, 9))
	if a := balance(t, s, 1); a.Available != 16 || a.DailyAvailable != 5 {
		t.Fatal("uncapped bonus or duplicate approval", a)
	}
}

func TestDailyCreditsReservationRefundAndRefill(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		if err := GrantDailyLogin(ctx, s.db, 1, now.AddDate(0, 0, i), DailyCreditCap); err != nil {
			t.Fatal(err)
		}
	}
	grant(t, s, 1, 10)
	task, err := s.Submit(ctx, 1, inputFor(p, session, "daily-credit-submit-001"))
	if err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.DailyAvailable != 3 || a.DailyReserved != 2 || a.Available != 13 {
		t.Fatal("daily first", a)
	}
	if err := GrantDailyLogin(ctx, s.db, 1, now.AddDate(0, 0, 5), DailyCreditCap); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.Available != 13 {
		t.Fatal("reserved credits bypassed cap", a)
	}
	if err := s.complete(task, "queued", "failed", "test", nil, 0); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.DailyAvailable != 5 || a.DailyReserved != 0 || a.Available != 15 {
		t.Fatal("refund source", a)
	}
	task, err = s.Submit(ctx, 1, inputFor(p, session, "daily-credit-submit-002"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.complete(task, "queued", "succeeded", "", nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := GrantDailyLogin(ctx, s.db, 1, now.AddDate(0, 0, 6), DailyCreditCap); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.DailyAvailable != 5 || a.DailyReserved != 0 || a.Available != 15 {
		t.Fatal("daily refill tops up to the cap", a)
	}
}

func TestDailyLoginUsesConfiguredCap(t *testing.T) {
	s, _, _, _ := testService(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if _, err := s.SaveDailyQuota(ctx, DailyQuota{Cap: 0}); err == nil {
		t.Fatal("zero cap must be rejected")
	}
	if _, err := s.SaveDailyQuota(ctx, DailyQuota{Cap: 8}); err != nil {
		t.Fatal(err)
	}
	if quota, err := s.DailyQuota(ctx); err != nil || quota.Cap != 8 {
		t.Fatal("saved cap lost", quota, err)
	}
	if err := s.GrantDailyLogin(ctx, 1, now); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.Available != 8 || a.DailyAvailable != 8 {
		t.Fatal("login must fill to the configured cap", a)
	}
	if err := s.db.Model(&Account{}).Where("user_id = ?", 1).Update("daily_available", 3).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.GrantDailyLogin(ctx, 1, now.AddDate(0, 0, 1)); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.Available != 13 || a.DailyAvailable != 8 {
		t.Fatal("login must only top up the missing part", a)
	}
}

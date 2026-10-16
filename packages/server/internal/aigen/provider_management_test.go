package aigen

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestSiteProviderEnableDisableAndDelete(t *testing.T) {
	s, _, provider, _ := testService(t)
	ctx := context.Background()
	if err := s.SetSiteProviderEnabled(ctx, provider.ID, false); err != nil {
		t.Fatal(err)
	}
	var saved ProviderConfig
	if err := s.Database().First(&saved, provider.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Enabled || saved.CipherKey == "" {
		t.Fatalf("disable should retain encrypted key: %#v", saved)
	}
	if err := s.SetSiteProviderEnabled(ctx, provider.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSiteProvider(ctx, provider.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Database().First(&ProviderConfig{}, provider.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("provider should be deleted, err=%v", err)
	}
}

func TestSiteProviderDeleteRejectsActiveTask(t *testing.T) {
	s, _, provider, session := testService(t)
	task := Task{UserID: session.UserID, SessionID: session.ID, ProviderID: provider.ID, DispatchProviderID: provider.ID, Status: "queued", Snapshot: `{}`}
	if err := s.Database().Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSiteProvider(context.Background(), provider.ID); err == nil {
		t.Fatal("active provider task should block deletion")
	}
}

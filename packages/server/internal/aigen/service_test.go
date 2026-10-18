package aigen

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fakeProvider struct {
	calls atomic.Int32
	err   error
	check func(Input, []string)
}

func (p *fakeProvider) Generate(_ context.Context, _ ProviderConfig, _ string, input Input, files []string) ([]Image, error) {
	p.calls.Add(1)
	if p.check != nil {
		p.check(input, files)
	}
	if p.err != nil {
		return nil, p.err
	}
	return []Image{{Bytes: testPNG()}}, nil
}
func testPNG() []byte {
	var out bytes.Buffer
	_ = png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	return out.Bytes()
}
func testService(t *testing.T) (*Service, *fakeProvider, ProviderConfig, Session) {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { sql.Close() })
	if err = db.AutoMigrate(&store.User{}); err != nil {
		t.Fatal(err)
	}
	for i := uint64(1); i <= 20; i++ {
		if err = db.Create(&store.User{ID: i, Username: fmtName(i)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	fake := &fakeProvider{}
	s := NewWithProvider(db, config.Config{AIImageDir: filepath.Join(dir, "private")}, fake)
	if err = s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	p, err := s.SaveProvider(context.Background(), 0, ProviderConfig{Name: "test", Endpoint: "https://api.slomerex.xyz/v1/images/generations", Model: "documented-model", Enabled: true, Default: true, Sizes: `["1024x1024"]`, Qualities: `[]`, MaxCount: 1, MaxReferences: 2, EditTransport: "multipart", TimeoutSeconds: 10, Concurrency: 1, Cost: 2, QualityCosts: `{}`}, "fake-secret-key")
	if err != nil {
		t.Fatal(err)
	}
	session := Session{UserID: 1, Name: "test", Draft: `{}`, Version: 1}
	if err = db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	return s, fake, p, session
}
func fmtName(id uint64) string { return "user-" + string(rune('A'+id)) }
func inputFor(p ProviderConfig, session Session, key string) Input {
	return Input{SessionID: session.ID, Key: key, Prompt: "A pink circle", ProviderID: p.ID, Mode: "site", Size: "1024x1024", Count: 1, References: []Reference{}, ExpectedCost: 2}
}
func grant(t *testing.T, s *Service, uid uint64, amount int64) {
	t.Helper()
	if err := s.Grant(context.Background(), 20, uid, amount, "grant-key-000000-"+fmtName(uid), "test credit"); err != nil {
		t.Fatal(err)
	}
}
func balance(t *testing.T, s *Service, uid uint64) Account {
	t.Helper()
	var row Account
	if err := s.db.First(&row, "user_id = ?", uid).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

func TestReservationIdempotencyAndImmutableSuccess(t *testing.T) {
	s, fake, p, session := testService(t)
	grant(t, s, 1, 10)
	ctx := context.Background()
	input := inputFor(p, session, "submit-key-00000001")
	task, err := s.Submit(ctx, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	input.Prompt = "changed draft"
	again, err := s.Submit(ctx, 1, input)
	if err != nil || again.ID != task.ID || again.Prompt != task.Prompt {
		t.Fatal(again, err)
	}
	if a := balance(t, s, 1); a.Available != 8 || a.Reserved != 2 {
		t.Fatal(a)
	}
	// Editing provider settings after submission cannot change this request.
	p.Model = "changed-model"
	if _, err = s.SaveProvider(ctx, 0, p, ""); err != nil {
		t.Fatal(err)
	}
	fake.check = func(input Input, _ []string) {
		if input.Prompt != "A pink circle" {
			t.Fatal(input)
		}
	}
	s.runOne(ctx)
	var done Task
	s.db.First(&done, task.ID)
	if done.Status != "succeeded" || done.Model != "documented-model" || fake.calls.Load() != 1 {
		t.Fatal(done)
	}
	if a := balance(t, s, 1); a.Available != 8 || a.Reserved != 0 {
		t.Fatal(a)
	}
	if err = s.complete(done, "running", "succeeded", "", nil, true, 0); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	var ledger int64
	s.db.Model(&Ledger{}).Where("task_id = ?", task.ID).Count(&ledger)
	if ledger != 2 {
		t.Fatal(ledger)
	}
	var assets []Asset
	s.db.Where("task_id = ?", task.ID).Find(&assets)
	if len(assets) != 1 {
		t.Fatal(assets)
	}
	path, _ := s.AssetPath(assets[0])
	if _, err = os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestUserImageSizeIsNotBoundByProviderList(t *testing.T) {
	s, _, p, session := testService(t)
	grant(t, s, 1, 10)
	ctx := context.Background()

	custom := inputFor(p, session, "custom-size-key-0001")
	custom.Size = "3840x2160"
	customTask, err := s.Submit(ctx, 1, custom)
	if err != nil {
		t.Fatal(err)
	}
	if customTask.Size != "3840x2160" {
		t.Fatalf("custom size was changed: %q", customTask.Size)
	}

	automatic := inputFor(p, session, "auto-size-key-00001")
	automatic.Size = "AUTO"
	autoTask, err := s.Submit(ctx, 1, automatic)
	if err != nil {
		t.Fatal(err)
	}
	if autoTask.Size != "auto" {
		t.Fatalf("automatic size was not canonicalized: %q", autoTask.Size)
	}
}

func TestFailureUnknownRecoveryAndManualSettlement(t *testing.T) {
	for _, tc := range []struct {
		name                string
		err                 error
		status              string
		available, reserved int64
	}{{"rejected", KnownFailure{"rejected"}, "failed", 10, 0}, {"timeout", context.DeadlineExceeded, "unknown", 8, 2}} {
		t.Run(tc.name, func(t *testing.T) {
			s, fake, p, session := testService(t)
			grant(t, s, 1, 10)
			task, err := s.Submit(context.Background(), 1, inputFor(p, session, "failure-key-000001"))
			if err != nil {
				t.Fatal(err)
			}
			fake.err = tc.err
			s.runOne(context.Background())
			s.db.First(&task, task.ID)
			if task.Status != tc.status {
				t.Fatal(task)
			}
			if a := balance(t, s, 1); a.Available != tc.available || a.Reserved != tc.reserved {
				t.Fatal(a)
			}
			if task.Status == "unknown" {
				if err = s.Resolve(context.Background(), 20, task.ID, false, "supplier confirmed no execution"); err != nil {
					t.Fatal(err)
				}
				if a := balance(t, s, 1); a.Available != 10 || a.Reserved != 0 {
					t.Fatal(a)
				}
				if err = s.Resolve(context.Background(), 20, task.ID, false, "duplicate"); !errors.Is(err, ErrConflict) {
					t.Fatal(err)
				}
			}
		})
	}
	s, _, p, session := testService(t)
	grant(t, s, 1, 10)
	task, _ := s.Submit(context.Background(), 1, inputFor(p, session, "crashed-key-000001"))
	s.db.Model(&task).Updates(map[string]any{"status": "running", "updated_at": time.Now().Add(-11 * time.Minute)})
	s.recoverStale(context.Background())
	s.db.First(&task, task.ID)
	if task.Status != "unknown" {
		t.Fatal(task)
	}
	if a := balance(t, s, 1); a.Reserved != 2 {
		t.Fatal(a)
	}
	if err := s.Resolve(context.Background(), 20, task.ID, true, "upstream billing confirmed"); err != nil {
		t.Fatal(err)
	}
	if a := balance(t, s, 1); a.Available != 8 || a.Reserved != 0 {
		t.Fatal(a)
	}
}

func TestArchiveTaskOnlyTerminalFailures(t *testing.T) {
	s, fake, p, session := testService(t)
	grant(t, s, 1, 10)
	ctx := context.Background()
	fake.err = KnownFailure{"provider rejected the request"}
	task, err := s.Submit(ctx, 1, inputFor(p, session, "archive-failure-0001"))
	if err != nil {
		t.Fatal(err)
	}
	s.runOne(ctx)
	if err = s.ArchiveTask(ctx, 1, task.ID); err != nil {
		t.Fatal(err)
	}
	var archived Task
	if err = s.db.First(&archived, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !archived.Archived || archived.Status != "failed" {
		t.Fatal(archived)
	}

	fake.err = nil
	succeeded, err := s.Submit(ctx, 1, inputFor(p, session, "archive-success-0001"))
	if err != nil {
		t.Fatal(err)
	}
	s.runOne(ctx)
	if err = s.ArchiveTask(ctx, 1, succeeded.ID); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
}

func TestAdminArchiveTaskHidesTerminalResultsAndProtectsReservedWork(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	grant(t, s, 1, 10)
	task, err := s.Submit(ctx, 1, inputFor(p, session, "admin-delete-success-0001"))
	if err != nil {
		t.Fatal(err)
	}
	s.runOne(ctx)
	var asset Asset
	if err = s.db.Where("task_id = ?", task.ID).First(&asset).Error; err != nil {
		t.Fatal(err)
	}
	if err = s.ArchiveTaskAsAdmin(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	var archived Task
	if err = s.db.First(&archived, task.ID).Error; err != nil || !archived.Archived {
		t.Fatal(archived, err)
	}
	if err = s.db.First(&asset, asset.ID).Error; err != nil || !asset.Archived {
		t.Fatal(asset, err)
	}

	pending, err := s.Submit(ctx, 1, inputFor(p, session, "admin-delete-pending-0001"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ArchiveTaskAsAdmin(ctx, pending.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("queued task deletion error = %v", err)
	}
	s.db.Model(&pending).Update("status", "unknown")
	if err = s.ArchiveTaskAsAdmin(ctx, pending.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown task deletion error = %v", err)
	}
}

func TestDeleteIdeaRemovesProposalButRetainsCompletedTask(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	idea, err := s.CreateIdea(ctx, 1, IdeaInput{
		Title: "待删除投稿", Input: inputFor(p, session, ""), PublicConsent: true, ReferenceRights: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteIdea(ctx, idea.ID); err != nil {
		t.Fatal(err)
	}
	var deleted Idea
	if err = s.db.First(&deleted, idea.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("idea still exists: %v", err)
	}
	var events int64
	if err = s.db.Model(&IdeaEvent{}).Where("idea_id = ?", idea.ID).Count(&events).Error; err != nil || events != 0 {
		t.Fatalf("idea events remain: %d, %v", events, err)
	}
}
func TestQuotaOwnershipAndReferenceRetention(t *testing.T) {
	s, fake, p, session := testService(t)
	ctx := context.Background()
	input := inputFor(p, session, "reference-key-0001")
	if _, err := s.Submit(ctx, 1, input); !errors.Is(err, ErrQuota) {
		t.Fatal(err)
	}
	grant(t, s, 1, 10)
	if _, err := s.Submit(ctx, 2, input); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.Upload(ctx, 1, strings.NewReader("not a png")); err == nil {
		t.Fatal("spoof accepted")
	}
	asset, err := s.Upload(ctx, 1, bytes.NewReader(testPNG()))
	if err != nil {
		t.Fatal(err)
	}
	input.References = []Reference{{AssetID: asset.ID, Note: "pink"}, {AssetID: asset.ID}}
	if _, err = s.Quote(ctx, 2, input); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	task, err := s.Submit(ctx, 1, input)
	if err != nil {
		t.Fatal(err)
	}
	s.db.Model(&asset).Update("archived", true)
	fake.check = func(_ Input, files []string) {
		if len(files) != 1 {
			t.Fatal(files)
		}
		if _, err := os.ReadFile(files[0]); err != nil {
			t.Fatal(err)
		}
	}
	s.runOne(ctx)
	s.db.First(&task, task.ID)
	if task.Status != "succeeded" {
		t.Fatal(task)
	}
	if _, err = s.Submit(ctx, 1, inputFor(p, session, "pending-key-0000001")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Submit(ctx, 1, inputFor(p, session, "pending-key-0000002")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Submit(ctx, 1, inputFor(p, session, "pending-key-0000003")); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
}
func TestConcurrentSponsorsCreateOneChargeAndOneTask(t *testing.T) {
	s, fake, p, session := testService(t)
	ctx := context.Background()
	request := IdeaInput{Title: "Shared idea", Input: inputFor(p, session, ""), PublicConsent: true, ReferenceRights: true, CharacterIDs: []uint64{}}
	idea, err := s.CreateIdea(ctx, 1, request)
	if err != nil {
		t.Fatal(err)
	}
	for uid := uint64(2); uid <= 16; uid++ {
		grant(t, s, uid, 10)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for uid := uint64(2); uid <= 16; uid++ {
		wg.Add(1)
		go func(uid uint64) {
			defer wg.Done()
			if _, err := s.Sponsor(ctx, uid, idea.ID, "sponsor-unique-key-"+fmtName(uid), 2); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrBusy) {
				t.Error(err)
			}
		}(uid)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal(successes.Load())
	}
	var tasks int64
	s.db.Model(&Task{}).Where("idea_id = ?", idea.ID).Count(&tasks)
	if tasks != 1 {
		t.Fatal(tasks)
	}
	s.runOne(ctx)
	s.db.First(&idea, idea.ID)
	if idea.Status != "realized" || fake.calls.Load() != 1 {
		t.Fatal(idea, fake.calls.Load())
	}
	if _, err = s.Sponsor(ctx, 2, idea.ID, "another-sponsor-key", 2); !errors.Is(err, ErrBusy) {
		t.Fatal(err)
	}
}
func TestGrantAndBYOKKeyIsolation(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	grant(t, s, 1, 5)
	grant(t, s, 1, 5)
	if a := balance(t, s, 1); a.Available != 5 {
		t.Fatal(a)
	}
	byok, err := s.SaveProvider(ctx, 1, ProviderConfig{Name: "my key", AllowedProviderID: p.ID, Default: true}, "private-byok-secret")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(byok.CipherKey, "private-byok-secret") {
		t.Fatal("plaintext stored")
	}
	plain, err := s.decrypt(byok.CipherKey)
	if err != nil || plain != "private-byok-secret" {
		t.Fatal("cipher roundtrip failed")
	}
	input := inputFor(p, session, "byok-submit-0000001")
	input.Mode = "byok"
	input.ProviderID = byok.ID
	input.ExpectedCost = 0
	task, err := s.Submit(ctx, 1, input)
	if err != nil || task.Cost != 0 || task.DispatchProviderID != p.ID {
		t.Fatal(task, err)
	}
	if a := balance(t, s, 1); a.Available != 5 || a.Reserved != 0 {
		t.Fatal(a)
	}
	input.ProviderID = p.ID
	if _, err = s.Submit(ctx, 1, inputFor(p, session, "valid-site-key-0001")); err != nil {
		t.Fatal(err)
	}
}

func TestPaidReceiptRecoveryDoesNotRepeatProvider(t *testing.T) {
	s, fake, p, session := testService(t)
	grant(t, s, 1, 10)
	ctx := context.Background()
	task, err := s.Submit(ctx, 1, inputFor(p, session, "receipt-recovery-0001"))
	if err != nil {
		t.Fatal(err)
	}
	asset, err := s.saveImage(bytes.NewReader(testPNG()), task.UserID, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.writeReceipt(task, "succeeded", "", []Asset{asset}); err != nil {
		t.Fatal(err)
	}
	s.db.Model(&task).Updates(map[string]any{"status": "running", "updated_at": time.Now().Add(-11 * time.Minute)})
	s.recoverStale(ctx)
	s.recoverStale(ctx)
	s.db.First(&task, task.ID)
	var assets int64
	s.db.Model(&Asset{}).Where("task_id = ?", task.ID).Count(&assets)
	if task.Status != "succeeded" || assets != 1 || fake.calls.Load() != 0 {
		t.Fatal(task.Status, assets, fake.calls.Load())
	}
	if a := balance(t, s, 1); a.Available != 8 || a.Reserved != 0 {
		t.Fatal(a)
	}
	if _, err = os.Stat(s.receiptPath(task.ID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
}

func TestPartialResultsStayPrivateAndHoldReservation(t *testing.T) {
	s, _, p, session := testService(t)
	grant(t, s, 1, 10)
	p.MaxCount = 2
	var err error
	p, err = s.SaveProvider(context.Background(), 0, p, "")
	if err != nil {
		t.Fatal(err)
	}
	input := inputFor(p, session, "partial-result-0001")
	input.Count, input.ExpectedCost = 2, 4
	task, err := s.Submit(context.Background(), 1, input)
	if err != nil {
		t.Fatal(err)
	}
	s.runOne(context.Background())
	s.db.First(&task, task.ID)
	var assets []Asset
	s.db.Where("task_id = ?", task.ID).Find(&assets)
	if task.Status != "unknown" || len(assets) != 1 || assets[0].Public {
		t.Fatal(task.Status, assets)
	}
	if a := balance(t, s, 1); a.Available != 6 || a.Reserved != 4 {
		t.Fatal(a)
	}
	for _, name := range []string{".encryption-key", "../secret.png", ".task-1-result.json"} {
		if _, err := s.AssetPath(Asset{FileName: name}); err == nil {
			t.Fatal("private control file reachable", name)
		}
	}
}

func TestBlueprintWithoutSiteKeyAndPriceRevision(t *testing.T) {
	s, _, p, session := testService(t)
	ctx := context.Background()
	grant(t, s, 1, 10)
	input := inputFor(p, session, "price-confirm-00001")
	input.ExpectedCost = 1
	if _, err := s.Submit(ctx, 1, input); !errors.Is(err, ErrConflict) {
		t.Fatal("stale price accepted", err)
	}
	input.ExpectedCost = 2
	input.ProviderRevision = p.Revision + 1
	if _, err := s.Submit(ctx, 1, input); !errors.Is(err, ErrConflict) {
		t.Fatal("stale capability accepted", err)
	}
	s.db.Model(&ProviderConfig{}).Where("id = ?", p.ID).Update("cipher_key", "")
	input.ProviderRevision = 0
	idea, err := s.CreateIdea(ctx, 1, IdeaInput{Title: "Future idea", Input: input, PublicConsent: true, ReferenceRights: true})
	if err != nil || idea.Status != "waiting" {
		t.Fatal(idea, err)
	}
	if _, err = s.Submit(ctx, 1, input); !errors.Is(err, ErrUnavailable) {
		t.Fatal("unconfigured provider executed", err)
	}
	if a := balance(t, s, 1); a.Available != 10 || a.Reserved != 0 {
		t.Fatal(a)
	}
}

func TestMissingMasterKeyDoesNotReplaceEncryptedTaskKey(t *testing.T) {
	s, _, p, session := testService(t)
	grant(t, s, 1, 10)
	if _, err := s.Submit(context.Background(), 1, inputFor(p, session, "lost-master-000001")); err != nil {
		t.Fatal(err)
	}
	s.db.Model(&ProviderConfig{}).Where("id = ?", p.ID).Update("cipher_key", "")
	path := filepath.Join(s.dir, ".encryption-key")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := s.Init(context.Background()); err == nil {
		t.Fatal("silently replaced encryption key for historical tasks")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("replacement key created", err)
	}
}

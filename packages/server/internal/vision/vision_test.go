package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "test.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	s := New(db, nil, dir)
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&store.Content{}); err != nil {
		t.Fatal(err)
	}
	return s
}

func createJob(t *testing.T, s *Service) Job {
	t.Helper()
	path := "work.png"
	f, err := os.Create(filepath.Join(s.uploadDir, path))
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.Set(10, 10, color.RGBA{R: 255, A: 255})
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	row := store.Content{Title: PlaceholderTitle, Tags: `["待生成标签"]`, AuditStatus: "pending", FilePath: &path}
	job := Job{}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		job = NewJob(row)
		return tx.Create(&job).Error
	}); err != nil {
		t.Fatal(err)
	}
	return job
}

func configure(t *testing.T, s *Service, base string) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.BaseURL = base
	key := "test-only-key"
	if _, err := s.SaveSettings(context.Background(), SettingsUpdate{Config: cfg, Key: &key}); err != nil {
		t.Fatal(err)
	}
}

func TestTagNormalizationAndReferences(t *testing.T) {
	tags := NormalizeTags([]string{"#黑色长发, 校园制服，图片", "黑色长发", "CAT", "猫耳;未知;https://bad.test", "这是一张超级超级超级超级超级超级超级长的描述句"}, []string{"Cat"}, 8)
	if !reflect.DeepEqual(tags, []string{"黑色长发", "校园制服", "Cat", "猫耳"}) {
		t.Fatal(tags)
	}
	refs := []Reference{{Name: "小泉"}, {Name: "严小希"}}
	r, err := parseResult("```json\n"+`{"title":"小泉的校园午后","tags":["小泉","严小希","校园制服","图片"],"characters":[{"name":"小泉","confidence":0.95},{"name":"严小希","confidence":0.4}]}`+"\n```", nil, 8, refs...)
	if err != nil || !reflect.DeepEqual(r.Tags, []string{"小泉", "校园制服"}) {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err := parseResult(`{"title":"严小希的校园午后","tags":["校园制服"],"characters":[]}`, nil, 8, refs...); err == nil {
		t.Fatal("unconfirmed role survived")
	}
	for _, raw := range []string{`{"title":"","tags":["黑发"]}`, `{"title":"校园午后","tags":["图片","未知"]}`, `some prose {"title":"test"}`, `{"title":"校园午后","tags":[12]}`} {
		if _, err := parseResult(raw, nil, 8); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestClientVisionWireAndSecretRedaction(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer private-key" {
			t.Error("wrong request path/auth")
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"title":"小泉在校园","tags":["蓝色外套","小泉"],"characters":[{"name":"小泉","confidence":0.92}]}`}}}})
	}))
	defer srv.Close()
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL + "/v1/"
	cfg.APIKey = "private-key"
	r, err := Generate(context.Background(), cfg, "data:image/jpeg;base64,TARGET", []string{"蓝色外套"}, Reference{Name: "小泉", DataURL: "data:image/jpeg;base64,REF"})
	if err != nil || r.Tags[0] != "小泉" {
		t.Fatalf("%+v %v", r, err)
	}
	messages := received["messages"].([]any)
	parts := messages[1].(map[string]any)["content"].([]any)
	if len(parts) != 4 || parts[1].(map[string]any)["image_url"].(map[string]any)["url"] != "data:image/jpeg;base64,REF" || parts[3].(map[string]any)["image_url"].(map[string]any)["url"] != "data:image/jpeg;base64,TARGET" {
		t.Fatal(parts)
	}
	if received["enable_thinking"] != false || received["max_tokens"] != float64(1024) {
		t.Fatal(received)
	}
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte("private-key"))
	})
	if _, err := Generate(context.Background(), cfg, imageSample(), nil); err == nil || strings.Contains(err.Error(), "private-key") {
		t.Fatal(err)
	}
	b, _ := json.Marshal(SettingsUpdate{Config: cfg})
	if strings.Contains(string(b), "private-key") {
		t.Fatal(string(b))
	}
}

func TestClientCancellationAndRedirectBoundary(t *testing.T) {
	started := make(chan struct{})
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
	}))
	defer blocked.Close()
	cfg := DefaultConfig()
	cfg.BaseURL = blocked.URL
	cfg.APIKey = "test-only"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := Generate(ctx, cfg, imageSample(), nil); done <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled request succeeded")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request was not cancelled")
	}
	forwarded := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { forwarded = true }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	cfg.BaseURL = redirect.URL
	if _, err := Generate(context.Background(), cfg, imageSample(), nil); err == nil || forwarded {
		t.Fatal("redirect could disclose bearer token")
	}
}

func TestSettingsPreserveClearAndResume(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	initial, err := s.Settings(ctx)
	if err != nil || initial.Model != "Qwen/Qwen3.5-4B" || initial.HasAPIKey || initial.Enabled {
		t.Fatalf("%+v %v", initial, err)
	}
	key := "never-return-this"
	cfg := initial.Config
	if _, err := s.SaveSettings(ctx, SettingsUpdate{Config: cfg, Key: &key}); err != nil {
		t.Fatal(err)
	}
	cfg.Enabled = true
	saved, err := s.SaveSettings(ctx, SettingsUpdate{Config: cfg})
	if err != nil || !saved.HasAPIKey || !s.Ready(ctx) {
		t.Fatalf("%+v %v", saved, err)
	}
	b, _ := json.Marshal(saved)
	if strings.Contains(string(b), key) {
		t.Fatal("secret leaked")
	}
	if _, err := s.SaveSettings(ctx, SettingsUpdate{Config: cfg, ClearAPIKey: true}); err == nil {
		t.Fatal("enabled without key")
	}
	cfg.Enabled = false
	if _, err := s.SaveSettings(ctx, SettingsUpdate{Config: cfg, ClearAPIKey: true}); err != nil {
		t.Fatal(err)
	}
	if s.Ready(ctx) {
		t.Fatal("still enabled")
	}
	job := createJob(t, s)
	if err := s.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var pending Job
	s.db.First(&pending, job.ContentID)
	if pending.Attempts != 0 || pending.Status != "pending" {
		t.Fatal(pending)
	}
}

func TestDurableWorkerSuccessAndCrashRecovery(t *testing.T) {
	s := testService(t)
	job := createJob(t, s)
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"校园中的少年\",\"tags\":[\"校园\",\"短发\"]}"}}]}`))
	}))
	defer server.Close()
	configure(t, s, server.URL+"/v1")
	s.db.Model(&Job{}).Where("content_id = ?", job.ContentID).Updates(map[string]any{"status": "running", "lease": "old-process", "attempts": 1, "next_at": time.Now().UTC().Add(-time.Minute)})
	restarted := New(s.db, nil, s.uploadDir)
	if err := restarted.tick(ctx); err != nil {
		t.Fatal(err)
	}
	var got Job
	s.db.First(&got, job.ContentID)
	if got.Status != "succeeded" || got.Attempts != 2 {
		t.Fatal(got)
	}
	var row store.Content
	s.db.First(&row, job.ContentID)
	if row.Title != "校园中的少年" || row.AuditStatus != "pending" {
		t.Fatal(row)
	}
}

func TestCompletionPreservesOnlyDeclaredAIGC(t *testing.T) {
	for _, declared := range []bool{false, true} {
		t.Run(fmt.Sprint(declared), func(t *testing.T) {
			s := testService(t)
			job := createJob(t, s)
			job.Lease = "aigc-claim"
			if declared {
				job.ExpectedTags = `["待生成标签","AIGC"]`
			}
			s.db.Model(&store.Content{}).Where("id = ?", job.ContentID).Update("tags", job.ExpectedTags)
			s.db.Model(&Job{}).Where("content_id = ?", job.ContentID).Updates(map[string]any{"expected_tags": job.ExpectedTags, "status": "running", "lease": job.Lease})
			if err := s.complete(context.Background(), job, Result{Title: "校园午后", Tags: []string{"校园", "aigc", "AIGC"}}); err != nil {
				t.Fatal(err)
			}
			var row store.Content
			s.db.First(&row, job.ContentID)
			want := `["校园"]`
			if declared {
				want = `["校园","AIGC"]`
			}
			if row.Tags != want {
				t.Fatalf("declared=%v tags=%s", declared, row.Tags)
			}
		})
	}
	if got := NormalizeTags([]string{"AIGC", "aigc", "校园"}, nil, 1); !reflect.DeepEqual(got, []string{"校园"}) {
		t.Fatal(got)
	}
}

func TestHumanEditsDeletionAndLeasePreventWriteback(t *testing.T) {
	for _, mode := range []string{"edit", "delete", "lease"} {
		t.Run(mode, func(t *testing.T) {
			s := testService(t)
			job := createJob(t, s)
			job.Lease = "claim"
			s.db.Model(&Job{}).Where("content_id = ?", job.ContentID).Updates(map[string]any{"status": "running", "lease": job.Lease})
			switch mode {
			case "edit":
				s.db.Model(&store.Content{}).Where("id = ?", job.ContentID).Update("title", "手写标题")
			case "delete":
				s.db.Delete(&store.Content{}, job.ContentID)
			case "lease":
				s.db.Model(&Job{}).Where("content_id = ?", job.ContentID).Update("lease", "new-claim")
			}
			if err := s.complete(context.Background(), job, Result{Title: "模型标题", Tags: []string{"校园"}}); err != nil {
				t.Fatal(err)
			}
			var row store.Content
			err := s.db.First(&row, job.ContentID).Error
			if mode == "delete" {
				if !isMissing(err) {
					t.Fatal("resurrected")
				}
			} else if row.Title == "模型标题" {
				t.Fatal("overwrote current owner")
			}
		})
	}
}

func TestFailureRetryAndCancellation(t *testing.T) {
	s := testService(t)
	job := createJob(t, s)
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(429) }))
	defer server.Close()
	configure(t, s, server.URL)
	for i := 0; i < 3; i++ {
		s.db.Model(&Job{}).Where("content_id = ?", job.ContentID).Update("next_at", time.Now().UTC().Add(-time.Minute))
		if err := s.tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var failed Job
	s.db.First(&failed, job.ContentID)
	if failed.Status != "failed" || failed.Attempts != 3 || !strings.Contains(failed.Error, "429") {
		t.Fatal(failed)
	}
	if err := s.Retry(ctx, job.ContentID); err != nil {
		t.Fatal(err)
	}
	s.db.Model(&store.Content{}).Where("id = ?", job.ContentID).Update("tags", `["人工标签"]`)
	if err := s.tick(ctx); err != nil {
		t.Fatal(err)
	}
	s.db.First(&failed, job.ContentID)
	if failed.Status != "cancelled" {
		t.Fatal(failed)
	}
}

func TestReferenceCompressionAndLocalBoundary(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	file := filepath.Join(src, "小泉.png")
	f, _ := os.Create(file)
	img := image.NewRGBA(image.Rect(0, 0, 1500, 2000))
	_ = png.Encode(f, img)
	_ = f.Close()
	before, _ := os.ReadFile(file)
	refs, err := ImportReferences(context.Background(), src, dst)
	if err != nil || len(refs) != 1 || refs[0].Name != "小泉" {
		t.Fatalf("%+v %v", refs, err)
	}
	after, _ := os.ReadFile(file)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("original modified")
	}
	loaded, err := loadReferences(context.Background(), dst)
	if err != nil || len(loaded) != 1 || loaded[0].DataURL == "" {
		t.Fatal(err)
	}
	s := testService(t)
	job := createJob(t, s)
	job.FilePath = "../private.png"
	if _, err := s.generateJob(context.Background(), DefaultConfig(), job); err == nil {
		t.Fatal("accepted external path")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := imageDataURL(ctx, file); err == nil {
		t.Fatal("ignored cancellation")
	}
}

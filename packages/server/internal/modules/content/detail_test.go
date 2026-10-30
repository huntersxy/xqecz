package content

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func detailFixture(t testing.TB) (*Handler, *gin.Engine, *atomic.Int32) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "detail.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&store.User{}, &store.Content{}, &store.ContentLike{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]store.User{{ID: 7, Username: "owner"}, {ID: 8, Username: "admin", IsAdmin: 1}, {ID: 9, Username: "reader"}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.Content{ID: 42, Title: "shared detail", UserID: 7, Tags: "[]", AuditStatus: "approved"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&store.ContentLike{ContentID: 42, UserID: 8}).Error; err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	port, err := strconv.Atoi(mr.Port())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Prefix = mr.Host(), port, "detail-test:"
	client := cache.Open(cfg)
	t.Cleanup(func() { _ = client.Raw().Close() })
	deps := app.Deps{DB: db, Redis: client}
	h := New(deps)
	router := gin.New()
	router.GET("/content/:id", web.OptionalAuth(deps), h.detail)
	reads := &atomic.Int32{}
	if err := db.Callback().Query().Before("gorm:query").Register("test:detail-cold-read", func(tx *gorm.DB) {
		if tx.Statement.Table == "contents" {
			reads.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
	}); err != nil {
		t.Fatal(err)
	}
	return h, router, reads
}

func detailBurst(router *gin.Engine, callers int) []*httptest.ResponseRecorder {
	responses := make([]*httptest.ResponseRecorder, callers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range responses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			responses[i] = httptest.NewRecorder()
			router.ServeHTTP(responses[i], httptest.NewRequest("GET", "/content/42", nil))
		}(i)
	}
	close(start)
	wg.Wait()
	return responses
}

func TestDetailCoalescesColdReadsAndCountsEveryView(t *testing.T) {
	h, router, reads := detailFixture(t)
	const callers = 8
	for _, response := range detailBurst(router, callers) {
		var envelope struct {
			Data Item `json:"data"`
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &envelope) != nil || envelope.Data.LikeCount != 1 || envelope.Data.User.Username != "owner" {
			t.Fatalf("unexpected detail: %d %s", response.Code, response.Body.String())
		}
	}
	if got := reads.Load(); got != 1 {
		t.Fatalf("cold burst read content %d times, want 1", got)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/content/42", nil))
	if response.Code != 200 || reads.Load() != 1 {
		t.Fatalf("warm detail missed the cache: %d, reads=%d", response.Code, reads.Load())
	}
	var row store.Content
	if err := h.deps.DB.First(&row, 42).Error; err != nil {
		t.Fatal(err)
	}
	if row.ViewCount != callers+1 {
		t.Fatalf("persisted views = %d, want %d", row.ViewCount, callers+1)
	}
	key := "views:date:" + time.Now().Format("2006-01-02") + ":42"
	if got, err := h.deps.Redis.Get(t.Context(), key); err != nil || got != strconv.Itoa(callers+1) {
		t.Fatalf("daily views = %q, err=%v", got, err)
	}
}

func TestDetailSilentAndAuthenticatedReadsBypassSharedCache(t *testing.T) {
	h, router, _ := detailFixture(t)
	ctx := t.Context()
	if err := h.deps.Redis.SetJSON(ctx, "content:42", Item{ID: 42, Title: "cached"}, ContentTTL); err != nil {
		t.Fatal(err)
	}
	if err := h.deps.DB.Model(&store.Content{}).Where("id = ?", 42).Update("title", "current").Error; err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/content/42?silent=1", nil))
	var envelope struct {
		Data Item `json:"data"`
	}
	if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &envelope) != nil || envelope.Data.Title != "current" {
		t.Fatalf("silent detail used shared cache: %s", response.Body.String())
	}
	var row store.Content
	if err := h.deps.DB.First(&row, 42).Error; err != nil || row.ViewCount != 0 {
		t.Fatalf("silent read changed views: %d, err=%v", row.ViewCount, err)
	}
	if err := h.deps.DB.Model(&store.Content{}).Where("id = ?", 42).Update("audit_status", "rejected").Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		uid    uint64
		status int
	}{{7, 200}, {8, 200}, {9, 404}} {
		sid := fmt.Sprintf("session-%d", tc.uid)
		if err := h.deps.Redis.SetSession(ctx, sid, tc.uid); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/content/42", nil)
		req.Header.Set("Cookie", web.SessionCookie+"="+sid)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != tc.status {
			t.Fatalf("user %d: status=%d, body=%s", tc.uid, response.Code, response.Body.String())
		}
	}
	h.deps.Redis.ClearContentCache(ctx, 42)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/content/42", nil))
	if response.Code != 404 {
		t.Fatalf("anonymous rejected detail: %d %s", response.Code, response.Body.String())
	}
	if _, err := h.deps.Redis.Get(ctx, "content:42"); err == nil {
		t.Fatal("rejected detail entered the shared cache")
	}
}

func BenchmarkDetailColdBurst(b *testing.B) {
	h, router, reads := detailFixture(b)
	const callers = 8
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.deps.Redis.ClearContentCache(b.Context(), 42)
		for _, response := range detailBurst(router, callers) {
			if response.Code != 200 {
				b.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		}
	}
	b.ReportMetric(float64(reads.Load())/float64(b.N), "content-reads/burst")
}

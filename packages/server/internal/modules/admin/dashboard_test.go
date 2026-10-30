package admin

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
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
	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDashboardCoalescesColdReadsAndFreshRefreshesCache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "dashboard.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&store.Content{}, &store.User{}, &store.Comment{}, &store.Claim{}, &store.CommentReport{}, &store.Poll{}, &store.ContentLike{}); err != nil {
		t.Fatal(err)
	}
	mr := miniredis.RunT(t)
	port, err := strconv.Atoi(mr.Port())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Prefix = mr.Host(), port, "dashboard-test:"
	client := cache.Open(cfg)
	t.Cleanup(func() { _ = client.Raw().Close() })
	deps := app.Deps{DB: db, Redis: client}
	h := New(deps, content.New(deps))
	var computations atomic.Int32
	if err := db.Callback().Row().Before("gorm:row").Register("test:dashboard-cold-read", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "FROM contents") {
			computations.Add(1)
			time.Sleep(10 * time.Millisecond)
		}
	}); err != nil {
		t.Fatal(err)
	}
	request := func(path string) (dashboardData, error) {
		response := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(response)
		c.Request = httptest.NewRequest("GET", path, nil)
		h.dashboard(c)
		var envelope struct {
			Data dashboardData `json:"data"`
		}
		if response.Code != 200 {
			return dashboardData{}, fmt.Errorf("dashboard status=%d, body=%s", response.Code, response.Body.String())
		}
		err := json.Unmarshal(response.Body.Bytes(), &envelope)
		return envelope.Data, err
	}
	const callers = 8
	errs := make([]error, callers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = request("/dashboard")
		}(i)
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := computations.Load(); got != 1 {
		t.Fatalf("cold burst computed dashboard %d times, want 1", got)
	}
	if err := db.Create(&store.Content{Title: "new content", Tags: "[]", AuditStatus: "approved"}).Error; err != nil {
		t.Fatal(err)
	}
	if data, err := request("/dashboard"); err != nil || data.Content.Total != 0 || computations.Load() != 1 {
		t.Fatalf("warm cache: total=%d computations=%d err=%v", data.Content.Total, computations.Load(), err)
	}
	if data, err := request("/dashboard?fresh=1"); err != nil || data.Content.Total != 1 || computations.Load() != 2 {
		t.Fatalf("fresh dashboard: total=%d computations=%d err=%v", data.Content.Total, computations.Load(), err)
	}
	if data, err := request("/dashboard"); err != nil || data.Content.Total != 1 || computations.Load() != 2 {
		t.Fatalf("cache after fresh: total=%d computations=%d err=%v", data.Content.Total, computations.Load(), err)
	}
}

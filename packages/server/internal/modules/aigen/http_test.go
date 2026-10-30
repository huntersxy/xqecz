package aigen

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	core "github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTaskViewsExposeDiagnosticOnlyForAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "views.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&store.User{}, &core.Task{}, &core.Asset{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&store.User{ID: 1, Username: "owner"}).Error; err != nil {
		t.Fatal(err)
	}
	const diagnostic = "time: 2026-01-01T00:00:00Z\ncategory: provider_http\nhttp_status: 502"
	snapshot, err := json.Marshal(map[string]any{
		"input":      map[string]any{"prompt": "pink circle", "base_prompt": "pink circle"},
		"diagnostic": diagnostic,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := core.Task{UserID: 1, Prompt: "pink circle", Status: "unknown", Error: "供应商结果不明（任务 #1）", Snapshot: string(snapshot)}
	if err = db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	assets := []core.Asset{{TaskID: task.ID, UserID: 1}, {TaskID: task.ID, UserID: 1, Archived: true}}
	if err = db.Create(&assets).Error; err != nil {
		t.Fatal(err)
	}
	var loadedAssets int64
	if err = db.Callback().Query().After("gorm:query").Register("test:loaded-assets", func(tx *gorm.DB) {
		if tx.Statement.Table == "ai_assets" {
			loadedAssets = tx.RowsAffected
		}
	}); err != nil {
		t.Fatal(err)
	}
	h := &Handler{s: core.New(db, config.Config{})}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/admin/ai/tasks", nil)

	adminViews, err := h.taskViews(c, []core.Task{task}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(adminViews) != 1 {
		t.Fatalf("admin views = %d", len(adminViews))
	}
	adminItem, ok := adminViews[0].(gin.H)
	if !ok {
		t.Fatalf("admin view type = %T", adminViews[0])
	}
	if got := adminItem["diagnostic"]; got != diagnostic {
		t.Fatalf("admin diagnostic = %#v", got)
	}
	if got := adminItem["assets"].([]core.Asset); len(got) != 1 || got[0].ID != assets[0].ID || loadedAssets != 1 {
		t.Fatalf("admin assets=%+v, loaded=%d, want only the active asset", got, loadedAssets)
	}
	rawAdmin, err := json.Marshal(adminItem)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawAdmin), `"diagnostic":"time:`) {
		t.Fatalf("admin JSON lacks the diagnostic field: %s", rawAdmin)
	}

	userViews, err := h.taskViews(c, []core.Task{task}, false)
	if err != nil {
		t.Fatal(err)
	}
	userItem, ok := userViews[0].(gin.H)
	if !ok {
		t.Fatalf("user view type = %T", userViews[0])
	}
	if _, exists := userItem["diagnostic"]; exists {
		t.Fatal("user view exposed the administrator diagnostic")
	}
	if got := userItem["assets"].([]core.Asset); len(got) != 1 || got[0].ID != assets[0].ID || loadedAssets != 1 {
		t.Fatalf("user assets=%+v, loaded=%d, want only the active asset", got, loadedAssets)
	}
	rawUser, err := json.Marshal(userItem)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rawUser), `"diagnostic"`) {
		t.Fatalf("user JSON leaked the diagnostic field: %s", rawUser)
	}
	if _, exists := userItem["task"]; !exists {
		t.Fatal("user view lost the task field")
	}
}

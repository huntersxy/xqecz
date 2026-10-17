package comment

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestQueryDeduplicatesAuthorIDsBeforeBatchLookup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&store.User{}, &store.Comment{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Create(&store.User{ID: 7, Username: "author"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now().UTC()
	top := store.Comment{ID: 1, ContentID: 9, UserID: 7, Text: "top", CreatedAt: now}
	if err := db.Create(&top).Error; err != nil {
		t.Fatalf("create top: %v", err)
	}
	parent := top.ID
	for id := uint64(2); id <= 4; id++ {
		row := store.Comment{ID: id, ContentID: 9, UserID: 7, Text: "reply", ParentID: &parent, CreatedAt: now.Add(time.Duration(id) * time.Second)}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create reply %d: %v", id, err)
		}
		parent = id
	}

	var mu sync.Mutex
	var userVars int
	if err := db.Callback().Query().After("gorm:query").Register("test:comment-user-vars", func(tx *gorm.DB) {
		if tx.Statement.Table != "users" {
			return
		}
		mu.Lock()
		userVars = len(tx.Statement.Vars)
		mu.Unlock()
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
	h := New(app.Deps{DB: db})
	page, err := h.query(context.Background(), 9, 1, 20)
	if err != nil {
		t.Fatalf("query comments: %v", err)
	}
	if len(page.List) != 1 || len(page.List[0].Replies) != 3 {
		t.Fatalf("unexpected thread shape: %+v", page)
	}
	mu.Lock()
	got := userVars
	mu.Unlock()
	if got != 1 {
		t.Fatalf("author lookup used %d bind values, want one unique user id", got)
	}
}

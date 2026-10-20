package content

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPublicListIncludesViewCount(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&store.User{}, &store.Content{}, &store.ContentLike{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now().UTC()
	row := store.Content{ID: 21, Title: "viewed", Tags: "[]", AuditStatus: "approved", ViewCount: 5, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("create content: %v", err)
	}

	h := &Handler{deps: app.Deps{DB: db}}
	page, err := h.queryDB(context.Background(), listOpts{
		Page: 1, PageSize: 20, SortBy: "created_at", Order: "desc",
		AuditStatuses: []string{"approved"},
	})
	if err != nil {
		t.Fatalf("query public list: %v", err)
	}
	if len(page.List) != 1 {
		t.Fatalf("got %d items, want 1", len(page.List))
	}
	if page.List[0].ViewCount == nil || *page.List[0].ViewCount != 5 {
		t.Fatalf("public list view_count = %v, want 5", page.List[0].ViewCount)
	}
}

func TestDecorateManyBatchesRelatedData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&store.User{}, &store.Content{}, &store.ContentLike{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	email := "author@example.com"
	if err := db.Create(&store.User{ID: 7, Username: "author", Email: &email}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now().UTC()
	path1, path2 := "one.webp", "two.webp"
	rows := []store.Content{
		{ID: 11, Title: "one", FilePath: &path1, Tags: "[]", AuditStatus: "approved", UserID: 7, ViewCount: 3, CreatedAt: now, UpdatedAt: now},
		{ID: 12, Title: "two", FilePath: &path2, Tags: "[]", AuditStatus: "approved", UserID: 7, ViewCount: 8, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create contents: %v", err)
	}
	if err := db.Create(&store.ContentLike{ContentID: 11, UserID: 7}).Error; err != nil {
		t.Fatalf("create like: %v", err)
	}

	h := &Handler{deps: app.Deps{DB: db}}
	var queryCount int32
	if err := db.Callback().Query().Before("gorm:query").Register("test:count-decorate-queries", func(_ *gorm.DB) {
		atomic.AddInt32(&queryCount, 1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	if err := db.Callback().Row().Before("gorm:row").Register("test:count-decorate-rows", func(_ *gorm.DB) {
		atomic.AddInt32(&queryCount, 1)
	}); err != nil {
		t.Fatalf("register row counter: %v", err)
	}
	got, err := h.DecorateMany(context.Background(), rows, true)
	if err != nil {
		t.Fatalf("decorate many: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d items, want 2", len(got))
	}
	if n := atomic.LoadInt32(&queryCount); n != 2 {
		t.Fatalf("decorate many executed %d related queries, want 2", n)
	}
	if got[0].User.Username != "author" || got[1].User.Username != "author" {
		t.Fatalf("author lookup was not applied to every item: %+v", got)
	}
	if got[0].LikeCount != 1 || got[1].LikeCount != 0 {
		t.Fatalf("like counts = %d, %d; want 1, 0", got[0].LikeCount, got[1].LikeCount)
	}
	if got[0].ViewCount == nil || *got[0].ViewCount != 3 || got[1].ViewCount == nil || *got[1].ViewCount != 8 {
		t.Fatalf("view counts were not retained: %+v", got)
	}
}

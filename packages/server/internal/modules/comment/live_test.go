package comment

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/project"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
	"github.com/joho/godotenv"
	"gorm.io/gorm/logger"
)

// 显式启用时在真实库的单个事务中验证查询和删除；测试行始终回滚，不留下数据。
func TestReplyChainLive(t *testing.T) {
	if os.Getenv("XQECZ_COMMENT_LIVE") != "1" {
		t.Skip("set XQECZ_COMMENT_LIVE=1 to run the rollback-only database regression")
	}
	root := project.FindRoot(".")
	_ = godotenv.Load(filepath.Join(root, ".env"))
	_ = godotenv.Overload(filepath.Join(root, ".env.local"))
	cfg := config.Load()
	db, err := store.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	db.Logger = logger.Default.LogMode(logger.Silent)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	suffix := time.Now().Format("150405000000")
	email := "comment-regression-" + suffix + "@example.invalid"
	user := store.User{Username: "__reply_" + suffix, Email: &email, Password: "unusable-test-password"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	content := store.Content{Title: "reply-chain-regression", UserID: user.ID, Tags: "[]", AuditStatus: "pending"}
	if err := tx.Create(&content).Error; err != nil {
		t.Fatal(err)
	}
	other := store.Content{Title: "reply-boundary-regression", UserID: user.ID, Tags: "[]", AuditStatus: "pending"}
	if err := tx.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	cfg.Redis.Prefix = "reply-regression:" + suffix + ":"
	redis := cache.Open(cfg)
	defer redis.Raw().Close()
	h := New(app.Deps{DB: tx, Redis: redis})
	created := time.Now().Add(-time.Hour)
	a := store.Comment{ContentID: content.ID, UserID: user.ID, Text: "A", CreatedAt: created}
	if err := tx.Create(&a).Error; err != nil {
		t.Fatal(err)
	}
	b := store.Comment{ContentID: content.ID, UserID: user.ID, Text: "B", ParentID: &a.ID, CreatedAt: created.Add(time.Second)}
	if err := tx.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	c := store.Comment{ContentID: content.ID, UserID: user.ID, Text: "C", ParentID: &b.ID, CreatedAt: created.Add(2 * time.Second)}
	if err := tx.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	d := store.Comment{ContentID: content.ID, UserID: user.ID, Text: "D", ParentID: &c.ID, CreatedAt: created.Add(3 * time.Second)}
	if err := tx.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	nextTop := store.Comment{ContentID: content.ID, UserID: user.ID, Text: "another top", CreatedAt: created.Add(4 * time.Second)}
	if err := tx.Create(&nextTop).Error; err != nil {
		t.Fatal(err)
	}
	page, err := h.query(context.Background(), content.ID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.List) != 1 || len(page.List[0].Replies) != 3 {
		t.Fatalf("分页不能截断 A/B/C/D 回复链: %+v", page)
	}
	if page.List[0].Replies[1].Parent.ID != b.ID {
		t.Fatal("C 引用的应是 B")
	}
	t.Log("A/B/C/D 全部可见，直接父评论引用正确，顶层分页保持完整回复链")

	// 回复其他内容下的父评论必须业务失败，且不产生额外行。
	body, _ := json.Marshal(addReq{ContentID: other.ID, Text: "invalid", ParentID: &b.ID})
	record := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(record)
	ctx.Request = httptest.NewRequest("POST", "/comment/add", strings.NewReader(string(body)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("xqecz.identity", web.Identity{UID: user.ID})
	h.add(ctx)
	if record.Code != 200 || !strings.Contains(record.Body.String(), "\"code\":400") {
		t.Fatalf("跨内容回复必须 HTTP 200 + 业务码 400: %s", record.Body.String())
	}

	record = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(record)
	ctx.Request = httptest.NewRequest("DELETE", "/comment/"+suffix, nil)
	ctx.Params = gin.Params{{Key: "id", Value: fmtID(b.ID)}}
	ctx.Set("xqecz.identity", web.Identity{UID: user.ID})
	h.remove(ctx)
	if !strings.Contains(record.Body.String(), "\"code\":200") {
		t.Fatalf("删除 B 失败: %s", record.Body.String())
	}
	page, err = h.query(context.Background(), content.ID, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, thread := range page.List {
		if thread.ID == c.ID {
			found = len(thread.Replies) == 1 && thread.Replies[0].ID == d.ID
		}
	}
	if !found {
		t.Fatal("删除 B 后 C/D 必须仍可见")
	}
	t.Log("删除 B 后 C 提升为顶层，D 继续展示；所有测试记录将在事务回滚时清理")
}

func fmtID(id uint64) string { return strconv.FormatUint(id, 10) }

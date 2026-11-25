package compress

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
)

func TestPickCandidateSkipsUnsupportedPrivateAndCoolingFiles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "queue.db")))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&store.Content{}); err != nil {
		t.Fatal(err)
	}

	var rows []store.Content
	for _, fixture := range []struct {
		path, status string
		size         int64
	}{
		{"unsupported.bmp", "approved", 5000},
		{"private.webp", "rejected", 4000},
		{"pending.webp", "pending", 3500},
		{"cooling.JPG", "approved", 3000},
		{"ready.PNG", "approved", 2000},
	} {
		rows = append(rows, store.Content{FilePath: &fixture.path, AuditStatus: fixture.status, FileSize: fixture.size})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	worker := NewWorker(app.Deps{DB: db, Cfg: config.Config{CompressMinSize: 1000}})
	worker.markFailed(rows[3].ID)

	row, found, err := worker.pickCandidate(t.Context())
	if err != nil || !found || row.ID != rows[4].ID {
		t.Fatalf("candidate=%d found=%t err=%v", row.ID, found, err)
	}
}

type testTransport func(*http.Request) (*http.Response, error)

func (transport testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func TestShrinkPreservesOriginalAndChecksOutputSize(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "sample.png")
	worker := NewWorker(app.Deps{Cfg: config.Config{BinDir: filepath.Join(dir, "bin"), TinyPNGAPIKey: "fixture"}})
	expectedSize := "3"
	worker.client.http.Transport = testTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body: io.NopCloser(strings.NewReader(`{"output":{"url":"https://api.tinify.com/output/fixture","size":` +
					expectedSize + `}}`)),
			}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("new"))}, nil
	})
	if err := os.WriteFile(original, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	written, archive, err := worker.shrinkInPlace(t.Context(), original, 8)
	if err != nil || written != 3 {
		t.Fatalf("written=%d err=%v", written, err)
	}
	if data, err := os.ReadFile(archive); err != nil || string(data) != "original" {
		t.Fatalf("archive=%q err=%v", data, err)
	}
	expectedSize = "4"
	if _, _, err := worker.shrinkInPlace(t.Context(), original, 8); err == nil {
		t.Fatal("accepted a truncated response")
	}
	if data, err := os.ReadFile(original); err != nil || string(data) != "new" {
		t.Fatalf("current image=%q err=%v", data, err)
	}
}

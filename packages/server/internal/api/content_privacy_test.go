package api

import (
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/modules/content"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

func TestGuestAvatarReferenceFollowsContentVisibility(t *testing.T) {
	deps, _ := visionFixture(t)
	deps.Cfg.AvatarOrigin = "https://api.example.com"
	deps.Cfg.AvatarKey = strings.Repeat("59", 32)
	handler := New(deps)
	nickname, email := "guest fixture", "guest@example.com"
	row := store.Content{Title: "guest work fixture", GuestNickname: &nickname, GuestEmail: &email, AuditStatus: "pending", Tags: "[]"}
	if err := deps.DB.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := deps.Redis.SetJSON(t.Context(), "content:"+strconv.FormatUint(row.ID, 10), map[string]any{
		"id": row.ID, "audit_status": "pending", "avatar_url": "https://elsewhere.example.com/avatar/legacy",
	}, time.Minute); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/content/"+strconv.FormatUint(row.ID, 10), nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	var envelope struct {
		Data content.Item `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || response.Code != http.StatusOK {
		t.Fatalf("public detail failed: status=%d err=%v", response.Code, err)
	}
	ref, err := url.Parse(envelope.Data.AvatarURL)
	if err != nil || ref.Host != "api.example.com" || !strings.HasPrefix(ref.Path, "/api/avatars/") ||
		strings.Contains(response.Body.String(), email) {
		t.Fatalf("invalid guest avatar reference: %v", err)
	}
	if err := deps.DB.Model(&row).Update("audit_status", "rejected").Error; err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, ref.Path, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("moderated guest avatar status=%d", response.Code)
	}
}

func TestRejectedContentIsPrivateAcrossPublicRoutes(t *testing.T) {
	deps, handler := visionFixture(t)
	fixtureFile := "reviewed.png"
	fixtureThumb := "thumbs/reviewed.png"
	row := store.Content{Title: "fixture moderation", UserID: 2, FilePath: &fixtureFile, ThumbPath: &fixtureThumb, AuditStatus: "approved", Tags: "[]"}
	if err := deps.DB.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := deps.DB.Create(&store.Comment{ContentID: row.ID, UserID: 2, Text: "fixture comment"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(deps.Cfg.UploadDir, fixtureFile),
		filepath.Join(deps.Cfg.ThumbDir, fixtureFile),
		filepath.Join(deps.Cfg.UploadDir, "unattached.png"),
		filepath.Join(deps.Cfg.ImagesDir, "site", "favicon-1-2.png"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, image.NewRGBA(image.Rect(0, 0, 2, 2)))
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("create fixture: encode=%v close=%v", err, closeErr)
		}
	}
	call := func(method, path, session string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, nil)
		if session != "" {
			req.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: session})
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	id := strconv.FormatUint(row.ID, 10)
	protected := []string{
		"/api/content/" + id,
		"/api/comment/list/" + id,
		"/api/comment/count/" + id,
		"/uploads/" + fixtureFile,
		"/" + fixtureThumb,
	}
	for _, path := range protected {
		if response := call(http.MethodGet, path, ""); response.Code != http.StatusOK {
			t.Fatalf("public fixture %s: status=%d", path, response.Code)
		}
	}
	if err := deps.DB.Model(&row).Update("audit_status", "rejected").Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range protected {
		if response := call(http.MethodGet, path, ""); response.Code != http.StatusNotFound {
			t.Fatalf("anonymous %s: status=%d", path, response.Code)
		}
		for _, session := range []string{"fixture-author", "fixture-admin"} {
			response := call(http.MethodGet, path, session)
			if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Cache-Control"), "no-store") {
				t.Fatalf("authorized %s: status=%d cache=%q", path, response.Code, response.Header().Get("Cache-Control"))
			}
		}
	}
	if response := call(http.MethodHead, "/uploads/"+fixtureFile, ""); response.Code != http.StatusNotFound {
		t.Fatalf("anonymous media HEAD: status=%d", response.Code)
	}
	if response := call(http.MethodGet, "/uploads/unattached.png", "fixture-admin"); response.Code != http.StatusNotFound {
		t.Fatalf("unattached file: status=%d", response.Code)
	}
	if response := call(http.MethodGet, "/images/site/favicon-1-2.png", ""); response.Code != http.StatusOK {
		t.Fatalf("site asset: status=%d", response.Code)
	}
	response := call(http.MethodGet, "/api/content/list?audit_status=rejected", "")
	var envelope struct {
		Data content.Page `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || response.Code != http.StatusOK ||
		envelope.Data.Total != 0 || envelope.Data.List == nil || len(envelope.Data.List) != 0 {
		t.Fatalf("public list returned private entries: status=%d err=%v", response.Code, err)
	}
}

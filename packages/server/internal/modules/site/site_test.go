package site

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/migrate"
	sitecfg "github.com/huntersxy/xqecz/server/internal/site"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testHandler(t *testing.T) (*Handler, *sitecfg.Service, config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "site-http.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("CREATE TABLE contents (id BIGINT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	if err := migrate.Run(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{ImagesDir: filepath.Join(t.TempDir(), "images")}
	service := sitecfg.New(db, cfg)
	if err := service.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return &Handler{deps: app.Deps{Site: service}}, service, cfg
}

func call(t *testing.T, handler func(*gin.Context), method, target string, body *bytes.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, target, nil)
	} else {
		request = httptest.NewRequest(method, target, body)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	c.Request = request
	handler(c)
	return recorder
}

func envelope(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v (%s)", err, recorder.Body.String())
	}
	return payload
}

func TestPublicConfigAndManifest(t *testing.T) {
	h, service, _ := testHandler(t)
	saved := sitecfg.Defaults()
	saved.Name = "模板宇宙"
	saved.Tagline = "自定义画廊"
	saved.Texts = map[string]string{"nav.gallery": "作品集"}
	if _, err := service.Save(context.Background(), saved); err != nil {
		t.Fatal(err)
	}

	configRecorder := call(t, h.config, http.MethodGet, "/site/config", nil, "")
	if configRecorder.Code != http.StatusOK {
		t.Fatalf("config status = %d", configRecorder.Code)
	}
	body := envelope(t, configRecorder)
	data := body["data"].(map[string]any)
	if data["name"] != "模板宇宙" || data["tagline"] != "自定义画廊" {
		t.Fatalf("config data = %#v", data)
	}
	texts := data["texts"].(map[string]any)
	if texts["nav.gallery"] != "作品集" {
		t.Fatalf("texts = %#v", texts)
	}
	if title := data["browser_title"].(string); !strings.Contains(title, "模板宇宙") {
		t.Fatalf("browser title = %q", title)
	}

	manifestRecorder := call(t, h.manifest, http.MethodGet, "/site/manifest", nil, "")
	if manifestRecorder.Code != http.StatusOK {
		t.Fatalf("manifest status = %d", manifestRecorder.Code)
	}
	if got := manifestRecorder.Header().Get("Content-Type"); !strings.Contains(got, "application/manifest+json") {
		t.Fatalf("manifest content type = %q", got)
	}
	manifest := envelope(t, manifestRecorder)
	if manifest["name"] != "模板宇宙 · 小泉动漫二创站" || manifest["short_name"] != "模板宇宙" {
		t.Fatalf("manifest = %#v", manifest)
	}
	icons := manifest["icons"].([]any)
	if len(icons) != 2 || icons[0].(map[string]any)["src"] != "/favicon.webp" {
		t.Fatalf("manifest icons = %#v", icons)
	}
}

func TestAdminSaveAndUpload(t *testing.T) {
	h, service, cfg := testHandler(t)
	saved := sitecfg.Defaults()
	saved.Name = "后台改名"
	payload, err := json.Marshal(saved)
	if err != nil {
		t.Fatal(err)
	}
	saveRecorder := call(t, h.save, http.MethodPut, "/admin/site/settings", bytes.NewReader(payload), "application/json")
	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("save status = %d body=%s", saveRecorder.Code, saveRecorder.Body.String())
	}
	body := envelope(t, saveRecorder)
	data := body["data"].(map[string]any)
	if data["name"] != "后台改名" {
		t.Fatalf("saved data = %#v", data)
	}
	if elements := data["elements"].([]any); len(elements) < 10 {
		t.Fatalf("elements = %d", len(elements))
	}
	if service.Public(context.Background()).Name != "后台改名" {
		t.Fatal("service did not update")
	}

	img := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 20, B: 80, A: 255})
		}
	}
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	var uploadBody bytes.Buffer
	writer := multipart.NewWriter(&uploadBody)
	part, err := writer.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(pngData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/site/assets/logo", bytes.NewReader(uploadBody.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Params = gin.Params{{Key: "kind", Value: "logo"}}
	h.upload(c)
	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	uploadData := envelope(t, recorder)["data"].(map[string]any)
	url := uploadData["url"].(string)
	if !strings.HasPrefix(url, "/images/site/logo-") {
		t.Fatalf("upload url = %q", url)
	}
	if _, err := os.Stat(filepath.Join(cfg.ImagesDir, "site", strings.TrimPrefix(url, "/images/site/"))); err != nil {
		t.Fatalf("uploaded file missing: %v", err)
	}
}

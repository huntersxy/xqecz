package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"github.com/huntersxy/xqecz/server/internal/web"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func visionFixture(t *testing.T) (app.Deps, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "fixture.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&store.User{}, &store.Content{}, &store.Comment{}, &store.ContentLike{}, &store.ContentFavorite{}, &store.CommentReport{}, &store.Claim{}, &store.APIKey{}, &store.Poll{}, &store.PollVote{}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Env: "production", UploadDir: filepath.Join(dir, "uploads"), ThumbDir: filepath.Join(dir, "thumbs"), ImagesDir: filepath.Join(dir, "images"), BinDir: filepath.Join(dir, "bin"), CORSOrigins: []string{"http://localhost:5178"}}
	mr := miniredis.RunT(t)
	cfg.Redis.Host = mr.Host()
	cfg.Redis.Port, _ = strconv.Atoi(mr.Port())
	redis := cache.Open(cfg)
	t.Cleanup(func() { _ = redis.Raw().Close() })
	s := vision.New(db, redis, cfg.UploadDir, filepath.Join(dir, "references"))
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	password, _ := bcrypt.GenerateFromPassword([]byte("fixture-password"), bcrypt.MinCost)
	email := "fixture@example.com"
	users := []store.User{{ID: 1, Username: "测试管理员", Password: string(password), Email: &email, IsAdmin: 1}, {ID: 2, Username: "测试作者", Password: string(password), Email: &email}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	for i, sid := range []string{"fixture-admin", "fixture-author"} {
		if err := redis.SetSession(context.Background(), sid, uint64(i+1)); err != nil {
			t.Fatal(err)
		}
	}
	text := "本地隔离测试作品"
	seed := store.Content{Title: "校园中的小泉", Content: &text, UserID: 1, AuditStatus: "approved", Tags: `["校园"]`}
	db.Create(&seed)
	deps := app.Deps{Cfg: cfg, DB: db, Redis: redis, Vision: s}
	return deps, New(deps)
}

func multipartRequest(t *testing.T, path, session, mime string, fields map[string]string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := mw.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	if mime != "" {
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition", `form-data; name="file"; filename="fixture.png"`)
		hdr.Set("Content-Type", mime)
		part, err := mw.CreatePart(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if mime == "image/png" {
			_ = png.Encode(part, image.NewRGBA(image.Rect(0, 0, 8, 8)))
		} else {
			_, _ = part.Write([]byte("video"))
		}
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if session != "" {
		req.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: session})
	}
	return req
}

func TestAIGCUploadMerging(t *testing.T) {
	deps, handler := visionFixture(t)
	for _, path := range []string{"/api/content/quick-upload", "/api/content/upload"} {
		for _, tc := range []struct {
			name, ai, auto, rawTags string
			want                    []string
		}{
			{"manual checked", "true", "false", `["校园"]`, []string{"校园", "AIGC"}},
			{"manual duplicate", "true", "false", `["校园","aigc","AIGC"]`, []string{"校园", "AIGC"}},
			{"manual owned", "false", "false", `["校园","aigc"]`, []string{"校园", "AIGC"}},
			{"unmarked", "false", "false", `["校园"]`, []string{"校园"}},
			{"auto checked", "true", "true", `["校园"]`, []string{vision.PlaceholderTag, "AIGC"}},
			{"auto manual owned", "false", "true", `["#AIGC","校园"]`, []string{vision.PlaceholderTag, "AIGC"}},
			{"auto unmarked", "false", "true", `["校园"]`, []string{vision.PlaceholderTag}},
		} {
			t.Run(path+tc.name, func(t *testing.T) {
				mime := ""
				if tc.auto == "true" {
					mime = "image/png"
				}
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, multipartRequest(t, path, "fixture-author", mime, map[string]string{"title": "手动标题", "content": "描述", "tags": tc.rawTags, "auto_metadata": tc.auto, "is_ai_generated": tc.ai}))
				var res struct {
					Code int `json:"code"`
					Data struct {
						ID   uint64   `json:"id"`
						Tags []string `json:"tags"`
					} `json:"data"`
				}
				if json.Unmarshal(w.Body.Bytes(), &res) != nil || res.Code != 200 || !reflect.DeepEqual(res.Data.Tags, tc.want) {
					t.Fatal(w.Body.String())
				}
				if tc.auto == "true" {
					var job vision.Job
					deps.DB.First(&job, res.Data.ID)
					var tags []string
					_ = json.Unmarshal([]byte(job.ExpectedTags), &tags)
					if !reflect.DeepEqual(tags, tc.want) {
						t.Fatal(job)
					}
				}
			})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, multipartRequest(t, path, "fixture-author", "", map[string]string{"title": "标题", "content": "描述", "is_ai_generated": "maybe"}))
		var res struct {
			Code int `json:"code"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res.Code != 400 || w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
}

func TestVisionUploadOutboxAndAdminAuthorization(t *testing.T) {
	deps, handler := visionFixture(t)
	var createdID uint64
	for _, path := range []string{"/api/content/quick-upload", "/api/content/upload"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, multipartRequest(t, path, "fixture-author", "image/png", map[string]string{"auto_metadata": "true", "title": "ignored", "tags": "ignored"}))
		var result struct {
			Code int `json:"code"`
			Data struct {
				ID    uint64   `json:"id"`
				Title string   `json:"title"`
				Tags  []string `json:"tags"`
				Audit string   `json:"audit_status"`
			} `json:"data"`
		}
		if json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Code != 200 || result.Data.Title != vision.PlaceholderTitle || result.Data.Tags[0] != vision.PlaceholderTag || result.Data.Audit != "pending" {
			t.Fatal(w.Body.String())
		}
		var job vision.Job
		if deps.DB.First(&job, result.Data.ID).Error != nil || job.Status != "pending" {
			t.Fatal(job)
		}
		createdID = result.Data.ID
	}
	jobRequest := httptest.NewRequest("GET", "/api/admin/vision/jobs", nil)
	jobRequest.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: "fixture-admin"})
	jobResponse := httptest.NewRecorder()
	handler.ServeHTTP(jobResponse, jobRequest)
	var jobs struct {
		Data []map[string]any `json:"data"`
	}
	if json.Unmarshal(jobResponse.Body.Bytes(), &jobs) != nil || len(jobs.Data) != 2 {
		t.Fatal(jobResponse.Body.String())
	}
	for _, job := range jobs.Data {
		for _, field := range []string{"created_at", "updated_at"} {
			if ok, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`, job[field].(string)); !ok {
				t.Fatal(job)
			}
		}
	}
	for _, title := range []string{"人工修改", vision.PlaceholderTitle} {
		req := multipartRequest(t, fmt.Sprintf("/api/content/%d", createdID), "fixture-author", "", map[string]string{"title": title})
		req.Method = http.MethodPut
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var job vision.Job
		deps.DB.First(&job, createdID)
		if job.Status != "cancelled" {
			t.Fatal(job)
		}
	}
	for _, mime := range []string{"", "video/mp4"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, multipartRequest(t, "/api/content/quick-upload", "", mime, map[string]string{"auto_metadata": "true", "title": "title", "content": "text"}))
		var response map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		if w.Code != 200 || response["code"] != float64(400) {
			t.Fatal(w.Body.String())
		}
	}
	for _, session := range []string{"", "fixture-author", "fixture-admin"} {
		req := httptest.NewRequest("GET", "/api/admin/vision/settings", nil)
		if session != "" {
			req.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: session})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		want := 401
		if session == "fixture-author" {
			want = 403
		}
		if session == "fixture-admin" {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("%s %d", session, w.Code)
		}
	}
	deps.DB.Migrator().DropTable(&vision.Job{})
	var before, after int64
	deps.DB.Model(&store.Content{}).Count(&before)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, multipartRequest(t, "/api/content/upload", "fixture-author", "image/png", map[string]string{"auto_metadata": "true"}))
	deps.DB.Model(&store.Content{}).Count(&after)
	if w.Code != 500 || before != after {
		t.Fatalf("rollback failed: %d %d %s", before, after, w.Body.String())
	}
}

func TestVisionBrowserFixture(t *testing.T) {
	address := os.Getenv("XQECZ_VISION_FIXTURE_ADDR")
	if address == "" {
		t.Skip("interactive local fixture")
	}
	deps, handler := visionFixture(t)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Messages) > 0 && bytes.Contains(body.Messages[0].Content, []byte("description")) {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"description":"黑色短发，蓝色瞳孔，额前有一缕挑染；使用发型、瞳色与配饰共同判读，服饰可随作品变化。"}`}}}})
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"title\":\"小泉在校园的午后\",\"tags\":[\"小泉\",\"校园\",\"蓝色外套\"],\"characters\":[{\"name\":\"小泉\",\"confidence\":0.95}]}"}}]}`))
	}))
	defer model.Close()
	cfg := vision.DefaultConfig()
	cfg.BaseURL = model.URL + "/v1"
	key := "fixture-only-key"
	if _, err := deps.Vision.SaveSettings(context.Background(), vision.SettingsUpdate{Config: cfg, Key: &key}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	deps.Vision.Start(ctx)
	defer deps.Vision.Close()
	if root := os.Getenv("XQECZ_VISION_FIXTURE_REFS"); root != "" {
		if _, err := vision.ImportReferences(ctx, root, filepath.Join(filepath.Dir(deps.Cfg.UploadDir), "references")); err != nil {
			t.Fatal(err)
		}
	}
	done := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/api/__fixture/session", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: web.SessionCookie, Value: "fixture-admin", Path: "/", HttpOnly: true})
		_, _ = w.Write([]byte("fixture session ready"))
	})
	mux.HandleFunc("/__fixture/stop", func(w http.ResponseWriter, r *http.Request) {
		select {
		case done <- struct{}{}:
		default:
		}
		_, _ = w.Write([]byte("stopping"))
	})
	ln, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	defer srv.Close()
	go func() { _ = srv.Serve(ln) }()
	fmt.Printf("ISOLATED_VISION_FIXTURE http://%s model=%s\n", address, model.URL)
	select {
	case <-done:
	case <-time.After(25 * time.Minute):
		t.Fatal("fixture timeout")
	}
}

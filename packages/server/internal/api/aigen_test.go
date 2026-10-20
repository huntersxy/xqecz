package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/huntersxy/xqecz/server/internal/aigen"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/creative"
	"github.com/huntersxy/xqecz/server/internal/store"
	"github.com/huntersxy/xqecz/server/internal/web"
)

type fixtureImages struct{}

func TestLoginAndContentApprovalCreditRewards(t *testing.T) {
	deps, handler := aiFixture(t)
	account := func() aigen.Account {
		t.Helper()
		var row aigen.Account
		if err := deps.DB.First(&row, "user_id = ?", 2).Error; err != nil {
			t.Fatal(err)
		}
		return row
	}
	for i := 0; i < 2; i++ {
		jsonData(t, aiRequest(t, handler, "GET", "/api/auth/me", "fixture-author", nil))
	}
	if a := account(); a.Available != 21 || a.DailyAvailable != 1 {
		t.Fatal(a)
	}
	jsonData(t, aiRequest(t, handler, "POST", "/api/auth/login", "", map[string]string{"username": "测试作者", "password": "fixture-password"}))
	if a := account(); a.Available != 21 {
		t.Fatal("login duplicated reward", a)
	}
	row := store.Content{Title: "审核奖励", UserID: 2, AuditStatus: "pending", Tags: "[]"}
	if err := deps.DB.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/audit/%d", row.ID)
	for _, status := range []string{"rejected", "approved", "approved", "rejected", "approved"} {
		jsonData(t, aiRequest(t, handler, "POST", path, "fixture-admin", map[string]string{"status": status}))
	}
	if a := account(); a.Available != 22 || a.DailyAvailable != 1 {
		t.Fatal("approval reward must be uncapped and once per work", a)
	}
	broken := store.Content{Title: "rollback", UserID: 2, AuditStatus: "pending", Tags: "[]"}
	deps.DB.Create(&broken)
	deps.DB.Migrator().DropTable(&aigen.Ledger{})
	w := aiRequest(t, handler, "POST", fmt.Sprintf("/api/admin/audit/%d", broken.ID), "fixture-admin", map[string]string{"status": "approved"})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":500`)) {
		t.Fatal(w.Body.String())
	}
	deps.DB.First(&broken, broken.ID)
	if broken.AuditStatus != "pending" {
		t.Fatal("approval committed without reward")
	}
}

func TestAIWorkspaceAndDiagnosticsIsolation(t *testing.T) {
	deps, handler := aiFixture(t)
	var workspace aigen.Session
	for i := 0; i < 2; i++ {
		var row aigen.Session
		json.Unmarshal(jsonData(t, aiRequest(t, handler, "POST", "/api/ai/sessions", "fixture-admin", map[string]any{"name": "workspace"})), &row)
		if i > 0 && row.ID != workspace.ID {
			t.Fatal("created a second workspace")
		}
		workspace = row
	}
	var provider aigen.ProviderConfig
	deps.DB.Where("user_id = 0").First(&provider)
	task, err := deps.AI.QueueTest(context.Background(), 1, provider.ID, true, "diagnostic-isolation-001")
	if err != nil {
		t.Fatal(err)
	}
	if task.SessionID != 0 {
		t.Fatal("diagnostic created a workspace", task.SessionID)
	}
	asset := aigen.Asset{UserID: 1, TaskID: task.ID, Mime: "image/png"}
	deps.DB.Create(&asset)
	var tasks []struct {
		Task aigen.Task `json:"task"`
	}
	json.Unmarshal(jsonData(t, aiRequest(t, handler, "GET", "/api/ai/tasks", "fixture-admin", nil)), &tasks)
	if len(tasks) != 0 {
		t.Fatal("diagnostic leaked into studio", tasks)
	}
	var assets []aigen.Asset
	json.Unmarshal(jsonData(t, aiRequest(t, handler, "GET", "/api/ai/assets", "fixture-admin", nil)), &assets)
	for _, row := range assets {
		if row.ID == asset.ID {
			t.Fatal("diagnostic leaked into library")
		}
	}
}

func (fixtureImages) Generate(ctx context.Context, _ aigen.ProviderConfig, _ string, _ aigen.Input, _ []string) ([]aigen.Image, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(80 * time.Millisecond):
	}
	return []aigen.Image{{Bytes: fixturePNG()}}, nil
}
func fixturePNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := 0; y < 512; y++ {
		for x := 0; x < 512; x++ {
			pixel := color.RGBA{255, 250, 252, 255}
			if (x-256)*(x-256)+(y-256)*(y-256) < 160*160 {
				pixel = color.RGBA{219, 128, 156, 255}
			}
			img.SetRGBA(x, y, pixel)
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, img)
	return out.Bytes()
}
func aiFixture(t *testing.T) (app.Deps, http.Handler) {
	t.Helper()
	deps, _ := visionFixture(t)
	deps.Cfg.AIImageDir = filepath.Join(filepath.Dir(deps.Cfg.UploadDir), "ai-private")
	s := aigen.NewWithProvider(deps.DB, deps.Cfg, fixtureImages{})
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	deps.AI = s
	if err := creative.Init(context.Background(), deps.DB); err != nil {
		t.Fatal(err)
	}
	p, err := s.SaveProvider(context.Background(), 0, aigen.ProviderConfig{Name: "隔离测试供应商", Endpoint: "https://api.slomerex.xyz/v1/images/generations", Model: "fixture-image-model", Enabled: true, Default: true, Sizes: `["1024x1024"]`, Qualities: `[]`, MaxCount: 1, MaxReferences: 2, EditTransport: "multipart", TimeoutSeconds: 10, Concurrency: 1, Cost: 2, QualityCosts: `{}`}, "fixture-only-secret")
	if err != nil {
		t.Fatal(err)
	}
	for _, uid := range []uint64{1, 2} {
		if err = s.Grant(context.Background(), 1, uid, 20, fmt.Sprintf("fixture-grant-0000%d", uid), "isolated fixture"); err != nil {
			t.Fatal(err)
		}
	}
	asset, err := s.Upload(context.Background(), 1, bytes.NewReader(fixturePNG()))
	if err != nil {
		t.Fatal(err)
	}
	if err = deps.DB.Model(&asset).Updates(map[string]any{"public": true, "title": "标准设定图", "purpose": "标准设定", "license": "测试素材，仅用于本地验收", "download_allowed": true}).Error; err != nil {
		t.Fatal(err)
	}
	character := creative.Resource{Kind: "character", Slug: "fixture-character", Title: "小希", Summary: "本地隔离验收角色", Status: "published", Version: 1, Payload: fmt.Sprintf(`{"description":"角色设定介绍","appearance":"粉色圆形测试标记","asset_ids":[%d],"character_ids":[],"tags":[],"variables":[],"preset_id":0,"parameters":{}}`, asset.ID)}
	deps.DB.Create(&character)
	template := creative.Resource{Kind: "template", Slug: "fixture-scene", Title: "角色场景模板", Summary: "填写角色和场景", Status: "published", Version: 1, Payload: fmt.Sprintf(`{"body":"保持{角色}的形象，位于{场景}。","asset_ids":[%d],"character_ids":[%d],"tags":[],"variables":[{"name":"角色","label":"角色名称","type":"text","default":"小希","required":true,"options":[],"help":""},{"name":"场景","label":"场景","type":"select","default":"公园","required":true,"options":["公园","学校"],"help":""}],"preset_id":0,"parameters":{}}`, asset.ID, character.ID)}
	deps.DB.Create(&template)
	_, err = s.CreateIdea(context.Background(), 2, aigen.IdeaInput{Title: "公园里的午后", Description: "一起实现这份灵感", Input: aigen.Input{Prompt: "A calm afternoon", ProviderID: p.ID, Mode: "site", Size: "1024x1024", Count: 1, References: []aigen.Reference{{AssetID: asset.ID}}}, CharacterIDs: []uint64{character.ID}, PublicConsent: true, ReferenceRights: true})
	if err != nil {
		t.Fatal(err)
	}
	return deps, New(deps)
}
func aiRequest(t *testing.T, handler http.Handler, method, path, session string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var data bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&data).Encode(body)
	}
	req := httptest.NewRequest(method, path, &data)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: session})
	}
	out := httptest.NewRecorder()
	handler.ServeHTTP(out, req)
	return out
}
func jsonData(t *testing.T, w *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var res struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(w.Body.Bytes(), &res) != nil || res.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	return res.Data
}
func TestAIHTTPPrivacyAuthorizationAndTimeContract(t *testing.T) {
	deps, handler := aiFixture(t)
	private, err := deps.AI.Upload(context.Background(), 2, bytes.NewReader(fixturePNG()))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, session string
		want          int
	}{{fmt.Sprintf("/api/ai/assets/%d", private.ID), "", 401}, {fmt.Sprintf("/api/creative/assets/%d", private.ID), "", 404}, {fmt.Sprintf("/api/ai/assets/%d", private.ID), "fixture-author", 200}, {fmt.Sprintf("/api/admin/ai/providers"), "fixture-author", 403}} {
		w := aiRequest(t, handler, "GET", tc.path, tc.session, nil)
		if w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	if w := aiRequest(t, handler, "GET", "/uploads/"+private.FileName, "", nil); w.Code != 404 {
		t.Fatal("private file mounted publicly")
	}
	w := aiRequest(t, handler, "GET", "/api/admin/ai/providers", "fixture-admin", nil)
	if bytes.Contains(w.Body.Bytes(), []byte("fixture-only-secret")) || bytes.Contains(w.Body.Bytes(), []byte("cipher_key")) {
		t.Fatal("key disclosed")
	}
	sessionRaw := jsonData(t, aiRequest(t, handler, "POST", "/api/ai/sessions", "fixture-author", map[string]any{"name": "saved session"}))
	var session aigen.Session
	_ = json.Unmarshal(sessionRaw, &session)
	if !regexp.MustCompile(`\.\d{3}Z"`).Match(sessionRaw) {
		t.Fatal("nonstandard timestamp", string(sessionRaw))
	}
	jsonData(t, aiRequest(t, handler, "PATCH", fmt.Sprintf("/api/ai/sessions/%d", session.ID), "fixture-author", map[string]any{"name": "saved session", "draft": map[string]any{"prompt": "private draft"}, "version": 1}))
	w = aiRequest(t, handler, "PATCH", fmt.Sprintf("/api/ai/sessions/%d", session.ID), "fixture-author", map[string]any{"name": "stale", "draft": map[string]any{}, "version": 1})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":400`)) {
		t.Fatal(w.Body.String())
	}
}
func TestAIHTTPPublishDedupAndIdeaVisibility(t *testing.T) {
	deps, handler := aiFixture(t)
	asset, err := deps.AI.Upload(context.Background(), 2, bytes.NewReader(fixturePNG()))
	if err != nil {
		t.Fatal(err)
	}
	deps.DB.Model(&asset).Update("task_id", 999)
	body := map[string]any{"title": "generated work", "description": "description", "public_consent": true}
	var first uint64
	for i := 0; i < 2; i++ {
		raw := jsonData(t, aiRequest(t, handler, "POST", fmt.Sprintf("/api/ai/assets/%d/publish", asset.ID), "fixture-author", body))
		var row struct {
			ContentID uint64 `json:"content_id"`
		}
		_ = json.Unmarshal(raw, &row)
		if i == 0 {
			first = row.ContentID
		} else if row.ContentID != first {
			t.Fatal("duplicate post")
		}
	}
	var count int64
	deps.DB.Model(&store.Content{}).Where("title = ?", "generated work").Count(&count)
	if count != 1 {
		t.Fatal(count)
	}
	var idea aigen.Idea
	deps.DB.First(&idea)
	jsonData(t, aiRequest(t, handler, "GET", fmt.Sprintf("/api/ideas/%d", idea.ID), "", nil))
	jsonData(t, aiRequest(t, handler, "POST", fmt.Sprintf("/api/admin/ai/ideas/%d/hide", idea.ID), "fixture-admin", map[string]string{"note": "test moderation"}))
	if w := aiRequest(t, handler, "GET", fmt.Sprintf("/api/ideas/%d", idea.ID), "", nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
	var input aigen.Input
	_ = json.Unmarshal([]byte(idea.Snapshot), &input)
	if w := aiRequest(t, handler, "GET", fmt.Sprintf("/api/ideas/%d/assets/%d", idea.ID, input.References[0].AssetID), "", nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestAIResolutionNoteFeedback(t *testing.T) {
	deps, handler := aiFixture(t)
	task := aigen.Task{UserID: 2, Status: "unknown", Cost: 2, Snapshot: `{}`}
	if err := deps.DB.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := deps.DB.Model(&aigen.Account{}).Where("user_id = ?", 2).Updates(map[string]any{"available": 18, "reserved": 2}).Error; err != nil {
		t.Fatal(err)
	}
	path := fmt.Sprintf("/api/admin/ai/tasks/%d/resolve", task.ID)
	for _, note := range []string{"", "  ", "确认", strings.Repeat("核", 301)} {
		response := aiRequest(t, handler, "POST", path, "fixture-admin", map[string]any{"consumed": false, "note": note})
		if response.Code != 200 || !bytes.Contains(response.Body.Bytes(), []byte(`"code":400`)) || !bytes.Contains(response.Body.Bytes(), []byte("核查依据与说明须填写 3–300 个字")) {
			t.Fatal("resolution note validation was not actionable", response.Body.String())
		}
	}
	var account aigen.Account
	deps.DB.First(&account, "user_id = ?", 2)
	deps.DB.First(&task, task.ID)
	if account.Reserved != 2 || account.Available != 18 || task.Status != "unknown" {
		t.Fatal("invalid note changed settlement", account, task)
	}
	jsonData(t, aiRequest(t, handler, "POST", path, "fixture-admin", map[string]any{"consumed": false, "note": "  供应商确认未执行  "}))
	deps.DB.First(&account, "user_id = ?", 2)
	deps.DB.First(&task, task.ID)
	if account.Reserved != 0 || account.Available != 20 || task.Status != "failed" || task.Error != "供应商确认未执行" {
		t.Fatal("valid note did not release quota and retain evidence", account, task)
	}
}

func TestAIBrowserFixture(t *testing.T) {
	address := os.Getenv("XQECZ_AI_FIXTURE_ADDR")
	if address == "" {
		t.Skip("interactive local fixture")
	}
	deps, handler := aiFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	deps.AI.Start(ctx)
	defer func() { cancel(); deps.AI.Close() }()
	done := make(chan struct{}, 1)
	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/api/__fixture/session", func(w http.ResponseWriter, r *http.Request) {
		sid := "fixture-admin"
		if r.URL.Query().Get("user") == "author" {
			sid = "fixture-author"
		}
		http.SetCookie(w, &http.Cookie{Name: web.SessionCookie, Value: sid, Path: "/", HttpOnly: true})
		_, _ = w.Write([]byte("isolated AI fixture session"))
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
	fmt.Printf("ISOLATED_AI_FIXTURE http://%s\n", address)
	select {
	case <-done:
	case <-time.After(45 * time.Minute):
		t.Fatal("fixture timeout")
	}
}

func TestCreativeResourceAdministrationContract(t *testing.T) {
	_, handler := aiFixture(t)
	body := map[string]any{"kind": "template", "slug": "fixed-prompt", "title": "Fixed prompt", "summary": "test", "status": "draft", "sort_order": 1, "version": 0, "payload": map[string]any{"body": "A peaceful park", "asset_ids": []uint64{}, "variables": []any{}, "parameters": map[string]any{}}}
	raw := jsonData(t, aiRequest(t, handler, "POST", "/api/admin/creative/resources", "fixture-admin", body))
	var row creative.Resource
	_ = json.Unmarshal(raw, &row)
	if w := aiRequest(t, handler, "GET", fmt.Sprintf("/api/creative/resources/%d", row.ID), "", nil); w.Code != 404 {
		t.Fatal("draft leaked")
	}
	body["version"] = 1
	body["status"] = "published"
	jsonData(t, aiRequest(t, handler, "PUT", fmt.Sprintf("/api/admin/creative/resources/%d", row.ID), "fixture-admin", body))
	jsonData(t, aiRequest(t, handler, "GET", fmt.Sprintf("/api/creative/resources/%d", row.ID), "", nil))
	if w := aiRequest(t, handler, "GET", "/api/creative/resources?kind=template", "", nil); strings.Contains(w.Body.String(), "fixture-only-secret") {
		t.Fatal("secret leaked")
	}
}

func TestAIHTTPPrivacyFiltersAndPagination(t *testing.T) {
	deps, handler := aiFixture(t)
	ctx := context.Background()
	private, err := deps.AI.Upload(ctx, 2, bytes.NewReader(fixturePNG()))
	if err != nil {
		t.Fatal(err)
	}
	w := aiRequest(t, handler, "PATCH", fmt.Sprintf("/api/admin/ai/assets/%d", private.ID), "fixture-admin", map[string]any{"public": true, "license": "test consent"})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":400`)) {
		t.Fatal("administrator exposed another user's private image", w.Body.String())
	}
	var idea aigen.Idea
	deps.DB.First(&idea)
	deps.DB.Create(&aigen.IdeaEvent{IdeaID: idea.ID, Action: "billing-note", Note: "private supplier investigation"})
	if raw := jsonData(t, aiRequest(t, handler, "GET", fmt.Sprintf("/api/ideas/%d", idea.ID), "", nil)); bytes.Contains(raw, []byte("private supplier investigation")) {
		t.Fatal("moderation note disclosed")
	}
	jsonData(t, aiRequest(t, handler, "GET", fmt.Sprintf("/api/admin/ai/ideas/%d/events", idea.ID), "fixture-admin", nil))
	session := aigen.Session{UserID: 2, Name: "private task", Draft: `{}`, Version: 1}
	deps.DB.Create(&session)
	task, err := deps.AI.Submit(ctx, 2, aigen.Input{SessionID: session.ID, Key: "private-task-key-01", Prompt: "private prompt", ProviderID: 1, Mode: "site", Size: "1024x1024", Count: 1, ExpectedCost: 2, References: []aigen.Reference{{AssetID: private.ID, Note: "reference note"}}})
	if err != nil {
		t.Fatal(err)
	}
	rawTasks := jsonData(t, aiRequest(t, handler, "GET", "/api/admin/ai/tasks", "fixture-admin", nil))
	if bytes.Contains(rawTasks, []byte("cipher_key")) || bytes.Contains(rawTasks, []byte("fixture-only-secret")) || bytes.Contains(rawTasks, []byte(private.FileName)) {
		t.Fatal("private task snapshot key or filename disclosed")
	}
	var taskViews []struct {
		References []aigen.Asset `json:"references"`
		User       struct {
			ID       uint64 `json:"id"`
			Username string `json:"username"`
		} `json:"user"`
	}
	if json.Unmarshal(rawTasks, &taskViews) != nil || len(taskViews) != 1 || len(taskViews[0].References) != 1 || taskViews[0].References[0].ID != private.ID {
		t.Fatal("task reference metadata unavailable", string(rawTasks))
	}
	if taskViews[0].User.ID != 2 || taskViews[0].User.Username != "测试作者" || bytes.Contains(rawTasks, []byte("fixture@example.com")) {
		t.Fatal("task owner display name missing or private account data disclosed", string(rawTasks))
	}
	if err = deps.AI.Cancel(ctx, 2, task.ID); err != nil {
		t.Fatal(err)
	}
	deps.DB.Create(&aigen.Idea{UserID: 2, Title: "Hidden", Status: "hidden", CharacterIDs: "[1]", Snapshot: idea.Snapshot})
	deps.DB.Create(&aigen.Idea{UserID: 2, Title: "Eleven", Status: "waiting", CharacterIDs: "[11]", Snapshot: idea.Snapshot})
	var views []struct {
		Idea aigen.Idea `json:"idea"`
	}
	json.Unmarshal(jsonData(t, aiRequest(t, handler, "GET", "/api/ideas?character=1", "", nil)), &views)
	if len(views) != 1 || views[0].Idea.ID != idea.ID {
		t.Fatal("character filter exposed hidden or partial ID matches", views)
	}
	rows := make([]aigen.Session, 101)
	for i := range rows {
		rows[i] = aigen.Session{UserID: 2, Name: fmt.Sprintf("session %d", i), Draft: `{}`, Version: 1}
	}
	if err = deps.DB.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	var page []aigen.Session
	json.Unmarshal(jsonData(t, aiRequest(t, handler, "GET", "/api/ai/sessions", "fixture-author", nil)), &page)
	if len(page) != 1 || page[0].ID != rows[100].ID {
		t.Fatal(page)
	}
	var template creative.Resource
	deps.DB.Where("kind = ?", "template").First(&template)
	w = aiRequest(t, handler, "POST", fmt.Sprintf("/api/creative/templates/%d/expand", template.ID), "", map[string]any{"version": 0, "values": map[string]string{"角色": "小希", "场景": "公园"}})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":409`)) {
		t.Fatal("stale template accepted", w.Body.String())
	}
	previewPath, err := deps.AI.AssetPath(aigen.Asset{FileName: private.PreviewFileName})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(previewPath); err != nil {
		t.Fatal(err)
	}
	w = aiRequest(t, handler, "GET", fmt.Sprintf("/api/ai/assets/%d?preview=1", private.ID), "fixture-author", nil)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatal(w.Code, w.Header())
	}
}

func TestAIHTTPAssetPublicationValidation(t *testing.T) {
	deps, handler := aiFixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		name, license, message string
		owner, task            uint64
	}{
		{"missing license", "", "公开素材前请填写来源与授权说明", 1, 0},
		{"blank license", " \n\t", "公开素材前请填写来源与授权说明", 1, 0},
		{"another owner", "授权用于二创", "只能将自己上传的设定图设为公开素材", 2, 0},
		{"generated result", "授权用于二创", "生成结果不能设为公开素材，请在 AI 工作台发布到站内", 1, 42},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset, err := deps.AI.Upload(ctx, tc.owner, bytes.NewReader(fixturePNG()))
			if err != nil {
				t.Fatal(err)
			}
			if err := deps.DB.Model(&asset).Update("task_id", tc.task).Error; err != nil {
				t.Fatal(err)
			}
			w := aiRequest(t, handler, "PATCH", fmt.Sprintf("/api/admin/ai/assets/%d", asset.ID), "fixture-admin", map[string]any{"public": true, "license": tc.license})
			var response struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || response.Code != 400 || response.Message != tc.message {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
			}
			if err := deps.DB.First(&asset, asset.ID).Error; err != nil {
				t.Fatal(err)
			}
			if asset.Public {
				t.Fatal("invalid publication disclosed asset")
			}
			if w := aiRequest(t, handler, "GET", fmt.Sprintf("/api/creative/assets/%d", asset.ID), "", nil); w.Code != 404 {
				t.Fatal("private asset became anonymously readable", w.Code)
			}
		})
	}
	asset, err := deps.AI.Upload(ctx, 1, bytes.NewReader(fixturePNG()))
	if err != nil {
		t.Fatal(err)
	}
	jsonData(t, aiRequest(t, handler, "PATCH", fmt.Sprintf("/api/admin/ai/assets/%d", asset.ID), "fixture-admin", map[string]any{"title": "公开设定图", "public": true, "license": "原创素材，授权用于二创", "download_allowed": true}))
	if err := deps.DB.First(&asset, asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !asset.Public || asset.License != "原创素材，授权用于二创" {
		t.Fatal("valid metadata was not persisted", asset)
	}
	if w := aiRequest(t, handler, "GET", fmt.Sprintf("/api/creative/assets/%d", asset.ID), "", nil); w.Code != 200 {
		t.Fatal("published asset unavailable", w.Code)
	}
}

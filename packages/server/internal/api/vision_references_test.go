package api

import (
	"bytes"
	"encoding/json"
	"github.com/huntersxy/xqecz/server/internal/vision"
	"github.com/huntersxy/xqecz/server/internal/web"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminReferenceRoutes(t *testing.T) {
	_, handler := visionFixture(t)
	const root = "/api/admin/vision/references"
	for _, session := range []string{"", "fixture-author"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, multipartRequest(t, root, session, "image/png", map[string]string{"name": "角色甲"}))
		if w.Code != 401 && w.Code != 403 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	upload := func(method, path, mime string) (int, vision.Reference) {
		req := multipartRequest(t, path, "fixture-admin", mime, map[string]string{"name": "角色甲"})
		req.Method = method
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		var res struct {
			Code int              `json:"code"`
			Data vision.Reference `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			t.Fatal(w.Body.String())
		}
		return res.Code, res.Data
	}
	code, ref := upload("POST", root, "image/png")
	if code != 200 || ref.Version == "" {
		t.Fatal(code, ref)
	}
	if code, _ := upload("POST", root, "image/png"); code != 400 {
		t.Fatal(code)
	}
	call := func(method, path, session string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if session != "" {
			req.AddCookie(&http.Cookie{Name: web.SessionCookie, Value: session})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	preview := call("GET", root+"/角色甲/image", "fixture-admin", nil)
	if preview.Code != 200 || preview.Header().Get("Content-Type") != "image/jpeg" || preview.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(preview.Code, preview.Header())
	}
	for _, path := range []string{root + "/角色甲/image", root + "/角色甲/description", root + "/角色甲/describe", root + "/角色甲"} {
		for _, method := range []string{"GET", "PUT", "POST", "PATCH", "DELETE"} {
			w := call(method, path, "fixture-author", nil)
			if w.Code == 200 {
				t.Fatal(method, path, w.Body.String())
			}
		}
	}
	w := call("PUT", root+"/角色甲/description", "fixture-admin", map[string]string{"description": "蓝色短发", "version": ref.Version})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":200`)) {
		t.Fatal(w.Body.String())
	}
	w = call("PATCH", root+"/角色甲", "fixture-admin", map[string]string{"name": "角色乙"})
	if !bytes.Contains(w.Body.Bytes(), []byte(`"code":200`)) {
		t.Fatal(w.Body.String())
	}
	if code, _ := upload("PUT", root+"/角色乙/image", "video/mp4"); code != 400 {
		t.Fatal(code)
	}
	if w = call("GET", root+"/角色乙/image", "fixture-admin", nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = call("DELETE", root+"/角色乙", "fixture-admin", nil); !bytes.Contains(w.Body.Bytes(), []byte(`"code":200`)) {
		t.Fatal(w.Body.String())
	}
	if w = call("GET", root+"/角色乙/image", "fixture-admin", nil); w.Code != 404 {
		t.Fatal(w.Code)
	}
}

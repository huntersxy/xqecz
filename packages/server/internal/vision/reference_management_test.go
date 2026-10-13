package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func referencePNG(t *testing.T, red uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1200, 20))
	img.Set(0, 0, color.RGBA{R: red, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestReferenceManagementVersionAndRecovery(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	ref, err := s.PutReference(ctx, "小泉", bytes.NewReader(referencePNG(t, 255)), false)
	if err != nil || !ref.NeedsDescription || ref.Version == "" || ref.Bytes > 512<<10 {
		t.Fatal(ref, err)
	}
	if _, err = s.PutReference(ctx, "小泉", bytes.NewReader(referencePNG(t, 1)), false); err != ErrReferenceExists {
		t.Fatal(err)
	}
	for _, name := range []string{"../bad", "CON", "小泉/abc", "小泉.", ".notes"} {
		if _, err = s.PutReference(ctx, name, bytes.NewReader(referencePNG(t, 1)), false); err == nil {
			t.Fatal(name)
		}
	}
	ready, err := s.SaveReferenceDescription(ctx, ref.Name, "黑色短发，蓝色瞳孔，星形发饰", ref.Version)
	if err != nil || ready.NeedsDescription {
		t.Fatal(ready, err)
	}
	if _, err = s.PutReference(ctx, ref.Name, strings.NewReader("not image"), true); err == nil {
		t.Fatal("invalid replacement accepted")
	}
	unchanged, _ := s.References(ctx)
	if unchanged[0].Version != ref.Version || unchanged[0].NeedsDescription {
		t.Fatal(unchanged)
	}
	renamed, err := s.RenameReference(ctx, ref.Name, "角色甲")
	if err != nil || renamed.NeedsDescription || renamed.Description != ready.Description {
		t.Fatal(renamed, err)
	}
	replaced, err := s.PutReference(ctx, "角色甲", bytes.NewReader(referencePNG(t, 12)), true)
	if err != nil || !replaced.NeedsDescription || replaced.Version == ref.Version {
		t.Fatal(replaced, err)
	}
	if _, err = s.SaveReferenceDescription(ctx, "角色甲", "old result", ref.Version); err == nil {
		t.Fatal("stale description accepted")
	}
	if err = s.DeleteReference(ctx, "角色甲"); err != nil {
		t.Fatal(err)
	}
	refs, err := s.References(ctx)
	if err != nil || len(refs) != 0 {
		t.Fatal(refs, err)
	}
	archived, _ := os.ReadDir(filepath.Join(s.referenceDir, ".archive"))
	if len(archived) != 2 {
		t.Fatal(archived)
	}
	entries, _ := os.ReadDir(s.referenceDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".incoming-") || strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Fatal(entry.Name())
		}
	}
}
func TestTextReferenceModeSendsOnlyTarget(t *testing.T) {
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !strings.Contains(string(body.Messages[0].Content), "星形发饰") {
			t.Error("missing textual reference")
		}
		var parts []struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(body.Messages[1].Content, &parts)
		images := 0
		for _, part := range parts {
			if part.Type == "image_url" {
				images++
			}
		}
		if images != 1 || strings.Contains(string(body.Messages[1].Content), "private-reference") {
			t.Error(string(body.Messages[1].Content))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"title":"小泉在窗边","tags":["窗边","小泉"],"characters":[{"name":"小泉","confidence":0.95}]}`}}}})
	}))
	defer provider.Close()
	cfg := DefaultConfig()
	cfg.BaseURL = provider.URL
	cfg.APIKey = "test"
	cfg.ReferenceMode = "text"
	ref := Reference{Name: "小泉", Description: "黑色短发，星形发饰", DataURL: "private-reference"}
	result, err := Generate(context.Background(), cfg, "target-image", nil, ref)
	if err != nil || result.Title != "小泉在窗边" {
		t.Fatal(result, err)
	}
	for _, bad := range []Reference{{Name: "角色甲"}, {Name: "角色甲", Description: "旧特征", NeedsDescription: true}} {
		if _, err := Generate(context.Background(), cfg, "target-image", nil, bad); err == nil {
			t.Fatal("missing/stale reference accepted")
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
func TestDescribeReferencePersistsAndRejectsChangedImage(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	ref, err := s.PutReference(ctx, "小泉", bytes.NewReader(referencePNG(t, 255)), false)
	if err != nil {
		t.Fatal(err)
	}
	replaceDuringCall := false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if !strings.Contains(stringMustJSON(body), "description") {
			t.Error("missing schema contract")
		}
		if replaceDuringCall {
			if _, err := s.PutReference(ctx, "小泉", bytes.NewReader(referencePNG(t, 12)), true); err != nil {
				t.Error(err)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": "```json\n{\"description\":\"黑色短发、蓝色瞳孔\"}\n```"}}}})
	}))
	defer provider.Close()
	configure(t, s, provider.URL)
	described, err := s.DescribeReference(ctx, ref.Name, ref.Version)
	if err != nil || described.NeedsDescription || described.Description != "黑色短发、蓝色瞳孔" {
		t.Fatal(described, err)
	}
	replaceDuringCall = true
	if _, err = s.DescribeReference(ctx, ref.Name, ref.Version); err == nil {
		t.Fatal("stale generated features saved")
	}
}
func stringMustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestReferenceUploadBoundsAndCancellation(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	if _, err := s.PutReference(ctx, "大图", strings.NewReader(strings.Repeat("x", maxReferenceUpload+1)), false); err == nil {
		t.Fatal("oversized reference accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.PutReference(cancelled, "取消", bytes.NewReader(referencePNG(t, 255)), false); err == nil {
		t.Fatal("cancelled upload accepted")
	}
	img := referencePNG(t, 255)
	for i := range 12 {
		if _, err := s.PutReference(ctx, fmt.Sprintf("角色%d", i), bytes.NewReader(img), false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.PutReference(ctx, "第十三个", bytes.NewReader(img), false); err == nil {
		t.Fatal("library limit bypassed")
	}
	entries, _ := os.ReadDir(s.referenceDir)
	if len(entries) != 12 {
		t.Fatal("temporary files leaked", entries)
	}
}

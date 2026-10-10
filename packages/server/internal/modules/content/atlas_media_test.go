package content

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/huntersxy/xqecz/server/internal/app"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/media"
)

func TestServeAtlasVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	source := filepath.Join(dir, "a.webp")
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	img.Set(0, 0, color.White)
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	h := &Handler{deps: app.Deps{Cfg: config.Config{ThumbDir: dir}}}
	r := gin.New()
	r.GET("/thumbs/*filepath", func(c *gin.Context) { h.serveMedia(c, dir, true) })
	for _, variant := range []string{media.AtlasDirectory, media.MontageDirectory} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/thumbs/"+variant+"/a.webp?texture_origin=https%3A%2F%2Fxq.xiey.work", nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", variant, w.Code)
		}
		if w.Header().Get("Cache-Control") == "" {
			t.Fatal("cache header missing")
		}
		decoded, _, err := image.Decode(w.Body)
		if err != nil {
			t.Fatal(err)
		}
		expected := media.AtlasMaxEdge
		if variant == media.MontageDirectory {
			expected = media.MontageMaxEdge
		}
		if decoded.Bounds().Dx() != expected {
			t.Fatal(decoded.Bounds())
		}
	}
	for _, path := range []string{"/thumbs/atlas-v1/missing.webp", "/thumbs/atlas-v1/nested/a.webp", "/thumbs/atlas-v1/a.png"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatalf("%s: %d", path, w.Code)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/thumbs/atlas-v1/a.webp", nil).WithContext(ctx))
	if w.Code != 404 {
		t.Fatal("cancelled generation must not write")
	}
}

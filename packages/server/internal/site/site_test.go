package site

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/migrate"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testService(t *testing.T) (*Service, config.Config) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "site.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	service := New(db, cfg)
	if err := service.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	return service, cfg
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 40), G: uint8(y * 60), B: 120, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDefaultsResolveTextPlaceholders(t *testing.T) {
	cfg := PublicDefaults()
	if cfg.Name == "" || cfg.ShortName == "" {
		t.Fatal("default site name is empty")
	}
	if !strings.Contains(cfg.BrowserTitle, cfg.Name) {
		t.Fatalf("browser title %q does not include site name", cfg.BrowserTitle)
	}
	if got := cfg.Texts["login.storyTitle"]; !strings.Contains(got, cfg.Name) || strings.Contains(got, "{name}") {
		t.Fatalf("story title was not resolved: %q", got)
	}
	if got := cfg.Texts["characters.detailMeta"]; !strings.Contains(got, "{title}") {
		t.Fatalf("page-level placeholder must stay for the caller: %q", got)
	}
}

func TestSaveMergesOverridesAndIgnoresUnknownTexts(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	defaults := Defaults()

	in := defaults
	in.Name = "  测试宇宙  "
	in.ShortName = ""
	in.Tagline = "自定义画廊"
	in.LogoURL = "/images/site/logo.webp"
	in.Texts = map[string]string{
		"nav.gallery":     "  作品集  ",
		"login.note":      defaults.Texts["login.note"],
		"unknown.element": "应被忽略",
		"home.empty":      "",
	}
	in.FriendLinks = []FriendLink{{Name: "友站", URL: "https://example.com/", IconURL: "/friend-links/a.webp"}}

	saved, err := service.Save(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Name != "测试宇宙" {
		t.Fatalf("name = %q", saved.Name)
	}
	if saved.ShortName != "测试宇宙" {
		t.Fatalf("short name fallback = %q", saved.ShortName)
	}
	if got := saved.Texts["nav.gallery"]; got != "作品集" {
		t.Fatalf("custom text = %q", got)
	}
	if got := saved.Texts["login.note"]; got != defaults.Texts["login.note"] {
		t.Fatalf("default text should be restored from catalog: %q", got)
	}
	if _, ok := saved.Texts["unknown.element"]; ok {
		t.Fatal("unknown text key must be dropped")
	}
	if len(saved.FriendLinks) != 1 || saved.FriendLinks[0].Name != "友站" {
		t.Fatalf("friend links = %#v", saved.FriendLinks)
	}

	public := service.Public(ctx)
	if public.Name != "测试宇宙" || !strings.Contains(public.Texts["login.submitLogin"], "测试宇宙") {
		t.Fatalf("public config not resolved: %#v", public.Texts["login.submitLogin"])
	}
}

func TestSaveAllowsClearingTaglineAndDescription(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	cfg := Defaults()
	cfg.Tagline = ""
	cfg.Description = ""
	saved, err := service.Save(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Tagline != "" || saved.Description != "" {
		t.Fatalf("tagline=%q description=%q", saved.Tagline, saved.Description)
	}
	if got := service.Public(ctx); got.Tagline != "" || got.Description != "" {
		t.Fatalf("public tagline=%q description=%q", got.Tagline, got.Description)
	}
}

func TestSaveRejectsInvalidConfig(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	cases := []struct {
		name  string
		apply func(*Config)
	}{
		{"empty name", func(c *Config) { c.Name = " " }},
		{"long tagline", func(c *Config) { c.Tagline = strings.Repeat("长", 41) }},
		{"bad logo", func(c *Config) { c.LogoURL = "javascript:alert(1)" }},
		{"bad friend url", func(c *Config) { c.FriendLinks = []FriendLink{{Name: "友站", URL: "ftp://example.com"}} }},
		{"too many friends", func(c *Config) {
			for i := 0; i < maxFriendLinks+1; i++ {
				c.FriendLinks = append(c.FriendLinks, FriendLink{Name: "友站", URL: "https://example.com"})
			}
		}},
		{"long text", func(c *Config) { c.Texts = map[string]string{"home.featuredTitle": strings.Repeat("字", 41)} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Defaults()
			tc.apply(&cfg)
			if _, err := service.Save(ctx, cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestSaveAssetStoresLogoAndSquareFavicon(t *testing.T) {
	service, cfg := testService(t)
	pngBytes := testPNG(t, 8, 4)

	logoURL, err := service.SaveAsset(AssetLogo, bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(logoURL, "/images/site/logo-") || !strings.HasSuffix(logoURL, ".webp") {
		t.Fatalf("logo url = %q", logoURL)
	}
	logoPath := filepath.Join(cfg.ImagesDir, "site", strings.TrimPrefix(logoURL, "/images/site/"))
	data, err := os.ReadFile(logoPath)
	if err != nil || len(data) == 0 {
		t.Fatalf("logo file missing: %v", err)
	}
	if _, format, err := image.Decode(bytes.NewReader(data)); err != nil || format != "webp" {
		t.Fatalf("logo format = %q err=%v", format, err)
	}

	iconURL, err := service.SaveAsset(AssetFavicon, bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatal(err)
	}
	iconPath := filepath.Join(cfg.ImagesDir, "site", strings.TrimPrefix(iconURL, "/images/site/"))
	iconData, err := os.ReadFile(iconPath)
	if err != nil {
		t.Fatal(err)
	}
	icon, format, err := image.Decode(bytes.NewReader(iconData))
	if err != nil || format != "png" {
		t.Fatalf("favicon format = %q err=%v", format, err)
	}
	if b := icon.Bounds(); b.Dx() != faviconSize || b.Dy() != faviconSize {
		t.Fatalf("favicon size = %dx%d", b.Dx(), b.Dy())
	}
}

func TestSaveAssetRejectsUnsupportedInput(t *testing.T) {
	service, _ := testService(t)
	if _, err := service.SaveAsset("banner", bytes.NewReader(testPNG(t, 2, 2))); err == nil {
		t.Fatal("unknown kind must fail")
	}
	if _, err := service.SaveAsset(AssetLogo, strings.NewReader("not an image")); err == nil {
		t.Fatal("non-image must fail")
	}
	if _, err := service.SaveAsset(AssetLogo, bytes.NewReader(nil)); err == nil {
		t.Fatal("empty input must fail")
	}
}

func TestSaveRemovesReplacedAsset(t *testing.T) {
	service, cfg := testService(t)
	ctx := context.Background()

	first, err := service.SaveAsset(AssetLogo, bytes.NewReader(testPNG(t, 8, 8)))
	if err != nil {
		t.Fatal(err)
	}
	cfgWithFirst := Defaults()
	cfgWithFirst.LogoURL = first
	if _, err := service.Save(ctx, cfgWithFirst); err != nil {
		t.Fatal(err)
	}
	firstPath := filepath.Join(cfg.ImagesDir, "site", strings.TrimPrefix(first, "/images/site/"))
	if _, err := os.Stat(firstPath); err != nil {
		t.Fatalf("first asset missing: %v", err)
	}
	savedFirst, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if savedFirst.LogoURL != first {
		t.Fatalf("saved logo = %q, want %q", savedFirst.LogoURL, first)
	}

	second, err := service.SaveAsset(AssetLogo, bytes.NewReader(testPNG(t, 6, 6)))
	if err != nil {
		t.Fatal(err)
	}
	cfgWithSecond := Defaults()
	cfgWithSecond.LogoURL = second
	if _, err := service.Save(ctx, cfgWithSecond); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("replaced asset should be removed, stat err = %v", err)
	}
}

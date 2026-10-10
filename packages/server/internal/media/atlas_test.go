package media

import (
	"bytes"
	"context"
	"image"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/deepteams/webp"
)

func TestAtlasDimensionsSizeAndReuse(t *testing.T) {
	dir := t.TempDir()
	name := "sample_thumb.webp"
	// 内容是高频彩色细节，验证真实编码体积而非只检查扩展名。
	source := filepath.Join(dir, name)
	writeSourcePNG(t, source, 800, 1200)
	before, _ := os.Stat(source)
	rel, err := EnsureAtlasThumbnail(context.Background(), dir, name)
	if err != nil {
		t.Fatal(err)
	}
	if rel != "thumbs/atlas-v1/"+name {
		t.Fatal(rel)
	}
	output := filepath.Join(dir, AtlasDirectory, name)
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 171 || img.Bounds().Dy() != 256 {
		t.Fatal(img.Bounds())
	}
	if int64(len(data)) >= before.Size()/2 {
		t.Fatalf("atlas %d bytes, source %d", len(data), before.Size())
	}
	st, _ := os.Stat(output)
	if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err != nil {
		t.Fatal(err)
	}
	cached, _ := os.Stat(output)
	if !cached.ModTime().Equal(st.ModTime()) {
		t.Fatal("cache regenerated")
	}
	t.Logf("source=%d bytes atlas=%d bytes dimensions=%v", before.Size(), len(data), img.Bounds())
}

func TestAtlasDoesNotUpscaleAndRejectsUnsafeNames(t *testing.T) {
	dir := t.TempDir()
	name := "small.webp"
	writeSourcePNG(t, filepath.Join(dir, name), 40, 20)
	if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err != nil {
		t.Fatal(err)
	}
	f, _ := os.Open(filepath.Join(dir, AtlasDirectory, name))
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 40 || img.Bounds().Dy() != 20 {
		t.Fatal(img.Bounds())
	}
	for _, name := range []string{"../x.webp", "a/b.webp", "a\\b.webp", ".x.webp", "x.mp4", "x.webp?q=1"} {
		if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if AtlasThumbnailURL("/uploads/x.webp") != "" || AtlasThumbnailURL("https://example/thumbs/a.webp") != "" {
		t.Fatal("guessed unsupported media")
	}
}

func TestAtlasFailureCancellationAndUpdatedSource(t *testing.T) {
	dir := t.TempDir()
	name := "test.webp"
	source := filepath.Join(dir, name)
	writeSourcePNG(t, source, 300, 200)
	if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, AtlasDirectory, name)
	initial, _ := os.ReadFile(output)
	if err := os.WriteFile(source, []byte("broken"), 0644); err != nil {
		t.Fatal(err)
	}
	newer := time.Now().Add(time.Second)
	_ = os.Chtimes(source, newer, newer)
	if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err == nil {
		t.Fatal("accepted corrupt source")
	}
	preserved, _ := os.ReadFile(output)
	if !bytes.Equal(initial, preserved) {
		t.Fatal("destroyed previous file")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := EnsureAtlasThumbnail(ctx, dir, name); err == nil {
		t.Fatal("ignored cancellation")
	}
	writeSourcePNG(t, source, 180, 180)
	_ = os.Chtimes(source, newer, newer)
	if _, err := EnsureAtlasThumbnail(context.Background(), dir, name); err != nil {
		t.Fatal(err)
	}
	updated, _ := os.ReadFile(output)
	if bytes.Equal(initial, updated) {
		t.Fatal("did not regenerate changed source")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, AtlasDirectory))
	if len(entries) != 1 {
		t.Fatal("temporary files leaked")
	}
}

func TestAtlasConcurrentRequestsAndSweep(t *testing.T) {
	dir := t.TempDir()
	writeSourcePNG(t, filepath.Join(dir, "a.webp"), 320, 320)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := EnsureAtlasThumbnail(context.Background(), dir, "a.webp"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	writeSourcePNG(t, filepath.Join(dir, "b.webp"), 320, 200)
	_ = os.WriteFile(filepath.Join(dir, "broken.webp"), []byte("broken"), 0644)
	_ = os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("text"), 0644)
	ok, fail, err := SweepAtlasThumbnails(context.Background(), dir)
	if err != nil || ok != 2 || fail != 1 {
		t.Fatalf("%d %d %v", ok, fail, err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, AtlasDirectory))
	if len(entries) != 2 {
		t.Fatal(entries)
	}
}

func TestMontageNeverEnlargesCompactWebP(t *testing.T) {
	dir := t.TempDir()
	name := "compact.webp"
	var source bytes.Buffer
	if err := webp.Encode(&source, image.NewRGBA(image.Rect(0, 0, 800, 800)), &webp.EncoderOptions{Quality: 10}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), source.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureMontageThumbnail(context.Background(), dir, name); err != nil {
		t.Fatal(err)
	}
	result, err := os.ReadFile(filepath.Join(dir, MontageDirectory, name))
	if err != nil {
		t.Fatal(err)
	}
	if len(result) > source.Len() {
		t.Fatalf("marquee enlarged: %d > %d", len(result), source.Len())
	}
	if _, _, err := image.Decode(bytes.NewReader(result)); err != nil {
		t.Fatalf("invalid marquee WebP: %v", err)
	}
}

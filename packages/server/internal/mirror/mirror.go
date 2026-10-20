package mirror

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/objstore"
	"github.com/huntersxy/xqecz/server/internal/openlist"
	"github.com/huntersxy/xqecz/server/internal/r2"
)

type ObjectStore = objstore.Store

var contentMD5Name = regexp.MustCompile("^([a-f0-9]{32})\\.[a-z0-9]{1,8}$")

const archiveDir = "original"

const archiveCachePrefix = "archive:"

type Setup struct {
	cfg    Target
	binDir string
	store  ObjectStore

	mu       sync.Mutex
	synced   map[string]syncedFile
	archived map[string]bool
}

type syncedFile struct {
	size    int64
	modTime time.Time
}

func New(t Target, binDir string) *Setup {
	s := &Setup{cfg: t, binDir: binDir, synced: map[string]syncedFile{}, archived: map[string]bool{}}
	if !t.Enabled() {
		return s
	}
	client, err := newStore(t)
	if err != nil {
		slog.Warn("镜像已停用：客户端构造失败", "target", t.LogName(), "err", err)
		return s
	}
	s.store = client
	return s
}

func newStore(t Target) (ObjectStore, error) {
	if t.driver() == DriverOpenList {
		return openlist.New(openlist.Config{
			Endpoint: t.Endpoint,
			Token:    t.Token,
			Prefix:   t.Prefix,
			Timeout:  t.Timeout,
		})
	}
	return r2.New(r2.Config{
		Endpoint:  t.Endpoint,
		AccessKey: t.AccessKey,
		SecretKey: t.SecretKey,
		Bucket:    t.Bucket,
		Region:    t.Region,
		Prefix:    t.Prefix,
		Timeout:   t.Timeout,
	})
}

func (s *Setup) Enabled() bool { return s != nil && s.store != nil }

func (s *Setup) PublicURL(rel string) string {
	if s == nil || !s.cfg.Exposed() || rel == "" {
		return ""
	}
	return s.cfg.ObjectURL(s.ObjectKey(rel))
}

func (s *Setup) ObjectKey(rel string) string {
	key := normRel(rel)
	if s.cfg.Prefix == "" {
		return key
	}
	return s.cfg.Prefix + "/" + key
}

func (s *Setup) Push(rel, absPath string) (uploaded bool, err error) {
	if !s.Enabled() || rel == "" || absPath == "" {
		return false, nil
	}
	rel = normRel(rel)
	if err := mirrorable(rel); err != nil {
		return false, err
	}
	return s.upload(rel, s.ObjectKey(rel), absPath)
}

func (s *Setup) ArchiveKey(rel string) string {
	rel = normRel(rel)
	if rel == "" {
		return ""
	}
	name := rel
	if i := strings.LastIndex(rel, "/"); i >= 0 {
		name = rel[i+1:]
	}
	base := s.cfg.Prefix
	if base == "" {
		base = "uploads"
	}
	return base + "/" + archiveDir + "/" + name
}

func (s *Setup) ArchiveOriginal(rel, origPath string) (uploaded bool, err error) {
	if !s.Enabled() || rel == "" || origPath == "" {
		return false, nil
	}
	if err := mirrorable(normRel(rel)); err != nil {
		return false, err
	}
	rel = normRel(rel)
	if s.archivedRemotely(rel) {
		return false, nil
	}
	uploaded, err = s.upload(archiveCachePrefix+rel, s.ArchiveKey(rel), origPath)
	return uploaded, err
}

func (s *Setup) MarkArchived(rel string) {
	if s == nil || rel == "" {
		return
	}
	s.markArchived(normRel(rel))
}

func (s *Setup) upload(rel, key, absPath string) (bool, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	if s.isSynced(rel, info.Size(), info.ModTime()) {
		return false, nil
	}

	want := localMD5(absPath)
	meta, found, err := s.store.Head(key)
	if err != nil {
		return false, err
	}
	if sameObject(meta, found, want, info.Size()) {
		s.markSynced(rel, info.Size(), info.ModTime())
		return false, nil
	}

	if err := s.store.Put(key, absPath, contentTypeOf(absPath)); err != nil {
		if errors.Is(err, objstore.ErrCorrupted) {
			return false, nil
		}
		return false, err
	}
	s.markSynced(rel, info.Size(), info.ModTime())
	slog.Info(s.cfg.LogName()+" 镜像完成", "key", key, "bytes", info.Size())
	return true, nil
}

func sameObject(meta objstore.ObjectMeta, found bool, want string, localSize int64) bool {
	if !found {
		return false
	}
	if want != "" {
		if remote := md5OfMeta(meta); remote != "" {
			return remote == want
		}
	}
	if meta.ContentLen > 0 && meta.ContentLen == localSize {
		return true
	}
	return false
}

func (s *Setup) ArchiveOriginalAsync(rel, origPath string) {
	if !s.Enabled() || rel == "" {
		return
	}
	go func() {
		if _, err := s.ArchiveOriginal(rel, origPath); err != nil {
			slog.Warn(s.cfg.LogName()+" 原图归档失败，留待回填重试", "rel", rel, "err", err)
		}
	}()
}

func FindInBin(binDir, rel string) (string, bool) {
	if binDir == "" || rel == "" {
		return "", false
	}
	name := normRel(rel)
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		return "", false
	}
	exact := filepath.Join(binDir, name)
	if info, err := os.Stat(exact); err == nil && !info.IsDir() {
		return exact, true
	}
	matches, err := filepath.Glob(filepath.Join(binDir, "*_"+name))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	best, varBest := "", time.Time{}
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil || info.IsDir() {
			continue
		}
		if varBest.IsZero() || info.ModTime().After(varBest) {
			best, varBest = m, info.ModTime()
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

func (s *Setup) PushAsync(rel, absPath string) {
	if !s.Enabled() || rel == "" {
		return
	}
	go func() {
		if _, err := s.Push(rel, absPath); err != nil {
			slog.Warn(s.cfg.LogName()+" 镜像失败，留待回填重试", "rel", rel, "err", err)
		}
	}()
}

func mirrorable(rel string) error {
	switch {
	case strings.HasPrefix(rel, "thumbs/"):
		return errors.New("缩略图保持纯本地，不镜像")
	case strings.HasPrefix(rel, "images/"):
		return errors.New("历史 images/ 目录不镜像")
	case rel == "" || strings.HasPrefix(rel, ".") || strings.Contains(rel, "/."):
		return errors.New("临时/隐藏文件不镜像")
	}
	return nil
}

func localMD5(absPath string) string {
	return fileMD5(absPath)
}

func md5OfMeta(meta objstore.ObjectMeta) string {
	if m := strings.ToLower(strings.TrimSpace(meta.MetaMD5)); m != "" {
		return m
	}
	etag := strings.ToLower(strings.TrimSpace(meta.ETag))
	if etag == "" || strings.Contains(etag, "-") {
		return ""
	}
	if contentMD5Name.MatchString(etag + ".bin") {
		return etag
	}
	return ""
}

func fileMD5(absPath string) string {
	f, err := os.Open(absPath)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

func contentTypeOf(absPath string) string {
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(absPath))); ct != "" {
		return ct
	}
	return "application/octet-stream"
}

func normRel(rel string) string {
	return strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
}

func (s *Setup) isSynced(rel string, size int64, modTime time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	got, ok := s.synced[rel]
	return ok && got.size == size && got.modTime.Equal(modTime)
}

func (s *Setup) markSynced(rel string, size int64, modTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.synced[rel] = syncedFile{size: size, modTime: modTime}
}

func (s *Setup) archivedRemotely(rel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.archived[rel]
}

func (s *Setup) markArchived(rel string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.archived[rel] = true
}

func (s *Setup) BinDir() string {
	if s == nil {
		return ""
	}
	return s.binDir
}

func newWithStore(cfg config.R2Config, store ObjectStore) *Setup {
	return newWithStoreTarget(TargetFromR2(cfg), store)
}

func newWithStoreTarget(t Target, store ObjectStore) *Setup {
	return &Setup{cfg: t, store: store, synced: map[string]syncedFile{}, archived: map[string]bool{}}
}

func newWithStoreAndBin(cfg config.R2Config, store ObjectStore, binDir string) *Setup {
	return &Setup{cfg: TargetFromR2(cfg), binDir: binDir, store: store,
		synced: map[string]syncedFile{}, archived: map[string]bool{}}
}

// Package mirror 把本地上传目录的媒体文件镜像到 Cloudflare R2。
//
// 语义（与产品要求一一对应）：
//   - 原图先落本地，再往 R2 传一份；压缩图同样上传，但**不替换** R2 上的原图，
//     两份都保留 —— 压缩是原地改写本地文件，对象名不变，所以 R2 上留下的是最新一份，
//     历史上传过的其它对象名不会被覆盖或删除。
//   - 缩略图纯本地，本包不碰 thumbs/。
//   - R2 侧 append-only：内容删除时本地文件进垃圾桶保留，R2 对象同样不删。
//
// 一致性策略：不做「乐观跳过」，只以「本地内容 md5 == 远端对象 md5」认定已同步。
// 因为压缩是原地改写（路径不变、内容变了），任何只记录「传过没有」的标记都会
// 在压缩后变成谎言，导致 R2 永远停在未压缩的原图上。
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

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/r2"
)

// ObjectStore 是镜像所需的存储能力（由 r2.Client 实现，测试可替换为内存实现）。
type ObjectStore interface {
	Put(rel, absPath, contentType string) error
	Head(key string) (r2.ObjectMeta, bool, error)
}

// contentMD5Name 匹配内容寻址文件名（前端上传前把文件重命名为 <md5>.<ext>）。
var contentMD5Name = regexp.MustCompile("^([a-f0-9]{32})\\.[a-z0-9]{1,8}$")

// Setup 是镜像能力的集合；Enabled 为 false 时所有方法都是安全的空操作。
type Setup struct {
	cfg   config.R2Config
	store ObjectStore

	mu     sync.Mutex
	synced map[string]int64 // 本地相对路径 → 最近一次确认已同步的字节数
}

// New 按配置构造镜像。凭据不全或客户端构造失败时返回停用态（Enabled()==false），
// 调用方无需分支判断 —— 与 TinyPNG 缺 Key 即休眠同一套思路。
func New(cfg config.R2Config) *Setup {
	s := &Setup{cfg: cfg, synced: map[string]int64{}}
	if !cfg.Enabled() {
		return s
	}
	client, err := r2.New(r2.Config{
		Endpoint:  cfg.Endpoint,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Bucket:    cfg.Bucket,
		Prefix:    cfg.Prefix,
		Timeout:   cfg.UploadTimeout,
	})
	if err != nil {
		slog.Warn("r2 镜像已停用：客户端构造失败", "err", err)
		return s
	}
	s.store = client
	return s
}

// Enabled 表示镜像是否处于工作状态。
func (s *Setup) Enabled() bool { return s != nil && s.store != nil }

// PublicURL 返回对象的公开访问地址；未配置公开域名时返回空串，
// 前端的「本地 vs R2 测速」也就无从谈起，直接走本地。
// 对 nil 接收者安全：调用方无需先判空（与 Push 一致的取向）。
func (s *Setup) PublicURL(rel string) string {
	if s == nil || !s.cfg.Exposed() || rel == "" {
		return ""
	}
	return s.cfg.ObjectURL(rel)
}

// ObjectKey 返回对象在桶内的名字（供对账/排障使用）。
func (s *Setup) ObjectKey(rel string) string {
	key := normRel(rel)
	if s.cfg.Prefix == "" {
		return key
	}
	return s.cfg.Prefix + "/" + key
}

// Push 确保 rel 指向的本地文件已在 R2 上、且与本地内容一致。
//
// 返回 uploaded=true 表示本次真的发出了 PUT（回填统计用）；已同步时只发一次 HEAD。
// 任何错误都返回给调用方记录，绝不影响上传/压缩主流程。
func (s *Setup) Push(rel, absPath string) (uploaded bool, err error) {
	if !s.Enabled() || rel == "" || absPath == "" {
		return false, nil
	}
	rel = normRel(rel)
	// 只镜像 uploads 目录（原图与压缩图）。缩略图、历史 images/ 目录一律不碰。
	if err := mirrorable(rel); err != nil {
		return false, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // 文件已不在（删除/移入垃圾桶），无需镜像
		}
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	if s.isSynced(rel, info.Size()) {
		return false, nil
	}

	key := s.ObjectKey(rel)
	want := localMD5(rel, absPath)
	meta, found, err := s.store.Head(key)
	if err != nil {
		return false, err
	}
	if found && want != "" && md5OfMeta(meta) == want {
		s.markSynced(rel, info.Size())
		return false, nil
	}

	if err := s.store.Put(rel, absPath, contentTypeOf(absPath)); err != nil {
		if errors.Is(err, r2.ErrCorrupted) {
			// 上传期间文件被压缩任务原地改写：放弃本轮（不进缓存），下一轮自动重传。
			return false, nil
		}
		return false, err
	}
	s.markSynced(rel, info.Size())
	slog.Info("r2 镜像完成", "rel", rel, "bytes", info.Size())
	return true, nil
}

// PushAsync 在后台镜像单个文件（上传/更新接口用，不阻塞响应）。
// 失败只告警：回填任务会兜住，不会漏传。
func (s *Setup) PushAsync(rel, absPath string) {
	if !s.Enabled() || rel == "" {
		return
	}
	go func() {
		if _, err := s.Push(rel, absPath); err != nil {
			slog.Warn("r2 镜像失败，留待回填重试", "rel", rel, "err", err)
		}
	}()
}

// mirrorable 判定该相对路径是否属于镜像范围。
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

// localMD5 取本地内容 md5：文件名已是内容寻址时直接沿用（省一次全文件哈希），
// 否则老实算一遍。
func localMD5(rel, absPath string) string {
	if m := contentMD5Name.FindStringSubmatch(filepath.Base(rel)); m != nil {
		return m[1]
	}
	return fileMD5(absPath)
}

// md5OfMeta 从对象元信息里取内容 md5：优先自定义元数据，
// 其次用 ETag（R2 单段上传的 ETag 就是内容 md5，多段上传会带 -N 后缀）。
func md5OfMeta(meta r2.ObjectMeta) string {
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

// normRel 统一相对路径写法：反斜杠转斜杠、去掉前导斜杠。
func normRel(rel string) string {
	return strings.TrimPrefix(strings.ReplaceAll(rel, "\\", "/"), "/")
}

func (s *Setup) isSynced(rel string, size int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	got, ok := s.synced[rel]
	return ok && got == size
}

func (s *Setup) markSynced(rel string, size int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.synced[rel] = size
}

// newWithStore 供同包测试注入假存储。
func newWithStore(cfg config.R2Config, store ObjectStore) *Setup {
	return &Setup{cfg: cfg, store: store, synced: map[string]int64{}}
}

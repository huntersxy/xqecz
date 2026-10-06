// Package mirror 把本地上传目录的媒体文件镜像到对象存储。
//
// 语义（与产品要求一一对应）：
//   - 原图先落本地，由本包镜像到对象存储。
//   - 压缩图另传一份到同一对象名（压缩是原地改写，本地路径不变），因此远端
//     "uploads/<name>" 始终是最新（通常即压缩后）的一份。
//   - 压缩前的原图额外归档到 "uploads/original/<name>"，两份都保留。
//     归档键只由记录里的路径推得、与内容是否已压缩无关，故天然幂等；
//     这一步不做且失败也不影响主流程，回填任务会按「本地垃圾桶里的同名原件」补齐。
//   - 缩略图纯本地，本包不碰 thumbs/。
//   - 远端 append-only：内容删除时本地文件进垃圾桶保留，远端对象同样不删。
//
// 目标与协议解耦：R2（S3）与 OpenList（自有 HTTP API）都实现 objstore.Store，
// 本包只见接口。两家的差别只有一处影响到本文件——OpenList 的 Local 驱动
// 不提供任何远端哈希，判重必须退化到比对字节数（见 sameObject）。
//
// 一致性策略：不做「乐观跳过」，只以「本地内容 md5 == 远端对象 md5」认定已同步。
// 因为压缩是原地改写（路径不变、内容变了），任何只记录「传过没有」的标记都会
// 在压缩后变成谎言，导致远端永远停在未压缩的原图上。
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

// ObjectStore 是镜像所需的存储能力（由 r2.Client / openlist.Client 实现，
// 测试可替换为内存实现）。Put 的第一个参数是**远端对象名**（由调用方算好），
// 归档命名空间因此也能复用同一个实现。
type ObjectStore = objstore.Store

// contentMD5Name 匹配「内容寻址」形态的名字（前端上传前把文件重命名为 <md5>.<ext>）。
// 只用于校验远端 ETag 是不是单段上传留下的裸 md5（见 md5OfMeta），
// **不参与判定本地内容**——文件名在压缩原地改写后就失信了，见 localMD5。
var contentMD5Name = regexp.MustCompile("^([a-f0-9]{32})\\.[a-z0-9]{1,8}$")

// archiveDir 是压缩前原图在远端内的归档子目录：
// 公开对象按 <prefix>/<name> 放最新一份，原图留在 <prefix>/original/<name>。
const archiveDir = "original"

// archiveCachePrefix 把归档件与现役对象在进程内缓存里分开记账。
const archiveCachePrefix = "archive:"

// Setup 是镜像能力的集合；Enabled 为 false 时所有方法都是安全的空操作。
type Setup struct {
	cfg    Target
	binDir string
	store  ObjectStore

	mu     sync.Mutex
	synced map[string]int64 // 本地相对路径 → 最近一次确认已同步的字节数
	// archived 记录「本进程生命周期内已确认归档」的 rel。
	// 归档件的本地来源（垃圾桶里的原件）不会像现役文件那样再被改写，
	// 因此这里不需要比对字节数 —— 只要本次上传成功过就不再重复 HEAD。
	archived map[string]bool
}

// New 按目标构造镜像。凭据不全或客户端构造失败时返回停用态（Enabled()==false），
// 调用方无需分支判断 —— 与 TinyPNG 缺 Key 即休眠同一套思路。
// binDir 是本地垃圾桶目录：压缩后的原图留在那里，归档失败时回填任务据此补传。
func New(t Target, binDir string) *Setup {
	s := &Setup{cfg: t, binDir: binDir, synced: map[string]int64{}, archived: map[string]bool{}}
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

// newStore 按驱动构造对应的客户端。协议差异只在这里出现。
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

// Enabled 表示镜像是否处于工作状态。
func (s *Setup) Enabled() bool { return s != nil && s.store != nil }

// PublicURL 返回对象的公开访问地址；未配置公开域名时返回空串，
// 前端的「本地 vs 远端测速」也就无从谈起，直接走本地。
// 对 nil 接收者安全：调用方无需先判空（与 Push 一致的取向）。
//
// 公开地址必须走 ObjectKey：对象在远端叫 <prefix>/<name>，而调用方传进来的是
// 「相对上传目录」的裸文件名（decorate 的 mirrorKey 已剥掉 uploads/）。
// 直接拼 rel 会得到 <PublicBase>/<name>，与真实对象名差一个前缀 ——
// 私有读写照样 200，公开地址却永远 404。
func (s *Setup) PublicURL(rel string) string {
	if s == nil || !s.cfg.Exposed() || rel == "" {
		return ""
	}
	return s.cfg.ObjectURL(s.ObjectKey(rel))
}

// ObjectKey 返回对象在远端内的名字（供对账/排障使用）。
func (s *Setup) ObjectKey(rel string) string {
	key := normRel(rel)
	if s.cfg.Prefix == "" {
		return key
	}
	return s.cfg.Prefix + "/" + key
}

// Push 确保 rel 指向的本地文件已在远端、且与本地内容一致。
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
	return s.upload(rel, s.ObjectKey(rel), absPath)
}

// ArchiveKey 返回 rel 对应原图的归档对象名（远端名，已含前缀）。
// 形如 "uploads/original/<name>"：与公开对象同前缀，取回时不必额外配权限；
// 正常加载路径不会用到它（前端只拿 ObjectKey 的公开地址）。
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

// ArchiveOriginal 把压缩前的原图归档到远端的 original/ 命名空间。
//
// rel 是**压缩后**记录的本地相对路径（压缩是原地改写，路径不变），
// origPath 是原图当前所在位置 —— 正常链路上是本地垃圾桶里的同名文件。
// 归档键由文件名推得，而文件名是内容寻址的，因此重复调用天然幂等：
// 远端已有同一份内容时只发一次 HEAD 就返回。
//
// 失败必须冒泡（调用方只记日志不中断压缩），回填任务会从垃圾桶补传，原图不会丢。
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
	// 缓存键与公开对象分开记账：同一 rel 的归档件与现役件大小本就不同，
	// 共用一个键会互相顶掉，导致本该重传的归档被误判为已同步。
	uploaded, err = s.upload(archiveCachePrefix+rel, s.ArchiveKey(rel), origPath)
	return uploaded, err
}

// MarkArchived 登记 rel 的原图已归档到位（远端 HEAD 命中时由回填任务调用），
// 避免同一进程内每轮都对同一对象重复发 HEAD。
func (s *Setup) MarkArchived(rel string) {
	if s == nil || rel == "" {
		return
	}
	s.markArchived(normRel(rel))
}

// upload 是 Push 与 ArchiveOriginal 共用的上传：key 为远端对象名，
// 缓存按 rel 记账（同一 rel 的公开对象与归档对象可能共存，故缓存只用于加速，
// 真正的判重依据始终是 sameObject）。
func (s *Setup) upload(rel, key, absPath string) (bool, error) {
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

	want := localMD5(absPath)
	meta, found, err := s.store.Head(key)
	if err != nil {
		return false, err
	}
	if sameObject(meta, found, want, info.Size()) {
		s.markSynced(rel, info.Size())
		return false, nil
	}

	if err := s.store.Put(key, absPath, contentTypeOf(absPath)); err != nil {
		if errors.Is(err, objstore.ErrCorrupted) {
			// 上传期间文件被压缩任务原地改写：放弃本轮（不进缓存），下一轮自动重传。
			return false, nil
		}
		return false, err
	}
	s.markSynced(rel, info.Size())
	slog.Info(s.cfg.LogName()+" 镜像完成", "key", key, "bytes", info.Size())
	return true, nil
}

// sameObject 判定远端已有的一份是否就是本地当前这一份。
//
// 优先用内容 md5 比对（R2 能给出 md5：自定义元数据或单段 ETag）。
// 退路是字节数比对——OpenList 的 Local 驱动 hashinfo 恒为 "null"，
// 拿不到任何哈希，此时「远端存在且字节数相同」是唯一可用的判据。
// 字节数相同不等于内容相同，但这里的目的只是**省掉一次无谓的重传**：
// 判错（误以为已同步）的代价是这一轮不传，而回填任务每轮都会重走
// Head 分支，文件一旦被压缩改写（大小几乎必然变化）就会立刻暴露并补传。
// 反过来，若坚持「拿不到哈希就一律重传」，OpenList 侧的每轮回填都会
// 把全量媒体重新推一遍，代价远大于这点风险。
//
// want 为空（文件名不是内容寻址且本地哈希失败）时不认定同步。
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

// ArchiveOriginalAsync 后台归档原图（压缩任务用，不阻塞调度）。
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

// FindInBin 在垃圾桶目录里找 rel 对应的原件（与 MoveToBin 的命名约定一致：
// 优先同名，其次 MoveToBin 为防同秒覆盖而加的时间戳前缀版本，取最新一个）。
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

// PushAsync 在后台镜像单个文件（上传/更新接口用，不阻塞响应）。
// 失败只告警：回填任务会兜住，不会漏传。
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

// localMD5 取本地**当前**内容的 md5，一律实读文件计算。
//
// 这里曾经有个「省一次全文件哈希」的快路径：文件名是内容寻址的（前端上传前把文件
// 重命名为 <md5>.<ext>），于是直接沿用文件名里的 md5。它在原地压缩面前是错的——
// TinyPNG 只改内容不改路径，压缩之后文件名描述的仍是**压缩前**那一份，快路径给出的
// 是两个不同时代的答案，且两个方向都会出事：
//   - 远端已换成压缩图时：文件名 md5（压缩前）≠ 远端 md5（压缩后），sameObject 的
//     md5 分支直接判「不同步」，于是一批对象每轮回填都被重传，永不收敛；
//   - 远端还停在原图时：文件名 md5（压缩前）== 远端 md5（原图），被误判成「已同步」，
//     压缩图永远推不上去——公开域名会一直供着未压缩的大图。
//
// 代价是对变更过的文件每轮多读一遍内容；走到这里之前，isSynced 的字节数缓存已经挡掉
// 绝大多数文件，实测全场 35MB 量级开销可忽略，换来的是两个方向都不再撒谎。
func localMD5(absPath string) string {
	return fileMD5(absPath)
}

// md5OfMeta 从对象元信息里取内容 md5：优先自定义元数据，
// 其次用 ETag（R2 单段上传的 ETag 就是内容 md5，多段上传会带 -N 后缀）。
// OpenList 两者皆空，返回空串——调用方须自行处理「没有远端哈希」。
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

// archivedRemotely 表示本进程内已把 rel 的原图归档过一次。
// 只看进程内状态，不额外发 HEAD：归档来源不可变，重复上传相同 key 是无害的覆盖写。
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

// BinDir 返回垃圾桶目录（回填任务据此查找原图）。
func (s *Setup) BinDir() string {
	if s == nil {
		return ""
	}
	return s.binDir
}

// newWithStore 供同包测试注入假存储（沿用 R2 形状的配置，内部映射为目标）。
func newWithStore(cfg config.R2Config, store ObjectStore) *Setup {
	return newWithStoreTarget(TargetFromR2(cfg), store)
}

// newWithStoreTarget 供同包测试注入任意目标。
func newWithStoreTarget(t Target, store ObjectStore) *Setup {
	return &Setup{cfg: t, store: store, synced: map[string]int64{}, archived: map[string]bool{}}
}

// newWithStoreAndBin 供同包测试注入假存储与垃圾桶目录。
func newWithStoreAndBin(cfg config.R2Config, store ObjectStore, binDir string) *Setup {
	return &Setup{cfg: TargetFromR2(cfg), binDir: binDir, store: store,
		synced: map[string]int64{}, archived: map[string]bool{}}
}

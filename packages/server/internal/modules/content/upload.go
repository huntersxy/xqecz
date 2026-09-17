package content

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// 单文件上限 20MB，与前端一致（快速上传/普通上传/富文本插图同规格）。
const maxUploadSize = 20 << 20

// 表单字段值上限 1MB，避免超大字段占用内存。
const maxFieldSize = 1 << 20

// md5NamePattern 匹配前端上传前的重命名结果（<md5>.<ext>）。
var md5NamePattern = regexp.MustCompile(`^[a-f0-9]{32}\.[a-z0-9]{1,8}$`)

var (
	errBadMultipart = errors.New("bad multipart body")
	errTooLarge     = errors.New("file too large")
	errBadFileType  = errors.New("unsupported media type")
)

// uploadedFile 是落盘后的上传文件。
type uploadedFile struct {
	AbsPath string
	RelPath string
	Size    int64
	Mime    string
}

// form 是解析后的 multipart 表单。
type form struct {
	Fields map[string][]string
	File   *uploadedFile
}

func (f *form) value(key string) string {
	if v := f.Fields[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// parseMultipart 流式解析 multipart 表单：文件直接落盘（不经过内存缓冲），
// 字段值按名收集（同名多值保留，用于 tags 这类可重复字段）。
func (h *Handler) parseMultipart(c *gin.Context) (*form, error) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		return nil, errBadMultipart
	}

	out := &form{Fields: map[string][]string{}}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errBadMultipart
		}

		name := part.FormName()
		if name == "file" && part.FileName() != "" {
			file, err := h.savePart(part)
			if err != nil {
				_ = part.Close()
				return nil, err
			}
			out.File = file
			_ = part.Close()
			continue
		}

		buf, _ := io.ReadAll(io.LimitReader(part, maxFieldSize))
		_ = part.Close()
		if name != "" {
			out.Fields[name] = append(out.Fields[name], string(buf))
		}
	}
	return out, nil
}

// savePart 把一个文件分区流式写入 UPLOAD_DIR。
//
// 落盘一律先写同目录临时文件、边写边算内容 md5，全部成功后才原子改名到最终路径。
// 因此中断、截断或半途失败的上传绝不会触碰已存在的文件——同名文件只有在内容校验
// 一致时才会被替换，不会再出现「打开即清空、写了一半」把旧文件毁掉的情况。
//
// 文件名规则：
//   - 客户端按内容寻址传了 <md5>.<ext> 且与服务端算出的内容 md5 一致 → 沿用该名（同内容天然去重）；
//   - 传了 <md5>.<ext> 但与内容不符（旧客户端、被截断或篡改）→ 改用服务端算出的 md5，
//     绝不写入可能已被其它内容占用的路径；
//   - 其他名字 → 随机名兜底，保证任何来源都能落盘。
func (h *Handler) savePart(part *multipart.Part) (*uploadedFile, error) {
	mime := part.Header.Get("Content-Type")
	if !strings.HasPrefix(mime, "image/") && !strings.HasPrefix(mime, "video/") {
		return nil, errBadFileType
	}

	dir := h.deps.Cfg.UploadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	declared := strings.ToLower(filepath.Base(part.FileName()))
	ext := strings.ToLower(filepath.Ext(declared))
	if ext == "" {
		ext = ".bin"
	}

	// 临时文件与最终文件同目录，保证 rename 是同一文件系统内的原子操作。
	tmp, err := os.CreateTemp(dir, ".upload-*.part")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	committed := false
	// 任何提前返回都清掉临时文件；改名成功后它已不存在，这里不再命中。
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	sum := md5.New()
	// 多读 1 字节用于判定超限；超限即整单放弃，磁盘上不留残文件。
	n, err := io.Copy(io.MultiWriter(tmp, sum), io.LimitReader(part, maxUploadSize+1))
	closeErr := tmp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	if n > maxUploadSize {
		return nil, errTooLarge
	}

	digest := hex.EncodeToString(sum.Sum(nil))
	filename := randomName() + ext
	if md5NamePattern.MatchString(declared) {
		if strings.TrimSuffix(declared, filepath.Ext(declared)) == digest {
			filename = declared
		} else {
			filename = digest + ext
		}
	}

	abs := filepath.Join(dir, filename)
	if st, statErr := os.Stat(abs); statErr == nil && st.Size() == n {
		// 同内容文件已存在，直接复用，不做无谓替换。
		return &uploadedFile{AbsPath: abs, RelPath: filename, Size: n, Mime: mime}, nil
	}
	// 文件属主可能被替换，保持与目录一致的权限，避免 Web 进程读不到。
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpPath, abs); err != nil {
		return nil, err
	}
	committed = true

	return &uploadedFile{AbsPath: abs, RelPath: filename, Size: n, Mime: mime}, nil
}

// parseTagsField 解析 tags 字段：支持重复字段、JSON 数组字符串与逗号分隔三种写法。
func (f *form) parseTagsField() []string {
	raw := f.Fields["tags"]
	out := []string{}
	seen := map[string]bool{}
	add := func(t string) {
		t = strings.TrimSpace(t)
		if t != "" && !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "[") {
			for _, t := range ParseTags(item) {
				add(t)
			}
			continue
		}
		for _, t := range strings.Split(item, ",") {
			add(t)
		}
	}
	return out
}

// relToUpload 把绝对路径转为相对上传目录的斜杠路径。
func (h *Handler) relToUpload(abs string) string {
	rel, err := filepath.Rel(h.deps.Cfg.UploadDir, abs)
	if err != nil {
		return filepath.Base(abs)
	}
	return filepath.ToSlash(rel)
}

// respondUploadError 把上传解析错误映射为与旧实现一致的响应。
func respondUploadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errTooLarge):
		softFail(c, 400, "文件大小超过 20MB 限制")
	case errors.Is(err, errBadFileType):
		softFail(c, 400, "仅支持图片或视频文件")
	default:
		softFail(c, 400, "请求参数格式错误")
	}
}

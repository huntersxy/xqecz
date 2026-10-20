package vision

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

var ErrReferenceNotFound = errors.New("角色参照不存在")
var ErrReferenceExists = errors.New("角色名已存在，请选择替换图片或使用其他名字")

type referenceNote struct {
	Text    string `json:"text"`
	Version string `json:"version"`
}

const referenceNotesFile = ".descriptions.json"
const maxReferenceUpload = 20 << 20

func validateReferenceName(name string) error {
	if name != strings.TrimSpace(name) || name == "" || len([]rune(name)) > 30 || strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".") || strings.ContainsAny(name, `<>:"/\|?*`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errors.New("角色名需为1到30字，不能含路径、控制字符或文件名特殊符号")
	}
	base := strings.ToUpper(strings.SplitN(name, ".", 2)[0])
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
		return errors.New("角色名不能使用系统保留文件名")
	}
	return nil
}
func referenceVersion(b []byte) string { sum := sha256.Sum256(b); return hex.EncodeToString(sum[:]) }

func loadReferenceNotes(dir string) (map[string]referenceNote, error) {
	notes := map[string]referenceNote{}
	f, err := os.Open(filepath.Join(dir, referenceNotesFile))
	if os.IsNotExist(err) {
		return notes, nil
	}
	if err != nil {
		return nil, errors.New("角色描述文件不可读取")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 || json.Unmarshal(b, &notes) != nil || notes == nil {
		return nil, errors.New("角色描述文件格式错误")
	}
	return notes, nil
}

func atomicReferenceWrite(ctx context.Context, dir, name string, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".reference-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(b)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}
func saveReferenceNotes(ctx context.Context, dir string, notes map[string]referenceNote) error {
	b, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	if len(b) > 1<<20 {
		return errors.New("角色描述文件过大")
	}
	return atomicReferenceWrite(ctx, dir, referenceNotesFile, b)
}

func compressedReference(ctx context.Context, path string) ([]byte, error) {
	url, err := imageDataURL(ctx, path)
	if err != nil {
		return nil, err
	}
	b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, "data:image/jpeg;base64,"))
	if err != nil {
		return nil, err
	}
	if len(b) <= 512<<10 {
		return b, nil
	}
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	for _, q := range []int{75, 65, 55, 45} {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: q}); err != nil {
			return nil, err
		}
		if out.Len() <= 512<<10 {
			return out.Bytes(), nil
		}
	}
	return nil, errors.New("压缩后仍超过512KB，请选择画面更简洁的设定图")
}

type contextReferenceReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReferenceReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(b)
}

func (s *Service) PutReference(ctx context.Context, name string, source io.Reader, replace bool) (Reference, error) {
	mode := "create"
	if replace {
		mode = "replace"
	}
	return s.putReference(ctx, name, source, mode)
}
func (s *Service) putReference(ctx context.Context, name string, source io.Reader, mode string) (Reference, error) {
	if err := validateReferenceName(name); err != nil {
		return Reference{}, err
	}
	if err := os.MkdirAll(s.referenceDir, 0o700); err != nil {
		return Reference{}, err
	}
	tmp, err := os.CreateTemp(s.referenceDir, ".incoming-*.part")
	if err != nil {
		return Reference{}, err
	}
	defer os.Remove(tmp.Name())
	n, copyErr := io.Copy(tmp, io.LimitReader(contextReferenceReader{ctx, source}, maxReferenceUpload+1))
	closeErr := tmp.Close()
	if copyErr != nil {
		return Reference{}, copyErr
	}
	if closeErr != nil {
		return Reference{}, closeErr
	}
	if n > maxReferenceUpload {
		return Reference{}, errors.New("参照图不能超过20MB")
	}
	b, err := compressedReference(ctx, tmp.Name())
	if err != nil {
		return Reference{}, err
	}
	s.referenceMu.Lock()
	defer s.referenceMu.Unlock()
	entries, err := os.ReadDir(s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	count, exists := 0, false
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".jpg") {
			continue
		}
		count++
		other := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if strings.EqualFold(other, name) {
			if other != name {
				return Reference{}, ErrReferenceExists
			}
			exists = true
		}
	}
	if mode == "create" && exists {
		return Reference{}, ErrReferenceExists
	}
	if mode == "replace" && !exists {
		return Reference{}, ErrReferenceNotFound
	}
	if !exists && count >= 12 {
		return Reference{}, errors.New("角色参照最多12张，请先删除不再需要的角色")
	}
	notes, err := loadReferenceNotes(s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	if exists {
		if err := s.backupReferenceLocked(ctx, name, false); err != nil {
			return Reference{}, err
		}
	}
	if err := atomicReferenceWrite(ctx, s.referenceDir, name+".jpg", b); err != nil {
		return Reference{}, err
	}
	note := notes[name]
	version := referenceVersion(b)
	return Reference{Name: name, Bytes: int64(len(b)), Version: version, Description: note.Text, NeedsDescription: note.Text == "" || note.Version != version}, nil
}

func (s *Service) referenceLocked(ctx context.Context, name string) (Reference, error) {
	if err := validateReferenceName(name); err != nil {
		return Reference{}, err
	}
	refs, err := loadReferences(ctx, s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	for _, ref := range refs {
		if ref.Name == name {
			return ref, nil
		}
	}
	return Reference{}, ErrReferenceNotFound
}

func (s *Service) ReferenceImage(ctx context.Context, name string) ([]byte, error) {
	s.referenceMu.RLock()
	defer s.referenceMu.RUnlock()
	ref, err := s.referenceLocked(ctx, name)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(strings.TrimPrefix(ref.DataURL, "data:image/jpeg;base64,"))
}

func (s *Service) backupReferenceLocked(ctx context.Context, name string, remove bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(s.referenceDir, name+".jpg")
	st, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return ErrReferenceNotFound
	}
	if err != nil || !st.Mode().IsRegular() {
		return errors.New("参照图需为普通文件")
	}
	archive := filepath.Join(s.referenceDir, ".archive")
	if err := os.MkdirAll(archive, 0o700); err != nil {
		return err
	}
	target := fmt.Sprintf("%d-%s.jpg", time.Now().UTC().UnixNano(), name)
	if remove {
		return os.Rename(path, filepath.Join(archive, target))
	}
	if st.Size() > 512<<10 {
		return errors.New("原参照图超过512KB，请先移出后重新上传")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return atomicReferenceWrite(ctx, archive, target, b)
}

func (s *Service) DeleteReference(ctx context.Context, name string) error {
	if err := validateReferenceName(name); err != nil {
		return err
	}
	s.referenceMu.Lock()
	defer s.referenceMu.Unlock()
	return s.backupReferenceLocked(ctx, name, true)
}

func (s *Service) RenameReference(ctx context.Context, name, newName string) (Reference, error) {
	if err := validateReferenceName(newName); err != nil {
		return Reference{}, err
	}
	s.referenceMu.Lock()
	defer s.referenceMu.Unlock()
	ref, err := s.referenceLocked(ctx, name)
	if err != nil {
		return Reference{}, err
	}
	if name == newName {
		return ref, nil
	}
	refs, err := loadReferences(ctx, s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	for _, other := range refs {
		if strings.EqualFold(other.Name, newName) {
			return Reference{}, ErrReferenceExists
		}
	}
	notes, err := loadReferenceNotes(s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	if note, ok := notes[name]; ok {
		notes[newName] = note
		if err := saveReferenceNotes(ctx, s.referenceDir, notes); err != nil {
			return Reference{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Reference{}, err
	}
	if err := os.Rename(filepath.Join(s.referenceDir, name+".jpg"), filepath.Join(s.referenceDir, newName+".jpg")); err != nil {
		return Reference{}, err
	}
	ref.Name = newName
	return ref, nil
}

func (s *Service) SaveReferenceDescription(ctx context.Context, name, description, version string) (Reference, error) {
	description = strings.TrimSpace(description)
	if len([]rune(description)) > 2000 || strings.IndexFunc(description, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' }) >= 0 {
		return Reference{}, errors.New("描述最多2000字，不能含控制字符")
	}
	s.referenceMu.Lock()
	defer s.referenceMu.Unlock()
	ref, err := s.referenceLocked(ctx, name)
	if err != nil {
		return Reference{}, err
	}
	if version == "" || version != ref.Version {
		return Reference{}, errors.New("参照图已更新，请刷新后再保存描述")
	}
	notes, err := loadReferenceNotes(s.referenceDir)
	if err != nil {
		return Reference{}, err
	}
	notes[name] = referenceNote{Text: description, Version: version}
	if err := saveReferenceNotes(ctx, s.referenceDir, notes); err != nil {
		return Reference{}, err
	}
	ref.Description = description
	ref.NeedsDescription = description == ""
	return ref, nil
}

func (s *Service) DescribeReference(ctx context.Context, name, version string) (Reference, error) {
	s.referenceMu.RLock()
	ref, err := s.referenceLocked(ctx, name)
	s.referenceMu.RUnlock()
	if err != nil {
		return Reference{}, err
	}
	if ref.Version != version {
		return Reference{}, errors.New("参照图已更新，请刷新后重新转述")
	}
	cfg, err := s.config(ctx)
	if err != nil {
		return Reference{}, errors.New("模型配置读取失败")
	}
	if cfg.APIKey == "" {
		return Reference{}, errors.New("请先保存视觉模型API Key")
	}
	text, err := chatCompletion(ctx, cfg, cfg.ReferencePrompt+"\n只输出JSON：{\"description\":\"文字特征\"}，不要Markdown或解释。", []any{map[string]any{"type": "image_url", "image_url": map[string]string{"url": ref.DataURL}}})
	if err != nil {
		return Reference{}, err
	}
	var out struct {
		Description string `json:"description"`
	}
	if json.Unmarshal([]byte(trimModelJSON(text)), &out) != nil || strings.TrimSpace(out.Description) == "" {
		return Reference{}, errors.New("模型未返回有效的角色描述JSON")
	}
	return s.SaveReferenceDescription(ctx, name, out.Description, version)
}

package aigen

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func loadCipher(dir string) (cipher.AEAD, error) {
	path := filepath.Join(dir, ".encryption-key")
	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		f, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(createErr, os.ErrExist) {
			key, err = os.ReadFile(path)
		} else if createErr != nil {
			return nil, createErr
		} else {
			_, err = f.Write(key)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, errors.New("AI encryption key is invalid; preserve the original key when restoring a backup")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Service) encrypt(raw string) (string, error) {
	if s.cipher == nil {
		return "", ErrUnavailable
	}
	nonce := make([]byte, s.cipher.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(s.cipher.Seal(nonce, nonce, []byte(raw), []byte("xqecz-ai-v1"))), nil
}
func (s *Service) decrypt(encoded string) (string, error) {
	if s.cipher == nil {
		return "", ErrUnavailable
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < s.cipher.NonceSize() {
		return "", ErrInvalid
	}
	plain, err := s.cipher.Open(nil, raw[:s.cipher.NonceSize()], raw[s.cipher.NonceSize():], []byte("xqecz-ai-v1"))
	return string(plain), err
}

func stringList(raw string) []string {
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
func encode(v any) string { raw, _ := json.Marshal(v); return string(raw) }

func invalidProvider(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalid, message)
}

const defaultAdapter = "openai"

func normalizedAdapter(adapter string) string {
	if strings.TrimSpace(adapter) == "" {
		return defaultAdapter
	}
	return strings.TrimSpace(adapter)
}

func adapterNeeds(path, suffix string) bool {
	return strings.HasSuffix(strings.TrimSuffix(path, "/"), suffix)
}

func validateProvider(p ProviderConfig) error {
	if err := validateURL(p.Endpoint); err != nil {
		return invalidProvider(err.Error())
	}
	p.Adapter = normalizedAdapter(p.Adapter)
	switch p.Adapter {
	case "openai", "volcengine_ark", "gitee_ai", "siliconflow_adapter", "agnes_ai", "grok":
		if !adapterNeeds(p.Endpoint, "/images/generations") {
			return invalidProvider("Images 供应商地址必须以 /images/generations 结尾")
		}
	case "openai_chat":
		if !adapterNeeds(p.Endpoint, "/chat/completions") {
			return invalidProvider("Chat 供应商地址必须以 /chat/completions 结尾")
		}
	case "gemini":
		if !adapterNeeds(p.Endpoint, ":generateContent") {
			return invalidProvider("Gemini 地址必须以 :generateContent 结尾")
		}
	case "codex_responses":
		if !adapterNeeds(p.Endpoint, "/responses") {
			return invalidProvider("Codex Responses 地址必须以 /responses 结尾")
		}
	default:
		return invalidProvider("不支持的供应商适配器")
	}
	if len(p.Name) == 0 || len([]rune(p.Name)) > 100 {
		return invalidProvider("供应商名称无效")
	}
	if len(p.Model) == 0 || len(p.Model) > 160 {
		return invalidProvider("模型 ID 无效")
	}
	if p.MaxCount < 1 || p.MaxCount > 2 || p.MaxReferences < 0 || p.MaxReferences > 8 || p.TimeoutSeconds < 10 || p.TimeoutSeconds > 300 || p.Concurrency < 1 || p.Concurrency > 2 || p.Cost < 1 || p.Cost > 10000 {
		return invalidProvider("供应商参数范围无效")
	}
	if len(stringList(p.Qualities)) > 8 {
		return invalidProvider("允许画质过多")
	}
	if p.EditTransport != "none" && p.EditTransport != "multipart" && p.EditTransport != "json" {
		return invalidProvider("参考图传输方式无效")
	}
	if p.EditTransport == "none" && p.MaxReferences != 0 {
		return invalidProvider("关闭参考图传输时，最多参考图必须为 0")
	}
	if (p.Adapter == "openai_chat" || p.Adapter == "gemini") && p.EditTransport == "multipart" {
		return invalidProvider("该适配器只支持 JSON 多模态参考图传输")
	}
	if p.Adapter == "gemini" && p.MaxCount != 1 {
		return invalidProvider("Gemini 适配器当前每次只能返回 1 张图片")
	}
	if p.Adapter == "codex_responses" && p.MaxCount != 1 {
		return invalidProvider("Codex Responses 适配器当前每次只能返回 1 张图片")
	}
	if strings.TrimSpace(p.Options) != "" {
		var options map[string]any
		if json.Unmarshal([]byte(p.Options), &options) != nil || options == nil {
			return invalidProvider("适配器选项必须是 JSON 对象")
		}
	}
	var costs map[string]int64
	if p.QualityCosts != "" && json.Unmarshal([]byte(p.QualityCosts), &costs) != nil {
		return invalidProvider("画质额度覆盖必须是 JSON 对象")
	}
	for quality, cost := range costs {
		if !contains(stringList(p.Qualities), quality) || cost < 1 || cost > 10000 {
			return invalidProvider("画质额度覆盖包含未允许的画质或无效额度")
		}
	}
	return nil
}

// BYOK can select only an administrator-approved provider. Users never get a
// URL input, and a queued task captures the configuration and encrypted key.
func (s *Service) SaveProvider(ctx context.Context, uid uint64, row ProviderConfig, key string) (ProviderConfig, error) {
	key = strings.TrimSpace(key)
	if len(key) > 500 || strings.ContainsAny(key, "\r\n") {
		return row, invalidInput("API Key 无效：长度不能超过 500 个字符且不能包含换行")
	}
	var old ProviderConfig
	if row.ID != 0 {
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", row.ID, uid).First(&old).Error; err != nil {
			return row, err
		}
		if uid == 0 && row.Revision != old.Revision {
			return row, ErrConflict
		}
	}
	row.UserID = uid
	row.Adapter = normalizedAdapter(row.Adapter)
	if uid != 0 {
		var allowed ProviderConfig
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = 0 AND enabled = ?", row.AllowedProviderID, true).First(&allowed).Error; err != nil {
			return row, invalidInput("所选站点供应商不存在、已停用或未获管理员批准")
		}
		name, id, def := row.Name, row.ID, row.Default
		row = allowed
		row.ID, row.UserID, row.Name, row.Default, row.AllowedProviderID = id, uid, name, def, allowed.ID
		row.CipherKey = old.CipherKey
	}
	if err := validateProvider(row); err != nil {
		return row, err
	}
	if key != "" {
		encrypted, err := s.encrypt(key)
		if err != nil {
			return row, err
		}
		row.CipherKey = encrypted
	} else {
		row.CipherKey = old.CipherKey
	}
	if uid != 0 && row.Enabled && row.CipherKey == "" {
		return row, invalidInput("启用供应商前请设置 API Key")
	}
	row.Revision = old.Revision + 1
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if old.ID != 0 {
			var current ProviderConfig
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, old.ID).Error; err != nil {
				return err
			}
			if current.Revision != old.Revision {
				return ErrConflict
			}
		}
		if row.Default {
			if err := tx.Model(&ProviderConfig{}).Where("user_id = ?", uid).Update("default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&row).Error
	})
	return row, err
}

// SetSiteProviderEnabled changes only the site's dispatch availability. The
// encrypted key is retained so an administrator can re-enable a provider
// without re-entering credentials; enabling still requires a configured key.
func (s *Service) SetSiteProviderEnabled(ctx context.Context, id uint64, enabled bool) error {
	var row ProviderConfig
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = 0", id).First(&row).Error; err != nil {
		return err
	}
	if enabled && row.CipherKey == "" {
		return invalidInput("启用供应商前请设置 API Key")
	}
	updates := map[string]any{"enabled": enabled}
	if !enabled {
		updates["default"] = false
	}
	return s.db.WithContext(ctx).Model(&ProviderConfig{}).Where("id = ? AND user_id = 0", id).Updates(updates).Error
}

// DeleteSiteProvider removes a site provider only when no queued/running task
// still needs its dispatch row and no BYOK row points at it. Completed task
// snapshots remain self-contained, while active work receives a clear action
// error instead of failing later with a missing provider.
func (s *Service) DeleteSiteProvider(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ProviderConfig
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = 0", id).First(&row).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&Task{}).Where("(provider_id = ? OR dispatch_provider_id = ?) AND status IN ?", id, id, []string{"queued", "running"}).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return invalidInput("供应商仍有排队或运行中的任务，请先等待任务结束或取消后再删除")
		}
		var byok int64
		if err := tx.Model(&ProviderConfig{}).Where("allowed_provider_id = ? AND user_id <> 0", id).Count(&byok).Error; err != nil {
			return err
		}
		if byok > 0 {
			return invalidInput("仍有用户自备密钥配置依赖此供应商，请先删除这些配置")
		}
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		if row.Default {
			var next ProviderConfig
			if err := tx.Where("user_id = 0 AND enabled = ? AND cipher_key <> ?", true, "").Order("id asc").First(&next).Error; err == nil {
				if err := tx.Model(&ProviderConfig{}).Where("id = ?", next.ID).Update("default", true).Error; err != nil {
					return err
				}
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		return nil
	})
}

func (s *Service) providerFor(ctx context.Context, uid uint64, input Input) (ProviderConfig, error) {
	owner := uint64(0)
	if input.Mode == "byok" {
		owner = uid
	} else if input.Mode != "site" {
		return ProviderConfig{}, ErrInvalid
	}
	var p ProviderConfig
	query := s.db.WithContext(ctx).Where("user_id = ? AND enabled = ?", owner, true)
	if owner == 0 {
		query = query.Where("cipher_key <> ?", "")
	}
	if input.ProviderID != 0 {
		query = query.Where("id = ?", input.ProviderID)
	} else {
		query = query.Order("`default` desc, id asc")
	}
	if err := query.First(&p).Error; err != nil {
		return p, ErrUnavailable
	}
	if owner != 0 {
		var allowed ProviderConfig
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = 0 AND enabled = ?", p.AllowedProviderID, true).First(&allowed).Error; err != nil {
			return p, ErrUnavailable
		}
		p = mergeBYOK(p, allowed)
	}
	p.Adapter = normalizedAdapter(p.Adapter)
	return p, nil
}

func mergeBYOK(own, allowed ProviderConfig) ProviderConfig {
	result := allowed
	result.ID, result.UserID, result.Name, result.CipherKey, result.Default, result.AllowedProviderID = own.ID, own.UserID, own.Name, own.CipherKey, own.Default, allowed.ID
	result.Enabled = own.Enabled && allowed.Enabled
	return result
}

// Capabilities always come from the current approved provider, while the
// private name/key remain owned by the user. Never display stale BYOK copies.
func (s *Service) VisibleProviders(ctx context.Context, uid uint64) ([]ProviderConfig, error) {
	rows := []ProviderConfig{}
	err := s.db.WithContext(ctx).Where("user_id = 0 OR (user_id = ? AND cipher_key <> ?)", uid, "").Order("`default` desc,id asc").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	allowed := map[uint64]ProviderConfig{}
	for _, row := range rows {
		if row.UserID == 0 {
			allowed[row.ID] = row
		}
	}
	out := []ProviderConfig{}
	for _, row := range rows {
		if row.UserID == 0 {
			out = append(out, row)
		} else if parent, ok := allowed[row.AllowedProviderID]; ok {
			out = append(out, mergeBYOK(row, parent))
		}
	}
	return out, nil
}

package aigen

import (
	"bytes"
	"context"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var ErrInvalid = errors.New("参数、会话或参考图片无效")
var ErrQuota = errors.New("生成额度不足")
var ErrBusy = errors.New("任务数量已达上限或状态已改变")
var ErrUnavailable = errors.New("供应商未配置、已停用或密钥不可用")
var ErrConflict = errors.New("数据已更新，请刷新后重试")

type Service struct {
	db         *gorm.DB
	cfg        config.Config
	dir        string
	cipher     cipher.AEAD
	provider   Provider
	wg         sync.WaitGroup
	imageSlots chan struct{}
}

func New(db *gorm.DB, cfg config.Config) *Service {
	return NewWithProvider(db, cfg, OpenAIProvider{})
}
func NewWithProvider(db *gorm.DB, cfg config.Config, provider Provider) *Service {
	return &Service{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), cfg: cfg, dir: cfg.AIImageDir, provider: provider, imageSlots: make(chan struct{}, 2)}
}
func (s *Service) Database() *gorm.DB { return s.db }

// The secret file and all images live outside every public media mount. Init
// migrates feature tables only; tests and migration tools use isolated databases.
func (s *Service) Init(ctx context.Context) error {
	if s.dir == "" {
		return errors.New("AI private directory is missing")
	}
	private, err := canonicalPath(s.dir)
	if err != nil {
		return err
	}
	for _, public := range []string{s.cfg.UploadDir, s.cfg.ThumbDir, s.cfg.ImagesDir} {
		if public == "" {
			continue
		}
		abs, err := canonicalPath(public)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(abs, private)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return errors.New("AI private directory cannot be inside a public media directory")
		}
	}
	if err = os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	if _, statErr := os.Stat(filepath.Join(s.dir, ".encryption-key")); errors.Is(statErr, os.ErrNotExist) && s.db.Migrator().HasTable(&ProviderConfig{}) {
		var saved int64
		if err := s.db.Model(&ProviderConfig{}).Where("cipher_key <> ?", "").Count(&saved).Error; err != nil {
			return err
		}
		if saved > 0 {
			return errors.New("AI encryption key is missing; restore it with the database backup")
		}
		// Removing a BYOK config does not remove encrypted keys from old task snapshots.
		if s.db.Migrator().HasTable(&Task{}) {
			if err := s.db.Model(&Task{}).Count(&saved).Error; err != nil {
				return err
			}
			if saved > 0 {
				return errors.New("AI encryption key is missing; restore it with the task database backup")
			}
		}
	}
	s.cipher, err = loadCipher(s.dir)
	if err != nil {
		return err
	}
	if err = s.db.WithContext(ctx).AutoMigrate(&Session{}, &Account{}, &Task{}, &Ledger{}, &Asset{}, &ProviderConfig{}, &Idea{}, &IdeaEvent{}); err != nil {
		return err
	}
	var count int64
	if err = s.db.Model(&ProviderConfig{}).Where("user_id = 0").Count(&count).Error; err != nil {
		return err
	}
	if count == 0 && s.cfg.AIImageBaseURL != "" && s.cfg.AIImageAPIKey != "" && s.cfg.AIImageModel != "" {
		_, err = s.SaveProvider(ctx, 0, ProviderConfig{Name: "站点生图", Endpoint: s.cfg.AIImageBaseURL, Model: s.cfg.AIImageModel, Enabled: true, Default: true, Sizes: `["1024x1024"]`, Qualities: `[]`, MaxCount: 1, EditTransport: "none", TimeoutSeconds: 180, Concurrency: 1, Cost: 1, QualityCosts: `{}`}, s.cfg.AIImageAPIKey)
	}
	return err
}

// Resolve existing parents as well as junctions/symlinks before creating a new
// directory. A lexical sibling must not secretly resolve into a public mount.
func canonicalPath(path string) (string, error) {
	current, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	suffix := ""
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Join(resolved, suffix), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		suffix = filepath.Join(filepath.Base(current), suffix)
		current = parent
	}
}

type prepareOptions struct {
	historical     bool
	allowNoSiteKey bool
}

func (s *Service) prepare(ctx context.Context, uid uint64, input Input, options prepareOptions) (taskSnapshot, int64, error) {
	input.Prompt = strings.TrimSpace(input.Prompt)
	if uid == 0 || len([]rune(input.Prompt)) < 1 || len([]rune(input.Prompt)) > 8000 {
		return taskSnapshot{}, 0, ErrInvalid
	}
	p, err := s.providerFor(ctx, uid, input)
	if options.allowNoSiteKey && input.Mode == "site" && errors.Is(err, ErrUnavailable) {
		query := s.db.WithContext(ctx).Where("user_id = 0 AND enabled = ?", true)
		if input.ProviderID != 0 {
			query = query.Where("id = ?", input.ProviderID)
		} else {
			query = query.Order("`default` desc,id asc")
		}
		err = query.First(&p).Error
	}
	if err != nil {
		return taskSnapshot{}, 0, err
	}
	if !options.historical && input.ProviderRevision != 0 && input.ProviderRevision != p.Revision {
		return taskSnapshot{}, 0, ErrConflict
	}
	input.ProviderID = p.ID
	if !contains(stringList(p.Sizes), input.Size) || input.Count < 1 || input.Count > p.MaxCount || (input.Quality != "" && !contains(stringList(p.Qualities), input.Quality)) {
		return taskSnapshot{}, 0, ErrInvalid
	}
	refs, files, seen := []Asset{}, []string{}, map[uint64]bool{}
	digests := map[string]bool{}
	var totalBytes int64
	clean := []Reference{}
	for _, ref := range input.References {
		if seen[ref.AssetID] {
			continue
		}
		seen[ref.AssetID] = true
		if len([]rune(ref.Note)) > 300 {
			return taskSnapshot{}, 0, ErrInvalid
		}
		var asset Asset
		q := s.db.WithContext(ctx).Where("id = ? AND (user_id = ? OR public = ?)", ref.AssetID, uid, true)
		if !options.historical {
			q = q.Where("archived = ?", false)
		}
		if q.First(&asset).Error != nil {
			return taskSnapshot{}, 0, ErrInvalid
		}
		if asset.Public && asset.UserID != uid && !asset.DownloadAllowed {
			return taskSnapshot{}, 0, ErrInvalid
		}
		if asset.Digest != "" {
			if digests[asset.Digest] {
				if ref.AssetID == input.ParentAssetID {
					for index, existing := range refs {
						if existing.Digest == asset.Digest {
							refs[index] = asset
							files[index] = asset.FileName
							clean[index] = ref
							break
						}
					}
				}
				continue
			}
			digests[asset.Digest] = true
		}
		path, err := s.AssetPath(asset)
		if err != nil {
			return taskSnapshot{}, 0, err
		}
		if _, err = os.Stat(path); err != nil {
			return taskSnapshot{}, 0, ErrInvalid
		}
		totalBytes += asset.Bytes
		if totalBytes > 32<<20 {
			return taskSnapshot{}, 0, errors.New("参考图总大小不能超过 32MB")
		}
		refs = append(refs, asset)
		files = append(files, asset.FileName)
		clean = append(clean, ref)
	}
	input.References = clean
	if len(refs) > p.MaxReferences || (len(refs) > 0 && p.EditTransport == "none") {
		return taskSnapshot{}, 0, errors.New("参考图数量超出供应商已配置能力")
	}
	if input.ParentAssetID != 0 {
		found := false
		for _, ref := range refs {
			if ref.ID == input.ParentAssetID && ref.UserID == uid && ref.TaskID != 0 {
				found = true
			}
		}
		if !found {
			return taskSnapshot{}, 0, ErrInvalid
		}
	}
	cost := p.Cost
	var prices map[string]int64
	_ = json.Unmarshal([]byte(p.QualityCosts), &prices)
	if v := prices[input.Quality]; v > 0 {
		cost = v
	}
	cost *= int64(input.Count)
	if input.Mode == "byok" {
		cost = 0
	}
	return taskSnapshot{Input: input, Provider: p, CipherKey: p.CipherKey, References: refs, Files: files}, cost, nil
}

func lockAccount(tx *gorm.DB, uid uint64) (Account, error) {
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Account{UserID: uid}).Error; err != nil {
		return Account{}, err
	}
	var account Account
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&account, "user_id = ?", uid).Error
	return account, err
}
func (s *Service) Quote(ctx context.Context, uid uint64, input Input) (int64, error) {
	_, cost, err := s.prepare(ctx, uid, input, prepareOptions{})
	return cost, err
}

func (s *Service) Submit(ctx context.Context, uid uint64, input Input) (Task, error) {
	if len(input.Key) < 16 || len(input.Key) > 80 || input.SessionID == 0 {
		return Task{}, ErrInvalid
	}
	var existing Task
	if s.db.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", uid, input.Key).First(&existing).Error == nil {
		return existing, nil
	}
	snapshot, cost, err := s.prepare(ctx, uid, input, prepareOptions{})
	if err != nil {
		return Task{}, err
	}
	if input.ExpectedCost != cost {
		return Task{}, ErrConflict
	}
	var result Task
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		account, err := lockAccount(tx, uid)
		if err != nil {
			return err
		}
		if tx.Where("user_id = ? AND idempotency_key = ?", uid, input.Key).First(&result).Error == nil {
			return nil
		}
		var session Session
		if tx.Where("id = ? AND user_id = ?", input.SessionID, uid).First(&session).Error != nil {
			return ErrInvalid
		}
		result, err = s.insertTask(tx, uid, account, snapshot, cost, 0)
		return err
	})
	if err != nil && s.db.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", uid, input.Key).First(&existing).Error == nil {
		return existing, nil
	}
	return result, err
}

func (s *Service) insertTask(tx *gorm.DB, uid uint64, account Account, snap taskSnapshot, cost int64, ideaID uint64) (Task, error) {
	var recent int64
	if err := tx.Model(&Task{}).Where("user_id = ? AND created_at >= ?", uid, time.Now().Add(-time.Hour)).Count(&recent).Error; err != nil {
		return Task{}, err
	}
	if recent >= 20 {
		return Task{}, ErrBusy
	}
	var pending int64
	if err := tx.Model(&Task{}).Where("user_id = ? AND status IN ?", uid, []string{"queued", "running", "unknown"}).Count(&pending).Error; err != nil {
		return Task{}, err
	}
	if pending >= 2 {
		return Task{}, ErrBusy
	}
	if account.Available < cost {
		return Task{}, ErrQuota
	}
	p := snap.Provider
	group := p.ID
	if p.AllowedProviderID != 0 {
		group = p.AllowedProviderID
	}
	task := Task{UserID: uid, SessionID: snap.Input.SessionID, IdempotencyKey: snap.Input.Key, Prompt: snap.Input.Prompt, Size: snap.Input.Size, Quality: snap.Input.Quality, Count: snap.Input.Count, Model: p.Model, Mode: snap.Input.Mode, Status: "queued", Cost: cost, Snapshot: encode(snap), ProviderID: p.ID, DispatchProviderID: group, ParentAssetID: snap.Input.ParentAssetID, IdeaID: ideaID}
	for _, ref := range snap.References {
		if ref.ID == task.ParentAssetID {
			task.ParentTaskID = ref.TaskID
		}
	}
	if err := tx.Create(&task).Error; err != nil {
		return task, err
	}
	if cost == 0 {
		return task, nil
	}
	if err := tx.Model(&Account{}).Where("user_id = ?", uid).Updates(map[string]any{"available": account.Available - cost, "reserved": account.Reserved + cost}).Error; err != nil {
		return task, err
	}
	return task, tx.Create(&Ledger{UserID: uid, TaskID: task.ID, Key: fmt.Sprintf("reserve:%d", task.ID), Kind: "reserve", Amount: -cost}).Error
}

func (s *Service) Grant(ctx context.Context, actor, uid uint64, amount int64, key, note string) error {
	if uid == 0 || amount < 1 || amount > 10000 || len(key) < 16 || len(key) > 80 || len([]rune(note)) > 300 {
		return ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user store.User
		if tx.First(&user, uid).Error != nil {
			return ErrInvalid
		}
		account, err := lockAccount(tx, uid)
		if err != nil {
			return err
		}
		ledgerKey := fmt.Sprintf("grant:%d:%s", actor, key)
		var prior Ledger
		if tx.Where("`key` = ?", ledgerKey).First(&prior).Error == nil {
			if prior.UserID != uid || prior.Amount != amount {
				return ErrConflict
			}
			return nil
		}
		if err = tx.Model(&Account{}).Where("user_id = ?", uid).Update("available", account.Available+amount).Error; err != nil {
			return err
		}
		return tx.Create(&Ledger{UserID: uid, ActorID: actor, Key: ledgerKey, Kind: "grant", Amount: amount, Note: note}).Error
	})
}

func (s *Service) Start(ctx context.Context) {
	for i := 0; i < 2; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				if ctx.Err() != nil {
					return
				}
				s.recoverStale(ctx)
				s.runOne(ctx)
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
}
func (s *Service) Close() { s.wg.Wait() }

func (s *Service) recoverStale(ctx context.Context) {
	var rows []Task
	if s.db.WithContext(ctx).Where("status = ? AND updated_at < ?", "running", time.Now().Add(-10*time.Minute)).Limit(20).Find(&rows).Error != nil {
		return
	}
	for _, row := range rows {
		status, message, assets, err := s.readReceipt(row)
		if err == nil {
			if s.complete(row, "running", status, message, assets, status == "succeeded", 0) == nil {
				s.removeReceipt(row.ID)
			}
		} else {
			_ = s.complete(row, "running", "unknown", "服务中断；结果不明，额度仍预留", nil, false, 0)
		}
	}
}
func (s *Service) claim(ctx context.Context) (Task, error) {
	var candidates []Task
	if err := s.db.WithContext(ctx).Where("status = ?", "queued").Order("id asc").Limit(20).Find(&candidates).Error; err != nil {
		return Task{}, err
	}
	for _, task := range candidates {
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var provider ProviderConfig
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&provider, task.DispatchProviderID).Error; err != nil {
				return err
			}
			var running int64
			if err := tx.Model(&Task{}).Where("dispatch_provider_id = ? AND status = ?", task.DispatchProviderID, "running").Count(&running).Error; err != nil {
				return err
			}
			if running >= int64(provider.Concurrency) {
				return ErrBusy
			}
			result := tx.Model(&Task{}).Where("id = ? AND status = ?", task.ID, "queued").Update("status", "running")
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrBusy
			}
			return nil
		})
		if err == nil {
			task.Status = "running"
			return task, nil
		}
	}
	return Task{}, gorm.ErrRecordNotFound
}
func (s *Service) runOne(ctx context.Context) {
	task, err := s.claim(ctx)
	if err != nil {
		return
	}
	var owner store.User
	if s.db.WithContext(ctx).First(&owner, task.UserID).Error != nil || owner.IsBanned == 1 {
		_ = s.complete(task, "running", "failed", "账号不可用，生成未执行", nil, false, 0)
		return
	}
	if task.IdeaID != 0 {
		var idea Idea
		if s.db.WithContext(ctx).First(&idea, task.IdeaID).Error != nil || idea.Status == "hidden" || idea.Status == "withdrawn" {
			_ = s.complete(task, "running", "cancelled", "创意已撤回或下架，生成未执行", nil, false, 0)
			return
		}
	}
	var snap taskSnapshot
	if json.Unmarshal([]byte(task.Snapshot), &snap) != nil {
		_ = s.complete(task, "running", "failed", "任务快照无法读取", nil, false, 0)
		return
	}
	key, err := s.decrypt(snap.CipherKey)
	if err != nil {
		_ = s.complete(task, "running", "failed", "供应商密钥无法读取", nil, false, 0)
		return
	}
	files := []string{}
	for _, name := range snap.Files {
		path, err := s.AssetPath(Asset{FileName: name})
		if err != nil {
			_ = s.complete(task, "running", "failed", "参考图不可读取", nil, false, 0)
			return
		}
		files = append(files, path)
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(snap.Provider.TimeoutSeconds)*time.Second)
	images, err := s.provider.Generate(requestCtx, snap.Provider, key, snap.Input, files)
	cancel()
	if err != nil && len(images) == 0 {
		var known KnownFailure
		if errors.As(err, &known) {
			_ = s.complete(task, "running", "failed", known.Message, nil, false, 0)
		} else {
			_ = s.complete(task, "running", "unknown", "供应商结果不明；额度仍预留，请联系管理员核查", nil, false, 0)
		}
		return
	}
	status, message := "succeeded", ""
	if err != nil || len(images) != task.Count {
		status, message = "unknown", "供应商只返回部分结果；已保存可用图片，额度仍预留"
	}
	assets := []Asset{}
	for _, img := range images {
		asset, err := s.saveImage(bytes.NewReader(img.Bytes), task.UserID, task.ID)
		if err != nil {
			status, message = "unknown", "供应商已完成，但部分图片保存失败；额度仍预留"
			break
		}
		assets = append(assets, asset)
	}
	if err := s.writeReceipt(task, status, message, assets); err != nil {
		slog.Warn("AI result recovery receipt unavailable", "task_id", task.ID)
	}
	if err = s.complete(task, "running", status, message, assets, status == "succeeded", 0); err != nil {
		slog.Error("AI result transaction failed", "task_id", task.ID)
	} else {
		s.removeReceipt(task.ID)
	}
}

// Every completion locks idea -> account -> task, matching sponsorship and
// cancellation. State comparison and unique ledger keys make settlement idempotent.
func (s *Service) complete(task Task, expected, status, message string, assets []Asset, consumed bool, actor uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var idea Idea
		if task.IdeaID != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&idea, task.IdeaID).Error; err != nil {
				return err
			}
		}
		account, err := lockAccount(tx, task.UserID)
		if err != nil {
			return err
		}
		var current Task
		if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, task.ID).Error; err != nil {
			return err
		}
		if current.Status != expected {
			return ErrConflict
		}
		if len(assets) > 0 {
			for i := range assets {
				if err = tx.Create(&assets[i]).Error; err != nil {
					return err
				}
			}
		}
		if err = tx.Model(&current).Updates(map[string]any{"status": status, "error": message}).Error; err != nil {
			return err
		}
		if status != "unknown" && current.Cost > 0 {
			if account.Reserved < current.Cost {
				return errors.New("额度预留记录不一致")
			}
			available, kind, amount := account.Available, "settle", int64(0)
			if !consumed {
				available += current.Cost
				kind, amount = "release", current.Cost
			}
			if err = tx.Model(&Account{}).Where("user_id = ?", current.UserID).Updates(map[string]any{"available": available, "reserved": account.Reserved - current.Cost}).Error; err != nil {
				return err
			}
			if err = tx.Create(&Ledger{UserID: current.UserID, TaskID: current.ID, ActorID: actor, Key: fmt.Sprintf("finish:%d", current.ID), Kind: kind, Amount: amount, Note: message}).Error; err != nil {
				return err
			}
		}
		if task.IdeaID != 0 {
			ideaStatus := idea.Status
			if idea.Status != "hidden" && idea.Status != "withdrawn" {
				switch status {
				case "succeeded":
					ideaStatus = "realized"
				case "unknown":
					ideaStatus = "unknown"
				default:
					if !consumed {
						ideaStatus = "waiting"
					} else {
						ideaStatus = "unavailable"
					}
				}
			}
			if err = tx.Model(&idea).Update("status", ideaStatus).Error; err != nil {
				return err
			}
			return tx.Create(&IdeaEvent{IdeaID: idea.ID, TaskID: task.ID, ActorID: actor, Action: status, Note: message}).Error
		}
		return nil
	})
}

func (s *Service) Cancel(ctx context.Context, uid, id uint64) error {
	var task Task
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, uid).First(&task).Error; err != nil {
		return err
	}
	return s.complete(task, "queued", "cancelled", "用户取消排队任务", nil, false, uid)
}
func (s *Service) Resolve(ctx context.Context, actor, id uint64, consumed bool, note string) error {
	if len([]rune(strings.TrimSpace(note))) < 3 || len([]rune(note)) > 300 {
		return ErrInvalid
	}
	var task Task
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return err
	}
	return s.complete(task, "unknown", "failed", note, nil, consumed, actor)
}

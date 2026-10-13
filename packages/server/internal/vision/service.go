package vision

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/cache"
	"github.com/huntersxy/xqecz/server/internal/contenttags"
	"github.com/huntersxy/xqecz/server/internal/store"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Job is a transactional outbox: upload and its generation intent commit together.
// Leases survive crashes; the claim token prevents an expired worker overwriting a newer attempt.
type Job struct {
	ContentID     uint64    `gorm:"primaryKey;autoIncrement:false" json:"content_id"`
	Status        string    `gorm:"size:20;index:vision_due" json:"status"`
	Attempts      int       `json:"attempts"`
	NextAt        time.Time `gorm:"index:vision_due" json:"next_at"`
	Lease         string    `gorm:"size:64" json:"-"`
	ExpectedTitle string    `gorm:"size:200" json:"-"`
	ExpectedTags  string    `gorm:"type:text" json:"-"`
	FilePath      string    `gorm:"type:text" json:"-"`
	Error         string    `gorm:"type:text" json:"error"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (Job) TableName() string { return "vision_jobs" }

func NewJob(row store.Content) Job {
	file := ""
	if row.FilePath != nil {
		file = *row.FilePath
	}
	return Job{ContentID: row.ID, Status: "pending", NextAt: time.Now().UTC(), ExpectedTitle: row.Title, ExpectedTags: row.Tags, FilePath: file}
}

type Service struct {
	db           *gorm.DB
	redis        *cache.Client
	uploadDir    string
	referenceDir string
	stop         context.CancelFunc
	done         chan struct{}
}

func New(db *gorm.DB, redis *cache.Client, uploadDir string, referenceDir ...string) *Service {
	// Settings contain a secret: never let GORM interpolate their payload into a logged SQL statement.
	dir := filepath.Join(filepath.Dir(uploadDir), "vision-references")
	if len(referenceDir) > 0 {
		dir = referenceDir[0]
	}
	return &Service{db: db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), redis: redis, uploadDir: uploadDir, referenceDir: dir}
}

// Init migrates only the two feature-owned tables, never the existing application schema.
func (s *Service) Init(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return s.db.WithContext(ctx).AutoMigrate(&settingRow{}, &Job{})
}

func (s *Service) config(ctx context.Context) (Config, error) {
	var row settingRow
	err := s.db.WithContext(ctx).First(&row, 1).Error
	if isMissing(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	// Explicit wire envelope keeps APIKey out of Config's public JSON representation.
	var payload struct {
		Config Config `json:"config"`
		Key    string `json:"key"`
	}
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		return Config{}, err
	}
	payload.Config.APIKey = payload.Key
	return payload.Config, nil
}

func (s *Service) Settings(ctx context.Context) (Settings, error) {
	c, err := s.config(ctx)
	out := publicSettings(c)
	out.HasAPIKey = c.APIKey != ""
	return out, err
}

func (s *Service) SaveSettings(ctx context.Context, req SettingsUpdate) (Settings, error) {
	old, err := s.config(ctx)
	if err != nil {
		return Settings{}, err
	}
	c := req.Config
	c.APIKey = old.APIKey
	if req.Key != nil && strings.TrimSpace(*req.Key) != "" {
		c.APIKey = strings.TrimSpace(*req.Key)
	}
	if req.ClearAPIKey {
		c.APIKey = ""
	}
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.Model = strings.TrimSpace(c.Model)
	if err := validateConfig(c); err != nil {
		return Settings{}, err
	}
	payload, _ := json.Marshal(struct {
		Config Config `json:"config"`
		Key    string `json:"key"`
	}{c, c.APIKey})
	row := settingRow{ID: 1, Payload: string(payload)}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error; err != nil {
		return Settings{}, errors.New("视觉配置保存失败")
	}
	return s.Settings(ctx)
}

func (s *Service) Ready(ctx context.Context) bool {
	c, err := s.config(ctx)
	return err == nil && c.Enabled && c.APIKey != ""
}

func (s *Service) Jobs(ctx context.Context) ([]Job, error) {
	rows := []Job{}
	err := s.db.WithContext(ctx).Order("created_at DESC").Limit(30).Find(&rows).Error
	return rows, err
}

func (s *Service) Retry(ctx context.Context, id uint64) error {
	r := s.db.WithContext(ctx).Model(&Job{}).Where("content_id = ? AND status = ?", id, "failed").Updates(map[string]any{"status": "pending", "attempts": 0, "next_at": time.Now().UTC(), "lease": "", "error": ""})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return errors.New("仅失败任务可重试")
	}
	return nil
}

func duration(seconds int) time.Duration { return time.Duration(seconds) * time.Second }

func (s *Service) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.stop = cancel
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.tick(ctx); err != nil && ctx.Err() == nil {
					slog.Warn("视觉任务执行失败", "err", err)
				}
			}
		}
	}()
}
func (s *Service) Close() {
	if s.stop != nil {
		s.stop()
		<-s.done
	}
}

func (s *Service) tick(ctx context.Context) error {
	cfg, err := s.config(ctx)
	if err != nil {
		return err
	}
	if !cfg.Enabled || cfg.APIKey == "" {
		return nil
	}
	now := time.Now().UTC()
	// A repeatedly crashed task also reaches a terminal state instead of cycling forever.
	if err := s.db.WithContext(ctx).Model(&Job{}).Where("status = ? AND next_at <= ? AND attempts >= 3", "running", now).
		Updates(map[string]any{"status": "failed", "error": "任务多次中断，可由管理员重试", "lease": ""}).Error; err != nil {
		return err
	}
	var job Job
	err = s.db.WithContext(ctx).Where("status IN ? AND next_at <= ? AND attempts < 3", []string{"pending", "running"}, now).Order("next_at ASC").First(&job).Error
	if isMissing(err) {
		return nil
	}
	if err != nil {
		return err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)
	r := s.db.WithContext(ctx).Model(&Job{}).Where("content_id = ? AND status = ? AND lease = ? AND next_at <= ?", job.ContentID, job.Status, job.Lease, now).
		Updates(map[string]any{"status": "running", "lease": token, "attempts": job.Attempts + 1, "next_at": now.Add(5 * time.Minute)})
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return nil
	}
	job.Lease = token
	job.Attempts++
	result, err := s.generateJob(ctx, cfg, job)
	if ctx.Err() != nil {
		return ctx.Err()
	} // Keep the lease; restart safely recovers after expiry.
	if err == nil {
		return s.complete(ctx, job, result)
	}
	status := "pending"
	if errors.Is(err, errConflict) {
		status = "cancelled"
	} else if job.Attempts >= 3 {
		status = "failed"
	}
	return s.db.WithContext(ctx).Model(&Job{}).Where("content_id = ? AND lease = ? AND status = ?", job.ContentID, job.Lease, "running").
		Updates(map[string]any{"status": status, "error": err.Error(), "lease": "", "next_at": time.Now().UTC().Add(duration(job.Attempts * 15))}).Error
}

func matches(row store.Content, job Job) bool {
	return row.FilePath != nil && *row.FilePath == job.FilePath && row.Title == job.ExpectedTitle && row.Tags == job.ExpectedTags
}

func (s *Service) generateJob(ctx context.Context, cfg Config, job Job) (Result, error) {
	var row store.Content
	err := s.db.WithContext(ctx).First(&row, job.ContentID).Error
	if isMissing(err) {
		return Result{}, errConflict
	}
	if err != nil {
		return Result{}, errors.New("内容读取失败")
	}
	if !matches(row, job) {
		return Result{}, errConflict
	}
	// FilePath originates in savePart, but reject traversal before reading even if the DB was edited externally.
	if filepath.Base(job.FilePath) != job.FilePath || strings.ContainsAny(job.FilePath, "/\\") {
		return Result{}, errors.New("原图路径无效")
	}
	ctx, cancel := context.WithTimeout(ctx, duration(cfg.TimeoutSeconds))
	defer cancel()
	url, err := imageDataURL(ctx, filepath.Join(s.uploadDir, job.FilePath))
	if err != nil {
		return Result{}, err
	}
	vocabulary, err := s.vocabulary(ctx)
	if err != nil {
		return Result{}, errors.New("已有标签读取失败")
	}
	refs := []Reference{}
	if cfg.UseReferences {
		refs, err = s.References(ctx)
		if err != nil {
			return Result{}, err
		}
	}
	return Generate(ctx, cfg, url, vocabulary, refs...)
}

func (s *Service) vocabulary(ctx context.Context) ([]string, error) {
	var raw []string
	if err := s.db.WithContext(ctx).Model(&store.Content{}).Where("audit_status = ?", "approved").Order("id DESC").Limit(500).Pluck("tags", &raw).Error; err != nil {
		return nil, err
	}
	freq := map[string]int{}
	for _, v := range raw {
		var tags []string
		if json.Unmarshal([]byte(v), &tags) != nil {
			tags = []string{v}
		}
		for _, t := range NormalizeTags(tags, nil, 12) {
			freq[t]++
		}
	}
	out := []string{}
	for t := range freq {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if freq[out[i]] == freq[out[j]] {
			return out[i] < out[j]
		}
		return freq[out[i]] > freq[out[j]]
	})
	if len(out) > 120 {
		out = out[:120]
	}
	return out, nil
}

func (s *Service) complete(ctx context.Context, job Job, result Result) error {
	var declared []string
	if err := json.Unmarshal([]byte(job.ExpectedTags), &declared); err != nil {
		return errors.New("任务原标签格式无效")
	}
	tags, _ := json.Marshal(contenttags.MergeGenerated(result.Tags, declared))
	changed := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Match the lock order of content edit/delete: content first, then its outbox job.
		var row store.Content
		rowErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, job.ContentID).Error
		if rowErr != nil && !isMissing(rowErr) {
			return rowErr
		}
		var claimed Job
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("content_id = ? AND status = ? AND lease = ?", job.ContentID, "running", job.Lease).First(&claimed).Error; err != nil {
			if isMissing(err) {
				return nil
			}
			return err
		}
		// Lock and compare in Go: MySQL's default collation is case-insensitive, while a human edit must match byte-for-byte.
		if rowErr == nil && matches(row, job) {
			r := tx.Model(&store.Content{}).Where("id = ?", job.ContentID).Updates(map[string]any{"title": result.Title, "tags": string(tags)})
			if r.Error != nil {
				return r.Error
			}
			changed = r.RowsAffected > 0
		}
		status, message := "succeeded", ""
		if !changed {
			status, message = "cancelled", errConflict.Error()
		}
		return tx.Model(&Job{}).Where("content_id = ? AND lease = ?", job.ContentID, job.Lease).Updates(map[string]any{"status": status, "error": message, "lease": ""}).Error
	})
	if err != nil {
		return fmt.Errorf("生成结果保存失败")
	}
	if changed && s.redis != nil {
		s.redis.ClearContentCache(ctx, job.ContentID)
		s.redis.ClearContentListCache(ctx)
		s.redis.Del(ctx, "admin:dashboard")
	}
	return nil
}

// Test runs the same vision/JSON/tag pipeline on an inline sample without writing content or jobs.
func (s *Service) Test(ctx context.Context) (Result, error) {
	cfg, err := s.config(ctx)
	if err != nil {
		return Result{}, errors.New("配置读取失败")
	}
	if cfg.APIKey == "" {
		return Result{}, errUnavailable
	}
	img := imageSample()
	return Generate(ctx, cfg, img, nil)
}

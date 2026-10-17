// Package aigen owns durable generation, immutable images and transactional credit accounting.
package aigen

import "time"

type Task struct {
	ID                 uint64    `gorm:"primaryKey" json:"id"`
	UserID             uint64    `gorm:"index;not null;uniqueIndex:ux_ai_submit,priority:1;index:idx_ai_user_time,priority:1" json:"user_id"`
	SessionID          uint64    `gorm:"index" json:"session_id"`
	IdempotencyKey     string    `gorm:"size:80;not null;uniqueIndex:ux_ai_submit,priority:2" json:"-"`
	Prompt             string    `gorm:"type:text" json:"prompt"`
	Size               string    `gorm:"size:32" json:"size"`
	Quality            string    `gorm:"size:32" json:"quality"`
	Count              int       `json:"count"`
	Model              string    `gorm:"size:160" json:"model"`
	Mode               string    `gorm:"size:16" json:"mode"`
	Status             string    `gorm:"size:20;index" json:"status"`
	Archived           bool      `gorm:"index" json:"-"`
	Cost               int64     `json:"cost"`
	DailyCost          int64     `gorm:"not null;default:0" json:"-"`
	Error              string    `gorm:"size:500" json:"error"`
	Snapshot           string    `gorm:"type:longtext" json:"-"`
	ProviderID         uint64    `gorm:"index" json:"provider_id"`
	DispatchProviderID uint64    `gorm:"index" json:"-"`
	ParentTaskID       uint64    `json:"parent_task_id"`
	ParentAssetID      uint64    `json:"parent_asset_id"`
	IdeaID             uint64    `gorm:"index" json:"idea_id"`
	CreatedAt          time.Time `gorm:"index:idx_ai_user_time,priority:2" json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (Task) TableName() string { return "ai_generation_tasks" }

type Session struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"index" json:"-"`
	Name      string    `gorm:"size:100" json:"name"`
	Draft     string    `gorm:"type:text" json:"draft"`
	Version   uint64    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Session) TableName() string { return "ai_sessions" }

type Account struct {
	UserID    uint64 `gorm:"primaryKey" json:"user_id"`
	Available int64  `json:"available"`
	Reserved  int64  `json:"reserved"`
	// Daily balances are a subset of Available/Reserved; existing credits remain uncapped.
	DailyAvailable int64     `gorm:"not null;default:0" json:"daily_available"`
	DailyReserved  int64     `gorm:"not null;default:0" json:"daily_reserved"`
	LastLoginDay   string    `gorm:"size:10" json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

func (Account) TableName() string { return "ai_quota_accounts" }

type Ledger struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	UserID    uint64    `gorm:"index" json:"user_id"`
	TaskID    uint64    `gorm:"index" json:"task_id"`
	ActorID   uint64    `json:"actor_id"`
	Key       string    `gorm:"size:160;uniqueIndex" json:"-"`
	Kind      string    `gorm:"size:24" json:"kind"`
	Amount    int64     `json:"amount"`
	Note      string    `gorm:"size:300" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (Ledger) TableName() string { return "ai_quota_ledger" }

// Image bytes and filenames never change. Archive removes a library entry,
// while existing tasks keep their validated reference snapshots readable.
type Asset struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	TaskID          uint64    `gorm:"index" json:"task_id"`
	UserID          uint64    `gorm:"index;index:idx_ai_asset_digest,priority:1" json:"user_id"`
	FileName        string    `gorm:"size:100" json:"-"`
	PreviewFileName string    `gorm:"size:120" json:"-"`
	Digest          string    `gorm:"size:64;index:idx_ai_asset_digest,priority:2" json:"-"`
	Mime            string    `gorm:"size:40" json:"mime"`
	Bytes           int64     `json:"bytes"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	Title           string    `gorm:"size:150" json:"title"`
	Description     string    `gorm:"type:text" json:"description"`
	License         string    `gorm:"size:300" json:"license"`
	Purpose         string    `gorm:"size:80" json:"purpose"`
	Public          bool      `gorm:"index" json:"public"`
	DownloadAllowed bool      `json:"download_allowed"`
	Archived        bool      `gorm:"index" json:"archived"`
	ContentID       *uint64   `json:"content_id"`
	CreatedAt       time.Time `json:"created_at"`
}

func (Asset) TableName() string { return "ai_assets" }

type ProviderConfig struct {
	ID                uint64 `gorm:"primaryKey" json:"id"`
	UserID            uint64 `gorm:"index" json:"-"`
	AllowedProviderID uint64 `json:"allowed_provider_id"`
	Name              string `gorm:"size:100" json:"name"`
	// Adapter selects the wire protocol while Endpoint remains the complete
	// provider URL. Empty values are read as the legacy OpenAI Images adapter.
	Adapter   string `gorm:"size:40" json:"adapter"`
	Endpoint  string `gorm:"size:500" json:"endpoint"`
	Model     string `gorm:"size:160" json:"model"`
	CipherKey string `gorm:"type:text" json:"-"`
	Enabled   bool   `json:"enabled"`
	Default   bool   `json:"default"`
	// Sizes remains in the schema for old clients and snapshots. Generation
	// accepts the user's validated size (or auto) independently of this field.
	Sizes          string `gorm:"type:text" json:"sizes"`
	Qualities      string `gorm:"type:text" json:"qualities"`
	MaxCount       int    `json:"max_count"`
	MaxReferences  int    `json:"max_references"`
	EditTransport  string `gorm:"size:32" json:"edit_transport"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	Concurrency    int    `json:"concurrency"`
	Cost           int64  `json:"cost"`
	QualityCosts   string `gorm:"type:text" json:"quality_costs"`
	// Options contains non-secret adapter-specific JSON. API keys stay in
	// CipherKey so snapshots and public views never expose credentials.
	Options   string    `gorm:"type:text" json:"options,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Revision  uint64    `json:"revision"`
}

func (ProviderConfig) TableName() string { return "ai_provider_configs" }

type Idea struct {
	ID                uint64    `gorm:"primaryKey" json:"id"`
	UserID            uint64    `gorm:"index" json:"user_id"`
	Title             string    `gorm:"size:150" json:"title"`
	Description       string    `gorm:"type:text" json:"description"`
	Snapshot          string    `gorm:"type:text" json:"snapshot"`
	ReferenceSnapshot string    `gorm:"type:text" json:"-"`
	Model             string    `gorm:"size:160" json:"model"`
	CharacterIDs      string    `gorm:"type:text" json:"character_ids"`
	Status            string    `gorm:"size:24;index" json:"status"`
	SponsorID         uint64    `json:"sponsor_id"`
	TaskID            uint64    `json:"task_id"`
	EstimatedCost     int64     `json:"estimated_cost"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (Idea) TableName() string { return "ai_ideas" }

type IdeaEvent struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	IdeaID    uint64    `gorm:"index" json:"idea_id"`
	ActorID   uint64    `json:"actor_id"`
	TaskID    uint64    `json:"task_id"`
	Action    string    `gorm:"size:40" json:"action"`
	Note      string    `gorm:"size:300" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (IdeaEvent) TableName() string { return "ai_idea_events" }

type Reference struct {
	AssetID uint64 `json:"asset_id"`
	Note    string `json:"note"`
}
type Input struct {
	SessionID        uint64      `json:"session_id"`
	Key              string      `json:"idempotency_key"`
	Prompt           string      `json:"prompt"`
	ProviderID       uint64      `json:"provider_id"`
	Mode             string      `json:"mode"`
	Size             string      `json:"size"`
	Quality          string      `json:"quality"`
	Count            int         `json:"count"`
	References       []Reference `json:"references"`
	ParentAssetID    uint64      `json:"parent_asset_id"`
	ExpectedCost     int64       `json:"expected_cost"`
	ProviderRevision uint64      `json:"provider_revision"`
	CharacterIDs     []uint64    `json:"character_ids"`
}
type taskSnapshot struct {
	Input      Input          `json:"input"`
	Provider   ProviderConfig `json:"provider"`
	CipherKey  string         `json:"key"`
	References []Asset        `json:"references"`
	Files      []string       `json:"files"`
}

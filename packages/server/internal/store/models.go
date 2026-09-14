package store

import (
	"time"

	"gorm.io/gorm"
)

// 表结构与现有 MySQL 完全一致（bigint 主键、tinyint 布尔、datetime(3)、软删除列）。

type User struct {
	ID        uint64         `gorm:"column:id;primaryKey" json:"id"`
	Username  string         `gorm:"column:username" json:"username"`
	Email     *string        `gorm:"column:email" json:"email"`
	Password  string         `gorm:"column:password" json:"-"`
	IsAdmin   int8           `gorm:"column:is_admin" json:"is_admin"`
	IsBanned  int8           `gorm:"column:is_banned" json:"is_banned"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (User) TableName() string { return "users" }

type Content struct {
	ID            uint64         `gorm:"column:id;primaryKey" json:"id"`
	Title         string         `gorm:"column:title" json:"title"`
	Content       *string        `gorm:"column:content" json:"content"`
	FilePath      *string        `gorm:"column:file_path" json:"file_path"`
	FileSize      int64          `gorm:"column:file_size" json:"file_size"`
	ThumbPath     *string        `gorm:"column:thumb_path" json:"thumb_path"`
	ViewCount     int64          `gorm:"column:view_count" json:"view_count"`
	UserID        uint64         `gorm:"column:user_id" json:"user_id"`
	GuestNickname *string        `gorm:"column:guest_nickname" json:"guest_nickname"`
	GuestEmail    *string        `gorm:"column:guest_email" json:"guest_email"`
	Tags          string         `gorm:"column:tags" json:"tags"`
	AuditStatus   string         `gorm:"column:audit_status" json:"audit_status"`
	CreatedAt     time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Content) TableName() string { return "contents" }

type Comment struct {
	ID        uint64         `gorm:"column:id;primaryKey" json:"id"`
	ContentID uint64         `gorm:"column:content_id" json:"content_id"`
	UserID    uint64         `gorm:"column:user_id" json:"user_id"`
	Text      string         `gorm:"column:text" json:"text"`
	ParentID  *uint64        `gorm:"column:parent_id" json:"parent_id"`
	IsBanned  int8           `gorm:"column:is_banned" json:"is_banned"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Comment) TableName() string { return "comments" }

type Claim struct {
	ID         uint64    `gorm:"column:id;primaryKey" json:"id"`
	ContentID  uint64    `gorm:"column:content_id" json:"content_id"`
	UserID     uint64    `gorm:"column:user_id" json:"user_id"`
	Reason     *string   `gorm:"column:reason" json:"reason"`
	Status     string    `gorm:"column:status" json:"status"`
	ApprovedBy *uint64   `gorm:"column:approved_by" json:"approved_by"`
	Remark     *string   `gorm:"column:remark" json:"remark"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Claim) TableName() string { return "claims" }

type Poll struct {
	ID          uint64         `gorm:"column:id;primaryKey" json:"id"`
	Title       string         `gorm:"column:title" json:"title"`
	Description *string        `gorm:"column:description" json:"description"`
	Options     string         `gorm:"column:options" json:"options"`
	VoteCount   int64          `gorm:"column:vote_count" json:"vote_count"`
	UserID      uint64         `gorm:"column:user_id" json:"user_id"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (Poll) TableName() string { return "polls" }

type PollVote struct {
	ID          uint64    `gorm:"column:id;primaryKey" json:"id"`
	PollID      uint64    `gorm:"column:poll_id" json:"poll_id"`
	UserID      *uint64   `gorm:"column:user_id" json:"user_id"`
	VisitorID   *string   `gorm:"column:visitor_id" json:"visitor_id"`
	OptionIndex int       `gorm:"column:option_index" json:"option_index"`
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
}

func (PollVote) TableName() string { return "poll_votes" }

type ContentLike struct {
	ID        uint64    `gorm:"column:id;primaryKey" json:"id"`
	ContentID uint64    `gorm:"column:content_id" json:"content_id"`
	UserID    uint64    `gorm:"column:user_id" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ContentLike) TableName() string { return "content_likes" }

type ContentFavorite struct {
	ID        uint64    `gorm:"column:id;primaryKey" json:"id"`
	ContentID uint64    `gorm:"column:content_id" json:"content_id"`
	UserID    uint64    `gorm:"column:user_id" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (ContentFavorite) TableName() string { return "content_favorites" }

type CommentReport struct {
	ID        uint64    `gorm:"column:id;primaryKey" json:"id"`
	CommentID uint64    `gorm:"column:comment_id" json:"comment_id"`
	UserID    uint64    `gorm:"column:user_id" json:"user_id"`
	Reason    string    `gorm:"column:reason" json:"reason"`
	Handled   int8      `gorm:"column:handled" json:"handled"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

func (CommentReport) TableName() string { return "comment_reports" }

type APIKey struct {
	ID          uint64         `gorm:"column:id;primaryKey" json:"id"`
	UserID      uint64         `gorm:"column:user_id" json:"user_id"`
	Name        string         `gorm:"column:name" json:"name"`
	KeyPrefix   string         `gorm:"column:key_prefix" json:"key_prefix"`
	KeyHash     string         `gorm:"column:key_hash" json:"-"`
	Permissions string         `gorm:"column:permissions" json:"permissions"`
	IsActive    int8           `gorm:"column:is_active" json:"is_active"`
	LastUsedAt  *time.Time     `gorm:"column:last_used_at" json:"last_used_at"`
	ExpiresAt   *time.Time     `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}

func (APIKey) TableName() string { return "api_keys" }

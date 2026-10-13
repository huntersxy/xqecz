// Package creative owns the editable public catalog; AI task snapshots never
// point at mutable template text or resource versions.
package creative

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Resource struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Kind      string    `gorm:"size:20;uniqueIndex:ux_creative_slug,priority:1" json:"kind"`
	Slug      string    `gorm:"size:80;uniqueIndex:ux_creative_slug,priority:2" json:"slug"`
	Title     string    `gorm:"size:150" json:"title"`
	Summary   string    `gorm:"size:1000" json:"summary"`
	Payload   string    `gorm:"type:text" json:"payload"`
	Status    string    `gorm:"size:20;index" json:"status"`
	SortOrder int       `json:"sort_order"`
	Version   uint64    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Resource) TableName() string { return "creative_resources" }

type Variable struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Default  string   `json:"default"`
	Required bool     `json:"required"`
	Options  []string `json:"options"`
	Help     string   `json:"help"`
}
type Payload struct {
	Description  string         `json:"description"`
	Appearance   string         `json:"appearance"`
	Body         string         `json:"body"`
	AssetIDs     []uint64       `json:"asset_ids"`
	CharacterIDs []uint64       `json:"character_ids"`
	Tags         []string       `json:"tags"`
	Variables    []Variable     `json:"variables"`
	PresetID     uint64         `json:"preset_id"`
	Parameters   map[string]any `json:"parameters"`
}

func Init(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).AutoMigrate(&Resource{})
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
var variablePattern = regexp.MustCompile(`\{([^{}]+)\}`)

func Validate(row Resource) (Payload, error) {
	var p Payload
	if json.Unmarshal([]byte(row.Payload), &p) != nil || !slugPattern.MatchString(row.Slug) || len([]rune(strings.TrimSpace(row.Title))) < 1 || len([]rune(row.Title)) > 150 || len([]rune(row.Summary)) > 1000 || len(row.Payload) > 64<<10 {
		return p, errors.New("资源字段无效")
	}
	if row.Kind != "character" && row.Kind != "template" && row.Kind != "preset" {
		return p, errors.New("资源类型无效")
	}
	if row.Status != "draft" && row.Status != "published" && row.Status != "archived" {
		return p, errors.New("发布状态无效")
	}
	if len(p.AssetIDs) > 40 || len(p.CharacterIDs) > 20 || len(p.Variables) > 20 || len(p.Tags) > 20 {
		return p, errors.New("资源关联过多")
	}
	if row.Kind == "template" {
		if len([]rune(strings.TrimSpace(p.Body))) < 1 || len([]rune(p.Body)) > 8000 {
			return p, errors.New("模板正文无效")
		}
		seen := map[string]bool{}
		for _, v := range p.Variables {
			if v.Name == "" || len([]rune(v.Name)) > 40 || strings.ContainsAny(v.Name, "{}") || seen[v.Name] || (v.Type != "text" && v.Type != "select") || len(v.Options) > 30 {
				return p, errors.New("模板变量无效")
			}
			seen[v.Name] = true
			if v.Type == "select" && len(v.Options) == 0 {
				return p, errors.New("选择变量缺少选项")
			}
		}
		for _, match := range variablePattern.FindAllStringSubmatch(p.Body, -1) {
			if !seen[match[1]] {
				return p, errors.New("模板含未定义变量")
			}
		}
	}
	return p, nil
}

// Expand substitutes literal user values, never executable expressions. Only
// defined variables are accepted; optional empty values stay plain strings.
func Expand(p Payload, values map[string]string) (string, error) {
	expanded := map[string]string{}
	for _, v := range p.Variables {
		value, ok := values[v.Name]
		if !ok {
			value = v.Default
		}
		if v.Required && strings.TrimSpace(value) == "" {
			return "", errors.New("请填写必填变量")
		}
		if len([]rune(value)) > 500 {
			return "", errors.New("变量内容过长")
		}
		if v.Type == "select" && value != "" {
			valid := false
			for _, option := range v.Options {
				if option == value {
					valid = true
				}
			}
			if !valid {
				return "", errors.New("变量选项无效")
			}
		}
		expanded[v.Name] = value
	}
	result := variablePattern.ReplaceAllStringFunc(p.Body, func(token string) string {
		if value, ok := expanded[token[1:len(token)-1]]; ok {
			return value
		}
		return token
	})
	if len([]rune(result)) > 8000 {
		return "", errors.New("展开后的提示词过长")
	}
	return result, nil
}

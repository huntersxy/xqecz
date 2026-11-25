package site

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/huntersxy/xqecz/server/internal/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type FriendLink struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	IconURL string `json:"icon_url"`
}

type Config struct {
	Name         string            `json:"name"`
	ShortName    string            `json:"short_name"`
	Tagline      string            `json:"tagline"`
	BrowserTitle string            `json:"browser_title"`
	Description  string            `json:"description"`
	LogoURL      string            `json:"logo_url"`
	FaviconURL   string            `json:"favicon_url"`
	FriendLinks  []FriendLink      `json:"friend_links"`
	Texts        map[string]string `json:"texts"`
}

type settingRow struct {
	ID        uint      `gorm:"primaryKey"`
	Payload   string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (settingRow) TableName() string { return "site_settings" }

const (
	defaultFavicon = "/favicon.webp"
	maxFriendLinks = 12
)

func Defaults() Config {
	texts := make(map[string]string, len(textElements))
	for _, element := range textElements {
		texts[element.Key] = element.Default
	}
	return Config{
		Name:         "小泉宇宙",
		ShortName:    "小泉宇宙",
		Tagline:      "二创画廊",
		BrowserTitle: "{name} · 小泉动漫二创站",
		Description:  "小泉动漫二创内容分享平台：上传你的作品，点亮属于你的那颗星球。",
		FaviconURL:   defaultFavicon,
		FriendLinks: []FriendLink{{
			Name:    "蓝色大肥鱼",
			URL:     "https://xn--pssy23gqgbz2d718b.com/",
			IconURL: "/friend-links/blue-fat-fish.webp",
		}},
		Texts: texts,
	}
}

func PublicDefaults() Config {
	return resolve(Defaults())
}

type Service struct {
	db        *gorm.DB
	imagesDir string
	mu        sync.RWMutex
	cached    Config
	cacheOK   bool
}

func New(db *gorm.DB, cfg config.Config) *Service {
	return &Service{
		db:        db.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}),
		imagesDir: cfg.ImagesDir,
	}
}

// Init loads the cached config. Schema comes from internal/migrate.
func (s *Service) Init(ctx context.Context) error {
	cfg, err := s.load(ctx)
	if err != nil {
		slog.Warn("site settings load failed, using defaults", "err", err)
		cfg = Defaults()
	}
	s.mu.Lock()
	s.cached = cfg
	s.cacheOK = true
	s.mu.Unlock()
	return nil
}

func (s *Service) load(ctx context.Context) (Config, error) {
	var row settingRow
	err := s.db.WithContext(ctx).First(&row, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Defaults(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var stored Config
	if err := json.Unmarshal([]byte(row.Payload), &stored); err != nil {
		return Config{}, errors.New("站点配置内容损坏")
	}
	return withDefaults(stored), nil
}

func (s *Service) Config(ctx context.Context) (Config, error) {
	s.mu.RLock()
	if s.cacheOK {
		out := clone(s.cached)
		s.mu.RUnlock()
		return out, nil
	}
	s.mu.RUnlock()

	cfg, err := s.load(ctx)
	if err != nil {
		return Config{}, err
	}
	s.mu.Lock()
	s.cached = cfg
	s.cacheOK = true
	s.mu.Unlock()
	return clone(cfg), nil
}

func (s *Service) Public(ctx context.Context) Config {
	cfg, err := s.Config(ctx)
	if err != nil {
		return PublicDefaults()
	}
	return resolve(cfg)
}

func (s *Service) Save(ctx context.Context, in Config) (Config, error) {
	next := normalize(in)
	if err := validate(next); err != nil {
		return Config{}, err
	}
	previous, _ := s.Config(ctx)
	payload, err := json.Marshal(next)
	if err != nil {
		return Config{}, err
	}

	var existing settingRow
	err = s.db.WithContext(ctx).First(&existing, 1).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		row := settingRow{ID: 1, Payload: string(payload)}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return Config{}, errors.New("站点配置保存失败")
		}
	case err != nil:
		return Config{}, errors.New("站点配置读取失败")
	default:
		if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{"payload": string(payload), "updated_at": time.Now()}).Error; err != nil {
			return Config{}, errors.New("站点配置保存失败")
		}
	}

	cfg := withDefaults(next)
	s.mu.Lock()
	s.cached = cfg
	s.cacheOK = true
	s.mu.Unlock()

	if previous.LogoURL != cfg.LogoURL {
		s.removeAsset(previous.LogoURL)
	}
	if previous.FaviconURL != cfg.FaviconURL {
		s.removeAsset(previous.FaviconURL)
	}
	return clone(cfg), nil
}

func withDefaults(cfg Config) Config {
	d := Defaults()
	defaultName := d.Name
	if cfg.Name != "" {
		d.Name = cfg.Name
	}
	if cfg.ShortName != "" && cfg.ShortName != defaultName {
		d.ShortName = cfg.ShortName
	} else {
		d.ShortName = d.Name
	}
	if cfg.Tagline != "" {
		d.Tagline = cfg.Tagline
	} else {
		d.Tagline = ""
	}
	if cfg.BrowserTitle != "" {
		d.BrowserTitle = cfg.BrowserTitle
	}
	if cfg.Description != "" {
		d.Description = cfg.Description
	} else {
		d.Description = ""
	}
	if cfg.LogoURL != "" {
		d.LogoURL = cfg.LogoURL
	}
	if cfg.FaviconURL != "" {
		d.FaviconURL = cfg.FaviconURL
	}
	if cfg.FriendLinks != nil {
		d.FriendLinks = append([]FriendLink(nil), cfg.FriendLinks...)
	}
	for key, value := range cfg.Texts {
		if _, ok := d.Texts[key]; ok && value != "" {
			d.Texts[key] = value
		}
	}
	return d
}

func normalize(cfg Config) Config {
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.ShortName = strings.TrimSpace(cfg.ShortName)
	cfg.Tagline = strings.TrimSpace(cfg.Tagline)
	cfg.BrowserTitle = strings.TrimSpace(cfg.BrowserTitle)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.LogoURL = strings.TrimSpace(cfg.LogoURL)
	cfg.FaviconURL = strings.TrimSpace(cfg.FaviconURL)

	texts := make(map[string]string, len(cfg.Texts))
	for key, value := range cfg.Texts {
		element, ok := textElement(key)
		if !ok {
			continue
		}
		value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
		if !element.Multiline {
			value = strings.Join(strings.Fields(value), " ")
		}
		if value == "" || value == element.Default {
			continue
		}
		texts[key] = value
	}
	cfg.Texts = texts

	links := make([]FriendLink, 0, len(cfg.FriendLinks))
	for _, link := range cfg.FriendLinks {
		link.Name = strings.TrimSpace(link.Name)
		link.URL = strings.TrimSpace(link.URL)
		link.IconURL = strings.TrimSpace(link.IconURL)
		if link.Name == "" && link.URL == "" && link.IconURL == "" {
			continue
		}
		links = append(links, link)
	}
	cfg.FriendLinks = links
	return cfg
}

func validate(cfg Config) error {
	if n := len([]rune(cfg.Name)); n < 1 || n > 40 {
		return errors.New("站点名称不能为空且不能超过 40 个字符")
	}
	if n := len([]rune(cfg.ShortName)); n > 20 {
		return errors.New("站点短名称不能超过 20 个字符")
	}
	if n := len([]rune(cfg.Tagline)); n > 40 {
		return errors.New("站点标语不能超过 40 个字符")
	}
	if n := len([]rune(cfg.BrowserTitle)); n > 120 {
		return errors.New("浏览器标题不能超过 120 个字符")
	}
	if n := len([]rune(cfg.Description)); n > 300 {
		return errors.New("站点描述不能超过 300 个字符")
	}
	if err := validateImageURL(cfg.LogoURL); err != nil {
		return errors.New("Logo 地址无效")
	}
	if err := validateImageURL(cfg.FaviconURL); err != nil {
		return errors.New("站点图标地址无效")
	}
	if len(cfg.FriendLinks) > maxFriendLinks {
		return errors.New("友情链接不能超过 12 条")
	}
	for _, link := range cfg.FriendLinks {
		if n := len([]rune(link.Name)); n < 1 || n > 40 {
			return errors.New("友情链接名称不能为空且不能超过 40 个字符")
		}
		if err := validateExternalURL(link.URL); err != nil {
			return errors.New("友情链接地址必须是 HTTP(S) 地址")
		}
		if err := validateImageURL(link.IconURL); err != nil {
			return errors.New("友情链接图标地址无效")
		}
	}
	for key, value := range cfg.Texts {
		element, ok := textElement(key)
		if !ok {
			return errors.New("包含未知的文案元素")
		}
		if n := len([]rune(value)); n > element.MaxLength {
			return errors.New(element.Label + "不能超过 " + strconv.Itoa(element.MaxLength) + " 个字符")
		}
	}
	return nil
}

func validateImageURL(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > 1000 {
		return errors.New("地址过长")
	}
	if strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n") {
			return errors.New("站点内路径无效")
		}
		return nil
	}
	return validateExternalURL(raw)
}

func validateExternalURL(raw string) error {
	if len(raw) > 1000 {
		return errors.New("地址过长")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return errors.New("地址无效")
	}
	return nil
}

func resolve(cfg Config) Config {
	out := clone(cfg)
	for key, value := range out.Texts {
		out.Texts[key] = resolveText(value, out)
	}
	out.BrowserTitle = resolveText(out.BrowserTitle, out)
	out.Description = resolveText(out.Description, out)
	return out
}

func resolveText(value string, cfg Config) string {
	replacer := strings.NewReplacer(
		"{name}", cfg.Name,
		"{short_name}", cfg.ShortName,
		"{tagline}", cfg.Tagline,
	)
	return replacer.Replace(value)
}

func clone(cfg Config) Config {
	out := cfg
	out.Texts = make(map[string]string, len(cfg.Texts))
	for key, value := range cfg.Texts {
		out.Texts[key] = value
	}
	out.FriendLinks = append([]FriendLink(nil), cfg.FriendLinks...)
	return out
}

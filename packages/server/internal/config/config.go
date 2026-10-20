package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/project"
)

type Config struct {
	Root           string
	Env            string
	Port           int
	CORSOrigins    []string
	UploadDir      string
	ThumbDir       string
	ImagesDir      string
	BinDir         string
	AIImageBaseURL string
	AIImageAPIKey  string
	AIImageModel   string
	AIImageDir     string

	R2 R2Config

	OpenList OpenListConfig

	TinyPNGAPIKey   string
	CompressMinSize int64
	CompressEvery   time.Duration

	MySQL struct {
		Host            string
		Port            int
		User            string
		Password        string
		Database        string
		PoolSize        int
		ConnectTimeout  time.Duration
		ConnMaxLifetime time.Duration
		TLS             string
	}

	Redis struct {
		Host     string
		Port     int
		Password string
		DB       int
		Prefix   string
	}
}

func Load() Config {
	root := project.FindRoot(".")
	var c Config
	c.Root = root
	c.Env = env("APP_ENV", "development")

	dataDir := filepath.Join(root, "data")
	c.UploadDir = env("UPLOAD_DIR", filepath.Join(dataDir, "uploads"))
	c.ThumbDir = env("THUMB_DIR", filepath.Join(dataDir, "thumbs"))
	c.ImagesDir = env("IMAGES_DIR", filepath.Join(dataDir, "images"))
	c.BinDir = env("BIN_DIR", filepath.Join(dataDir, "bin"))
	c.AIImageDir = env("AI_IMAGE_DIR", filepath.Join(dataDir, "ai-private"))
	c.AIImageBaseURL = strings.TrimSpace(os.Getenv("AI_IMAGE_BASE_URL"))
	c.AIImageAPIKey = strings.TrimSpace(os.Getenv("AI_IMAGE_API_KEY"))
	c.AIImageModel = strings.TrimSpace(os.Getenv("AI_IMAGE_MODEL"))

	c.R2 = loadR2()

	c.OpenList = loadOpenList()

	c.TinyPNGAPIKey = strings.TrimSpace(os.Getenv("TINIFY_API_KEY"))
	c.CompressMinSize = int64(envInt("COMPRESS_MIN_KB", 400)) << 10
	c.CompressEvery = time.Duration(envInt("COMPRESS_INTERVAL_SECONDS", 60)) * time.Second

	c.Port = envInt("PORT", 3000)
	for _, o := range strings.Split(env("CORS_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.CORSOrigins = append(c.CORSOrigins, o)
		}
	}

	c.MySQL.Host = env("MYSQL_HOST", "127.0.0.1")
	c.MySQL.Port = envInt("MYSQL_PORT", 3306)
	c.MySQL.User = env("MYSQL_USER", "root")
	c.MySQL.Password = os.Getenv("MYSQL_PASSWORD")
	c.MySQL.Database = env("MYSQL_DATABASE", "xqecz")
	c.MySQL.PoolSize = envInt("MYSQL_POOL_SIZE", 10)
	c.MySQL.ConnectTimeout = time.Duration(envInt("MYSQL_CONNECT_TIMEOUT", 10)) * time.Second
	c.MySQL.ConnMaxLifetime = c.MySQL.ConnectTimeout
	c.MySQL.TLS = strings.TrimSpace(os.Getenv("MYSQL_TLS"))

	c.Redis.Host = env("REDIS_HOST", "127.0.0.1")
	c.Redis.Port = envInt("REDIS_PORT", 6379)
	c.Redis.Password = os.Getenv("REDIS_PASSWORD")
	c.Redis.DB = envInt("REDIS_DB", 0)
	c.Redis.Prefix = env("REDIS_PREFIX", "xqecz:")

	return c
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

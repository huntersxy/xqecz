package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/huntersxy/xqecz/server/internal/project"
)

// Config 汇总进程运行所需配置。单一来源为项目根 .env，与旧 API/Worker 一致。
type Config struct {
	Root        string
	Env         string
	Port        int
	CORSOrigins []string
	UploadDir   string
	ThumbDir    string
	ImagesDir   string

	MySQL struct {
		Host            string
		Port            int
		User            string
		Password        string
		Database        string
		PoolSize        int
		ConnMaxLifetime time.Duration
	}

	Redis struct {
		Host     string
		Port     int
		Password string
		DB       int
		Prefix   string
	}
}

// Load 读取项目根 .env（不覆盖已存在的环境变量）并组装配置。
func Load() Config {
	root := project.FindRoot(".")
	var c Config
	c.Root = root
	c.Env = env("APP_ENV", "development")

	dataDir := filepath.Join(root, "data")
	c.UploadDir = env("UPLOAD_DIR", filepath.Join(dataDir, "uploads"))
	c.ThumbDir = env("THUMB_DIR", filepath.Join(dataDir, "thumbs"))
	c.ImagesDir = env("IMAGES_DIR", filepath.Join(dataDir, "images"))

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
	c.MySQL.ConnMaxLifetime = time.Duration(envInt("MYSQL_CONNECT_TIMEOUT", 10)) * time.Second

	c.Redis.Host = env("REDIS_HOST", "127.0.0.1")
	c.Redis.Port = envInt("REDIS_PORT", 6379)
	c.Redis.Password = os.Getenv("REDIS_PASSWORD")
	c.Redis.DB = envInt("REDIS_DB", 0)
	// 与旧 ioredis keyPrefix 对齐：所有业务 key 统一加前缀。
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

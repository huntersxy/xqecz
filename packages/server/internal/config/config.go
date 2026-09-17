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
	// BinDir 是压缩/清理后原图的垃圾桶目录，保留而非直接删除，便于人工回溯。
	BinDir string

	// R2 对象存储镜像：原图与压缩图各存一份，缩略图仍只留在本地。
	// 四项凭据任一为空即整体停用（与 TinyPNG 同样的「缺配置即休眠」策略），
	// 便于本地开发与未开通 R2 的环境照常运行。
	R2 R2Config

	// TinyPNG 定时压缩：API Key 为空时该任务自动休眠（不报错），便于本地与无配额环境。
	TinyPNGAPIKey string
	// CompressMinSize 以下的图片不压缩（字节）。
	CompressMinSize int64
	// CompressEvery 压缩任务的执行间隔，每轮只处理一张最大的待压缩图片。
	CompressEvery time.Duration

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
	c.BinDir = env("BIN_DIR", filepath.Join(dataDir, "bin"))

	// R2 镜像：凭据不全即停用（本地开发常未开通）。
	c.R2 = loadR2()

	// 压缩链路：Key 缺失即停用（本地开发常无配额）；阈值与节奏可按需覆盖。
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

package config

import (
	"os"
	"path/filepath"
)

// Config 是 worker 运行所需的全部配置，全部从环境变量读取。
// 架构约束（见 AGENTS.md）：worker 不直连 MySQL/Redis，仅做无状态计算/文件处理，
// 因此配置只涉及文件目录、外部服务（Tinify）与监听端口。
type Config struct {
	Tinify TinifyConfig
	Server ServerConfig
	Worker WorkerConfig
}

type TinifyConfig struct {
	APIKey string
}

type ServerConfig struct {
	UploadDir string
	ThumbDir  string
	ImagesDir string
}

type WorkerConfig struct {
	Port string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load 从环境变量构建配置，使用合理的本地默认。
func Load() *Config {
	uploadDir := getenv("UPLOAD_DIR", "/app/uploads")
	return &Config{
		Tinify: TinifyConfig{
			APIKey: getenv("TINIFY_API_KEY", ""),
		},
		Server: ServerConfig{
			UploadDir: uploadDir,
			// 默认与 uploads 同级（data/uploads、data/thumbs、data/images 三目录同级）。
			ThumbDir:  getenv("THUMB_DIR", filepath.Join(filepath.Dir(uploadDir), "thumbs")),
			ImagesDir: getenv("IMAGES_DIR", filepath.Join(filepath.Dir(uploadDir), "images")),
		},
		Worker: WorkerConfig{
			Port: getenv("WORKER_PORT", "50051"),
		},
	}
}

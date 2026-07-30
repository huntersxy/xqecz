package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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

// findProjectRoot 从 startDir 向上查找 pnpm-workspace.yaml 或 .git，
// 返回项目根目录路径。与 NestJS API 的 paths.ts 保持一致的发现逻辑。
func findProjectRoot(startDir string) string {
	dir := startDir
	for range 6 {
		if _, err := os.Stat(filepath.Join(dir, "pnpm-workspace.yaml")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// 兜底：返回当前工作目录
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return startDir
}

// loadDotEnv 从指定目录读取 .env 文件，将其中 KEY=VALUE 行注入环境变量。
// 仅注入当前 OS 环境变量中不存在的 key（OS 环境变量优先级高于 .env）。
func loadDotEnv(dir string) {
	path := filepath.Join(dir, ".env")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 允许值中包含 = 号（例如连接串）
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key == "" {
			continue
		}
		// OS 环境变量优先，不覆盖
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, val)
	}
}

// Load 从环境变量构建配置。优先 OS 环境变量，其次 .env 文件，最后兜底到项目根 data/ 目录。
func Load() *Config {
	// 1) 找到项目根目录（二进制所在目录或工作目录向上查找）
	root := "."
	if exe, err := os.Executable(); err == nil {
		root = findProjectRoot(filepath.Dir(exe))
	}
	if root == "." {
		if wd, err := os.Getwd(); err == nil {
			root = findProjectRoot(wd)
		}
	}

	// 2) 从项目根加载 .env（OS 环境变量已有的不会被覆盖）
	loadDotEnv(root)

	// 3) 默认值：项目根/data/{uploads,thumbs,images} 三目录同级
	dataDir := filepath.Join(root, "data")
	uploadDir := getenv("UPLOAD_DIR", filepath.Join(dataDir, "uploads"))

	return &Config{
		Tinify: TinifyConfig{
			APIKey: getenv("TINIFY_API_KEY", ""),
		},
		Server: ServerConfig{
			UploadDir: uploadDir,
			ThumbDir:  getenv("THUMB_DIR", filepath.Join(dataDir, "thumbs")),
			ImagesDir: getenv("IMAGES_DIR", filepath.Join(dataDir, "images")),
		},
		Worker: WorkerConfig{
			Port: getenv("WORKER_PORT", "50051"),
		},
	}
}

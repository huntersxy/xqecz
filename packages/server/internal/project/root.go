package project

import (
	"os"
	"path/filepath"
	"strings"
)

// rootMarkers 用于判定「项目根」的标记文件/目录，按可信度排序。
//
// pnpm-workspace.yaml / .git 是 monorepo 开发期的标记；生产形态是「宝塔 Go 项目 + 静态站点」，
// 只上传二进制与 .env，不携带包管理器清单，因此 .env 必须能独立作为根标记。
var rootMarkers = []string{"pnpm-workspace.yaml", ".git", ".env"}

// rootEnvKeys 显式指定项目根的环境变量，优先级高于目录探测，便于部署侧（如面板环境变量）覆盖。
var rootEnvKeys = []string{"PROJECT_ROOT", "XQECZ_ROOT"}

// FindRoot 返回项目根目录：优先环境变量，其次自 start 向上查找标记文件。
// 与前端/旧 API 的 PROJECT_ROOT 解析保持一致，保证 data 目录落在「项目根/data」。
func FindRoot(start string) string {
	for _, key := range rootEnvKeys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if abs, err := filepath.Abs(v); err == nil {
				return abs
			}
			return v
		}
	}

	dir, err := filepath.Abs(start)
	if err != nil {
		return start
	}
	for i := 0; i < 8; i++ {
		for _, marker := range rootMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dir
}

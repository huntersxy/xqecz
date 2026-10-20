package project

import (
	"os"
	"path/filepath"
	"strings"
)

var rootMarkers = []string{"pnpm-workspace.yaml", ".git", ".env"}

var rootEnvKeys = []string{"PROJECT_ROOT", "XQECZ_ROOT"}

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

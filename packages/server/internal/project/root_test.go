package project

import (
	"os"
	"path/filepath"
	"testing"
)

// 清掉显式根环境变量，避免宿主机环境影响探测类断言。
func clearRootEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("XQECZ_ROOT", "")
}

func TestFindRootByEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PROJECT_ROOT", dir)

	if got := FindRoot(t.TempDir()); got != dir {
		t.Fatalf("PROJECT_ROOT 覆盖失败: got %q want %q", got, dir)
	}
}

func TestFindRootByXqeczRootEnv(t *testing.T) {
	clearRootEnv(t)
	dir := t.TempDir()
	t.Setenv("XQECZ_ROOT", dir)

	if got := FindRoot(t.TempDir()); got != dir {
		t.Fatalf("XQECZ_ROOT 覆盖失败: got %q want %q", got, dir)
	}
}

// 生产形态只上传二进制与 .env（不带 pnpm-workspace.yaml），.env 必须能独立作为根标记。
func TestFindRootByMarkers(t *testing.T) {
	for _, marker := range []string{"pnpm-workspace.yaml", ".git", ".env"} {
		t.Run(marker, func(t *testing.T) {
			clearRootEnv(t)
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, marker), []byte("marker\n"), 0o644); err != nil {
				t.Fatalf("写入标记失败: %v", err)
			}
			nested := filepath.Join(root, "packages", "server")
			if err := os.MkdirAll(nested, 0o755); err != nil {
				t.Fatalf("创建嵌套目录失败: %v", err)
			}

			if got := FindRoot(nested); got != root {
				t.Fatalf("标记 %s 未生效: got %q want %q", marker, got, root)
			}
			// 生成目录内的相对起点同样应解析到根
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(nested); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Chdir(wd) }()

			if got := FindRoot("."); got != root {
				t.Fatalf("标记 %s 以 . 为起点失败: got %q want %q", marker, got, root)
			}
		})
	}
}

// 无任何标记时回退到向上遍历的终点，不应 panic 且返回绝对路径。
func TestFindRootFallbackWithoutMarkers(t *testing.T) {
	clearRootEnv(t)
	start := t.TempDir()

	got := FindRoot(start)
	if got == "" {
		t.Fatal("回退结果不应为空")
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("回退结果应为绝对路径: %q", got)
	}
}

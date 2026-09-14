package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// MoveToBin 把文件移入垃圾桶目录（保留文件名），同秒同名时加时间戳前缀避免互相覆盖。
// rename 跨设备会失败，此时退化为复制 + 删除源文件。
func MoveToBin(absPath, binDir string) error {
	f, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if f.IsDir() {
		return fmt.Errorf("not a file: %s", absPath)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	dst := filepath.Join(binDir, filepath.Base(absPath))
	if _, err := os.Stat(dst); err == nil {
		dst = filepath.Join(binDir, fmt.Sprintf("%d_%s", time.Now().UnixMilli(), filepath.Base(absPath)))
	}

	if err := os.Rename(absPath, dst); err == nil {
		return nil
	}
	if err := copyFile(absPath, dst); err != nil {
		return err
	}
	return os.Remove(absPath)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

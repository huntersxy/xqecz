package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

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

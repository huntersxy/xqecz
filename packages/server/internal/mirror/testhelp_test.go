package mirror

import (
	"crypto/md5"
	"encoding/hex"
)

// md5Sum 是测试侧独立实现的 md5（刻意不复用生产代码的 fileMD5）。
func md5Sum(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

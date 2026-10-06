package mirror

import (
	"crypto/md5"
	"encoding/hex"
	"testing"
	"time"
)

// md5Sum 是测试侧独立实现的 md5（刻意不复用生产代码的 fileMD5）。
func md5Sum(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}

// waitFor 轮询直到 cond 成立，用于断言异步推送（PushAsync 起 goroutine）。
// 超时即失败并打印 msg——避免测试用固定 sleep 换来不稳定。
func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("超时未满足条件：%s", msg)
}

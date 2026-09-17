package r2

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLiveRoundTrip 是针对**真实 R2**的端到端校验：PUT 一个对象、HEAD 回读、
// 再 PUT 不同内容确认覆盖生效、最后删除临时对象。
//
// 默认跳过：只有显式提供了凭据才会跑，避免 CI 与本地开发误触真桶。
//
//	R2_ACCOUNT_ID=... R2_ACCESS_KEY_ID=... R2_SECRET_ACCESS_KEY=... R2_BUCKET=... \
//	  go test ./internal/r2/ -run TestLiveRoundTrip -v
//
// 对象写在 <bucket>/_r2test/ 下并在结束时清理，不会碰到真实业务对象。
func TestLiveRoundTrip(t *testing.T) {
	cfg := Config{
		Endpoint:  os.Getenv("R2_ENDPOINT"),
		AccessKey: os.Getenv("R2_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		Bucket:    os.Getenv("R2_BUCKET"),
		Prefix:    "_r2test",
		Timeout:   30 * time.Second,
	}
	if account := os.Getenv("R2_ACCOUNT_ID"); account != "" && cfg.Endpoint == "" {
		cfg.Endpoint = "https://" + account + ".r2.cloudflarestorage.com"
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Endpoint == "" {
		t.Skip("未提供 R2_ACCESS_KEY_ID / R2_SECRET_ACCESS_KEY / R2_BUCKET / R2_ACCOUNT_ID，跳过真实 R2 校验")
	}

	client, err := New(cfg)
	if err != nil {
		t.Fatalf("构造客户端失败: %v", err)
	}

	dir := t.TempDir()
	rel := "live-check.webp"
	key := client.ObjectKey(rel)
	abs := filepath.Join(dir, rel)

	first := []byte("r2-live-check-v1-" + time.Now().Format(time.RFC3339Nano))
	if err := os.WriteFile(abs, first, 0o644); err != nil {
		t.Fatal(err)
	}

	// 从干净状态开始（上一次失败可能留下垃圾）。
	if _, found, err := client.Head(key); err != nil {
		t.Logf("预清理 HEAD 失败（忽略）: %v", err)
	} else if found {
		t.Logf("已存在残留对象，将由本次 PUT 覆盖: %s", key)
	}

	if err := client.Put(rel, abs, "image/webp"); err != nil {
		t.Fatalf("PUT 失败: %v", err)
	}
	t.Cleanup(func() { cleanupObject(t, client, key) })

	meta, found, err := client.Head(key)
	if err != nil {
		t.Fatalf("HEAD 失败: %v", err)
	}
	if !found {
		t.Fatal("PUT 之后 HEAD 应能找到对象")
	}
	if meta.ContentLen != int64(len(first)) {
		t.Fatalf("远端长度不符: got %d want %d", meta.ContentLen, len(first))
	}
	if meta.MetaMD5 == "" {
		t.Fatal("应带回 x-amz-meta-md5 自定义元数据（镜像判重依赖它）")
	}
	if meta.LastModified.IsZero() {
		t.Log("提示：Last-Modified 未解析出来（不影响镜像判重）")
	}

	// 覆盖写：模拟原地压缩后重传。
	second := []byte("r2-live-check-v2-shorter")
	if err := os.WriteFile(abs, second, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := client.Put(rel, abs, "image/webp"); err != nil {
		t.Fatalf("覆盖 PUT 失败: %v", err)
	}
	meta2, _, err := client.Head(key)
	if err != nil {
		t.Fatalf("覆盖后 HEAD 失败: %v", err)
	}
	if meta2.ContentLen != int64(len(second)) {
		t.Fatalf("覆盖后长度未更新: got %d want %d", meta2.ContentLen, len(second))
	}
	if meta2.MetaMD5 == meta.MetaMD5 {
		t.Fatal("覆盖后内容 md5 应变化，说明 PUT 没真的替换对象")
	}

	// 不存在的对象：应返回 found=false 而不是报错。
	if _, found, err := client.Head(client.ObjectKey("does-not-exist-" + time.Now().Format("150405"))); err != nil {
		t.Fatalf("HEAD 不存在的对象不应报错: %v", err)
	} else if found {
		t.Fatal("不存在的对象不应被判为存在")
	}

	// 中文/空格对象名：验证规范化 URI 未二次编码（否则服务端直接拒签）。
	cnRel := "中文 名.webp"
	cnKey := client.ObjectKey(cnRel)
	cnAbs := filepath.Join(dir, "cn.webp")
	if err := os.WriteFile(cnAbs, []byte("unicode-key"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := client.Put(cnRel, cnAbs, "image/webp"); err != nil {
		t.Fatalf("中文对象名 PUT 失败（多为签名 URI 编码问题）: %v", err)
	}
	t.Cleanup(func() { cleanupObject(t, client, cnKey) })
	if _, found, err := client.Head(cnKey); err != nil || !found {
		t.Fatalf("中文对象名 HEAD 失败: found=%v err=%v", found, err)
	}
}

// cleanupObject 删除本次校验写入的临时对象，保持桶干净。
func cleanupObject(t *testing.T, client *Client, key string) {
	t.Helper()
	if err := client.Delete(key); err != nil {
		t.Logf("清理临时对象失败（可手动删 _r2test/%s）: %v", key, err)
		return
	}
	if _, found, err := client.Head(key); err == nil && found {
		t.Logf("临时对象似乎仍在: %s", key)
	}
}

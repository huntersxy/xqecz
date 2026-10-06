package mirror

import (
	"testing"
)

// chainFixture 造一条双目标链（R2 为主、OpenList 为次），各用一个假存储，
// 便于分别断言谁被推送了。顺序只影响 Setups 的排列，不代表优先级。
func chainFixture(t *testing.T) (*Chain, *fakeStore, *fakeStore) {
	t.Helper()
	primaryStore, secondaryStore := newFakeStore(), newFakeStore()
	primary := newWithStoreTarget(Target{
		Name: "r2", Driver: DriverR2, Endpoint: "https://r2.example", AccessKey: "a",
		SecretKey: "b", Bucket: "bucket", Region: "auto", Prefix: "uploads",
		PublicBase: "https://img.r2.test",
	}, primaryStore)
	secondary := newWithStoreTarget(Target{
		Name: "openlist", Driver: DriverOpenList, Endpoint: "http://127.0.0.1:5244",
		Token: "tk", Prefix: "uploads", PublicBase: "https://drive.xiey.work/d",
	}, secondaryStore)
	return NewChain(primary, secondary), primaryStore, secondaryStore
}

func TestChainR2URLAlwaysPrimary(t *testing.T) {
	c, _, _ := chainFixture(t)
	got := c.R2URL("a.webp")
	if got != "https://img.r2.test/uploads/a.webp" {
		t.Fatalf("R2URL 应给 R2 那一级的地址，实际 %q", got)
	}
}

// TestChainOpenListURLWhenConfigured OpenList 那一级配了公开域名就该下发地址。
// 这里同时锁住两点：① 地址是 `PublicBase + "/" + key` 的形态——PublicBase 带 `/d`
// 是实例的免签直链前缀，少了它 OpenList 只会回 SPA 首页 HTML；
// ② 地址**不带任何签名参数**——七牛时代的时间戳防盗链已随之下线，
// OpenList 的直链靠服务端 sign_all=false 免签开放。
func TestChainOpenListURLWhenConfigured(t *testing.T) {
	c, _, _ := chainFixture(t)
	got := c.OpenListURL("a.webp")
	if got != "https://drive.xiey.work/d/uploads/a.webp" {
		t.Fatalf("OpenListURL 应给公开地址，实际 %q", got)
	}
	if want := "https://drive.xiey.work/d/uploads/a.webp"; got != want {
		t.Fatalf("地址应逐字节等于 %q，实际 %q", want, got)
	}
}

// TestChainOpenListURLWithoutPublicBase 只配了连接参数、没配公开域名时
// 这一级不给候选——前端因此直接落到 R2，不会被引到一个不存在的地址上。
func TestChainOpenListURLWithoutPublicBase(t *testing.T) {
	c, _, _ := chainFixture(t)
	c.secondary.cfg.PublicBase = ""
	if got := c.OpenListURL("a.webp"); got != "" {
		t.Fatalf("未配公开域名时不应下发地址，实际 %q", got)
	}
}

// TestChainOpenListURLDisabledWhenUnconfigured 凭据不全（未配 token）时该级停用。
func TestChainOpenListURLDisabledWhenUnconfigured(t *testing.T) {
	c, _, _ := chainFixture(t)
	c.secondary.cfg.Token = ""
	c.secondary.store = nil
	if got := c.OpenListURL("a.webp"); got != "" {
		t.Fatalf("目标停用时应返回空串，实际 %q", got)
	}
}

func TestChainSetupsFiltersDisabledTargets(t *testing.T) {
	c, _, _ := chainFixture(t)
	if len(c.Setups()) != 2 {
		t.Fatalf("两个目标都启用时应回 2 个，实际 %d", len(c.Setups()))
	}

	// 只配 R2：第二级用空 Target 构造（Enabled()==false）
	only := NewChain(newWithStoreTarget(Target{
		Name: "r2", Driver: DriverR2, Endpoint: "https://r2.example", AccessKey: "a",
		SecretKey: "b", Bucket: "b",
	}, newFakeStore()), New(Target{}, ""))
	if len(only.Setups()) != 1 {
		t.Fatalf("第二级未配时应回 1 个，实际 %d", len(only.Setups()))
	}
	if only.OpenListURL("a.webp") != "" {
		t.Fatal("第二级未配时不应下发地址")
	}
}

func TestChainDisabledIsSafe(t *testing.T) {
	var c *Chain
	if c.Enabled() || c.R2URL("a") != "" || c.OpenListURL("a") != "" || c.BinDir() != "" {
		t.Fatal("空链的读方法必须安全返回零值")
	}
	c.PushAsync("a", "/b")
	c.ArchiveOriginalAsync("a", "/b")
	if len(c.Setups()) != 0 {
		t.Fatal("空链不应给出目标")
	}
}

// TestChainBothTargetsPushed 推送侧不分主次：两个目标都该收到。
func TestChainBothTargetsPushed(t *testing.T) {
	dir := t.TempDir()
	abs := writeFile(t, dir, "x.webp", []byte("bytes"))

	c, primaryStore, secondaryStore := chainFixture(t)
	c.PushAsync("x.webp", abs)
	c.ArchiveOriginalAsync("x.webp", abs)

	// PushAsync 起 goroutine，等两个目标都收到为止（有超时保护）。
	waitFor(t, func() bool {
		return primaryStore.putCount() > 0 && secondaryStore.putCount() > 0
	}, "两个目标都应收到推送")
}

// TestChainSecondaryAbsentIsFine 第二级停用时 push 不应 panic，R2 照常收到。
func TestChainSecondaryAbsentIsFine(t *testing.T) {
	dir := t.TempDir()
	abs := writeFile(t, dir, "y.webp", []byte("bytes"))

	primaryStore := newFakeStore()
	c := NewChain(newWithStoreTarget(Target{
		Name: "r2", Driver: DriverR2, Endpoint: "https://r2.example", AccessKey: "a",
		SecretKey: "b", Bucket: "bucket", Prefix: "uploads", PublicBase: "https://img.r2.test",
	}, primaryStore), New(Target{}, ""))

	c.PushAsync("y.webp", abs)
	waitFor(t, func() bool { return primaryStore.putCount() > 0 }, "R2 应照常收到推送")
}

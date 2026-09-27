package mirror

import (
	"errors"
	"strings"
	"testing"
)

// gateStub 可切换放行状态，用于验证替补层受额度闸门约束。
type gateStub struct{ open bool }

func (g gateStub) Available() bool { return g.open }

// chainFixture 造一条双目标链，主/替补各用一个假存储，便于分别断言谁被推送了。
func chainFixture(t *testing.T, gate Availability, sign func(string) (string, error)) (*Chain, *fakeStore, *fakeStore) {
	t.Helper()
	primaryStore, secondaryStore := newFakeStore(), newFakeStore()
	primary := newWithStoreTarget(Target{
		Name: "r2", Endpoint: "https://r2.example", AccessKey: "a", SecretKey: "b",
		Bucket: "bucket", Region: "auto", Prefix: "uploads", PublicBase: "https://img.r2.test",
	}, primaryStore)
	secondary := newWithStoreTarget(Target{
		Name: "qiniu", Endpoint: "https://s3.cn-south-1.qiniucs.com", AccessKey: "a",
		SecretKey: "b", Bucket: "bucket", Region: "cn-south-1", Prefix: "uploads",
		PublicBase: "https://img.qiniu.test",
	}, secondaryStore)
	return NewChain(primary, secondary, gate, sign), primaryStore, secondaryStore
}

func TestChainPublicURLAlwaysPrimary(t *testing.T) {
	c, _, _ := chainFixture(t, gateStub{open: true}, nil)
	got := c.PublicURL("a.webp")
	if got != "https://img.r2.test/uploads/a.webp" {
		t.Fatalf("PublicURL 应给主目标地址，实际 %q", got)
	}
	// 主目标地址不签名——R2 没有时间戳防盗链
	if strings.Contains(got, "sign=") {
		t.Fatalf("主目标不应带签名: %s", got)
	}
}

// TestChainSecondaryClosedByGate 额度触顶时替补层一个地址都不下发，
// 前端连候选都拿不到，探测自然不会打到一个注定 403 的地址。
func TestChainSecondaryClosedByGate(t *testing.T) {
	c, _, _ := chainFixture(t, gateStub{open: false}, nil)
	if got := c.Mirror2URL("a.webp"); got != "" {
		t.Fatalf("闸门关闭时替补层地址应为空，实际 %q", got)
	}
}

func TestChainSecondarySignedWhenOpen(t *testing.T) {
	c, _, _ := chainFixture(t, gateStub{open: true}, func(raw string) (string, error) {
		return raw + "?sign=deadbeef&t=55bb9b80", nil
	})
	got := c.Mirror2URL("a.webp")
	if !strings.Contains(got, "https://img.qiniu.test/uploads/a.webp") {
		t.Fatalf("应给替补目标地址，实际 %q", got)
	}
	if !strings.Contains(got, "sign=deadbeef") {
		t.Fatalf("替补目标地址必须带签名，实际 %q", got)
	}
}

// TestChainSignFailureYieldsNoCandidate 签名失败宁可不下发，
// 也不给一个必然 403 的地址让前端白探测一轮。
func TestChainSignFailureYieldsNoCandidate(t *testing.T) {
	c, _, _ := chainFixture(t, gateStub{open: true}, func(string) (string, error) {
		return "", errors.New("boom")
	})
	if got := c.Mirror2URL("a.webp"); got != "" {
		t.Fatalf("签名失败应放弃下发，实际 %q", got)
	}
}

func TestChainSetupsFiltersDisabledTargets(t *testing.T) {
	c, _, _ := chainFixture(t, gateStub{open: true}, nil)
	if len(c.Setups()) != 2 {
		t.Fatalf("两个目标都启用时应回 2 个，实际 %d", len(c.Setups()))
	}

	// 只配主目标
	only := NewChain(newWithStoreTarget(Target{
		Name: "r2", Endpoint: "https://r2.example", AccessKey: "a", SecretKey: "b", Bucket: "b",
	}, newFakeStore()), New(Target{}, ""), gateStub{open: true}, nil)
	if len(only.Setups()) != 1 {
		t.Fatalf("替补未配时应回 1 个，实际 %d", len(only.Setups()))
	}
	if only.Mirror2URL("a.webp") != "" {
		t.Fatal("替补未配时不应下发地址")
	}
}

// TestChainNilGateMeansNoLimit 未接入额度管理的环境不设闸，替补层恒放行。
func TestChainNilGateMeansNoLimit(t *testing.T) {
	c, _, _ := chainFixture(t, nil, nil)
	if got := c.Mirror2URL("a.webp"); got == "" {
		t.Fatal("不设闸时替补层应放行")
	}
}

func TestChainDisabledIsSafe(t *testing.T) {
	var c *Chain
	if c.Enabled() || c.PublicURL("a") != "" || c.Mirror2URL("a") != "" || c.BinDir() != "" {
		t.Fatal("空链的读方法必须安全返回零值")
	}
	c.PushAsync("a", "/b")
	c.ArchiveOriginalAsync("a", "/b")
	if len(c.Setups()) != 0 {
		t.Fatal("空链不应给出目标")
	}
}

// TestSignerReturnsNilWithoutKey 未配密钥时不做签名（替补层地址原样下发）。
func TestSignerReturnsNilWithoutKey(t *testing.T) {
	if Signer("") != nil {
		t.Fatal("空密钥应返回 nil 签名函数")
	}
	got, err := Signer("k")("https://img.xiey.work/a.webp")
	if err != nil || !strings.Contains(got, "sign=") {
		t.Fatalf("配了密钥就该签出参数: url=%q err=%v", got, err)
	}
}

// TestGateBlocksSecondaryPush 闸门必须管到**推送本身**：触顶后回填仍在跑，
// 若只在取址层判闸，替补层会被继续塞数据，额度照样被吃掉。
// 关闭时是同步短路（goroutine 都不会起），所以断言不依赖时序。
func TestGateBlocksSecondaryPush(t *testing.T) {
	dir := t.TempDir()
	abs := writeFile(t, dir, "x.webp", []byte("bytes"))

	closed, _, secondaryStore := chainFixture(t, gateStub{open: false}, nil)
	closed.PushAsync("x.webp", abs)
	closed.ArchiveOriginalAsync("x.webp", abs)
	if secondaryStore.puts != 0 {
		t.Fatalf("闸门关闭时替补层一次都不该推，实际 %d 次", secondaryStore.puts)
	}
}

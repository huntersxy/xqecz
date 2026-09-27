package qiniu

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// pct 取 total 的 p 比例（向下取整），避免在用例里手写易错的整数运算。
func pct(total int64, p float64) int64 { return int64(float64(total) * p) }

func TestDefaultQuota(t *testing.T) {
	q := DefaultQuota()
	if q.StorageBytes != 10*GB || q.OriginFlowBytes != 10*GB || q.CDNFlowBytes != 10*GB {
		t.Fatalf("流量/存储额度应为 10GB: %+v", q)
	}
	if q.GETCount != 1_000_000 || q.PUTCount != 100_000 {
		t.Fatalf("请求次数额度错误: %+v", q)
	}
	if q.Threshold != 0.9 {
		t.Fatalf("默认阈值应为 0.9，实际 %v", q.Threshold)
	}
}

// TestEvaluateStorageRatio 存储量是时点值，按当月快照直接算比例。
func TestEvaluateStorageRatio(t *testing.T) {
	q := DefaultQuota()
	if r := q.Evaluate(Usage{StorageBytes: 5 * GB}); r.Storage != 0.5 {
		t.Fatalf("5GB/10GB 应为 0.5，实际 %v", r.Storage)
	}
}

// TestExceededBoundary 90% 即触顶（>=），差一点不触顶。
func TestExceededBoundary(t *testing.T) {
	q := DefaultQuota()
	if !q.Exceeded(Usage{StorageBytes: pct(q.StorageBytes, 0.9)}) {
		t.Fatal("达到 90% 应判定为触顶")
	}
	if q.Exceeded(Usage{StorageBytes: pct(q.StorageBytes, 0.899)}) {
		t.Fatal("未到 90% 不应判定为触顶")
	}
}

// TestCDNUnknownIsNotTreatedAsZero 查不到 CDN 流量时（未绑域名）该项不参与判定。
// 这不是「把未知当安全」：没绑域名就没有 CDN 流量这条路径可走。
func TestCDNUnknownIsNotTreatedAsZero(t *testing.T) {
	q := DefaultQuota()
	unknown := Usage{CDNFlowBytes: 99 * GB, CDNKnown: false}
	if q.Evaluate(unknown).CDNFlow != 0 {
		t.Fatal("CDNKnown=false 时 CDNFlow 比例应为 0")
	}
	if q.Exceeded(unknown) {
		t.Fatal("CDN 项未知时不应仅凭其绝对值触顶")
	}
	if !q.Exceeded(Usage{CDNFlowBytes: 99 * GB, CDNKnown: true}) {
		t.Fatal("CDN 流量已知且超额时必须触顶")
	}
}

// TestEgressCapOptional 七牛外网流出**没有**免费额度（0.26 元/GB 从第一字节起计），
// 只能自设上限；上限为 0 表示不判定——桶保持私有时这条路本就走不通。
func TestEgressCapOptional(t *testing.T) {
	q := DefaultQuota()
	if q.Exceeded(Usage{EgressBytes: 999 * GB}) {
		t.Fatal("未设外网流出上限时不应判定")
	}
	q.EgressBytes = 1 * GB
	if !q.Exceeded(Usage{EgressBytes: pct(1*GB, 0.95)}) {
		t.Fatal("设了上限后应参与判定")
	}
}

func TestBreachedListsOnlyOffenders(t *testing.T) {
	q := DefaultQuota()
	got := q.Breached(Usage{
		CDNFlowBytes: pct(q.CDNFlowBytes, 0.95),
		CDNKnown:     true,
		GETCount:     q.GETCount,
		StorageBytes: 1 * GB, // 远未触顶
	})
	joined := strings.Join(got, ",")
	if !strings.Contains(joined, "CDN下载") {
		t.Fatalf("应列出 CDN下载，实际 %v", got)
	}
	if !strings.Contains(joined, "GET") {
		t.Fatalf("应列出 GET，实际 %v", got)
	}
	if strings.Contains(joined, "存储") {
		t.Fatalf("存储未超额，不应列出，实际 %v", got)
	}
}

/* ---------------- 闸门 ---------------- */

// fakeSource 可切换「返回什么、是否报错」，用来验证闸门的状态保持语义。
type fakeSource struct {
	usage Usage
	err   error
}

func (f *fakeSource) Usage(context.Context, time.Time) (Usage, error) {
	if f.err != nil {
		return Usage{}, f.err
	}
	return f.usage, nil
}

// TestGateStartsClosed 启动时没查过就不开——宁可先少一层，也不要先超支。
func TestGateStartsClosed(t *testing.T) {
	if NewGate(&fakeSource{}, DefaultQuota()).Available() {
		t.Fatal("闸门初值必须是关闭")
	}
}

func TestGateOpensWhenUnderQuota(t *testing.T) {
	g := NewGate(&fakeSource{usage: Usage{StorageBytes: 100}}, DefaultQuota())
	if err := g.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !g.Available() {
		t.Fatal("额度充足时应放行")
	}
}

func TestGateClosesWhenBreached(t *testing.T) {
	q := DefaultQuota()
	g := NewGate(&fakeSource{usage: Usage{StorageBytes: pct(q.StorageBytes, 0.95)}}, q)
	if err := g.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if g.Available() {
		t.Fatal("触顶后必须停用")
	}
	on, maxRatio, breached, checkedAt, err := g.Status()
	if on || err != nil {
		t.Fatalf("状态摘要不对: on=%v err=%v", on, err)
	}
	if len(breached) == 0 || maxRatio < 0.9 || checkedAt.IsZero() {
		t.Fatalf("应留下可排障的摘要: max=%.3f breached=%v checkedAt=%v", maxRatio, breached, checkedAt)
	}
}

// TestGateKeepsStateOnQueryFailure 查询失败时保留上次结论——
// 拿半份数据开闸的代价是真金白银，拿旧结论顶着只是晚几分钟切换。
func TestGateKeepsStateOnQueryFailure(t *testing.T) {
	src := &fakeSource{usage: Usage{StorageBytes: 100}}
	g := NewGate(src, DefaultQuota())
	if err := g.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !g.Available() {
		t.Fatal("首轮应放行")
	}

	src.err = errors.New("boom")
	if err := g.Refresh(context.Background()); err == nil {
		t.Fatal("查询失败应返回错误")
	}
	if !g.Available() {
		t.Fatal("查询失败不得改变上次结论")
	}
	if _, _, _, _, err := g.Status(); err == nil {
		t.Fatal("错误应被记录，便于日志与健康检查暴露")
	}

	// 恢复后重新按用量判定
	src.err = nil
	src.usage = Usage{StorageBytes: pct(10*GB, 0.95)}
	if err := g.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if g.Available() {
		t.Fatal("恢复查询后应按新用量重新触顶")
	}
}

// TestGateRecoversNextMonth 用量按自然月统计，次月额度重发后闸门自动恢复，
// 不需要任何跨月的特殊处理。
func TestGateRecoversNextMonth(t *testing.T) {
	q := DefaultQuota()
	src := &fakeSource{usage: Usage{StorageBytes: pct(q.StorageBytes, 0.95)}}
	g := NewGate(src, q)
	_ = g.Refresh(context.Background())
	if g.Available() {
		t.Fatal("本月应触顶停用")
	}

	src.usage = Usage{StorageBytes: 0} // 次月额度重发
	_ = g.Refresh(context.Background())
	if !g.Available() {
		t.Fatal("次月用量归零后应自动恢复")
	}
}

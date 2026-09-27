package qiniu

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"
)

// Quota 是七牛**每月发放、不结转**的免费额度。
// 任何一项达到 Threshold 即判定「本月额度用尽」，替补层停用，次月自动恢复。
type Quota struct {
	// StorageBytes 是标准存储免费空间（时点值，不累计）。
	StorageBytes int64
	// OriginFlowBytes 是 CDN 回源免费流量（当月累计）。
	OriginFlowBytes int64
	// CDNFlowBytes 是 CDN 下载免费流量（当月累计）——CDN 路径下最先触顶的一项。
	CDNFlowBytes int64
	// GETCount / PUTCount 是免费请求次数（当月累计）。
	GETCount int64
	PUTCount int64
	// EgressBytes 是外网流出的自设上限。
	// 该项**七牛没有免费额度**（0.26 元/GB 从第一字节起计），所以不看百分比，
	// 只能由我们自己划一条线；0 表示不判定（例如桶保持私有、S3 直连走不通时）。
	EgressBytes int64
	// Threshold 是触顶比例，0 表示用默认 0.9。
	Threshold float64
}

// 免费额度与阈值。数据来源：developer.qiniu.com/af/12968/freequota（对象存储）
// 与 www.qiniu.com/prices/qcdn（CDN 下载流量）。
const (
	GB = 1 << 30

	freeStorageBytes    = 10 * GB
	freeOriginFlowBytes = 10 * GB
	freeCDNFlowBytes    = 10 * GB // 国内 10GB + 海外 10GB，取更紧的国内额度
	freeGETCount        = 1_000_000
	freePUTCount        = 100_000

	defaultThreshold = 0.9
)

// DefaultQuota 返回默认的免费额度判定。
func DefaultQuota() Quota {
	return Quota{
		StorageBytes:    freeStorageBytes,
		OriginFlowBytes: freeOriginFlowBytes,
		CDNFlowBytes:    freeCDNFlowBytes,
		GETCount:        freeGETCount,
		PUTCount:        freePUTCount,
		Threshold:       defaultThreshold,
	}
}

func (q Quota) threshold() float64 {
	if q.Threshold <= 0 || q.Threshold >= 1 {
		return defaultThreshold
	}
	return q.Threshold
}

// Ratio 是各项用量占其额度的比例。
type Ratio struct {
	Storage    float64
	OriginFlow float64
	CDNFlow    float64
	GET        float64
	PUT        float64
	Egress     float64
}

// Evaluate 把当月用量换算成占免费额度的比例。
// CDNKnown 为 false 时 CDNFlow 置 0——**查不到就按 0 计入**，因为没绑域名
// 时该指标本就无从产生用量；这不是「把未知当安全」，而是「没有该路径」。
func (q Quota) Evaluate(u Usage) Ratio {
	r := Ratio{
		Storage:    ratio(float64(u.StorageBytes), float64(q.StorageBytes)),
		OriginFlow: ratio(float64(u.OriginFlowBytes), float64(q.OriginFlowBytes)),
		GET:        ratio(float64(u.GETCount), float64(q.GETCount)),
		PUT:        ratio(float64(u.PUTCount), float64(q.PUTCount)),
	}
	if u.CDNKnown {
		r.CDNFlow = ratio(float64(u.CDNFlowBytes), float64(q.CDNFlowBytes))
	}
	if q.EgressBytes > 0 {
		r.Egress = ratio(float64(u.EgressBytes), float64(q.EgressBytes))
	}
	return r
}

func ratio(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total
}

// Max 返回各项比例的最大值。
func (r Ratio) Max() float64 {
	return math.Max(r.Storage, math.Max(r.OriginFlow, math.Max(r.CDNFlow,
		math.Max(r.GET, math.Max(r.PUT, r.Egress)))))
}

// Breached 列出已达到阈值的项（按字段名，日志与排障用）。
func (q Quota) Breached(u Usage) []string {
	t := q.threshold()
	r := q.Evaluate(u)
	var out []string
	check := func(name string, v float64) {
		if v >= t {
			out = append(out, fmt.Sprintf("%s=%.0f%%", name, v*100))
		}
	}
	check("存储", r.Storage)
	check("CDN回源", r.OriginFlow)
	check("CDN下载", r.CDNFlow)
	check("GET", r.GET)
	check("PUT", r.PUT)
	check("外网流出", r.Egress)
	return out
}

// Exceeded 表示任一项已触顶，替补层应当停用。
func (q Quota) Exceeded(u Usage) bool { return len(q.Breached(u)) > 0 }

/* ---------------- 闸门 ---------------- */

// UsageSource 是闸门对用量来源的全部需求，便于测试注入假实现。
type UsageSource interface {
	Usage(ctx context.Context, now time.Time) (Usage, error)
}

// Gate 决定替补层（七牛）当前是否可用。
//
// 结论由周期查询得出，期间只读内存；**查询失败一律保留上次结论**——
// 拿半份数据去开闸，代价是真金白银，拿旧结论顶着，代价只是晚几分钟切换。
// 启动时的初值是关闭：没查过就不开，宁可先少一层，也不要先超支。
//
// 用量本身按自然月统计，次月额度重发后 Evaluate 自然回落，闸门随之自动恢复，
// 不需要任何跨月的特殊处理。
type Gate struct {
	quota    Quota
	src      UsageSource
	mu       sync.RWMutex
	on       bool
	checked  time.Time
	lastErr  error
	lastMax  float64
	breached []string
}

// NewGate 构造闸门；初始为关闭态。
func NewGate(src UsageSource, quota Quota) *Gate {
	if quota.Threshold == 0 {
		quota.Threshold = defaultThreshold
	}
	return &Gate{quota: quota, src: src}
}

// Available 表示替补层当前是否放行。
func (g *Gate) Available() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.on
}

// Status 返回最近一次判定的摘要，供日志与健康检查使用。
func (g *Gate) Status() (on bool, maxRatio float64, breached []string, checkedAt time.Time, err error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.on, g.lastMax, append([]string(nil), g.breached...), g.checked, g.lastErr
}

// Refresh 立即查询一次并更新结论。查询失败时只记错误、不改结论。
func (g *Gate) Refresh(ctx context.Context) error {
	if g == nil || g.src == nil {
		return nil
	}
	u, err := g.src.Usage(ctx, time.Now())
	if err != nil {
		g.mu.Lock()
		g.lastErr = err
		g.mu.Unlock()
		return err
	}
	breached := g.quota.Breached(u)
	ratio := g.quota.Evaluate(u).Max()

	g.mu.Lock()
	g.on = len(breached) == 0
	g.breached = breached
	g.lastMax = ratio
	g.lastErr = nil
	g.checked = time.Now()
	on := g.on
	g.mu.Unlock()

	if len(breached) > 0 {
		slog.Warn("七牛额度触顶，替补层停用至次月",
			"breached", strings.Join(breached, ","), "threshold", g.quota.threshold())
	} else {
		slog.Info("七牛额度检查通过", "max_ratio", fmt.Sprintf("%.1f%%", ratio*100), "available", on)
	}
	return nil
}

// Start 周期刷新闸门：先立即查一轮，再按 interval 重复。
func (g *Gate) Start(ctx context.Context, interval time.Duration) {
	if g == nil || g.src == nil {
		return
	}
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	go func() {
		_ = g.Refresh(ctx)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := g.Refresh(ctx); err != nil {
					slog.Warn("七牛额度查询失败，沿用上次结论", "err", err)
				}
			}
		}
	}()
}

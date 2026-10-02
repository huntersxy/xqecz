package mirror

import (
	"log/slog"
	"time"

	"github.com/huntersxy/xqecz/server/internal/qiniu"
)

// Availability 决定替补层当前是否放行（由七牛额度闸门实现）。
// 用接口而非具体类型，避免镜像包反向依赖额度实现，也便于测试注入。
type Availability interface {
	Available() bool
}

// Chain 把两条镜像目标编排成一条取用链。
//
// **推送侧不分主次**：两边都推，取用优先级只体现在前端（详见
// `packages/frontend/src/utils/imageSource.ts` 的 `bestSource()`）。
// 这里的 primary/secondary 只区分「哪一路」，不代表谁优先：
//
//   - 推送：R2 那一路始终推；七牛那一路只在额度闸门放行时推（省出网与存储额度）；
//   - 取址：PublicURL 给 R2 那一路（无需签名）；
//     Mirror2URL 给七牛那一路（须按时间戳防盗链签名，闸门关闭时返回空串，
//     前端据此连候选都不拿到，探测自然不会打到一个注定 403 的地址）；
//   - 归档：两侧都留一份，七牛侧同样受闸门约束。
//
// 闸门为 nil 表示不设闸（七牛那一路恒放行），便于只配一个目标的环境。
type Chain struct {
	primary   *Setup
	secondary *Setup
	// sign 给替补层地址加时间戳防盗链签名；nil 表示不签名。
	sign func(rawURL string) (string, error)
}

// NewChain 构造取用链。primary 为 nil 时返回空链（所有方法安全空转）。
// gate / sign 均可为 nil。
func NewChain(primary, secondary *Setup, gate Availability, sign func(string) (string, error)) *Chain {
	if primary == nil && secondary == nil {
		return nil
	}
	// 闸门下沉到替补 Setup 上：回填任务直接持有 *Setup，
	// 若只在 Chain 层判闸，触顶后回填仍会继续往替补层推送。
	if secondary != nil {
		secondary.gate = gate
	}
	return &Chain{primary: primary, secondary: secondary, sign: sign}
}

// Enabled 表示链上至少有一个目标可用。
func (c *Chain) Enabled() bool {
	if c == nil {
		return false
	}
	return c.primary.Enabled() || c.secondary.Enabled()
}

// Setups 返回参与回填的目标（主在前、替补在后，已过滤停用者）。
// 每个目标跑自己的回填任务与分布式锁，互不阻塞。
func (c *Chain) Setups() []*Setup {
	if c == nil {
		return nil
	}
	var out []*Setup
	for _, s := range []*Setup{c.primary, c.secondary} {
		if s.Enabled() {
			out = append(out, s)
		}
	}
	return out
}

// secondaryOpen 表示替补层当前是否放行。
// 没配闸门时视为放行——不设闸就是不设限，由调用方（未接入额度管理的环境）负责。
func (c *Chain) secondaryOpen() bool {
	return c != nil && c.secondary != nil && c.secondary.Enabled() && c.secondary.open()
}

// PublicURL 返回 R2 那一路的公开地址，不签名——R2 没有时间戳防盗链。
// 注意它是**次选**：前端拿到后先看七牛，七牛不可达才用它。
func (c *Chain) PublicURL(rel string) string {
	if c == nil || c.primary == nil {
		return ""
	}
	return c.primary.PublicURL(rel)
}

// Mirror2URL 返回七牛那一路的公开地址（前端**首选**它）：闸门关闭时为空串，
// 开启时按时间戳防盗链签名。签名失败同样返回空串——
// 宁可少给一个候选，也不给一个必然 403 的地址让前端白探测一轮。
func (c *Chain) Mirror2URL(rel string) string {
	if !c.secondaryOpen() {
		return ""
	}
	raw := c.secondary.PublicURL(rel)
	if raw == "" || c.sign == nil {
		return raw
	}
	signed, err := c.sign(raw)
	if err != nil {
		slog.Warn("替补层地址签名失败，本轮不下发", "target", c.secondary.cfg.LogName(), "err", err)
		return ""
	}
	return signed
}

// PushAsync 后台推送：主目标始终推，替补目标受闸门约束。
// 任一目标失败只记日志，由各自回填任务兜底。
func (c *Chain) PushAsync(rel, absPath string) {
	if c == nil {
		return
	}
	if c.primary.Enabled() {
		c.primary.PushAsync(rel, absPath)
	}
	if c.secondaryOpen() {
		c.secondary.PushAsync(rel, absPath)
	}
}

// ArchiveOriginalAsync 后台归档压缩前的原图，规则与 PushAsync 一致。
func (c *Chain) ArchiveOriginalAsync(rel, origPath string) {
	if c == nil {
		return
	}
	if c.primary.Enabled() {
		c.primary.ArchiveOriginalAsync(rel, origPath)
	}
	if c.secondaryOpen() {
		c.secondary.ArchiveOriginalAsync(rel, origPath)
	}
}

// BinDir 返回垃圾桶目录（回填任务据此查找原图），取自主目标。
func (c *Chain) BinDir() string {
	if c == nil || c.primary == nil {
		return ""
	}
	return c.primary.BinDir()
}

// Signer 返回七牛时间戳签名函数；未配密钥时返回 nil（替补层地址不签名）。
func Signer(timeKey string) func(string) (string, error) {
	if timeKey == "" {
		return nil
	}
	return func(rawURL string) (string, error) {
		return qiniu.SignURL(rawURL, timeKey, time.Now().Add(qiniu.SignTTL))
	}
}

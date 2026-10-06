package mirror

// Chain 把若干镜像目标编排成一条链：推送侧不分主次（两边都推），
// 优先级只体现在前端取用顺序上（见 frontend/src/utils/imageSource.ts）。
//
// 两级各有独立的公开基址，某一级停用时只是少一个候选，不影响另一级：
// 前端据此逐级回退 OpenList → R2 → 源站。
//
// 历史上这里有第三个组件——一个七牛的额度闸门（internal/qiniu 的 Gate），
// 用来在免费额度触顶前整层停用。七牛已下线，闸门随之删除：
// 剩下的两个目标（自建 OpenList、Cloudflare R2）没有按量计费的免费额度，
// 不需要一个「到 90% 就关掉」的开关。
type Chain struct {
	// primary 是 R2，secondary 是 OpenList——顺序只影响日志与 Setups 的排列，
	// 不代表任何优先级。
	primary   *Setup
	secondary *Setup
}

// NewChain 组装镜像链；两级都不可用时返回 nil（调用方据此整体跳过镜像装配）。
func NewChain(primary, secondary *Setup) *Chain {
	if primary == nil && secondary == nil {
		return nil
	}
	return &Chain{primary: primary, secondary: secondary}
}

// Enabled 表示至少有一个目标可用。
func (c *Chain) Enabled() bool { return c != nil && (c.primary.Enabled() || c.secondary.Enabled()) }

// Setups 返回可用目标，供回填任务逐个启动（每个目标各持一把锁）。
func (c *Chain) Setups() []*Setup {
	if c == nil {
		return nil
	}
	var out []*Setup
	if c.primary.Enabled() {
		out = append(out, c.primary)
	}
	if c.secondary.Enabled() {
		out = append(out, c.secondary)
	}
	return out
}

// secondaryOpen 表示第二级可用于下发地址。
func (c *Chain) secondaryOpen() bool {
	return c != nil && c.secondary != nil && c.secondary.Enabled()
}

// R2URL 返回 R2 侧的公开地址（未配置公开基址时为空）。
func (c *Chain) R2URL(rel string) string {
	if c == nil || c.primary == nil {
		return ""
	}
	return c.primary.PublicURL(rel)
}

// OpenListURL 返回 OpenList 侧的公开地址（未配置公开基址时为空）。
//
// 注意这里不再有签名环节：七牛的替补层曾需要时间戳防盗链签名，
// OpenList 的直链靠服务端 sign_all=false 免签开放，
// 由 nginx 侧限制写入类接口（见 docs/deploy.md）。
func (c *Chain) OpenListURL(rel string) string {
	if !c.secondaryOpen() {
		return ""
	}
	return c.secondary.PublicURL(rel)
}

// PushAsync 异步把一份本地文件推到所有可用目标。
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

// ArchiveOriginalAsync 异步把原图归档到所有可用目标。
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

// BinDir 返回垃圾桶目录（取首个可用目标上的值，两级共享同一份配置）。
func (c *Chain) BinDir() string {
	if c == nil {
		return ""
	}
	if c.primary.Enabled() {
		return c.primary.BinDir()
	}
	if c.secondary.Enabled() {
		return c.secondary.BinDir()
	}
	return ""
}

package mirror

type Chain struct {
	primary   *Setup
	secondary *Setup
}

func NewChain(primary, secondary *Setup) *Chain {
	if primary == nil && secondary == nil {
		return nil
	}
	return &Chain{primary: primary, secondary: secondary}
}

func (c *Chain) Enabled() bool { return c != nil && (c.primary.Enabled() || c.secondary.Enabled()) }

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

func (c *Chain) secondaryOpen() bool {
	return c != nil && c.secondary != nil && c.secondary.Enabled()
}

func (c *Chain) R2URL(rel string) string {
	if c == nil || c.primary == nil {
		return ""
	}
	return c.primary.PublicURL(rel)
}

func (c *Chain) OpenListURL(rel string) string {
	if !c.secondaryOpen() {
		return ""
	}
	return c.secondary.PublicURL(rel)
}

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

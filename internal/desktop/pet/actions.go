package pet

// hide persists the same opt-out as the original panel action. An OS window
// close remains temporary; a failed save keeps the visible pet recoverable.
func (p *Companion) hide() {
	if p.Store == nil {
		return
	}
	cfg := p.Store.DesktopPet()
	cfg.Enabled = false
	if err := p.Store.SaveDesktopPet(cfg); err != nil {
		p.Error = err.Error()
		p.assetMu.Lock()
		p.feedMessage = "无法保存隐藏设置，请重试 / Could not save Hide; try again"
		p.assetMu.Unlock()
		return
	}
	p.Enabled = false
	p.Error = ""
	p.close()
	if p.OnEnabledChanged != nil {
		p.OnEnabledChanged()
	}
}

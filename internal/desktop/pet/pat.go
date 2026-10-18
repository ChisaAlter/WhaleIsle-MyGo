package pet

import "time"

// patPet is a fixed native action for the original PANEL_ACTIONS pat entry.
// No animation gesture, arbitrary care bridge or inference is needed to apply
// its real persistent effect. Native picker activity blocks a parallel action.
func (p *Companion) patPet() {
	if p.win == nil || p.win.IsDestroyed() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.feeding {
		return
	}
	if err := p.Store.PatPet(time.Now()); err != nil {
		p.feedMessage = "保存失败，未计入互动 / Save failed; unchanged"
		return
	}
	p.careAction = newCareAnimation("pat", "", 0)
	_ = p.rest.wake(nil)
	p.feedMessage = "已摸摸头：心情 +10；亲密按同类冷却计入 / Patted"
}

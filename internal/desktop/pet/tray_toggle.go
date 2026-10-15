package pet

import "errors"

// SetEnabled changes only the committed visibility preference. In particular a
// tray click must not save unrelated, uncommitted settings-panel drafts.
func (p *Companion) SetEnabled(enabled bool) error {
	if p.Store == nil {
		return errors.New("pet configuration is unavailable")
	}
	cfg := p.Store.DesktopPet()
	cfg.Enabled = enabled
	if err := p.Store.SaveDesktopPet(cfg); err != nil {
		p.Error = err.Error()
		return err
	}
	p.Enabled = enabled
	p.Error = ""
	p.sync()
	if p.OnEnabledChanged != nil {
		p.OnEnabledChanged()
	}
	return nil
}

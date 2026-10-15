package pet

import "whaleisle/internal/config"

// ResetAppearanceBehavior is called only after the settings confirmation. Like
// upstream applySettings({reset:true}), it resets preferences, not visibility,
// position or care/growth. MyGo's skin and renderer selection are also outside
// its behavior object. Unrelated settings-panel drafts are never committed.
func (p *Companion) ResetAppearanceBehavior() bool {
	if p == nil {
		return false
	}
	if p.Store == nil {
		p.Error = "桌宠配置不可用 / Pet configuration unavailable"
		return false
	}
	next := p.Store.DesktopPet()
	previousPersonality := next.Behavior.Personality
	next.Behavior = config.DefaultPetBehavior()
	next.PersonalityConfigured = true
	if err := p.Store.SaveDesktopPet(next); err != nil {
		p.Error = err.Error()
		return false
	}
	// Publish only after the conflict-checked atomic save succeeds. The renderer
	// already receives saved Behavior through task.json; no window recreation,
	// skin scan, model request or reset of the current rest state is needed.
	p.Behavior = next.Behavior
	p.Error = ""
	p.applyBehavior()
	p.CheckQuickChat()
	if previousPersonality != next.Behavior.Personality && p.OnPersonality != nil {
		p.OnPersonality(next.Behavior.Personality)
	}
	return true
}

package pet

import (
	"whaleisle/internal/config"
	"whaleisle/internal/store/storelocale"
)

// ResetAppearanceBehavior is called only after the settings confirmation. Like
// upstream applySettings({reset:true}), it resets preferences, not visibility,
// position or care/growth. MyGo's skin and renderer selection are also outside
// its behavior object. Unrelated settings-panel drafts are never committed.
func (p *Companion) ResetAppearanceBehavior() bool {
	if p == nil {
		return false
	}
	if p.Store == nil {
		p.Error = storelocale.T(storelocale.KeyDesktopPetPetConfigurationUnavailable)
		return false
	}
	next := p.Store.DesktopPet()
	previousBehavior := next.Behavior
	previousPersonality := previousBehavior.Personality
	next.Behavior = config.DefaultPetBehavior()
	next.PersonalityConfigured = true
	if err := p.Store.SaveDesktopPet(next); err != nil {
		p.Error = err.Error()
		return false
	}
	// Publish only after the conflict-checked atomic save succeeds. The renderer
	// already receives saved Behavior through task.json; no window recreation,
	// skin scan, model request or reset of the current rest state is needed.
	p.cancelChangedLook(previousBehavior, next.Behavior)
	p.Behavior = next.Behavior
	p.Error = ""
	p.applyBehavior()
	p.CheckQuickChat()
	if previousPersonality != next.Behavior.Personality && p.OnPersonality != nil {
		p.OnPersonality(next.Behavior.Personality)
	}
	return true
}

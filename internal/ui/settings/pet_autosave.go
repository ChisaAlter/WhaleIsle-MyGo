package settings

import "whaleisle/internal/desktop/pet"

// Original settings patches only the control that changed. Keep the native
// skin/position import draft out of these immediate writes; its explicit Save
// still commits that draft through Companion.Save.
func savePetSetting(p *pet.Companion, access *PetSettingsAccess, field string) {
	if p.Store == nil {
		if access != nil && access.OnSave != nil {
			access.OnSave()
		}
		return
	}
	enabled, skin, neural, behavior := p.Enabled, p.SkinID, p.Neural, p.Behavior
	position, report := p.PositionDraft, p.OriginalImportReport
	next := p.Store.DesktopPet()
	switch field {
	case "enabled":
		next.Enabled = enabled
	case "scale":
		next.Behavior.Scale = behavior.Scale
	case "opacity":
		next.Behavior.Opacity = behavior.Opacity
	case "personality":
		next.Behavior.Personality = behavior.Personality
	case "activity":
		next.Behavior.Activity = behavior.Activity
	case "selfTalk":
		next.Behavior.SelfTalk = behavior.SelfTalk
	case "wander":
		next.Behavior.Wander = behavior.Wander
	case "drag":
		next.Behavior.LockPosition, next.Behavior.ShiftToDrag = behavior.LockPosition, behavior.ShiftToDrag
	case "powerSave":
		next.Behavior.PowerSave = behavior.PowerSave
	case "clickSound":
		next.Behavior.ClickSound = behavior.ClickSound
	case "chatEnabled":
		next.Behavior.ChatEnabled = behavior.ChatEnabled
	default:
		return
	}
	p.Enabled, p.SkinID, p.Neural, p.Behavior = next.Enabled, next.SkinID, next.Neural, next.Behavior
	p.PositionDraft, p.OriginalImportReport = nil, nil
	p.Save()
	p.Enabled, p.SkinID, p.Neural, p.Behavior = enabled, skin, neural, behavior
	p.PositionDraft, p.OriginalImportReport = position, report
}

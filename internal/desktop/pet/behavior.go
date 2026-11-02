package pet

import (
	"encoding/json"
	"html"
	"strings"

	"whaleisle/internal/config"
)

// savedBehavior deliberately ignores pending UI edits when a store exists.
// The protocol handler can safely read Store without accessing mutable form
// fields, and a failed save leaves the running pet's preferences unchanged.
func (p *Companion) savedBehavior() config.PetBehavior {
	if p == nil {
		return config.DefaultPetBehavior()
	}
	if p.Store != nil {
		return p.Store.DesktopPet().Behavior
	}
	return config.NormalizePetBehavior(p.Behavior)
}

// ChatAllowed is checked at both the pet action and the main chat entry point.
func (p *Companion) ChatAllowed() bool {
	return p != nil && p.savedBehavior().ChatEnabled
}

func renderBehaviorPage(page []byte, behavior config.PetBehavior) []byte {
	data, _ := json.Marshal(config.NormalizePetBehavior(behavior))
	return []byte(strings.ReplaceAll(string(page), "{{PET_BEHAVIOR}}", html.EscapeString(string(data))))
}

// applyBehavior updates the existing native window. Display-only preference
// saves must not recreate the page, stop inference, or reset its rest state.
// The native overlay never resizes with growth. Only the character's logical
// rectangle and page layout change, preserving unclipped poses and floating UI.
func petWindowHeight(behavior config.PetBehavior) int {
	_ = config.NormalizePetBehavior(behavior) // validated on save; never a size input
	return PetWindowHeight
}

func (p *Companion) applyBehavior() {
	if p == nil || p.win == nil || p.win.IsDestroyed() {
		return
	}
	p.sizeStage()
	if area, ok := motionWorkArea(p.petBounds()); ok {
		b := p.petBounds()
		position := clampMotionPoint(p.petPosition(), float64(b.Width), float64(b.Height), area)
		p.stage.X, p.stage.Y = int(position.x), int(position.y)
	}
	p.publishLayout()
}

package pet

import (
	"encoding/json"
	"html"
	"math"
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
func petWindowHeight(behavior config.PetBehavior) int {
	behavior = config.NormalizePetBehavior(behavior)
	// Snapped tenths can carry binary floating-point noise.
	return 438 + int(math.Ceil(math.Max(0, behavior.Scale-1)*110-1e-9))
}

func (p *Companion) applyBehavior() {
	if p == nil || p.win == nil || p.win.IsDestroyed() {
		return
	}
	p.win.SetSize(220, petWindowHeight(p.savedBehavior()))
}

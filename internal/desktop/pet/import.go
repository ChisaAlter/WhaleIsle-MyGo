package pet

import (
	"encoding/json"
	"os"
	"strings"
	"whaleisle/internal/config"
)

// LoadOriginalDraft is called only by the explicit Settings import button.
// Discovery reads confined local resources; it never executes package code.
func (p *Companion) LoadOriginalDraft() {
	if p.Store == nil {
		return
	}
	skins, err := Discover(DefaultRoot())
	if err != nil {
		p.Error = err.Error()
		return
	}
	p.Skins = skins
	ids := originalSkinIDs(DefaultRoot(), skins)
	base := p.Store.DesktopPet()
	base.Enabled, base.Neural, base.SkinID, base.Behavior = p.Enabled, p.Neural, p.SkinID, p.Behavior
	base.Position = p.PositionDraft
	draft, report := p.Store.OriginalPetDraft(base, ids)
	p.Behavior, p.SkinID = draft.Behavior, draft.SkinID
	// Retain coordinates in the UI draft only. Save commits them atomically with
	// appearance; visibility remains a separate explicit opt-in.
	p.PositionDraft = draft.Position
	p.OriginalImportReport = report
	p.Error = ""
}

func originalSkinIDs(dir string, skins []Skin) map[string]string {
	out := map[string]string{}
	duplicates := map[string]bool{}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return out
	}
	defer root.Close()
	for _, skin := range skins {
		sub, err := root.OpenRoot(skin.ID)
		if err != nil {
			continue
		}
		data, err := limited(sub, "pet.json", 64<<10)
		sub.Close()
		if err != nil {
			continue
		}
		var manifest struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		id := strings.TrimSpace(manifest.ID)
		if id == "" {
			id = skin.ID
		}
		if _, exists := out[id]; exists {
			duplicates[id] = true
		}
		out[id] = skin.ID
	}
	for id := range duplicates {
		delete(out, id)
	}
	return out
}

// PositionForSave leaves a newer manual drag alone unless a position import is
// actually pending. The caller clears PositionDraft only after a successful save.
func (p *Companion) PositionForSave(previous *config.PetPosition) *config.PetPosition {
	if p.PositionDraft != nil {
		return p.PositionDraft
	}
	return previous
}

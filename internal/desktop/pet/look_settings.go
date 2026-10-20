package pet

import (
	"whaleisle/internal/config"
	"whaleisle/internal/petlook"
	"whaleisle/internal/store/storelocale"
)

// SaveLookRoute commits only this pet's pair; a catalog read or picker render
// never invokes it. An explicit Off remains available while Host is offline.
func (p *Companion) SaveLookRoute(provider, model string, catalog petlook.Catalog) bool {
	if p == nil {
		return false
	}
	if p.Store == nil {
		p.Error = storelocale.T(storelocale.KeyDesktopPetPetConfigurationUnavailable)
		return false
	}
	if (provider != "" || model != "") && !petlook.Resolve(provider, model, catalog) {
		p.Error = storelocale.T(storelocale.KeyDesktopPetRefreshTheImageCatalogOrTurnLookO)
		return false
	}
	next := p.Store.DesktopPet()
	previous := next.Behavior
	next.Behavior.LookProvider, next.Behavior.LookModel = provider, model
	if err := p.Store.SaveDesktopPet(next); err != nil {
		p.Error = err.Error()
		return false
	}
	p.Behavior.LookProvider, p.Behavior.LookModel = provider, model
	p.Error = ""
	p.cancelChangedLook(previous, next.Behavior)
	return true
}

func (p *Companion) cancelChangedLook(before, after config.PetBehavior) {
	if before.LookProvider == after.LookProvider && before.LookModel == after.LookModel {
		return
	}
	p.assetMu.Lock()
	p.look.close()
	p.assetMu.Unlock()
}

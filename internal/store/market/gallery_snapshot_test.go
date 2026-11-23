package market

import (
	"path/filepath"
	"testing"

	desktopmarket "whaleisle/internal/desktop/market"
)

// Discovery projections share the immutable catalog items instead of
// copying screenshot slices per read; a later details-profile update
// replaces the item in a fresh slice, so a held projection keeps its item.
func TestDiscoveryScreenshotSlicesSharedAndHeldProjectionStable(t *testing.T) {
	s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{{ID: "owned/plugin", InstallSpec: "owned-plugin", Screenshots: []string{"https://owned.invalid/original.png"}}}
	s.discovery.selected = "owned/plugin"
	s.republish()
	view := s.Discovery()
	if view.Selected == nil || len(view.Rows) != 1 || len(view.Catalog.Items) != 1 {
		t.Fatalf("fixture not present in every discovery projection: %+v", view)
	}
	if &view.Rows[0].Screenshots[0] != &view.Catalog.Items[0].Screenshots[0] || view.Selected != &view.Catalog.Items[0] {
		t.Fatal("discovery projection copied catalog items instead of sharing them")
	}
	s.SetCatalogService(stubMarketCatalog{details: desktopmarket.Details{CapabilityKind: desktopmarket.CapabilityHybrid, Capabilities: desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilityHybrid, NativeCompatible: true}}})
	s.mu.Lock()
	s.discovery.catalog.Items = view.Catalog.Items
	s.publishLocked()
	s.mu.Unlock()
	held := s.Discovery()
	s.SelectDetails("owned/plugin", false)
	waitFor(t, func() bool { return !s.Discovery().DetailsLoading })
	if held.Catalog.Items[0].CapabilityKind == desktopmarket.CapabilityHybrid {
		t.Fatal("details profile was applied to a published catalog item in place")
	}
	if got := s.Discovery().Catalog.Items[0].CapabilityKind; got != desktopmarket.CapabilityHybrid {
		t.Fatalf("details profile not published: %q", got)
	}
}

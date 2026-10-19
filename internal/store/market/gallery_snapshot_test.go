package market

import (
	"path/filepath"
	"testing"

	desktopmarket "whaleisle/internal/desktop/market"
)

func TestDiscoveryScreenshotSlicesDoNotMutateCatalogOrEachOther(t *testing.T) {
	s := NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{{ID: "owned/plugin", Screenshots: []string{"https://owned.invalid/original.png"}}}
	s.discovery.selected = "owned/plugin"
	view := s.Discovery()
	if view.Selected == nil || len(view.Rows) != 1 || len(view.Catalog.Items) != 1 {
		t.Fatalf("fixture not present in every discovery projection: %+v", view)
	}
	view.Catalog.Items[0].Screenshots[0] = "catalog snapshot mutation"
	if view.Rows[0].Screenshots[0] != "https://owned.invalid/original.png" || view.Selected.Screenshots[0] != "https://owned.invalid/original.png" {
		t.Fatal("catalog snapshot shared screenshot slices with rows or selection")
	}
	view.Rows[0].Screenshots[0] = "row snapshot mutation"
	view.Selected.Screenshots[0] = "selection snapshot mutation"
	if got := s.Discovery().Catalog.Items[0].Screenshots[0]; got != "https://owned.invalid/original.png" {
		t.Fatalf("caller mutated retained catalog screenshot metadata: %q", got)
	}
}

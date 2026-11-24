package market

import (
	"context"
	"testing"

	desktopmarket "whaleisle/internal/desktop/market"
)

type stubMarketCatalog struct {
	items   []desktopmarket.CatalogItem
	details desktopmarket.Details
}

func (c stubMarketCatalog) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{Items: c.items}, nil
}

func (c stubMarketCatalog) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return c.details, nil
}

func TestNativeCompatibleOnlyDefaultFilterAndDetailsSync(t *testing.T) {
	s := NewWithPersonalPath("")
	if d := s.Discovery(); !d.NativeCompatibleOnly {
		t.Fatal("expected NativeCompatibleOnly to default to true")
	}
	uiItem := desktopmarket.CatalogItem{ID: "a/ui", Category: "ui"}
	desktopmarket.ApplyCapabilityProfile(&uiItem, desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilityWebUIOnly})
	hostItem := desktopmarket.CatalogItem{ID: "b/host", Category: "tools"}
	desktopmarket.ApplyCapabilityProfile(&hostItem, desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilityPureHost})
	mcpItem := desktopmarket.CatalogItem{ID: "c/mcp", Category: "tools"}
	desktopmarket.ApplyCapabilityProfile(&mcpItem, desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilityMCP})
	skillItem := desktopmarket.CatalogItem{ID: "d/skill", Category: "skill"}
	desktopmarket.ApplyCapabilityProfile(&skillItem, desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilitySkill})

	s.SetCatalogService(stubMarketCatalog{
		items: []desktopmarket.CatalogItem{uiItem, hostItem, mcpItem, skillItem},
		details: desktopmarket.Details{
			CapabilityKind:   desktopmarket.CapabilityHybrid,
			NativeCompatible: true,
			Capabilities:     desktopmarket.PackageCapabilities{Kind: desktopmarket.CapabilityHybrid, NativeCompatible: true, CompatibilityLabel: "Hybrid"},
		},
	})
	s.discovery.catalog.Items = []desktopmarket.CatalogItem{uiItem, hostItem, mcpItem, skillItem}
	s.republish()

	filtered := s.Discovery()
	if !filtered.NativeCompatibleOnly || filtered.Total != 3 || len(filtered.Rows) != 3 {
		t.Fatalf("expected web-ui-only item hidden by default: total=%d rows=%+v", filtered.Total, filtered.Rows)
	}
	for _, row := range filtered.Rows {
		if row.ID == "a/ui" {
			t.Fatal("web-ui-only row leaked through NativeCompatibleOnly filter")
		}
	}

	s.SetNativeCompatibleOnly(false)
	all := s.Discovery()
	if all.NativeCompatibleOnly || all.Total != 4 || len(all.Rows) != 4 {
		t.Fatalf("expected all 4 items when NativeCompatibleOnly=false: total=%d", all.Total)
	}
}

package market

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

type capabilityCatalogFixture struct {
	items   []desktopmarket.CatalogItem
	details map[string]desktopmarket.Details
}

func (f capabilityCatalogFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{
		Items: f.items,
		Categories: []desktopmarket.Category{
			{ID: "all", Label: "All", Count: len(f.items)},
			{ID: "tools", Label: "Tools", Count: len(f.items)},
		},
	}, nil
}

func (f capabilityCatalogFixture) Details(_ context.Context, item desktopmarket.CatalogItem, _ bool) (desktopmarket.Details, error) {
	if d, ok := f.details[item.ID]; ok {
		return d, nil
	}
	return desktopmarket.Details{
		Version:                  "1.0.0",
		CapabilityKind:           item.CapabilityKind,
		NativeCompatible:         item.NativeCompatible,
		NativeCompatibilityLabel: item.NativeCompatibilityLabel,
		Capabilities:             item.Capabilities,
	}, nil
}

func TestMarketCapabilityFilterDefaultOnWebUIOnlyGuidanceDetailsAndInstalledBadges(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	hostItem := desktopmarket.CatalogItem{
		ID: "org/native-host", Repo: "native-host", Owner: "org",
		PackageName: "@org/native-host", InstallSpec: "@org/native-host",
		Category: "tools", Description: "Native host plugin.",
	}
	desktopmarket.ApplyCapabilityProfile(&hostItem, desktopmarket.PackageCapabilities{
		Kind:             desktopmarket.CapabilityPureHost,
		NativeCompatible: true,
	})

	webItem := desktopmarket.CatalogItem{
		ID: "org/web-panel", Repo: "web-panel", Owner: "org",
		PackageName: "@org/web-panel", InstallSpec: "@org/web-panel",
		Category: "tools", Description: "Legacy browser panel.",
	}
	desktopmarket.ApplyCapabilityProfile(&webItem, desktopmarket.PackageCapabilities{
		Kind: desktopmarket.CapabilityWebUIOnly,
	})

	s := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	s.SetCatalogService(capabilityCatalogFixture{
		items: []desktopmarket.CatalogItem{hostItem, webItem},
		details: map[string]desktopmarket.Details{
			hostItem.ID: {
				Version:                  "2.0.0",
				CapabilityKind:           desktopmarket.CapabilityHybrid,
				NativeCompatible:         true,
				NativeCompatibilityLabel: "Native + Web (Hybrid)",
			},
		},
	})
	wait := func(pred func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !pred() && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !pred() {
			t.Fatalf("market catalog/details did not settle: %+v", s.Discovery())
		}
	}
	s.RefreshCatalog(false, "en")
	wait(func() bool { return !s.Discovery().Loading && len(s.Discovery().Catalog.Items) == 2 })
	if !s.Discovery().NativeCompatibleOnly {
		t.Fatal("expected NativeCompatibleOnly default true")
	}

	m := Model{Store: s, ThemeID: "deepseek.light"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() { View(c, m) })
	}, 900, 900)

	// Default filter hides Web UI-only item and keeps native Host badge visible.
	if tt.HasText("web-panel") {
		t.Fatal("web-ui-only card leaked under default NativeCompatibleOnly=true filter")
	}
	if !tt.HasText("native-host") || !tt.HasText("Pure native (Host)") {
		t.Fatalf("native host badge missing from default market discovery view: %v", tt.Texts())
	}

	// Switching to All plugins reveals web-ui-only badge and compatibility workbench guidance.
	if err := tt.Click("All plugins"); err != nil {
		t.Fatalf("click All plugins failed: %v", err)
	}
	if s.Discovery().NativeCompatibleOnly {
		t.Fatal("clicking All plugins did not disable NativeCompatibleOnly")
	}
	if !tt.HasText("web-panel") || !tt.HasText("Web UI only") || !tt.HasText("compatibility workbench") {
		t.Fatalf("web-ui-only badge or guidance missing after toggling All plugins: %v", tt.Texts())
	}

	// Toggling back to Native compatible only hides web-ui-only again.
	if err := tt.Click("Native compatible only"); err != nil {
		t.Fatalf("click Native compatible only failed: %v", err)
	}
	if !s.Discovery().NativeCompatibleOnly || tt.HasText("web-panel") {
		t.Fatal("Native compatible only toggle did not re-filter web-ui-only item")
	}

	// Details dialog resolves capability metadata instead of hardcoded Unknown.
	s.SelectDetails(hostItem.ID, false)
	wait(func() bool { return !s.Discovery().DetailsLoading && s.Discovery().Details.Version == "2.0.0" })
	tt.Frame()
	if !tt.HasText("Native + Web (Hybrid)") || !tt.HasText("Host features run natively") {
		t.Fatalf("details dialog did not surface probed capability metadata: %v", tt.Texts())
	}
	s.CloseDetails()
	tt.Frame()

	// Installed rows render capability badge and web-ui-only guidance.
	inv := marketstore.ViewState{
		ClientReady: true,
		Rows: []marketstore.Entry{
			{
				Label:            "@org/mcp-helper",
				Detail:           "1.2.0",
				IdentityKnown:    true,
				Installed:        true,
				CapabilityKind:   desktopmarket.CapabilityMCP,
				NativeCompatible: true,
			},
			{
				Label:          "@org/legacy-web",
				Detail:         "0.9.0",
				IdentityKnown:  true,
				Installed:      true,
				CapabilityKind: desktopmarket.CapabilityWebUIOnly,
			},
		},
	}
	installedTester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			installedRows(c, testPage(m, marketstore.Discovery{}, inv))
		})
	}, 800, 500)
	if !installedTester.HasText("MCP server (Native)") || !installedTester.HasText("Web UI only") || !installedTester.HasText("compatibility workbench") {
		t.Fatalf("installed rows missing capability badges or web-ui-only guidance: %v", installedTester.Texts())
	}
}

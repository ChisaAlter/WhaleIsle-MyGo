package market

import (
	"path/filepath"
	"testing"

	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

// TestMarketSortSelectDoesNotExposeInternalCatalogKey verifies that the sort
// and period selects display and expose localized labels rather than raw
// internal keys like "catalog" or "all".
func TestMarketSortSelectDoesNotExposeInternalCatalogKey(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	d := marketstore.Discovery{
		Tab:          "discover",
		Category:     "all",
		Sort:         "catalog",
		Period:       "all",
		RuntimeReady: true,
		Catalog: desktopmarket.Catalog{
			Source: "registry",
			Items:  []desktopmarket.CatalogItem{{ID: "owner/demo", Repo: "Demo Plugin", Owner: "owner"}},
		},
		Rows:  []desktopmarket.CatalogItem{{ID: "owner/demo", Repo: "Demo Plugin", Owner: "owner"}},
		Total: 1,
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(16).Children(func() {
			p := testPage(Model{Store: store}, d, marketstore.ViewState{})
			discoveryView(c, p)
		})
	}, 900, 700)
	tt.Frame()

	for _, text := range tt.Texts() {
		if text == "catalog" {
			t.Fatalf("internal sort key 'catalog' leaked into rendered texts: %v", tt.Texts())
		}
	}
	if !tt.HasText(i18n.T(i18n.KeyW4MarketCatalogOrder)) {
		t.Fatalf("localized sort label %q missing: %v", i18n.T(i18n.KeyW4MarketCatalogOrder), tt.Texts())
	}
	if !tt.HasText(i18n.T(i18n.KeyW4MarketAnyTime)) {
		t.Fatalf("localized period label %q missing: %v", i18n.T(i18n.KeyW4MarketAnyTime), tt.Texts())
	}
}

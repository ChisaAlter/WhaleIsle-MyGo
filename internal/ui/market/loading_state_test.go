package market

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	desktopmarket "whaleisle/internal/desktop/market"

	"github.com/egoist/mygo/ui"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

// TestMarketLoadingAndEmptyAreDifferentStates covers what a live pass caught on
// one screen: the catalog was still loading, and the page said "0 项插件" beside
// an empty source and update date. Loading is not emptiness, and a fetched
// catalog with no rows is not emptiness either until it says which source it read.
func TestMarketLoadingAndEmptyAreDifferentStates(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	render := func(d marketstore.Discovery) []string {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(12).Children(func() {
				discoveryView(c, testPage(Model{Store: store}, d, marketstore.ViewState{}))
			})
		}, 800, 1400)
		return tt.Texts()
	}

	loading := render(marketstore.Discovery{Loading: true, Category: "all", Period: "all", Sort: "catalog"})
	shown := strings.Join(loading, "\n")
	if !strings.Contains(shown, "正在加载插件目录…") {
		t.Fatalf("the loading state is not stated: %q", shown)
	}
	if strings.Contains(shown, "0 项插件") {
		t.Fatalf("a loading catalog claims to hold 0 plugins: %q", shown)
	}
	if strings.Contains(shown, "目录来源") || strings.Contains(shown, "更新日期") {
		t.Fatalf("a loading catalog prints an empty source and date: %q", shown)
	}
	if strings.Contains(shown, "还没有可用的插件") || strings.Contains(shown, "No plugins available") {
		t.Fatalf("a loading catalog claims to be empty: %q", shown)
	}

	empty := render(marketstore.Discovery{Category: "all", Period: "all", Sort: "catalog", Catalog: desktopmarket.Catalog{FetchedAt: time.Now()}})
	shown = strings.Join(empty, "\n")
	if !strings.Contains(shown, "0 项插件") {
		t.Fatalf("a fetched empty catalog does not state its count: %q", shown)
	}
	if !strings.Contains(shown, "目录来源：未知") || !strings.Contains(shown, "更新日期：—") {
		t.Fatalf("a fetched catalog with no source does not say so: %q", shown)
	}
}

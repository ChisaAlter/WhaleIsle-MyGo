package market

import (
	"github.com/egoist/mygo/ui"
	"path/filepath"
	"testing"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

func TestHistoryExportInvokesNativeAction(t *testing.T) {
	store := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "market.json"))
	if err := store.RecordInstall("owner/plugin", "applied", ""); err != nil {
		t.Fatal(err)
	}
	store.SetTab("history")
	calls := 0
	tt := ui.NewTester(func(c *ui.Context) { View(c, Model{Store: store, OnExportHistory: func() { calls++ }}) }, 1200, 1800)
	if err := tt.Click(marketText(i18n.T(i18n.KeyW4MarketExportInstallHistory), "Export installation history")); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("export callback called %d times", calls)
	}
}

package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	desktopmarket "whaleisle/internal/desktop/market"
	"whaleisle/internal/dsh"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
)

type cardCatalogFixture struct{ item desktopmarket.CatalogItem }

func (f cardCatalogFixture) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{Items: []desktopmarket.CatalogItem{f.item}}, nil
}
func (f cardCatalogFixture) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{Version: "1.1.0", Readme: "Local fixture documentation."}, nil
}
func (f cardCatalogFixture) LatestVersion(context.Context, desktopmarket.CatalogItem) (string, error) {
	return "1.1.0", nil
}

func TestInstalledCatalogCardsKeepConfirmationAndVersionReceipts(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ RPCID, Method string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		var value any = map[string]any{}
		switch req.Method {
		case "pluginManager/listBundles":
			value = []any{map[string]any{"name": "fixture-plugin", "version": "1.0.0", "source": "fixture-plugin@1.0.0", "installed": true, "removable": true, "enabled": true}}
		case "pluginRegistryProbe/fastest":
			value = ""
		case "pluginManager/registries", "pluginManager/listVersionExemptions":
		default:
			t.Errorf("UI launched unexpected runtime operation %s", req.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
	}))
	defer server.Close()
	s := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: server.URL}}, nil)
	defer s.Detach()
	item := desktopmarket.CatalogItem{ID: "owner/fixture", Repo: "Fixture plugin", PackageName: "fixture-plugin", InstallSpec: "fixture-plugin", Description: "Fixture plugin update and uninstall actions.", Homepage: "https://example.invalid/fixture"}
	s.SetCatalogService(cardCatalogFixture{item})
	s.RefreshCatalog(false, "en")
	s.Refresh(context.Background())
	wait := func(predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for !predicate() && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if !predicate() {
			t.Fatal("market fixture did not become ready", s.Snapshot(), s.Updates())
		}
	}
	wait(func() bool { return len(s.Updates().Rows) == 1 && !s.Updates().Loading })
	var targets []marketstore.InstallTarget
	var removed []string
	opened := ""
	spec := "fixture-plugin@1.1.0"
	m := Model{Store: s, ThemeID: "deepseek.light", InstallSpec: &spec, OnInstallCatalog: func(target marketstore.InstallTarget) { targets = append(targets, target) }, OnRemoveBundle: func(name string) { removed = append(removed, name) }, OnOpenURL: func(url string) { opened = url }}
	detailsOnly := false
	tt := ui.NewTester(func(c *ui.Context) {
		if !detailsOnly {
			ui.Column(c).Fill().Children(func() { View(c, m) })
			return
		}
		// Scope duplicate Update labels to the active dialog, rather than
		// asking the headless first-label finder to click the masked card.
		owned := withMarketConfirmations(m)
		catalogDetailsDialog(c, owned, s.Discovery(), s.Snapshot(), c.Theme().Text, c.Theme().TextMuted, c.Theme().Surface, c.Theme().Accent)
		confirmationView(c, owned, c.Theme().Text, c.Theme().Accent)
	}, 800, 1000)
	click := func(label string) {
		t.Helper()
		if err := tt.Click(label); err != nil {
			t.Fatal(err)
		}
		tt.Frame()
	}
	for _, tab := range []string{"discover", "favorites"} {
		if tab == "favorites" {
			if err := s.ToggleFavorite(item.ID); err != nil {
				t.Fatal(err)
			}
			s.SetTab(tab)
			tt.Frame()
		}
		if !tt.HasText("1.0.0 → 1.1.0") {
			t.Fatal("installed catalog card lost update versions", tt.Texts())
		}
		click("Update")
		if len(targets) != 0 || !strings.Contains(s.Confirmation(), spec) {
			t.Fatal("card bypassed reviewed update confirmation")
		}
		click("Cancel")
		if len(targets) != 0 {
			t.Fatal("cancelled update reached runtime")
		}
	}
	click("Update")
	click("Confirm and run")
	if len(targets) != 1 || targets[0].Spec != spec || targets[0].CurrentVersion != "1.0.0" || s.ValidateTarget(targets[0]) != nil {
		t.Fatal("confirmed update lost pinned receipt", targets)
	}
	saveMarketSnapshot(t, tt, "market-installed-update-800.png")
	tt.SetSize(400, 1000)
	if update, ok := tt.Find("Update"); !ok || update.X < 0 || update.X+update.W > 400 {
		t.Fatal("narrow update escaped card", update)
	}
	saveMarketSnapshot(t, tt, "market-installed-update-400.png")
	m.InstallBusy = true
	tt.Frame()
	click("Updating…")
	if len(targets) != 1 || s.Confirmation() != "" {
		t.Fatal("busy card queued duplicate update")
	}
	m.InstallBusy = false
	tt.Frame()
	click("Details")
	wait(func() bool { return !s.Discovery().DetailsLoading })
	detailsOnly = true
	tt.Frame()
	if _, ok := tt.Find("Refresh details"); !ok {
		t.Fatal("details header refresh absent")
	}
	click(item.Homepage)
	if opened != item.Homepage {
		t.Fatal("details homepage did not use explicit opener", opened)
	}
	click("Update")
	if !strings.Contains(s.Confirmation(), spec) || len(targets) != 1 {
		t.Fatal("details update bypassed confirmation")
	}
	click("Cancel")
	s.CloseDetails()
	detailsOnly = false
	tt.Frame()
	click("Uninstall")
	if len(removed) != 0 || !strings.Contains(s.Confirmation(), "fixture-plugin") {
		t.Fatal("uninstall bypassed confirmation")
	}
	click("Cancel")
	click("Uninstall")
	click("Confirm and run")
	if fmt.Sprint(removed) != "[fixture-plugin]" {
		t.Fatal("confirmed uninstall did not preserve exact package", removed)
	}
}

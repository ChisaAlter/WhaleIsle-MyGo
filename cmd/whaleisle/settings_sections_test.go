package main

import (
	"context"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/deeplink"
	desktopmarket "whaleisle/internal/desktop/market"
	marketstore "whaleisle/internal/store/market"
	"whaleisle/internal/ui/i18n"
	marketui "whaleisle/internal/ui/market"
	"whaleisle/internal/ui/settings"
	"whaleisle/internal/ui/theme"
)

type sectionCatalog struct{}

func (sectionCatalog) List(context.Context, bool, string) (desktopmarket.Catalog, error) {
	return desktopmarket.Catalog{Categories: []desktopmarket.Category{{ID: "all", Label: "All", Count: 2}}, Items: []desktopmarket.CatalogItem{
		{ID: "fixture/first", Repo: "First plugin", Owner: "fixture", Stars: 123, Description: strings.Repeat("A readable plugin description. ", 8)},
		{ID: "fixture/second", Repo: "Second plugin", Owner: "fixture", Description: "Another isolated catalog entry."},
	}}, nil
}
func (sectionCatalog) Details(context.Context, desktopmarket.CatalogItem, bool) (desktopmarket.Details, error) {
	return desktopmarket.Details{}, nil
}

func TestSettingsMarketUsesModalShellAndReturnsToGeneral(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	catalog := marketstore.NewWithPersonalPath(filepath.Join(t.TempDir(), "personal.json"))
	catalog.SetCatalogService(sectionCatalog{})
	changed := make(chan struct{}, 4)
	catalog.SetNotify(func() { changed <- struct{}{} })
	catalog.RefreshCatalog(false, i18n.Locale())
	for catalog.Discovery().Loading {
		select {
		case <-changed:
		case <-time.After(time.Second):
			t.Fatal("isolated catalog did not settle")
		}
	}
	state := mainState{}
	open := true
	selectedGeneral := 0
	tt := ui.NewTester(func(c *ui.Context) {
		applyAppearanceTheme(c, theme.Defaults(), theme.Resolve(theme.IDDeepseekLight).Light)
		ui.Text(c, "Preserved workspace")
		if !open {
			return
		}
		m := settings.Model{Desktop: &settings.DesktopAccess{}, DesktopActive: true,
			OnSelectDesktop: func() { selectedGeneral++ }, OnClose: func() { open = false },
			OnOpenMarket: func() { state.settingsEmbeddedSection = settings.SectionMarket },
		}
		settings.ModalView(c, state.settingsSectionModel(m, func(c *ui.Context) {
			marketui.View(c, marketui.Model{Store: catalog, Embedded: true, ThemeID: theme.IDDeepseekLight})
		}))
	}, 1000, 760)
	if err := tt.Click(i18n.T(i18n.KeySettingsSectionMarket)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("First plugin") || !tt.HasText(i18n.T(i18n.KeySettingsTitle)) {
		t.Fatal("market content or settings navigation disappeared")
	}
	card, ok := tt.Find("First plugin")
	saveSettingsSectionEvidence(t, tt, "settings-market-1000")
	if !ok || card.W <= 0 || card.Y < 24 || card.Y+card.H > 736 || card.X < 288 || card.X+card.W > 900 {
		t.Fatalf("embedded catalog escaped the settings content pane: %+v", card)
	}
	if err := tt.Click("History"); err != nil {
		t.Fatal(err)
	}
	if catalog.Discovery().Tab != "history" || state.settingsEmbeddedSection != settings.SectionMarket {
		t.Fatal("market tab selection escaped its owner")
	}
	tt.SetSize(600, 400)
	close, ok := tt.Find(i18n.T(i18n.KeySettingsBack))
	if !ok || close.X+close.W > 600 || close.Y+close.H > 400 {
		t.Fatalf("settings close inaccessible in narrow viewport: %+v", close)
	}
	saveSettingsSectionEvidence(t, tt, "settings-market-600")
	if err := tt.Click(i18n.T(i18n.KeySettingsSectionGeneral)); err != nil {
		t.Fatal(err)
	}
	if selectedGeneral != 1 || state.settingsEmbeddedSection != "" || !tt.HasText(i18n.T(i18n.KeySettingsDesktopWorkspace)) {
		t.Fatal("General did not restore the existing desktop controls")
	}
	tt.Key(0, ui.KeyEscape)
	if open || !tt.HasText("Preserved workspace") {
		t.Fatalf("settings dismissal lost the mounted workspace: open=%v texts=%v", open, tt.Texts())
	}
}

func TestSettingsExternalSelectionReplacesEmbeddedMarket(t *testing.T) {
	for _, target := range []deeplink.Target{{Action: deeplink.ActionSettings, Section: "models"}, {Action: deeplink.ActionAuth}} {
		t.Run(string(target.Action)+target.Section, func(t *testing.T) {
			s := mainState{settingsEmbeddedSection: settings.SectionMarket}
			s.applyDeeplink(target)
			m := s.settingsSectionModel(settings.Model{ModelsActive: s.settingsModelsActive, AccountActive: s.settingsAccountActive}, func(*ui.Context) {
				t.Fatal("external settings entry still paints Market")
			})
			if s.settingsEmbeddedSection != "" || m.SectionContent != nil || m.ActiveSection != "" || !s.settingsOpen {
				t.Fatal("external selection retained embedded Market")
			}
			if target.Action == deeplink.ActionAuth && !m.AccountActive || target.Section == "models" && !m.ModelsActive {
				t.Fatal("external selection lost its requested dedicated page")
			}
		})
	}
}

func TestPetSettingsEntryReplacesEmbeddedMarket(t *testing.T) {
	s := mainState{settingsEmbeddedSection: settings.SectionMarket}
	s.openPetSettings()
	m := s.settingsSectionModel(settings.Model{}, func(*ui.Context) { t.Fatal("Pet entry still paints Market") })
	if !s.settingsOpen || s.settingsEmbeddedSection != "" || m.SectionContent != nil {
		t.Fatal("Pet entry retained the embedded Market selection")
	}
}

func saveSettingsSectionEvidence(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("SECONDARY_SNAPSHOT_DIR")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, tt.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

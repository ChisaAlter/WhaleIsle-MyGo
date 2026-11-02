package launcher

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/product"
	"whaleisle/internal/ui/i18n"
)

func TestViewLauncherElectronParityLabels(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	ws := "C:\\ws"
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:      "deepseek",
			Version:      "0.1.0",
			VendorReady:  true,
			WorkspacePtr: &ws,
			ActiveTab:    TabHome,
		})
	}, 1060, 660)
	for _, label := range []string{
		product.Name,
		i18n.T(i18n.KeyLauncherBrandRole),
		i18n.T(i18n.KeyLauncherTabHome),
		i18n.T(i18n.KeyLauncherTabComponents),
		i18n.T(i18n.KeyLauncherTabVersions),
		i18n.T(i18n.KeyLauncherTabImport),
		i18n.T(i18n.KeyLauncherTabPlugins),
		i18n.T(i18n.KeyLauncherTabSettings),
		i18n.T(i18n.KeyLauncherHomeLede),
		i18n.T(i18n.KeyLauncherDesktopTitle),
		i18n.T(i18n.KeyLauncherStartDesktop),
		i18n.T(i18n.KeyLauncherCheckUpdate),
		"v0.1.0", // Electron home badge is the version pill
	} {
		if !tt.HasText(label) {
			t.Fatalf("missing %q; texts=%q", label, tt.Texts())
		}
	}
}

func TestViewLauncherRetryAfterFailedStart(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:     "deepseek",
			VendorReady: true,
			StartFailed: true,
			ActiveTab:   TabHome,
		})
	}, 1060, 660)
	if !tt.HasText(i18n.T(i18n.KeyLauncherRetryStart)) {
		t.Fatalf("missing retry; texts=%q", tt.Texts())
	}
	if tt.HasText(i18n.T(i18n.KeyLauncherStartDesktop)) {
		t.Fatalf("fresh start label should not show after failure; texts=%q", tt.Texts())
	}
}

func TestViewLauncherPluginsTab(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{
			ThemeID:         "deepseek",
			ActiveTab:       TabPlugins,
			SkipUserPlugins: true,
			DisabledPlugins: []string{"demo"},
		})
	}, 1060, 660)
	if !tt.HasText(i18n.T(i18n.KeyLauncherSkipUserPluginsOn)) {
		t.Fatalf("missing recovery on; texts=%q", tt.Texts())
	}
	if !tt.HasText("demo") {
		t.Fatalf("missing disabled plugin; texts=%q", tt.Texts())
	}
}

// TestHomeCardProductGeometry pins the reference home card's product row
// (launcher.html:68, launcher.css .home-desktop/.home-product/.home-eyebrow):
// rail 188 + hairline, stage padding 24, card padding 24, a 48px brand avatar,
// 12px gap, then the WHALE ISLE eyebrow with the 16/24 title 2px under it.
func TestHomeCardProductGeometry(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ThemeID: "deepseek", ActiveTab: TabHome, VendorReady: true, Version: "0.3.9", SystemDark: false})
	}, 1060, 660)
	// "WHALE ISLE" also labels the rail wordmark, so the card's product row is
	// measured through its unique title: without the avatar the text column
	// would start at 249 instead of 297.
	title, ok := tt.Find(i18n.T(i18n.KeyLauncherDesktopTitle))
	if !ok {
		t.Fatalf("home card title missing; texts=%q", tt.Texts())
	}
	// 189 (rail + hairline) + 24 (stage) + 24 (card) + 48 (avatar) + 12 (gap).
	const wantX = float32(297)
	if title.X < wantX-3 || title.X > wantX+3 {
		t.Fatalf("home card title x = %.1f, want about %.0f (48px avatar + 12px gap)", title.X, wantX)
	}
	// The product row is centred on the 48px avatar, so the text column starts
	// inside the card and the title's line height stays the 24px row.
	if title.H < 16 || title.H > 26 {
		t.Fatalf("home card title height = %.1f, want the 16/24 line", title.H)
	}
}

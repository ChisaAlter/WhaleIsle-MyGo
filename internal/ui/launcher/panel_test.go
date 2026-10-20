package launcher

import (
	"testing"

	"github.com/egoist/mygo/ui"

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
		"鲸",
		"屿",
		"MYGO鲸屿",
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

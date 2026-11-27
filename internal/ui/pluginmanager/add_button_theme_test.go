package pluginmanager

import (
	"testing"

	"github.com/egoist/mygo/ui"
	manager "whaleisle/internal/pluginmanager"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/i18n"
)

func TestAddPluginButtonFollowsDarkThemeTokensNotPureWhite(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(prev) })

	accent := ui.RGB(122, 170, 255)
	onAccent := ui.RGB(15, 17, 21)

	// Enabled state: uses theme primary accent, not blinding pure white.
	mReady := pageModel(t)
	mReady.ThemeID = "deepseek.dark"
	mReady.SystemDark = true
	ttReady := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		design.Bind(c, design.Dark, accent, onAccent)
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Children(func() { View(c, mReady) })
	}, 960, 640)
	addBtn, ok := ttReady.Find("Add plugin")
	if !ok {
		t.Fatalf("Add plugin button missing: %v", ttReady.Texts())
	}
	px := ttReady.Image().RGBAAt(int(addBtn.X+4), int(addBtn.Y+4))
	if px.R > 240 && px.G > 240 && px.B > 240 {
		t.Fatalf("enabled Add plugin button rendered pure white in dark mode: %+v", px)
	}
	if px.B <= px.R+20 {
		t.Fatalf("enabled Add plugin button did not follow blue primary accent in dark mode: %+v", px)
	}

	// Disabled (loading/disconnected) state: still distinct from dark background.
	mDis := Model{Controller: manager.New(), ThemeID: "deepseek.dark", SystemDark: true}
	ttDis := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		design.Bind(c, design.Dark, accent, onAccent)
		ui.Column(c).Fill().Background(ui.RGB(21, 21, 23)).Children(func() { View(c, mDis) })
	}, 960, 640)
	disBtn, ok := ttDis.Find("Add plugin")
	if !ok {
		t.Fatalf("disabled Add plugin button missing: %v", ttDis.Texts())
	}
	disPx := ttDis.Image().RGBAAt(int(disBtn.X+4), int(disBtn.Y+4))
	if disPx.R > 240 && disPx.G > 240 && disPx.B > 240 {
		t.Fatalf("disabled Add plugin button rendered pure white in dark mode: %+v", disPx)
	}
	if disPx.B < 100 {
		t.Fatalf("disabled Add plugin button collapsed to dark-on-dark: %+v", disPx)
	}
}

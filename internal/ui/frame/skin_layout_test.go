package frame

import (
	"bytes"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// ExternalNavigation requires a real NavigationShell; shared and standalone
// frames must composite the common canvas exactly once over the same backdrop.
func TestSkinCanvasOwnedOnceAcrossNavigationAndStandalone(t *testing.T) {
	old := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	t.Cleanup(func() { theme.SetPreferences(old); chrome.ConfigureBackdrop("", wallpapers.Defaults()) })
	p := wallpapers.Defaults()
	p.Effect, p.EffectExplicit, p.HideSidebarMask = "gradient", true, true
	chrome.ConfigureBackdrop("", p)
	store := session.New()
	store.Select("fixture")
	paint := func(external bool) []byte {
		sizes := DefaultLayout()
		sizes.Bottom, sizes.Right = 0, 0
		draft := ""
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.LightTheme())
			ui.Column(c).Fill().Background(ui.RGB(45, 90, 135)).Children(func() {
				m := Model{Store: store, Sizes: &sizes, Draft: &draft, ThemeID: theme.IDDeepseekLight, ExternalNavigation: external}
				if external {
					NavigationShell(c, m, func() { View(c, m) })
				} else {
					View(c, m)
				}
			})
		}, 1000, 700)
		tt.Frame()
		im := tt.Image()
		// Both a clear rail seat and a clear center seat must include one canvas.
		for _, point := range [][2]int{{4, 400}, {700, 90}} {
			got := im.RGBAAt(point[0], point[1])
			if got.R <= 45 || got.G <= 90 || got.B <= 135 {
				t.Fatalf("missing canvas at%v color%v", point, got)
			}
		}
		return im.Pix
	}
	if !bytes.Equal(paint(false), paint(true)) {
		t.Fatal("shared vs standalone frame canvas differs (missing or double fill)")
	}
}

func TestSkinFrameSelectedHalfUses720Axis(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old); i18n.SetLocale(i18n.DefaultLocale) })
	p := theme.Defaults()
	p.Light, p.Dark = "codex", "claude"
	theme.SetPreferences(p)
	i18n.SetLocale("en")
	for _, dark := range []bool{false, true} {
		sizes := DefaultLayout()
		sizes.Center, sizes.Bottom, sizes.Right = 1158, 0, 0
		store := session.New()
		right := float32(0)
		colors := theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Children(func() {
				mainRow(c, Model{Store: store, Sizes: &sizes, FilePaneWidth: &right, ThemeID: theme.IDSystem, SystemDark: dark}, session.ViewState{}, colors, chrome.Color(colors.Border), chrome.Color(colors.Muted))
			})
		}, 1440, 864)
		card, ok := tt.Find("Composer card")
		if !ok || card.W != 752 {
			t.Fatalf("dark%v source720axis+32card=%+v", dark, card)
		}
		_, mark := tt.Find("Conversation hero fish")
		_, badge := tt.Find(i18n.T(i18n.KeyChatHeroPreview))
		if dark && (!mark || !badge) || !dark && (mark || badge) {
			t.Fatalf("skin mark/badge displaydark%v mark%v badge%v", dark, mark, badge)
		}
	}
}

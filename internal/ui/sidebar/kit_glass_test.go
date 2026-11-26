package sidebar

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
)

func TestSidebarHoverAndSelectedInteractiveTokens(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	design.SetGlass(true, nil)
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	for _, tc := range []struct {
		dark       bool
		wantHover  ui.Color
		wantActive ui.Color
	}{
		{false, ui.RGBA(38, 49, 72, 0.06), ui.RGBA(38, 49, 72, 0.10)},
		{true, ui.RGBA(255, 255, 255, 0.08), ui.RGBA(255, 255, 255, 0.14)},
	} {
		tt := ui.NewTester(func(c *ui.Context) {
			if tc.dark {
				c.SetTheme(ui.DarkTheme())
			} else {
				c.SetTheme(ui.LightTheme())
			}
			toks := design.For(c)
			if got := sbHoverBg(toks); got != tc.wantHover {
				t.Fatalf("dark=%v sbHoverBg = %+v, want %+v", tc.dark, got, tc.wantHover)
			}
			if got := sbSelectedBg(toks); got != tc.wantActive {
				t.Fatalf("dark=%v sbSelectedBg = %+v, want %+v", tc.dark, got, tc.wantActive)
			}
		}, 100, 60)
		tt.Frame()
	}
}

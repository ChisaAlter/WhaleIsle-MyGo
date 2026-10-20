package settings

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

// TestSettingsModalDrawsItsElevationHairline pins --dsw-elevation-stroke: the
// panel paints a 0.5px hairline after its glass, so the edge stays visible where
// an opaque fill would otherwise cover the element's border.
func TestSettingsModalDrawsItsElevationHairline(t *testing.T) {
	dark := Model{Desktop: &DesktopAccess{}, DesktopActive: true, ThemeID: theme.IDDeepseekDark, SystemDark: true, OnClose: func() {}}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		ModalView(c, dark)
	}, 1400, 900)
	// The overlay fades in; a settled clock shows the paint it settles on.
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))
	// settingsModalSize(1400, 900) is 800×800, centered: (300, 50).
	image := tt.Image()
	const panelLeft, panelTop = 300, 50
	inside := modalPixel(image.RGBAAt(panelLeft+400, panelTop+8))
	edge := modalPixel(image.RGBAAt(panelLeft+400, panelTop))
	contrast := chrome.ContrastRatio(edge, inside)
	t.Logf("settings hairline %v on fill %v = %.2f:1", edge, inside, contrast)
	if contrast < 1.15 {
		t.Fatalf("settings modal hairline invisible in the dark: %.2f:1 (edge %v on fill %v)", contrast, edge, inside)
	}
	if edge.R <= inside.R {
		t.Fatalf("settings modal hairline is darker than its fill: %v on %v", edge, inside)
	}
	// The hairline belongs to the panel, not to a stray pixel: it must run along
	// the whole top edge.
	for _, x := range []int{panelLeft + 120, panelLeft + 400, panelLeft + 660} {
		got := modalPixel(image.RGBAAt(x, panelTop))
		if chrome.ContrastRatio(got, inside) < 1.15 {
			t.Fatalf("hairline missing at x=%d: %v on %v", x, got, inside)
		}
	}
}

// modalPixel reads a rendered pixel as a theme color.
func modalPixel(c color.RGBA) ui.Color { return ui.RGBA(c.R, c.G, c.B, float32(c.A)/255) }

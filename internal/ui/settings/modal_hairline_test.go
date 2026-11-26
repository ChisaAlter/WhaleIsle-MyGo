package settings

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
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

// TestSettingsModalGlassClarityAndSingleHairline pins the glass polish
// contract: dark/light scrim opacities match Electron (0.48 / 0.24), dark glass
// modals paint a single outer hairline with no duplicate inner rim at y+1, and
// the left rail shares the cohesive dark card surface without a white fog fill.
func TestSettingsModalGlassClarityAndSingleHairline(t *testing.T) {
	if got := modalScrimAlpha(true); got != 0.48 {
		t.Fatalf("dark scrim alpha = %v, want 0.48", got)
	}
	if got := modalScrimAlpha(false); got != 0.24 {
		t.Fatalf("light scrim alpha = %v, want 0.24", got)
	}

	design.SetGlass(true, func(s design.Scheme, p *design.Palette) {
		if s == design.Dark {
			p.Card = ui.RGBA(32, 32, 35, 0.80)
			p.Popover = ui.RGBA(48, 49, 54, 0.64)
			p.Border = ui.RGBA(255, 255, 255, 0.07)
			p.SidebarBorder = ui.RGBA(255, 255, 255, 0.055)
		}
	})
	t.Cleanup(func() { design.SetGlass(false, nil) })

	dark := Model{
		Desktop:       &DesktopAccess{},
		DesktopActive: true,
		ThemeID:       theme.IDDeepseekDark,
		SystemDark:    true,
		OnClose:       func() {},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		c.SetTheme(ui.DarkTheme())
		design.Bind(c, design.Dark, ui.Color{}, ui.Color{})
		ModalView(c, dark)
	}, 1400, 900)
	start := time.Unix(1000, 0)
	tt.SetTime(start)
	tt.SetTime(start.Add(time.Second))

	img := tt.Image()
	const panelLeft, panelTop = 260, 50 // settingsModalSize(1400, 900) is 880×800 -> (260, 50)
	edge := img.RGBAAt(panelLeft+440, panelTop)
	belowEdge := img.RGBAAt(panelLeft+440, panelTop+1)
	inside := img.RGBAAt(panelLeft+440, panelTop+8)
	if edge.R <= inside.R {
		t.Fatalf("dark glass outer hairline missing: edge=%v inside=%v", edge, inside)
	}
	// No duplicate inner rim stroke at y = panelTop + 1.
	if absDiff(belowEdge.R, inside.R) > 3 || absDiff(belowEdge.G, inside.G) > 3 || absDiff(belowEdge.B, inside.B) > 3 {
		t.Fatalf("unexpected inner rim at y=panelTop+1: belowEdge=%v inside=%v (edge=%v)", belowEdge, inside, edge)
	}
	// Rail background stays cohesive with the options surface (no white/4% fog).
	railFill := img.RGBAAt(panelLeft+12, panelTop+8)
	if absDiff(railFill.R, inside.R) > 3 || absDiff(railFill.G, inside.G) > 3 || absDiff(railFill.B, inside.B) > 3 {
		t.Fatalf("rail painted extra fog fill over modal surface: rail=%v options=%v", railFill, inside)
	}
}

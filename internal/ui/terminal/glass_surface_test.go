package terminal

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/store/hostterminal"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestTerminalPanelAndHostSurfaceAcrylicShowThrough(t *testing.T) {
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })

	prefs := wallpapers.Defaults()
	prefs.Effect = "none"
	prefs.Glass = 80
	prefs.Terminal = 50
	chrome.ConfigureBackdrop("", prefs)

	colors := theme.Resolve(theme.IDDeepseekDark).Active(true)
	desktopBlue := ui.RGB(20, 110, 240)
	baseBG := chrome.Color(colors.BG)

	t.Run("local-panel", func(t *testing.T) {
		p := &Panel{}
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			ui.Box(c).Fill().Background(desktopBlue).Children(func() {
				View(c, Model{Panel: p, ThemeID: theme.IDDeepseekDark, SystemDark: true})
			})
		}, 240, 140)
		tt.Frame()
		got := tt.Image().RGBAAt(120, 100)
		wantB := float64(baseBG.B)*0.5 + float64(desktopBlue.B)*0.5
		if math.Abs(float64(got.B)-wantB) > 4 {
			t.Fatalf("local terminal acrylic pixel = %+v, want B~%.0f (50%% show-through)", got, wantB)
		}
	})

	t.Run("host-surface", func(t *testing.T) {
		h := &HostState{}
		store := &hostterminal.Store{}
		v := hostLayoutView()
		tt := ui.NewTester(func(c *ui.Context) {
			c.SetTheme(ui.DarkTheme())
			ui.Box(c).Fill().Background(desktopBlue).Children(func() {
				hostSurface(c, h, store, "acrylic-session", colors, v, HostActions{})
			})
		}, 240, 140)
		tt.Frame()
		got := tt.Image().RGBAAt(120, 100)
		wantB := float64(baseBG.B)*0.5 + float64(desktopBlue.B)*0.5
		if math.Abs(float64(got.B)-wantB) > 4 {
			t.Fatalf("host terminal acrylic pixel = %+v, want B~%.0f (50%% show-through)", got, wantB)
		}
	})
}

func TestTerminalPanelAndHostSurfaceGradientBlur(t *testing.T) {
	t.Cleanup(func() { chrome.ConfigureBackdrop("", wallpapers.Defaults()) })

	prefs := wallpapers.Defaults()
	prefs.Effect = "gradient"
	prefs.EffectExplicit = true
	prefs.Preset = "aurora"
	prefs.Terminal = 55

	paintStripes := func(p *ui.Painter, r ui.Rect) {
		c1 := ui.RGB(20, 120, 240)
		c2 := ui.RGB(240, 120, 20)
		for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
			col := c1
			if i%2 == 1 {
				col = c2
			}
			p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, col, 0)
		}
	}

	colors := theme.Resolve(theme.IDDeepseekDark).Active(true)
	channelSpan := func(tt *ui.Tester, y, x0, x1 int) int {
		img := tt.Image()
		loR, hiR := 255, 0
		loB, hiB := 255, 0
		for x := x0; x < x1; x++ {
			p := img.RGBAAt(x, y)
			loR, hiR = min(loR, int(p.R)), max(hiR, int(p.R))
			loB, hiB = min(loB, int(p.B)), max(hiB, int(p.B))
		}
		return max(hiR-loR, hiB-loB)
	}

	for _, name := range []string{"local-panel", "host-surface"} {
		t.Run(name, func(t *testing.T) {
			p := &Panel{}
			h := &HostState{}
			store := &hostterminal.Store{}
			v := hostLayoutView()
			tt := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(ui.DarkTheme())
				paint := chrome.WallpaperBackgroundPainter("", prefs, true)
				ui.Box(c).Fill().Draw(func(painter *ui.Painter, r ui.Rect) {
					paint(painter, r)
					paintStripes(painter, r)
				}).Children(func() {
					ui.Box(c).Absolute().Left(0).Top(40).Width(240).Height(160).Children(func() {
						if name == "local-panel" {
							View(c, Model{Panel: p, ThemeID: theme.IDDeepseekDark, SystemDark: true})
						} else {
							hostSurface(c, h, store, "grad-session", colors, v, HostActions{})
						}
					})
				})
			}, 240, 200)
			tt.Frame()
			rawSpan := channelSpan(tt, 10, 20, 220)
			blurredSpan := channelSpan(tt, 180, 40, 200)
			if rawSpan < 150 {
				t.Fatalf("expected vivid stripes outside terminal, got %d", rawSpan)
			}
			if blurredSpan > rawSpan/3 {
				t.Fatalf("%s did not blur gradient backdrop: blurredSpan=%d rawSpan=%d", name, blurredSpan, rawSpan)
			}
		})
	}
}

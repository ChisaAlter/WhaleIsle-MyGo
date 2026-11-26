package chat

import (
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/widgets/markdown"
)

func TestMarkdownInlineCodeAndDockSurfaceBackdrop(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	for _, dark := range []bool{false, true} {
		tt := ui.NewTester(func(c *ui.Context) {
			if dark {
				c.SetTheme(ui.DarkTheme())
			} else {
				c.SetTheme(ui.LightTheme())
			}
			st := transcriptMarkdownStyleFor(c, 14)
			want := markdown.InlineCodeBase(dark)
			if st.InlineCodeBG != want {
				t.Fatalf("dark=%v InlineCodeBG = %v, want %v", dark, st.InlineCodeBG, want)
			}
		}, 120, 80)
		tt.Frame()
	}

	// Verify chatCardSurface (used by TodoPanel / GoalDock / QueueDock) stays frosted
	// even when BackdropActive() is true without design.GlassEnabled().
	design.SetGlass(false, nil)
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	var rect ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for i, x := 0, float32(0); x < r.W; i, x = i+1, x+4 {
				col := color.RGBA{R: 20, G: 110, B: 240, A: 255}
				if i%2 == 1 {
					col = color.RGBA{R: 245, G: 130, B: 30, A: 255}
				}
				p.Fill(ui.Rect{X: r.X + x, Y: r.Y, W: 4, H: r.H}, ui.RGB(col.R, col.G, col.B), 0)
			}
		}).Children(func() {
			rect = chatCardSurface(c, ui.Box(c).Absolute().Left(20).Top(20).Width(120).Height(60), 12).Bounds()
		})
	}, 180, 120)
	tt.Frame()
	img := tt.Image()
	loR, hiR := 255, 0
	for x := int(rect.X) + 16; x < int(rect.X+rect.W)-16; x++ {
		r := int(img.RGBAAt(x, int(rect.Y+rect.H/2)).R)
		loR, hiR = min(loR, r), max(hiR, r)
	}
	if hiR-loR > 40 {
		t.Fatalf("chatCardSurface did not blur stripes under BackdropActive: span=%d", hiR-loR)
	}
}

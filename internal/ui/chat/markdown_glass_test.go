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

	// Verify chatCardSurface (used by TodoPanel / GoalDock / QueueDock) and CodeSurface
	// use clean translucent fills without GPU backdrop blur under BackdropActive(),
	// while chatPopupSurface retains frosted backdrop blur.
	design.SetGlass(false, nil)
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	var cardRect, codeRect, popupRect ui.Rect
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
			cardRect = chatCardSurface(c, ui.Box(c).Absolute().Left(10).Top(10).Width(100).Height(40), 12).Bounds()
			st := transcriptMarkdownStyleFor(c, 14)
			codeRect = st.CodeSurface(ui.Box(c).Absolute().Left(10).Top(60).Width(100).Height(40), 10).Bounds()
			popupRect = chatPopupSurface(c, ui.Box(c).Absolute().Left(120).Top(10).Width(100).Height(90), 12).Bounds()
		})
	}, 240, 120)
	tt.Frame()
	img := tt.Image()
	spanR := func(rect ui.Rect) int {
		loR, hiR := 255, 0
		for x := int(rect.X) + 12; x < int(rect.X+rect.W)-12; x++ {
			r := int(img.RGBAAt(x, int(rect.Y+rect.H/2)).R)
			loR, hiR = min(loR, r), max(hiR, r)
		}
		return hiR - loR
	}
	rawSpan := spanR(ui.Rect{X: 0, Y: 108, W: 220, H: 10})
	if rawSpan < 150 {
		t.Fatalf("expected crisp raw stripe span >= 150, got %d", rawSpan)
	}
	if s := spanR(cardRect); s == 0 || s >= rawSpan {
		t.Fatalf("chatCardSurface should stay translucent without blur under BackdropActive: span=%d raw=%d", s, rawSpan)
	}
	if s := spanR(codeRect); s == 0 || s >= rawSpan {
		t.Fatalf("CodeSurface should stay translucent without blur under BackdropActive: span=%d raw=%d", s, rawSpan)
	}
	if s := spanR(popupRect); s > 40 {
		t.Fatalf("chatPopupSurface did not blur stripes under BackdropActive: span=%d", s)
	}
}

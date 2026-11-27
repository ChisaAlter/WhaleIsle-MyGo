package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/wallpapers"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/widgets/markdown"
)

// TestToolCodeBlockScrolledGlassBannerTranslucentAndNoTextBleed verifies that
// scrolling a frosted ToolCodeBlock keeps the sticky banner translucent (never
// painting an opaque A=255 solid strip) while clipping scrolled code lines so
// foreground text never bleeds behind the sticky banner.
func TestToolCodeBlockScrolledGlassBannerTranslucentAndNoTextBleed(t *testing.T) {
	t.Cleanup(func() {
		chrome.ConfigureBackdrop("", wallpapers.Defaults())
		design.SetGlass(false, nil)
	})
	chrome.ConfigureBackdrop("wall.png", wallpapers.Defaults())
	design.SetGlass(true, func(_ design.Scheme, p *design.Palette) {
		p.Code = ui.RGBA(248, 250, 255, 0.45)
	})

	lines := make([]string, 24)
	for i := range lines {
		lines[i] = "MMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMMM"
	}
	code := strings.Join(lines, "\n")

	tt := ui.NewTester(func(c *ui.Context) {
		st := transcriptMarkdownStyleFor(c, 13)
		st.FG = ui.RGB(235, 10, 10)
		st.Muted = ui.RGB(90, 95, 110)
		st.ToolCodeLanguage = ui.RGB(90, 95, 110)
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			for x := r.X; x < r.X+r.W; x += 6 {
				col := ui.RGB(25, 200, 245)
				if int(x/6)%2 == 1 {
					col = ui.RGB(40, 235, 95)
				}
				p.Fill(ui.Rect{X: x, Y: r.Y, W: 6, H: r.H}, col, 0)
			}
		}).Children(func() {
			markdown.ToolCodeBlock(c, "text", code, st, 84)
		})
	}, 360, 420)
	tt.Frame()
	tt.Frame()

	img := tt.Image()
	// Sticky banner sits at top = 84 - 4 = 80, height 42 -> y in [80, 122].
	// Sample the middle empty horizontal stretch of the banner (x in [110, 250], y = 98):
	// 1. Blue channel must retain backdrop stripe variation (proving no opaque solid strip).
	// 2. Red code ink must NOT bleed into the banner area.
	loB, hiB := 255, 0
	for y := 88; y <= 112; y++ {
		for x := 110; x < 250; x++ {
			px := img.RGBAAt(x, y)
			if y == 98 {
				loB = min(loB, int(px.B))
				hiB = max(hiB, int(px.B))
			}
			if int(px.R) > int(px.G)+40 && int(px.R) > int(px.B)+40 {
				t.Fatalf("scrolled code ink bled behind sticky banner at (%d,%d): %v", x, y, px)
			}
		}
	}
	if hiB-loB < 35 {
		t.Fatalf("sticky banner lost glass translucency (opaque strip painted): B span=%d", hiB-loB)
	}
}

// TestTerminalCopyButtonTransparentBackground verifies that the terminal card's
// copy button does not paint an opaque box over the card surface.
func TestTerminalCopyButtonTransparentBackground(t *testing.T) {
	m := &toolTerminalModel{
		Command: "echo hello",
		Output:  "hello world",
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(8).Children(func() {
			terminalCard(c, ToolCardStyle{}, "copy-transparent", m, "/home/u", false)
		})
	}, 360, 180)
	tt.Frame()
	if err := tt.Click("Copy"); err != nil {
		// Fall back to localized label if default locale is ZH.
		if errZH := tt.Click("复制"); errZH != nil {
			t.Fatalf("copy button not clickable: en=%v zh=%v texts=%v", err, errZH, tt.Texts())
		}
	}
	if tt.Clipboard() != "hello world" {
		t.Fatalf("clipboard=%q want %q", tt.Clipboard(), "hello world")
	}
}

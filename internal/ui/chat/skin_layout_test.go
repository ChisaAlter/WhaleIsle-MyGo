package chat

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/theme"
)

func TestSkinUserBubbleUsesInstalledFourCornerRadiiAndSurface(t *testing.T) {
	for _, dark := range []bool{false, true} {
		bg := ui.RGB(0, 0, 0)
		scheme := design.Light
		if dark {
			scheme = design.Dark
		}
		body := "A coss chat bubble"
		tt := ui.NewTester(func(c *ui.Context) {
			if dark {
				c.SetTheme(ui.DarkTheme())
			}
			ui.Column(c).FillWidth().Padding(4).Background(bg).Children(func() {
				userMessageView(c, Model{ThemeID: theme.IDSystem, SystemDark: dark, ContentWidth: 720}, presentationMessage(body), theme.Colors{}, nil, body, nil)
			})
		}, 800, 100)
		r, ok := tt.Find("User message bubble")
		if !ok {
			t.Fatal("bubble missing")
		}
		im := tt.Image()
		fill := design.Base(scheme).Foreground.Alpha(0.12).Over(bg)
		center := im.RGBAAt(int(r.X+r.W/2), int(r.Y+3))
		if center.R != fill.R || center.G != fill.G || center.B != fill.B {
			t.Fatalf("surface=%v want%v", center, fill)
		}
		// Outgoing coss bubble squares top-right to RadiusMD (8) and keeps RadiusXL2 (16) elsewhere.
		topLeft := im.RGBAAt(int(r.X+2), int(r.Y+2))
		topRight := im.RGBAAt(int(r.X+r.W-4), int(r.Y+4))
		if topLeft == center || topRight != center {
			t.Fatalf("outgoing coss corner radii mismatch: topLeft=%v topRight=%v center=%v", topLeft, topRight, center)
		}
	}
}
